package s7

import (
	"context"

	"github.com/indugate/gateway/internal/protocol"
)

var defaultNodeCatalog = []struct {
	nodeID      string
	browseName  string
	description string
	dataType    string
	writable    bool
}{
	{nodeID: "db1:0", browseName: "Temperature", description: "DB1 offset 0 (uint16)", dataType: "uint16", writable: true},
	{nodeID: "db1:2", browseName: "Pressure", description: "DB1 offset 2 (uint16)", dataType: "uint16", writable: true},
	{nodeID: "db1:4.real", browseName: "FlowRate", description: "DB1 offset 4 (real)", dataType: "real", writable: true},
	{nodeID: "db1:8.dint", browseName: "Counter", description: "DB1 offset 8 (dint)", dataType: "dint", writable: true},
	{nodeID: "db1:12.bool.0", browseName: "AlarmActive", description: "DB1 offset 12 bit 0", dataType: "bool", writable: true},
	{nodeID: "m0.0", browseName: "MerkerBit0", description: "Merker M0.0", dataType: "bool", writable: true},
	{nodeID: "i0.0", browseName: "InputBit0", description: "Input I0.0 (read-only)", dataType: "bool", writable: false},
	{nodeID: "q0.0", browseName: "OutputBit0", description: "Output Q0.0", dataType: "bool", writable: true},
}

func (d *Driver) Browse(_ context.Context, _ string, _ int, _ bool) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	nodes := make([]protocol.NodeInfo, 0, len(defaultNodeCatalog))
	for _, item := range defaultNodeCatalog {
		nodes = append(nodes, protocol.NodeInfo{
			NodeID:      item.nodeID,
			BrowseName:  item.browseName,
			DisplayName: item.browseName,
			Description: item.description,
			NodeClass:   "DataBlock",
			DataType:    item.dataType,
			Writable:    item.writable,
			Path:        item.browseName,
		})
	}
	return nodes, nil
}

func (d *Driver) BrowseChildren(ctx context.Context, nodeID string) ([]protocol.NodeInfo, error) {
	return d.Browse(ctx, nodeID, 0, true)
}
