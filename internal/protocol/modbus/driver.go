package modbus

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/goburrow/modbus"
	"github.com/indugate/gateway/internal/protocol"
)

var (
	ErrNotConnected         = errors.New("modbus client not connected")
	ErrAlreadyConnected     = errors.New("modbus client already connected")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type Driver struct {
	address string
	cfg     *Config

	mu            sync.RWMutex
	handler       *modbus.TCPClientHandler
	client        modbus.Client
	connected     bool
	subscriptions map[string]*subscription
	subsMu        sync.RWMutex
}

func NewDriver(address string, cfg *Config) *Driver {
	if cfg == nil {
		cfg = &Config{UnitID: 1, TimeoutMS: 3000}
	}
	return &Driver{
		address:       address,
		cfg:           cfg,
		subscriptions: make(map[string]*subscription),
	}
}

func (d *Driver) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

func (d *Driver) Connect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return ErrAlreadyConnected
	}

	handler := modbus.NewTCPClientHandler(d.address)
	handler.Timeout = d.cfg.RequestTimeout()
	handler.SlaveId = d.cfg.UnitID

	connectCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- handler.Connect()
	}()

	select {
	case <-connectCtx.Done():
		return fmt.Errorf("connect modbus server: %w", connectCtx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("connect modbus server: %w", err)
		}
	}

	d.handler = handler
	d.client = modbus.NewClient(handler)
	d.connected = true
	return nil
}

func (d *Driver) Disconnect(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	d.subsMu.Lock()
	for id, sub := range d.subscriptions {
		sub.cancel()
		delete(d.subscriptions, id)
	}
	d.subsMu.Unlock()

	if d.handler != nil {
		_ = d.handler.Close()
		d.handler = nil
	}
	d.client = nil
	d.connected = false
	return nil
}

func (d *Driver) Read(ctx context.Context, nodeIDStr string) (*protocol.DataValue, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	addr, err := ParseAddress(nodeIDStr)
	if err != nil {
		return nil, err
	}

	readCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	type readResult struct {
		value any
		err   error
	}
	ch := make(chan readResult, 1)
	go func() {
		value, err := d.readRegister(addr)
		ch <- readResult{value: value, err: err}
	}()

	var value any
	select {
	case <-readCtx.Done():
		return nil, fmt.Errorf("read register: %w", readCtx.Err())
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		value = res.value
	}

	return &protocol.DataValue{
		NodeID:    addr.String(),
		Value:     value,
		DataType:  dataTypeFor(addr),
		Status:    "OK",
		Timestamp: time.Now(),
	}, nil
}

func (d *Driver) readRegister(addr Address) (any, error) {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	switch addr.Type {
	case RegisterCoil:
		results, err := client.ReadCoils(addr.Addr, 1)
		if err != nil {
			return nil, fmt.Errorf("read coil: %w", err)
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("empty coil response")
		}
		return (results[0] & 0x01) == 1, nil
	case RegisterDiscrete:
		results, err := client.ReadDiscreteInputs(addr.Addr, 1)
		if err != nil {
			return nil, fmt.Errorf("read discrete input: %w", err)
		}
		if len(results) == 0 {
			return nil, fmt.Errorf("empty discrete response")
		}
		return (results[0] & 0x01) == 1, nil
	case RegisterInput:
		results, err := client.ReadInputRegisters(addr.Addr, 1)
		if err != nil {
			return nil, fmt.Errorf("read input register: %w", err)
		}
		if len(results) < 2 {
			return nil, fmt.Errorf("empty input register response")
		}
		return bytesToUint16(results), nil
	case RegisterHolding:
		results, err := client.ReadHoldingRegisters(addr.Addr, 1)
		if err != nil {
			return nil, fmt.Errorf("read holding register: %w", err)
		}
		if len(results) < 2 {
			return nil, fmt.Errorf("empty holding register response")
		}
		return bytesToUint16(results), nil
	default:
		return nil, fmt.Errorf("unsupported register type")
	}
}

