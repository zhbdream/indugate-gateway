package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/protocol"
	bacnetdriver "github.com/indugate/gateway/internal/protocol/bacnet"
	modbusdriver "github.com/indugate/gateway/internal/protocol/modbus"
	mqttdriver "github.com/indugate/gateway/internal/protocol/mqtt"
	opcuadriver "github.com/indugate/gateway/internal/protocol/opcua"
	s7driver "github.com/indugate/gateway/internal/protocol/s7"
)

var ErrDeviceNotConnected = errors.New("device not connected")
var ErrUnsupportedProtocol = errors.New("unsupported protocol")

type DriverManager struct {
	mu     sync.RWMutex
	opcua  map[uint]*opcuadriver.Driver
	modbus map[uint]*modbusdriver.Driver
	mqtt   map[uint]*mqttdriver.Driver
	s7     map[uint]*s7driver.Driver
	bacnet map[uint]*bacnetdriver.Driver
}

func NewDriverManager() *DriverManager {
	return &DriverManager{
		opcua:  make(map[uint]*opcuadriver.Driver),
		modbus: make(map[uint]*modbusdriver.Driver),
		mqtt:   make(map[uint]*mqttdriver.Driver),
		s7:     make(map[uint]*s7driver.Driver),
		bacnet: make(map[uint]*bacnetdriver.Driver),
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
	case model.ProtocolModbus:
		cfg, err := modbusdriver.ParseConfig(device.Config)
		if err != nil {
			return err
		}
		driver := modbusdriver.NewDriver(device.Address, cfg)
		if err := driver.Connect(ctx); err != nil {
			return err
		}
		m.mu.Lock()
		if old, ok := m.modbus[device.ID]; ok {
			_ = old.Disconnect(ctx)
		}
		m.modbus[device.ID] = driver
		m.mu.Unlock()
		return nil
	case model.ProtocolMQTT:
		cfg, err := mqttdriver.ParseConfig(device.Config)
		if err != nil {
			return err
		}
		driver := mqttdriver.NewDriver(device.Address, cfg)
		if err := driver.Connect(ctx); err != nil {
			return err
		}
		m.mu.Lock()
		if old, ok := m.mqtt[device.ID]; ok {
			_ = old.Disconnect(ctx)
		}
		m.mqtt[device.ID] = driver
		m.mu.Unlock()
		return nil
	case model.ProtocolS7:
		cfg, err := s7driver.ParseConfig(device.Config)
		if err != nil {
			return err
		}
		driver := s7driver.NewDriver(device.Address, cfg)
		if err := driver.Connect(ctx); err != nil {
			return err
		}
		m.mu.Lock()
		if old, ok := m.s7[device.ID]; ok {
			_ = old.Disconnect(ctx)
		}
		m.s7[device.ID] = driver
		m.mu.Unlock()
		return nil
	case model.ProtocolBACnet:
		cfg, err := bacnetdriver.ParseConfig(device.Config)
		if err != nil {
			return err
		}
		driver := bacnetdriver.NewDriver(device.Address, cfg)
		if err := driver.Connect(ctx); err != nil {
			return err
		}
		m.mu.Lock()
		if old, ok := m.bacnet[device.ID]; ok {
			_ = old.Disconnect(ctx)
		}
		m.bacnet[device.ID] = driver
		m.mu.Unlock()
		return nil
	default:
		return ErrUnsupportedProtocol
	}
}

func (m *DriverManager) Disconnect(ctx context.Context, deviceID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if driver, ok := m.opcua[deviceID]; ok {
		delete(m.opcua, deviceID)
		return driver.Disconnect(ctx)
	}
	if driver, ok := m.modbus[deviceID]; ok {
		delete(m.modbus, deviceID)
		return driver.Disconnect(ctx)
	}
	if driver, ok := m.mqtt[deviceID]; ok {
		delete(m.mqtt, deviceID)
		return driver.Disconnect(ctx)
	}
	if driver, ok := m.s7[deviceID]; ok {
		delete(m.s7, deviceID)
		return driver.Disconnect(ctx)
	}
	if driver, ok := m.bacnet[deviceID]; ok {
		delete(m.bacnet, deviceID)
		return driver.Disconnect(ctx)
	}
	return nil
}

func (m *DriverManager) IsConnected(deviceID uint) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if driver, ok := m.opcua[deviceID]; ok {
		return driver.IsConnected()
	}
	if driver, ok := m.modbus[deviceID]; ok {
		return driver.IsConnected()
	}
	if driver, ok := m.mqtt[deviceID]; ok {
		return driver.IsConnected()
	}
	if driver, ok := m.s7[deviceID]; ok {
		return driver.IsConnected()
	}
	if driver, ok := m.bacnet[deviceID]; ok {
		return driver.IsConnected()
	}
	return false
}

