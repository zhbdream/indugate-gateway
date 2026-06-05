package modbus

import (
	"context"

	"github.com/indugate/gateway/internal/protocol"
)

var defaultRegisterCatalog = []struct {
	nodeID      string
	browseName  string
	description string
	dataType    string
	writable    bool
}{
	{nodeID: "holding:0", browseName: "Temperature", description: "Reactor temperature (value x100)", dataType: "uint16", writable: true},
	{nodeID: "holding:1", browseName: "Pressure", description: "Pipeline pressure (value x100)", dataType: "uint16", writable: true},
	{nodeID: "holding:2", browseName: "Flow", description: "Coolant flow rate (value x100)", dataType: "uint16", writable: true},
	{nodeID: "holding:3", browseName: "MotorSpeed", description: "Motor rotation speed (rpm)", dataType: "uint16", writable: true},
	{nodeID: "coil:0", browseName: "AlarmActive", description: "Alarm status (0=normal, 1=alarm)", dataType: "bool", writable: true},
	{nodeID: "input:0", browseName: "SensorInput0", description: "Read-only input register 0", dataType: "uint16", writable: false},
	{nodeID: "discrete:0", browseName: "SwitchInput0", description: "Read-only discrete input 0", dataType: "bool", writable: false},
}

func (d *Driver) Browse(_ context.Context, _ string, _ int, _ bool) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	nodes := make([]protocol.NodeInfo, 0, len(defaultRegisterCatalog))
	for _, item := range defaultRegisterCatalog {
		nodes = append(nodes, protocol.NodeInfo{
			NodeID:      item.nodeID,
			BrowseName:  item.browseName,
			DisplayName: item.browseName,
			Description: item.description,
			NodeClass:   "Register",
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