func (d *Driver) Write(ctx context.Context, nodeIDStr string, value any) error {
	if !d.IsConnected() {
		return ErrNotConnected
	}

	addr, err := ParseAddress(nodeIDStr)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.writeRegister(addr, value)
	}()

	select {
	case <-writeCtx.Done():
		return fmt.Errorf("write register: %w", writeCtx.Err())
	case err := <-errCh:
		return err
	}
}

func (d *Driver) writeRegister(addr Address, value any) error {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	switch addr.Type {
	case RegisterCoil:
		if values, ok := value.([]any); ok {
			return d.writeMultipleCoils(client, addr.Addr, values)
		}
		v, err := toBool(value)
		if err != nil {
			return err
		}
		var coilValue uint16
		if v {
			coilValue = 0xFF00
		}
		_, err = client.WriteSingleCoil(addr.Addr, coilValue)
		if err != nil {
			return fmt.Errorf("write coil: %w", err)
		}
		return nil
	case RegisterHolding:
		if values, ok := value.([]any); ok {
			return d.writeMultipleRegisters(client, addr.Addr, values)
		}
		v, err := toUint16(value)
		if err != nil {
			return err
		}
		_, err = client.WriteSingleRegister(addr.Addr, v)
		if err != nil {
			return fmt.Errorf("write holding register: %w", err)
		}
		return nil
	case RegisterDiscrete, RegisterInput:
		return fmt.Errorf("register %s is read-only", addr.String())
	default:
		return fmt.Errorf("unsupported register type")
	}
}

func (d *Driver) writeMultipleCoils(client modbus.Client, startAddr uint16, values []any) error {
	if len(values) == 0 {
		return fmt.Errorf("empty coil values")
	}
	bytes := make([]byte, (len(values)+7)/8)
	for i, v := range values {
		b, err := toBool(v)
		if err != nil {
			return fmt.Errorf("coil[%d]: %w", i, err)
		}
		if b {
			bytes[i/8] |= 1 << (i % 8)
		}
	}
	_, err := client.WriteMultipleCoils(startAddr, uint16(len(values)), bytes)
	if err != nil {
		return fmt.Errorf("write multiple coils: %w", err)
	}
	return nil
}

func (d *Driver) writeMultipleRegisters(client modbus.Client, startAddr uint16, values []any) error {
	if len(values) == 0 {
		return fmt.Errorf("empty register values")
	}
	payload := make([]byte, len(values)*2)
	for i, v := range values {
		n, err := toUint16(v)
		if err != nil {
			return fmt.Errorf("register[%d]: %w", i, err)
		}
		payload[i*2] = byte(n >> 8)
		payload[i*2+1] = byte(n)
	}
	_, err := client.WriteMultipleRegisters(startAddr, uint16(len(values)), payload)
	if err != nil {
		return fmt.Errorf("write multiple registers: %w", err)
	}
	return nil
}

func bytesToUint16(data []byte) uint16 {
	return uint16(data[0])<<8 | uint16(data[1])
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case string:
		switch v {
		case "true", "1", "on":
			return true, nil
		case "false", "0", "off":
			return false, nil
		default:
			return false, fmt.Errorf("invalid bool value %q", v)
		}
	default:
		return false, fmt.Errorf("invalid bool value %T", value)
	}
}

func toUint16(value any) (uint16, error) {
	switch v := value.(type) {
	case uint16:
		return v, nil
	case int:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value out of range: %d", v)
		}
		return uint16(v), nil
	case int64:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value out of range: %d", v)
		}
		return uint16(v), nil
	case float64:
		if v < 0 || v > 65535 {
			return 0, fmt.Errorf("value out of range: %v", v)
		}
		return uint16(v), nil
	default:
		return 0, fmt.Errorf("invalid uint16 value %T", value)
	}
}