func (m *DriverManager) Browse(ctx context.Context, device *model.Device, nodeID string, depth int, childrenOnly bool) ([]protocol.NodeInfo, error) {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		driver, err := m.OPCUADriver(device.ID)
		if err != nil {
			return nil, err
		}
		if childrenOnly {
			return driver.BrowseChildren(ctx, nodeID)
		}
		return driver.Browse(ctx, nodeID, depth)
	case model.ProtocolModbus:
		driver, err := m.ModbusDriver(device.ID)
		if err != nil {
			return nil, err
		}
		if childrenOnly {
			return driver.BrowseChildren(ctx, nodeID)
		}
		return driver.Browse(ctx, nodeID, depth, childrenOnly)
	case model.ProtocolMQTT:
		driver, err := m.MQTTDriver(device.ID)
		if err != nil {
			return nil, err
		}
		if childrenOnly {
			return driver.BrowseChildren(ctx, nodeID)
		}
		return driver.Browse(ctx, nodeID, depth, childrenOnly)
	case model.ProtocolS7:
		driver, err := m.S7Driver(device.ID)
		if err != nil {
			return nil, err
		}
		if childrenOnly {
			return driver.BrowseChildren(ctx, nodeID)
		}
		return driver.Browse(ctx, nodeID, depth, childrenOnly)
	case model.ProtocolBACnet:
		driver, err := m.BACnetDriver(device.ID)
		if err != nil {
			return nil, err
		}
		if childrenOnly {
			return driver.BrowseChildren(ctx, nodeID)
		}
		return driver.Browse(ctx, nodeID, depth, childrenOnly)
	default:
		return nil, ErrUnsupportedProtocol
	}
}

func (m *DriverManager) Read(ctx context.Context, device *model.Device, nodeID string) (*protocol.DataValue, error) {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		driver, err := m.OPCUADriver(device.ID)
		if err != nil {
			return nil, err
		}
		return driver.Read(ctx, nodeID)
	case model.ProtocolModbus:
		driver, err := m.ModbusDriver(device.ID)
		if err != nil {
			return nil, err
		}
		return driver.Read(ctx, nodeID)
	case model.ProtocolMQTT:
		driver, err := m.MQTTDriver(device.ID)
		if err != nil {
			return nil, err
		}
		return driver.Read(ctx, nodeID)
	case model.ProtocolS7:
		driver, err := m.S7Driver(device.ID)
		if err != nil {
			return nil, err
		}
		return driver.Read(ctx, nodeID)
	case model.ProtocolBACnet:
		driver, err := m.BACnetDriver(device.ID)
		if err != nil {
			return nil, err
		}
		return driver.Read(ctx, nodeID)
	default:
		return nil, ErrUnsupportedProtocol
	}
}

func (m *DriverManager) Write(ctx context.Context, device *model.Device, nodeID string, value any) error {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		driver, err := m.OPCUADriver(device.ID)
		if err != nil {
			return err
		}
		return driver.Write(ctx, nodeID, value)
	case model.ProtocolModbus:
		driver, err := m.ModbusDriver(device.ID)
		if err != nil {
			return err
		}
		return driver.Write(ctx, nodeID, value)
	case model.ProtocolMQTT:
		driver, err := m.MQTTDriver(device.ID)
		if err != nil {
			return err
		}
		return driver.Write(ctx, nodeID, value)
	case model.ProtocolS7:
		driver, err := m.S7Driver(device.ID)
		if err != nil {
			return err
		}
		return driver.Write(ctx, nodeID, value)
	case model.ProtocolBACnet:
		driver, err := m.BACnetDriver(device.ID)
		if err != nil {
			return err
		}
		return driver.Write(ctx, nodeID, value)
	default:
		return ErrUnsupportedProtocol
	}
}

