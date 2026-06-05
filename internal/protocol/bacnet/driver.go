package bacnet

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/indugate/gateway/internal/protocol"
)

var (
	ErrNotConnected         = errors.New("bacnet client not connected")
	ErrAlreadyConnected     = errors.New("bacnet client already connected")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type Driver struct {
	address string
	cfg     *Config

	mu            sync.RWMutex
	client        *UDPClient
	connected     bool
	invokeID      uint32
	subscriptions map[string]*subscription
	subsMu        sync.RWMutex
}

func NewDriver(address string, cfg *Config) *Driver {
	if cfg == nil {
		cfg = &Config{DeviceID: 1, TimeoutMS: 3000}
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

func (d *Driver) Connect(_ context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.connected {
		return ErrAlreadyConnected
	}
	host, port, err := parseTarget(d.address)
	if err != nil {
		return err
	}
	client, err := NewUDPClient(host, port)
	if err != nil {
		return err
	}
	d.client = client
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
	if d.client != nil {
		_ = d.client.Close()
		d.client = nil
	}
	d.connected = false
	return nil
}

func (d *Driver) Read(ctx context.Context, nodeIDStr string) (*protocol.DataValue, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	obj, err := ParseObjectRef(nodeIDStr)
	if err != nil {
		return nil, err
	}

	readCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	type result struct {
		value any
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		d.mu.RLock()
		client := d.client
		d.mu.RUnlock()
		val, err := client.ReadProperty(obj, d.cfg.RequestTimeout(), &d.invokeID)
		ch <- result{value: val, err: err}
	}()

	select {
	case <-readCtx.Done():
		return nil, fmt.Errorf("read bacnet property: %w", readCtx.Err())
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return &protocol.DataValue{
			NodeID:    obj.String(),
			Value:     res.value,
			DataType:  dataTypeFor(obj),
			Status:    "OK",
			Timestamp: time.Now(),
		}, nil
	}
}

func (d *Driver) Write(ctx context.Context, nodeIDStr string, value any) error {
	if !d.IsConnected() {
		return ErrNotConnected
	}
	obj, err := ParseObjectRef(nodeIDStr)
	if err != nil {
		return err
	}
	if obj.Type == 3 {
		return fmt.Errorf("binaryInput is read-only")
	}

	writeCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		d.mu.RLock()
		client := d.client
		d.mu.RUnlock()
		errCh <- client.WriteProperty(obj, value, d.cfg.RequestTimeout(), &d.invokeID)
	}()

	select {
	case <-writeCtx.Done():
		return fmt.Errorf("write bacnet property: %w", writeCtx.Err())
	case err := <-errCh:
		return err
	}
}
