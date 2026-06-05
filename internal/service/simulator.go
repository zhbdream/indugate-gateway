package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/indugate/gateway/internal/config"
	modbussim "github.com/indugate/gateway/internal/simulator/modbus"
	mqttsim "github.com/indugate/gateway/internal/simulator/mqtt"
	opcuasim "github.com/indugate/gateway/internal/simulator/opcua"
	"go.uber.org/zap"
)

type SimulatorService struct {
	log     *zap.Logger
	opcua   *opcuasim.Simulator
	modbus  *modbussim.Simulator
	mqtt    *mqttsim.Simulator
	mu      sync.RWMutex
	started map[string]bool
}

func NewSimulatorService(log *zap.Logger, cfg config.SimulatorConfig) *SimulatorService {
	svc := &SimulatorService{
		log: log,
		opcua: opcuasim.NewSimulator(log, opcuasim.SimulatorConfig{
			Host: cfg.OPCUA.Host,
			Port: cfg.OPCUA.Port,
		}),
		modbus: modbussim.NewSimulator(log, modbussim.SimulatorConfig{
			Host: cfg.Modbus.Host,
			Port: cfg.Modbus.Port,
		}),
		mqtt: mqttsim.NewSimulator(log, mqttsim.SimulatorConfig{
			Host:   cfg.MQTT.Host,
			Port:   cfg.MQTT.Port,
			Topics: cfg.MQTT.Topics,
		}),
		started: make(map[string]bool),
	}
	if cfg.OPCUA.AutoStart {
		if _, err := svc.Start("opcua"); err != nil {
			log.Warn("failed to auto-start opc ua simulator", zap.Error(err))
		}
	}
	if cfg.Modbus.AutoStart {
		if _, err := svc.Start("modbus"); err != nil {
			log.Warn("failed to auto-start modbus simulator", zap.Error(err))
		}
	}
	if cfg.MQTT.AutoStart {
		if _, err := svc.Start("mqtt"); err != nil {
			log.Warn("failed to auto-start mqtt simulator", zap.Error(err))
		}
	}
	return svc
}

func (s *SimulatorService) List() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	opcuaStatus := s.opcua.Status()
	modbusStatus := s.modbus.Status()
	mqttStatus := s.mqtt.Status()

	return []map[string]any{
		{
			"type":        "opcua",
			"status":      statusString(s.started["opcua"]),
			"description": "OPC UA simulator with Temperature/Pressure/Flow data points",
			"endpoint":    opcuaStatus.Endpoint,
		},
		{
			"type":        "modbus",
			"status":      statusString(s.started["modbus"]),
			"description": "Modbus TCP simulator with holding registers and coils",
			"endpoint":    modbusStatus.Endpoint,
			"nodes":       modbusStatus.NodeIDs,
		},
		{
			"type":        "mqtt",
			"status":      statusString(s.started["mqtt"]),
			"description": "Embedded MQTT broker with auto telemetry publishing",
			"endpoint":    mqttStatus.Endpoint,
			"topics":      mqttStatus.Topics,
		},
	}
}

func (s *SimulatorService) Start(simType string) (map[string]any, error) {
	switch simType {
	case "opcua":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.opcua.Start(); err != nil {
			return nil, err
		}
		s.started["opcua"] = true
		status := s.opcua.Status()
		return map[string]any{
			"type":     simType,
			"status":   "running",
			"endpoint": status.Endpoint,
			"nodes":    status.NodeIDs,
		}, nil
	case "modbus":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.modbus.Start(); err != nil {
			return nil, err
		}
		s.started["modbus"] = true
		status := s.modbus.Status()
		return map[string]any{
			"type":     simType,
			"status":   "running",
			"endpoint": status.Endpoint,
			"nodes":    status.NodeIDs,
		}, nil
	case "mqtt":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.mqtt.Start(); err != nil {
			return nil, err
		}
		s.started["mqtt"] = true
		status := s.mqtt.Status()
		return map[string]any{
			"type":     simType,
			"status":   "running",
			"endpoint": status.Endpoint,
			"topics":   status.Topics,
		}, nil
	default:
		return nil, fmt.Errorf("simulator type %q not implemented", simType)
	}
}

func (s *SimulatorService) Stop(simType string) (map[string]any, error) {
	switch simType {
	case "opcua":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.opcua.Stop(); err != nil {
			return nil, err
		}
		s.started["opcua"] = false
		return map[string]any{"type": simType, "status": "stopped"}, nil
	case "modbus":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.modbus.Stop(); err != nil {
			return nil, err
		}
		s.started["modbus"] = false
		return map[string]any{"type": simType, "status": "stopped"}, nil
	case "mqtt":
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.mqtt.Stop(); err != nil {
			return nil, err
		}
		s.started["mqtt"] = false
		return map[string]any{"type": simType, "status": "stopped"}, nil
	default:
		return nil, fmt.Errorf("simulator type %q not implemented", simType)
	}
}

func (s *SimulatorService) UpdateConfig(simType string, configJSON string) (map[string]any, error) {
	switch simType {
	case "opcua":
		if err := s.opcua.UpdateConfig(configJSON); err != nil {
			return nil, err
		}
		return map[string]any{"type": simType, "message": "config updated"}, nil
	case "modbus":
		if err := s.modbus.UpdateConfig(configJSON); err != nil {
			return nil, err
		}
		return map[string]any{"type": simType, "message": "config updated"}, nil
	case "mqtt":
		if err := s.mqtt.UpdateConfig(configJSON); err != nil {
			return nil, err
		}
		return map[string]any{"type": simType, "message": "config updated"}, nil
	default:
		return nil, fmt.Errorf("simulator type %q not implemented", simType)
	}
}

func (s *SimulatorService) Shutdown(_ context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.opcua.Stop()
	_ = s.modbus.Stop()
	_ = s.mqtt.Stop()
}

func statusString(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

var ErrSimulatorNotFound = errors.New("simulator not found")
