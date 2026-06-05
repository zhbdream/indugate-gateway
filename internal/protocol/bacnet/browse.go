package bacnet

import (
	"context"

	"github.com/indugate/gateway/internal/protocol"
)

var defaultObjectCatalog = []struct {
	nodeID      string
	browseName  string
	description string
	dataType    string
	writable    bool
}{
	{nodeID: "analogInput:1", browseName: "RoomTemp", description: "Room temperature sensor", dataType: "float", writable: false},
	{nodeID: "analogOutput:1", browseName: "DamperPos", description: "Damper position setpoint", dataType: "float", writable: true},
	{nodeID: "analogValue:1", browseName: "Setpoint", description: "Temperature setpoint", dataType: "float", writable: true},
	{nodeID: "binaryInput:1", browseName: "Occupancy", description: "Occupancy sensor", dataType: "bool", writable: false},
	{nodeID: "binaryOutput:1", browseName: "FanEnable", description: "Fan enable output", dataType: "bool", writable: true},
	{nodeID: "binaryValue:1", browseName: "AlarmState", description: "Alarm state flag", dataType: "bool", writable: true},
}

func (d *Driver) Browse(_ context.Context, _ string, _ int, _ bool) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	nodes := make([]protocol.NodeInfo, 0, len(defaultObjectCatalog))
	for _, item := range defaultObjectCatalog {
		nodes = append(nodes, protocol.NodeInfo{
			NodeID:      item.nodeID,
			BrowseName:  item.browseName,
			DisplayName: item.browseName,
			Description: item.description,
			NodeClass:   "Object",
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
