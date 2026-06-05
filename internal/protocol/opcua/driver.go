package opcua

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/indugate/gateway/internal/protocol"
)

var (
	ErrNotConnected          = errors.New("opc ua client not connected")
	ErrAlreadyConnected      = errors.New("opc ua client already connected")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
)

type Driver struct {
	endpoint string
	cfg      *Config

	mu            sync.RWMutex
	client        *opcua.Client
	connected     bool
	subscriptions map[string]*subscription
	subsMu        sync.RWMutex
}

func NewDriver(endpoint string, cfg *Config) *Driver {
	if cfg == nil {
		cfg = &Config{
			SecurityPolicy:   "None",
			SecurityMode:     "None",
			RequestTimeoutMS: 5000,
		}
	}
	return &Driver{
		endpoint:      endpoint,
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

	opts, err := d.clientOptions(ctx)
	if err != nil {
		return err
	}

	client, err := opcua.NewClient(d.endpoint, opts...)
	if err != nil {
		return fmt.Errorf("create opc ua client: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	if err := client.Connect(connectCtx); err != nil {
		return fmt.Errorf("connect opc ua server: %w", err)
	}

	d.client = client
	d.connected = true
	return nil
}

func (d *Driver) Disconnect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.connected {
		return nil
	}

	d.subsMu.Lock()
	for id, sub := range d.subscriptions {
		sub.cancel()
		if sub.sub != nil {
			_ = sub.sub.Cancel(ctx)
		}
		delete(d.subscriptions, id)
	}
	d.subsMu.Unlock()

	if d.client != nil {
		_ = d.client.Close(ctx)
		d.client = nil
	}
	d.connected = false
	return nil
}

func (d *Driver) Read(ctx context.Context, nodeIDStr string) (*protocol.DataValue, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid node id %q: %w", nodeIDStr, err)
	}

	readCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	req := &ua.ReadRequest{
		MaxAge: 2000,
		NodesToRead: []*ua.ReadValueID{
			{NodeID: nodeID, AttributeID: ua.AttributeIDValue},
		},
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	resp, err := d.client.Read(readCtx, req)
	if err != nil {
		return nil, fmt.Errorf("read node: %w", err)
	}
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("empty read response")
	}
	if resp.Results[0].Status != ua.StatusOK {
		return nil, fmt.Errorf("read failed: %s", resp.Results[0].Status)
	}

	result := resp.Results[0]
	value := &protocol.DataValue{
		NodeID:    nodeIDStr,
		Status:    result.Status.Error(),
		Timestamp: time.Now(),
	}
	if result.Value != nil {
		value.Value = result.Value.Value()
	}
	if !result.SourceTimestamp.IsZero() {
		value.Timestamp = result.SourceTimestamp
	}

	return value, nil
}

func (d *Driver) Write(ctx context.Context, nodeIDStr string, value any) error {
	if !d.IsConnected() {
		return ErrNotConnected
	}

	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return fmt.Errorf("invalid node id %q: %w", nodeIDStr, err)
	}

	variant, err := ua.NewVariant(value)
	if err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	req := &ua.WriteRequest{
		NodesToWrite: []*ua.WriteValue{
			{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
				Value: &ua.DataValue{
					EncodingMask: ua.DataValueValue,
					Value:        variant,
				},
			},
		},
	}

	resp, err := d.client.Write(writeCtx, req)
	if err != nil {
		return fmt.Errorf("write node: %w", err)
	}
	if len(resp.Results) == 0 {
		return fmt.Errorf("empty write response")
	}
	if resp.Results[0] != ua.StatusOK {
		return fmt.Errorf("write failed: %s", resp.Results[0])
	}
	return nil
}

func (d *Driver) clientOptions(ctx context.Context) ([]opcua.Option, error) {
	policy := d.cfg.SecurityPolicy
	mode := d.cfg.SecurityMode

	if policy == "" {
		policy = "None"
	}
	if mode == "" {
		mode = "None"
	}

	if policy == "None" && mode == "None" {
		return []opcua.Option{
			opcua.SecurityPolicy("None"),
			opcua.SecurityMode(ua.MessageSecurityModeNone),
			opcua.AuthAnonymous(),
			opcua.RequestTimeout(d.cfg.RequestTimeout()),
		}, nil
	}

	endpoints, err := opcua.GetEndpoints(ctx, d.endpoint)
	if err != nil {
		return nil, fmt.Errorf("get endpoints: %w", err)
	}

	ep, err := opcua.SelectEndpoint(endpoints, policy, ua.MessageSecurityModeFromString(mode))
	if err != nil {
		return nil, fmt.Errorf("select endpoint: %w", err)
	}
	ep.EndpointURL = d.endpoint

	opts := []opcua.Option{
		opcua.SecurityPolicy(policy),
		opcua.SecurityModeString(mode),
		opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		opcua.RequestTimeout(d.cfg.RequestTimeout()),
	}

	if d.cfg.CertFile != "" && d.cfg.KeyFile != "" {
		opts = append(opts,
			opcua.CertificateFile(d.cfg.CertFile),
			opcua.PrivateKeyFile(d.cfg.KeyFile),
		)
	}

	if d.cfg.Username != "" {
		opts = append(opts, opcua.AuthUsername(d.cfg.Username, d.cfg.Password))
	} else {
		opts = append(opts, opcua.AuthAnonymous())
	}

	return opts, nil
}
