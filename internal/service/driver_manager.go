package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/indugate/gateway/internal/model"
	opcuadriver "github.com/indugate/gateway/internal/protocol/opcua"
	"github.com/indugate/gateway/internal/protocol"
)

var ErrDeviceNotConnected = errors.New("device not connected")
var ErrUnsupportedProtocol = errors.New("unsupported protocol")

type DriverManager struct {
	mu     sync.RWMutex
	opcua  map[uint]*opcuadriver.Driver
}

func NewDriverManager() *DriverManager {
	return &DriverManager{
		opcua: make(map[uint]*opcuadriver.Driver),
	}
}

func (m *DriverManager) Connect(ctx context.Context, device *model.Device) error {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		cfg, err := opcuadriver.ParseConfig(device.Config)
		if err != nil {
			return err
		}
		driver := opcuadriver.NewDriver(device.Address, cfg)
		if err := driver.Connect(ctx); err != nil {
			return err
		}
		m.mu.Lock()
		if old, ok := m.opcua[device.ID]; ok {
			_ = old.Disconnect(ctx)
		}
		m.opcua[device.ID] = driver
		m.mu.Unlock()
		return nil
	default:
		return ErrUnsupportedProtocol
	}
}

func (m *DriverManager) Disconnect(ctx context.Context, deviceID uint) error {
	m.mu.Lock()
	driver, ok := m.opcua[deviceID]
	if ok {
		delete(m.opcua, deviceID)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}
	return driver.Disconnect(ctx)
}

func (m *DriverManager) OPCUADriver(deviceID uint) (*opcuadriver.Driver, error) {
	m.mu.RLock()
	driver, ok := m.opcua[deviceID]
	m.mu.RUnlock()
	if !ok || !driver.IsConnected() {
		return nil, ErrDeviceNotConnected
	}
	return driver, nil
}

func (m *DriverManager) IsConnected(deviceID uint) bool {
	m.mu.RLock()
	driver, ok := m.opcua[deviceID]
	m.mu.RUnlock()
	return ok && driver.IsConnected()
}

func (m *DriverManager) Browse(ctx context.Context, device *model.Device, nodeID string, depth int, childrenOnly bool) ([]protocol.NodeInfo, error) {
	driver, err := m.driverForDevice(device)
	if err != nil {
		return nil, err
	}
	opcua, ok := driver.(*opcuadriver.Driver)
	if !ok {
		return nil, ErrUnsupportedProtocol
	}
	if childrenOnly {
		return opcua.BrowseChildren(ctx, nodeID)
	}
	return opcua.Browse(ctx, nodeID, depth)
}

func (m *DriverManager) Read(ctx context.Context, device *model.Device, nodeID string) (*protocol.DataValue, error) {
	driver, err := m.driverForDevice(device)
	if err != nil {
		return nil, err
	}
	opcua, ok := driver.(*opcuadriver.Driver)
	if !ok {
		return nil, ErrUnsupportedProtocol
	}
	return opcua.Read(ctx, nodeID)
}

func (m *DriverManager) Write(ctx context.Context, device *model.Device, nodeID string, value any) error {
	driver, err := m.driverForDevice(device)
	if err != nil {
		return err
	}
	opcua, ok := driver.(*opcuadriver.Driver)
	if !ok {
		return ErrUnsupportedProtocol
	}
	return opcua.Write(ctx, nodeID, value)
}

func (m *DriverManager) Subscribe(ctx context.Context, device *model.Device, nodeIDs []string, interval time.Duration) (*protocol.SubscriptionInfo, error) {
	driver, err := m.driverForDevice(device)
	if err != nil {
		return nil, err
	}
	opcua, ok := driver.(*opcuadriver.Driver)
	if !ok {
		return nil, ErrUnsupportedProtocol
	}
	info, err := opcua.Subscribe(ctx, nodeIDs, interval)
	if err != nil {
		return nil, err
	}
	info.DeviceID = device.ID
	return info, nil
}

func (m *DriverManager) PollSubscription(deviceID uint, subID string, clear bool) ([]protocol.DataChangeEvent, error) {
	driver, err := m.OPCUADriver(deviceID)
	if err != nil {
		return nil, err
	}
	return driver.PollSubscription(subID, clear)
}

func (m *DriverManager) Unsubscribe(deviceID uint, subID string) error {
	driver, err := m.OPCUADriver(deviceID)
	if err != nil {
		return err
	}
	return driver.Unsubscribe(subID)
}

func (m *DriverManager) ListSubscriptions(deviceID uint) ([]protocol.SubscriptionInfo, error) {
	driver, err := m.OPCUADriver(deviceID)
	if err != nil {
		return nil, err
	}
	subs := driver.ListSubscriptions()
	for i := range subs {
		subs[i].DeviceID = deviceID
	}
	return subs, nil
}

func (m *DriverManager) driverForDevice(device *model.Device) (any, error) {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		return m.OPCUADriver(device.ID)
	default:
		return nil, ErrUnsupportedProtocol
	}
}
