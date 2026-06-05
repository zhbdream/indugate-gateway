package mqtt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/indugate/gateway/internal/protocol"
)

var (
	ErrNotConnected         = errors.New("mqtt client not connected")
	ErrAlreadyConnected     = errors.New("mqtt client already connected")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrTopicNotFound        = errors.New("topic has no cached message")
)

type cachedMessage struct {
	payload   string
	timestamp time.Time
}

type Driver struct {
	broker string
	cfg    *Config

	mu            sync.RWMutex
	client        pahomqtt.Client
	connected     bool
	messageCache  map[string]cachedMessage
	cacheMu       sync.RWMutex
	subscriptions map[string]*subscription
	subsMu        sync.RWMutex
}

func NewDriver(broker string, cfg *Config) *Driver {
	if cfg == nil {
		cfg = &Config{ClientID: "indugate-gateway", QoS: 1}
	}
	return &Driver{
		broker:        normalizeBroker(broker),
		cfg:           cfg,
		messageCache:  make(map[string]cachedMessage),
		subscriptions: make(map[string]*subscription),
	}
}

func normalizeBroker(address string) string {
	address = strings.TrimSpace(address)
	if address == "" {
		return "tcp://127.0.0.1:1883"
	}
	if strings.HasPrefix(address, "tcp://") || strings.HasPrefix(address, "ssl://") || strings.HasPrefix(address, "ws://") {
		return address
	}
	return "tcp://" + address
}

func (d *Driver) IsConnected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected && d.client != nil && d.client.IsConnected()
}

func (d *Driver) Connect(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.connected {
		return ErrAlreadyConnected
	}

	opts := pahomqtt.NewClientOptions().
		AddBroker(d.broker).
		SetClientID(d.cfg.ClientID).
		SetCleanSession(d.cfg.CleanSession).
		SetKeepAlive(time.Duration(d.cfg.KeepAliveSec) * time.Second).
		SetConnectTimeout(d.cfg.RequestTimeout())

	if d.cfg.Username != "" {
		opts.SetUsername(d.cfg.Username)
		opts.SetPassword(d.cfg.Password)
	}
	if d.cfg.Will != nil {
		opts.SetWill(d.cfg.Will.Topic, d.cfg.Will.Message, d.cfg.Will.QoS, d.cfg.Will.Retain)
	}

	opts.SetDefaultPublishHandler(func(_ pahomqtt.Client, msg pahomqtt.Message) {
		d.cacheMu.Lock()
		d.messageCache[msg.Topic()] = cachedMessage{
			payload:   string(msg.Payload()),
			timestamp: time.Now(),
		}
		d.cacheMu.Unlock()
	})

	client := pahomqtt.NewClient(opts)

	connectCtx, cancel := context.WithTimeout(ctx, d.cfg.RequestTimeout())
	defer cancel()

	token := client.Connect()
	if !token.WaitTimeout(d.cfg.RequestTimeout()) {
		return fmt.Errorf("connect mqtt broker: timeout")
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("connect mqtt broker: %w", err)
	}

	select {
	case <-connectCtx.Done():
		client.Disconnect(250)
		return fmt.Errorf("connect mqtt broker: %w", connectCtx.Err())
	default:
	}

	for _, topic := range d.cfg.Topics {
		subToken := client.Subscribe(topic, d.cfg.QoS, func(_ pahomqtt.Client, msg pahomqtt.Message) {
			d.cacheMu.Lock()
			d.messageCache[msg.Topic()] = cachedMessage{
				payload:   string(msg.Payload()),
				timestamp: time.Now(),
			}
			d.cacheMu.Unlock()
		})
		if !subToken.WaitTimeout(d.cfg.RequestTimeout()) {
			client.Disconnect(250)
			return fmt.Errorf("subscribe topic %q: timeout", topic)
		}
		if err := subToken.Error(); err != nil {
			client.Disconnect(250)
			return fmt.Errorf("subscribe topic %q: %w", topic, err)
		}
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
		d.client.Disconnect(250)
		d.client = nil
	}
	d.connected = false
	return nil
}

func (d *Driver) Read(_ context.Context, nodeID string) (*protocol.DataValue, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	topic := strings.TrimSpace(nodeID)
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	d.cacheMu.RLock()
	msg, ok := d.messageCache[topic]
	d.cacheMu.RUnlock()
	if !ok {
		return nil, ErrTopicNotFound
	}

	return &protocol.DataValue{
		NodeID:    topic,
		Value:     msg.payload,
		DataType:  "string",
		Status:    "OK",
		Timestamp: msg.timestamp,
	}, nil
}

func (d *Driver) Write(_ context.Context, nodeID string, value any) error {
	if !d.IsConnected() {
		return ErrNotConnected
	}

	topic := strings.TrimSpace(nodeID)
	if topic == "" {
		return fmt.Errorf("topic is required")
	}

	payload := fmt.Sprintf("%v", value)
	token := d.client.Publish(topic, d.cfg.QoS, false, payload)
	if !token.WaitTimeout(d.cfg.RequestTimeout()) {
		return fmt.Errorf("publish to %q: timeout", topic)
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish to %q: %w", topic, err)
	}

	d.cacheMu.Lock()
	d.messageCache[topic] = cachedMessage{
		payload:   payload,
		timestamp: time.Now(),
	}
	d.cacheMu.Unlock()
	return nil
}
