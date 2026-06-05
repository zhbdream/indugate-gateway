package bacnet

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/indugate/gateway/internal/protocol"
)

type subscription struct {
	id         string
	nodeIDs    []string
	interval   time.Duration
	mode       string
	events     []protocol.DataChangeEvent
	eventsMu   sync.Mutex
	cancel     context.CancelFunc
	createdAt  time.Time
	lastVals   map[string]any
	processIDs map[string]uint32
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
		id:         subID,
		nodeIDs:    append([]string(nil), nodeIDs...),
		interval:   interval,
		cancel:     cancel,
		createdAt:  time.Now(),
		lastVals:   make(map[string]any),
		processIDs: make(map[string]uint32),
	}

	if d.cfg.COVEnabled {
		if err := d.startCOVSubscription(subCtx, s); err == nil {
			s.mode = "cov"
		} else {
			s.mode = "poll"
			go d.pollSubscription(subCtx, s)
		}
	} else {
		s.mode = "poll"
		go d.pollSubscription(subCtx, s)
	}

	d.subsMu.Lock()
	d.subscriptions[subID] = s
	d.subsMu.Unlock()

	return &protocol.SubscriptionInfo{
		ID:        subID,
		NodeIDs:   s.nodeIDs,
		Interval:  interval.String(),
		CreatedAt: s.createdAt,
	}, nil
}

func (d *Driver) startCOVSubscription(ctx context.Context, s *subscription) error {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()
	if client == nil {
		return ErrNotConnected
	}

	lifetime := uint32(d.cfg.COVLifetimeSec)
	if lifetime == 0 {
		lifetime = 300
	}

	for _, nodeID := range s.nodeIDs {
		obj, err := ParseObjectRef(nodeID)
		if err != nil {
			return err
		}
		processID := atomic.AddUint32(&d.invokeID, 1)
		if processID == 0 {
			processID = 1
		}
		s.processIDs[nodeID] = processID
		nodeIDCopy := nodeID
		client.RegisterCOVHandler(processID, func(object ObjectRef, value any) {
			if object.String() != nodeIDCopy {
				return
			}
			d.appendSubscriptionEvent(s, nodeIDCopy, value, "OK")
		})
		if err := client.SubscribeCOV(obj, processID, lifetime, d.cfg.RequestTimeout(), &d.invokeID); err != nil {
			client.UnregisterCOVHandler(processID)
			return err
		}
	}

	go func() {
		<-ctx.Done()
		for _, processID := range s.processIDs {
			client.UnregisterCOVHandler(processID)
			_ = client.UnsubscribeCOV(processID, d.cfg.RequestTimeout(), &d.invokeID)
		}
	}()
	return nil
}

func (d *Driver) appendSubscriptionEvent(s *subscription, nodeID string, value any, status string) {
	s.eventsMu.Lock()
	defer s.eventsMu.Unlock()
	prev, seen := s.lastVals[nodeID]
	if seen && fmt.Sprintf("%v", prev) == fmt.Sprintf("%v", value) {
		return
	}
	s.lastVals[nodeID] = value
	s.events = append(s.events, protocol.DataChangeEvent{
		SubscriptionID: s.id,
		NodeID:         nodeID,
		Value:          value,
		Status:         status,
		Timestamp:      time.Now(),
	})
	if len(s.events) > 1000 {
		s.events = s.events[len(s.events)-500:]
	}
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
				d.appendSubscriptionEvent(s, nodeID, val.Value, val.Status)
			}
		}
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
