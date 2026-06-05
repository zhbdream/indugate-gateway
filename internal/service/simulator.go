package service

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/indugate/gateway/internal/config"
	opcuasim "github.com/indugate/gateway/internal/simulator/opcua"
	"go.uber.org/zap"
)

type SimulatorService struct {
	log     *zap.Logger
	opcua   *opcuasim.Simulator
	mu      sync.RWMutex
	started map[string]bool
}

func NewSimulatorService(log *zap.Logger, cfg config.SimulatorConfig) *SimulatorService {
	svc := &SimulatorService{
		log:     log,
		opcua:   opcuasim.NewSimulator(log, opcuasim.SimulatorConfig{Host: cfg.OPCUA.Host, Port: cfg.OPCUA.Port}),
		started: make(map[string]bool),
	}
	if cfg.OPCUA.AutoStart {
		if _, err := svc.Start("opcua"); err != nil {
			log.Warn("failed to auto-start opc ua simulator", zap.Error(err))
		}
	}
	return svc
}

func (s *SimulatorService) List() []map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	opcuaStatus := s.opcua.Status()
	return []map[string]any{
		{
			"type":        "opcua",
			"status":      statusString(s.started["opcua"]),
			"description": "OPC UA simulator with Temperature/Pressure/Flow data points",
			"endpoint":    opcuaStatus.Endpoint,
		},
		{
			"type":        "modbus",
			"status":      "stopped",
			"description": "Modbus TCP simulator (not implemented)",
		},
		{
			"type":        "mqtt",
			"status":      "stopped",
			"description": "MQTT simulator (not implemented)",
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
	default:
		return nil, fmt.Errorf("simulator type %q not implemented", simType)
	}
}

func (s *SimulatorService) Shutdown(_ context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.opcua.Stop()
}

func statusString(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

var ErrSimulatorNotFound = errors.New("simulator not found")
