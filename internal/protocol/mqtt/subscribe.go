package mqtt

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/indugate/gateway/internal/protocol"
)

type subscription struct {
	id        string
	nodeIDs   []string
	interval  time.Duration
	events    []protocol.DataChangeEvent
	eventsMu  sync.Mutex
	cancel    context.CancelFunc
	createdAt time.Time
}

func (d *Driver) Subscribe(_ context.Context, nodeIDs []string, interval time.Duration) (*protocol.SubscriptionInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("node_ids is required")
	}
	if interval <= 0 {
		interval = time.Second
	}

	subCtx, cancel := context.WithCancel(context.Background())
	subID := uuid.NewString()
	s := &subscription{
		id:        subID,
		nodeIDs:   append([]string(nil), nodeIDs...),
		interval:  interval,
		cancel:    cancel,
		createdAt: time.Now(),
	}

	d.subsMu.Lock()
	d.subscriptions[subID] = s
	d.subsMu.Unlock()

	for _, topic := range nodeIDs {
		token := d.client.Subscribe(topic, d.cfg.QoS, func(_ mqtt.Client, msg mqtt.Message) {
			event := protocol.DataChangeEvent{
				SubscriptionID: subID,
				NodeID:         msg.Topic(),
				Value:          string(msg.Payload()),
				Status:         "OK",
				Timestamp:      time.Now(),
			}

			s.eventsMu.Lock()
			s.events = append(s.events, event)
			if len(s.events) > 1000 {
				s.events = s.events[len(s.events)-500:]
			}
			s.eventsMu.Unlock()

			d.cacheMu.Lock()
			d.messageCache[msg.Topic()] = cachedMessage{
				payload:   string(msg.Payload()),
				timestamp: time.Now(),
			}
			d.cacheMu.Unlock()
		})
		if token.Wait() && token.Error() != nil {
			s.cancel()
			d.subsMu.Lock()
			delete(d.subscriptions, subID)
			d.subsMu.Unlock()
			return nil, fmt.Errorf("subscribe topic %q: %w", topic, token.Error())
		}
	}

	go func() {
		<-subCtx.Done()
		for _, topic := range nodeIDs {
			token := d.client.Unsubscribe(topic)
			token.Wait()
		}
	}()

	return &protocol.SubscriptionInfo{
		ID:        subID,
		NodeIDs:   s.nodeIDs,
		Interval:  interval.String(),
		CreatedAt: s.createdAt,
	}, nil
}

func (d *Driver) PollSubscription(subID string, clear bool) ([]protocol.DataChangeEvent, error) {
	d.subsMu.RLock()
	s, ok := d.subscriptions[subID]
	d.subsMu.RUnlock()
	if !ok {
		return nil, ErrSubscriptionNotFound
	}

	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	events := append([]protocol.DataChangeEvent(nil), s.events...)
	if clear {
		s.events = nil
	}
	return events, nil
}

func (d *Driver) Unsubscribe(subID string) error {
	d.subsMu.Lock()
	s, ok := d.subscriptions[subID]
	if ok {
		delete(d.subscriptions, subID)
	}
	d.subsMu.Unlock()

	if !ok {
		return ErrSubscriptionNotFound
	}
	s.cancel()
	return nil
}

func (d *Driver) ListSubscriptions() []protocol.SubscriptionInfo {
	d.subsMu.RLock()
	defer d.subsMu.RUnlock()

	result := make([]protocol.SubscriptionInfo, 0, len(d.subscriptions))
	for _, s := range d.subscriptions {
		result = append(result, protocol.SubscriptionInfo{
			ID:        s.id,
			NodeIDs:   append([]string(nil), s.nodeIDs...),
			Interval:  s.interval.String(),
			CreatedAt: s.createdAt,
		})
	}
	return result
}
