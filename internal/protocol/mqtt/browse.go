package mqtt

import (
	"context"
	"sort"

	"github.com/indugate/gateway/internal/protocol"
)

func (d *Driver) Browse(_ context.Context, _ string, _ int, _ bool) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	d.cacheMu.RLock()
	defer d.cacheMu.RUnlock()

	seen := make(map[string]struct{})
	for _, topic := range d.cfg.Topics {
		seen[topic] = struct{}{}
	}
	for topic := range d.messageCache {
		seen[topic] = struct{}{}
	}

	topics := make([]string, 0, len(seen))
	for topic := range seen {
		topics = append(topics, topic)
	}
	sort.Strings(topics)

	nodes := make([]protocol.NodeInfo, 0, len(topics))
	for _, topic := range topics {
		nodes = append(nodes, protocol.NodeInfo{
			NodeID:      topic,
			BrowseName:  topic,
			DisplayName: topic,
			Description: "MQTT topic",
			NodeClass:   "Topic",
			DataType:    "string",
			Writable:    true,
			Path:        topic,
		})
	}
	return nodes, nil
}

func (d *Driver) BrowseChildren(ctx context.Context, nodeID string) ([]protocol.NodeInfo, error) {
	return d.Browse(ctx, nodeID, 0, true)
}
