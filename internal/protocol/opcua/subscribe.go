package opcua

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/google/uuid"
	"github.com/indugate/gateway/internal/protocol"
)

type subscription struct {
	id         string
	nodeIDs    []string
	interval   time.Duration
	sub        *opcua.Subscription
	notifyCh   chan *opcua.PublishNotificationData
	events     []protocol.DataChangeEvent
	eventsMu   sync.Mutex
	cancel     context.CancelFunc
	createdAt  time.Time
}

func (d *Driver) Subscribe(ctx context.Context, nodeIDs []string, interval time.Duration) (*protocol.SubscriptionInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	if len(nodeIDs) == 0 {
		return nil, fmt.Errorf("node_ids is required")
	}
	if interval <= 0 {
		interval = opcua.DefaultSubscriptionInterval
	}

	subCtx, cancel := context.WithCancel(context.Background())
	notifyCh := make(chan *opcua.PublishNotificationData, 64)

	sub, err := d.client.Subscribe(subCtx, &opcua.SubscriptionParameters{
		Interval: interval,
	}, notifyCh)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create subscription: %w", err)
	}

	var requests []*ua.MonitoredItemCreateRequest
	handle := uint32(1)
	for _, nodeIDStr := range nodeIDs {
		nodeID, err := ua.ParseNodeID(nodeIDStr)
		if err != nil {
			sub.Cancel(subCtx)
			cancel()
			return nil, fmt.Errorf("invalid node id %q: %w", nodeIDStr, err)
		}
		requests = append(requests, opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, handle))
		handle++
	}

	res, err := sub.Monitor(subCtx, ua.TimestampsToReturnBoth, requests...)
	if err != nil {
		sub.Cancel(subCtx)
		cancel()
		return nil, fmt.Errorf("monitor nodes: %w", err)
	}
	for _, r := range res.Results {
		if r.StatusCode != ua.StatusOK {
			sub.Cancel(subCtx)
			cancel()
			return nil, fmt.Errorf("monitor node failed: %s", r.StatusCode)
		}
	}

	subID := uuid.NewString()
	s := &subscription{
		id:        subID,
		nodeIDs:   append([]string(nil), nodeIDs...),
		interval:  interval,
		sub:       sub,
		notifyCh:  notifyCh,
		cancel:    cancel,
		createdAt: time.Now(),
	}

	d.subsMu.Lock()
	d.subscriptions[subID] = s
	d.subsMu.Unlock()

	go d.consumeNotifications(subCtx, s)

	return &protocol.SubscriptionInfo{
		ID:        subID,
		NodeIDs:   s.nodeIDs,
		Interval:  interval.String(),
		CreatedAt: s.createdAt,
	}, nil
}

func (d *Driver) consumeNotifications(ctx context.Context, s *subscription) {
	defer s.cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case res := <-s.notifyCh:
			if res == nil {
				continue
			}
			if res.Error != nil {
				continue
			}
			notification, ok := res.Value.(*ua.DataChangeNotification)
			if !ok {
				continue
			}
			for _, item := range notification.MonitoredItems {
				if item.Value == nil {
					continue
				}
				var nodeID string
				if item.Value.Value != nil {
					nodeID = fmt.Sprintf("handle:%d", item.ClientHandle)
				}
				if idx := int(item.ClientHandle) - 1; idx >= 0 && idx < len(s.nodeIDs) {
					nodeID = s.nodeIDs[idx]
				}
				event := protocol.DataChangeEvent{
					SubscriptionID: s.id,
					NodeID:         nodeID,
					Value:          item.Value.Value.Value(),
					Status:         item.Value.Status.Error(),
					Timestamp:      time.Now(),
				}
				if !item.Value.SourceTimestamp.IsZero() {
					event.Timestamp = item.Value.SourceTimestamp
				}

				s.eventsMu.Lock()
				s.events = append(s.events, event)
				if len(s.events) > 1000 {
					s.events = s.events[len(s.events)-500:]
				}
				s.eventsMu.Unlock()
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.cancel()
	if s.sub != nil {
		_ = s.sub.Cancel(ctx)
	}
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
