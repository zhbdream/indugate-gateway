package modbus

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	lastVals  map[string]any
}

func (d *Driver) Subscribe(ctx context.Context, nodeIDs []string, interval time.Duration) (*protocol.SubscriptionInfo, error) {
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
		lastVals:  make(map[string]any),
	}

	d.subsMu.Lock()
	d.subscriptions[subID] = s
	d.subsMu.Unlock()

	go d.pollSubscription(subCtx, s)

	return &protocol.SubscriptionInfo{
		ID:        subID,
		NodeIDs:   s.nodeIDs,
		Interval:  interval.String(),
		CreatedAt: s.createdAt,
	}, nil
}

func (d *Driver) pollSubscription(ctx context.Context, s *subscription) {
	defer s.cancel()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, nodeID := range s.nodeIDs {
				val, err := d.Read(ctx, nodeID)
				if err != nil {
					continue
				}

				s.eventsMu.Lock()
				prev, seen := s.lastVals[nodeID]
				if !seen || !valuesEqual(prev, val.Value) {
					s.lastVals[nodeID] = val.Value
					s.events = append(s.events, protocol.DataChangeEvent{
						SubscriptionID: s.id,
						NodeID:         nodeID,
						Value:          val.Value,
						Status:         val.Status,
						Timestamp:      val.Timestamp,
					})
					if len(s.events) > 1000 {
						s.events = s.events[len(s.events)-500:]
					}
				}
				s.eventsMu.Unlock()
			}
		}
	}
}

func valuesEqual(a, b any) bool {
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case uint16:
		bv, ok := b.(uint16)
		return ok && av == bv
	default:
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
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
