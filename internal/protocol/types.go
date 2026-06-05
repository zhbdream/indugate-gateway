package protocol

import "time"

type NodeInfo struct {
	NodeID      string `json:"node_id"`
	BrowseName  string `json:"browse_name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description,omitempty"`
	NodeClass   string `json:"node_class"`
	DataType    string `json:"data_type,omitempty"`
	Writable    bool   `json:"writable"`
	Path        string `json:"path,omitempty"`
	HasChildren bool   `json:"has_children,omitempty"`
}

type DataValue struct {
	NodeID    string      `json:"node_id"`
	Value     interface{} `json:"value"`
	DataType  string      `json:"data_type,omitempty"`
	Status    string      `json:"status"`
	Timestamp time.Time   `json:"timestamp"`
}

type DataChangeEvent struct {
	SubscriptionID string    `json:"subscription_id"`
	NodeID         string    `json:"node_id"`
	Value          any       `json:"value"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"timestamp"`
}

type SubscriptionInfo struct {
	ID        string   `json:"id"`
	DeviceID  uint     `json:"device_id"`
	NodeIDs   []string `json:"node_ids"`
	Interval  string   `json:"interval"`
	CreatedAt time.Time `json:"created_at"`
}