func (m *DriverManager) Subscribe(ctx context.Context, device *model.Device, nodeIDs []string, interval time.Duration) (*protocol.SubscriptionInfo, error) {
	switch device.Protocol {
	case model.ProtocolOPCUA:
		driver, err := m.OPCUADriver(device.ID)
		if err != nil {
			return nil, err
		}
		info, err := driver.Subscribe(ctx, nodeIDs, interval)
		if err != nil {
			return nil, err
		}
		info.DeviceID = device.ID
		return info, nil
	case model.ProtocolModbus:
		driver, err := m.ModbusDriver(device.ID)
		if err != nil {
			return nil, err
		}
		info, err := driver.Subscribe(ctx, nodeIDs, interval)
		if err != nil {
			return nil, err
		}
		info.DeviceID = device.ID
		return info, nil
	case model.ProtocolMQTT:
		driver, err := m.MQTTDriver(device.ID)
		if err != nil {
			return nil, err
		}
		info, err := driver.Subscribe(ctx, nodeIDs, interval)
		if err != nil {
			return nil, err
		}
		info.DeviceID = device.ID
		return info, nil
	case model.ProtocolS7:
		driver, err := m.S7Driver(device.ID)
		if err != nil {
			return nil, err
		}
		info, err := driver.Subscribe(ctx, nodeIDs, interval)
		if err != nil {
			return nil, err
		}
		info.DeviceID = device.ID
		return info, nil
	case model.ProtocolBACnet:
		driver, err := m.BACnetDriver(device.ID)
		if err != nil {
			return nil, err
		}
		info, err := driver.Subscribe(ctx, nodeIDs, interval)
		if err != nil {
			return nil, err
		}
		info.DeviceID = device.ID
		return info, nil
	default:
		return nil, ErrUnsupportedProtocol
	}
}

func (m *DriverManager) PollSubscription(deviceID uint, subID string, clear bool) ([]protocol.DataChangeEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if driver, ok := m.opcua[deviceID]; ok {
		return driver.PollSubscription(subID, clear)
	}
	if driver, ok := m.modbus[deviceID]; ok {
		return driver.PollSubscription(subID, clear)
	}
	if driver, ok := m.mqtt[deviceID]; ok {
		return driver.PollSubscription(subID, clear)
	}
	if driver, ok := m.s7[deviceID]; ok {
		return driver.PollSubscription(subID, clear)
	}
	if driver, ok := m.bacnet[deviceID]; ok {
		return driver.PollSubscription(subID, clear)
	}
	return nil, ErrDeviceNotConnected
}

func (m *DriverManager) Unsubscribe(deviceID uint, subID string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if driver, ok := m.opcua[deviceID]; ok {
		return driver.Unsubscribe(subID)
	}
	if driver, ok := m.modbus[deviceID]; ok {
		return driver.Unsubscribe(subID)
	}
	if driver, ok := m.mqtt[deviceID]; ok {
		return driver.Unsubscribe(subID)
	}
	if driver, ok := m.s7[deviceID]; ok {
		return driver.Unsubscribe(subID)
	}
	if driver, ok := m.bacnet[deviceID]; ok {
		return driver.Unsubscribe(subID)
	}
	return ErrDeviceNotConnected
}

func (m *DriverManager) ListSubscriptions(deviceID uint) ([]protocol.SubscriptionInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var subs []protocol.SubscriptionInfo
	if driver, ok := m.opcua[deviceID]; ok {
		subs = driver.ListSubscriptions()
	} else if driver, ok := m.modbus[deviceID]; ok {
		subs = driver.ListSubscriptions()
	} else if driver, ok := m.mqtt[deviceID]; ok {
		subs = driver.ListSubscriptions()
	} else if driver, ok := m.s7[deviceID]; ok {
		subs = driver.ListSubscriptions()
	} else if driver, ok := m.bacnet[deviceID]; ok {
		subs = driver.ListSubscriptions()
	} else {
		return nil, ErrDeviceNotConnected
	}
	for i := range subs {
		subs[i].DeviceID = deviceID
	}
	return subs, nil
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

func (m *DriverManager) ModbusDriver(deviceID uint) (*modbusdriver.Driver, error) {
	m.mu.RLock()
	driver, ok := m.modbus[deviceID]
	m.mu.RUnlock()
	if !ok || !driver.IsConnected() {
		return nil, ErrDeviceNotConnected
	}
	return driver, nil
}

func (m *DriverManager) MQTTDriver(deviceID uint) (*mqttdriver.Driver, error) {
	m.mu.RLock()
	driver, ok := m.mqtt[deviceID]
	m.mu.RUnlock()
	if !ok || !driver.IsConnected() {
		return nil, ErrDeviceNotConnected
	}
	return driver, nil
}

func (m *DriverManager) S7Driver(deviceID uint) (*s7driver.Driver, error) {
	m.mu.RLock()
	driver, ok := m.s7[deviceID]
	m.mu.RUnlock()
	if !ok || !driver.IsConnected() {
		return nil, ErrDeviceNotConnected
	}
	return driver, nil
}

func (m *DriverManager) BACnetDriver(deviceID uint) (*bacnetdriver.Driver, error) {
	m.mu.RLock()
	driver, ok := m.bacnet[deviceID]
	m.mu.RUnlock()
	if !ok || !driver.IsConnected() {
		return nil, ErrDeviceNotConnected
	}
	return driver, nil
}
