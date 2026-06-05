package s7

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/indugate/gateway/internal/protocol"
	"github.com/robinson/gos7"
)

var (
	ErrNotConnected         = errors.New("s7 client not connected")
	ErrAlreadyConnected     = errors.New("s7 client already connected")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type Driver struct {
	address string
	cfg     *Config

	mu            sync.RWMutex
	handler       *gos7.TCPClientHandler
	client        gos7.Client
	helper        gos7.Helper
	connected     bool
	subscriptions map[string]*subscription
	subsMu        sync.RWMutex
}

func NewDriver(address string, cfg *Config) *Driver {
	if cfg == nil {
		cfg = &Config{Rack: 0, Slot: 1, TimeoutMS: 5000}
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

	host := parseHost(d.address)
	handler := gos7.NewTCPClientHandler(host, d.cfg.Rack, d.cfg.Slot)
	handler.Timeout = d.cfg.RequestTimeout()
	handler.IdleTimeout = d.cfg.RequestTimeout()

	connectCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- handler.Connect()
	}()

	select {
	case <-connectCtx.Done():
		return fmt.Errorf("connect s7 plc: %w", connectCtx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("connect s7 plc: %w", err)
		}
	}

	d.handler = handler
	d.client = gos7.NewClient(handler)
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
		value, err := d.readAddress(addr)
		ch <- readResult{value: value, err: err}
	}()

	var value any
	select {
	case <-readCtx.Done():
		return nil, fmt.Errorf("read s7 address: %w", readCtx.Err())
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

func (d *Driver) readAddress(addr Address) (any, error) {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	buf := make([]byte, addr.ReadSize())
	if err := d.readRaw(client, addr, buf); err != nil {
		return nil, err
	}
	return decodeValue(addr, buf, &d.helper)
}

func (d *Driver) readRaw(client gos7.Client, addr Address, buf []byte) error {
	switch addr.Area {
	case AreaDB:
		if err := client.AGReadDB(addr.DBNumber, addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("read db: %w", err)
		}
		return nil
	case AreaM:
		if err := client.AGReadMB(addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("read merker: %w", err)
		}
		return nil
	case AreaI:
		if err := client.AGReadEB(addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		return nil
	case AreaQ:
		if err := client.AGReadAB(addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("read output: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported area")
	}
}

func (d *Driver) writeRaw(client gos7.Client, addr Address, buf []byte) error {
	switch addr.Area {
	case AreaDB:
		if err := client.AGWriteDB(addr.DBNumber, addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("write db: %w", err)
		}
		return nil
	case AreaM:
		if err := client.AGWriteMB(addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("write merker: %w", err)
		}
		return nil
	case AreaI:
		return fmt.Errorf("input area is read-only")
	case AreaQ:
		if err := client.AGWriteAB(addr.Offset, len(buf), buf); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported area")
	}
}

func decodeValue(addr Address, buf []byte, helper *gos7.Helper) (any, error) {
	switch addr.Kind {
	case KindBool:
		return (buf[0]>>uint(addr.Bit))&1 == 1, nil
	case KindByte:
		return buf[0], nil
	case KindInt16:
		var v int16
		helper.GetValueAt(buf, 0, &v)
		return v, nil
	case KindInt32:
		var v int32
		helper.GetValueAt(buf, 0, &v)
		return v, nil
	case KindReal:
		var v float32
		helper.GetValueAt(buf, 0, &v)
		return float64(v), nil
	default:
		var v uint16
		helper.GetValueAt(buf, 0, &v)
		return v, nil
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
	if !addr.Writable() {
		return fmt.Errorf("address %s is read-only", addr.String())
	}

	writeCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.writeAddress(addr, value)
	}()

	select {
	case <-writeCtx.Done():
		return fmt.Errorf("write s7 address: %w", writeCtx.Err())
	case err := <-errCh:
		return err
	}
}

func (d *Driver) writeAddress(addr Address, value any) error {
	d.mu.RLock()
	client := d.client
	helper := d.helper
	d.mu.RUnlock()

	if addr.Kind == KindBool {
		buf := make([]byte, 1)
		if err := d.readRaw(client, addr, buf); err != nil {
			return err
		}
		b, err := toBool(value)
		if err != nil {
			return err
		}
		mask := byte(1 << addr.Bit)
		if b {
			buf[0] |= mask
		} else {
			buf[0] &^= mask
		}
		return d.writeRaw(client, addr, buf)
	}

	buf := make([]byte, addr.ReadSize())
	if err := encodeValue(addr, value, buf, &helper); err != nil {
		return err
	}
	return d.writeRaw(client, addr, buf)
}

func encodeValue(addr Address, value any, buf []byte, helper *gos7.Helper) error {
	switch addr.Kind {
	case KindByte:
		v, err := toByte(value)
		if err != nil {
			return err
		}
		buf[0] = v
		return nil
	case KindInt16:
		v, err := toInt16(value)
		if err != nil {
			return err
		}
		helper.SetValueAt(buf, 0, v)
		return nil
	case KindInt32:
		v, err := toInt32(value)
		if err != nil {
			return err
		}
		helper.SetValueAt(buf, 0, v)
		return nil
	case KindReal:
		v, err := toFloat32(value)
		if err != nil {
			return err
		}
		helper.SetValueAt(buf, 0, v)
		return nil
	default:
		v, err := toUint16(value)
		if err != nil {
			return err
		}
		helper.SetValueAt(buf, 0, v)
		return nil
	}
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	case string:
		switch v {
		case "true", "1", "on":
			return true, nil
		case "false", "0", "off":
			return false, nil
		}
	}
	return false, fmt.Errorf("invalid bool value %T", value)
}

func toByte(value any) (byte, error) {
	switch v := value.(type) {
	case byte:
		return v, nil
	case int:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value out of range")
		}
		return byte(v), nil
	case float64:
		if v < 0 || v > 255 {
			return 0, fmt.Errorf("value out of range")
		}
		return byte(v), nil
	default:
		return 0, fmt.Errorf("invalid byte value %T", value)
	}
}

func toInt16(value any) (int16, error) {
	switch v := value.(type) {
	case int16:
		return v, nil
	case int:
		return int16(v), nil
	case float64:
		return int16(v), nil
	default:
		return 0, fmt.Errorf("invalid int16 value %T", value)
	}
}

func toInt32(value any) (int32, error) {
	switch v := value.(type) {
	case int32:
		return v, nil
	case int:
		return int32(v), nil
	case float64:
		return int32(v), nil
	default:
		return 0, fmt.Errorf("invalid int32 value %T", value)
	}
}

func toFloat32(value any) (float32, error) {
	switch v := value.(type) {
	case float32:
		return v, nil
	case float64:
		return float32(v), nil
	case int:
		return float32(v), nil
	default:
		return 0, fmt.Errorf("invalid float value %T", value)
	}
}

func toUint16(value any) (uint16, error) {
	switch v := value.(type) {
	case uint16:
		return v, nil
	case int:
		if v < 0 || v > math.MaxUint16 {
			return 0, fmt.Errorf("value out of range")
		}
		return uint16(v), nil
	case float64:
		if v < 0 || v > math.MaxUint16 {
			return 0, fmt.Errorf("value out of range")
		}
		return uint16(v), nil
	default:
		return 0, fmt.Errorf("invalid uint16 value %T", value)
	}
}
