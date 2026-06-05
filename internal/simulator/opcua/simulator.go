package opcua

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"go.uber.org/zap"
)

type SimulatorConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func DefaultSimulatorConfig() SimulatorConfig {
	return SimulatorConfig{
		Host: "0.0.0.0",
		Port: 4840,
	}
}

type SimulatorStatus struct {
	Running    bool     `json:"running"`
	Endpoint   string   `json:"endpoint"`
	Port       int      `json:"port"`
	Namespace  uint16   `json:"namespace"`
	NodeIDs    []string `json:"node_ids,omitempty"`
}

type dataPoint struct {
	name        string
	unit        string
	description string
	value       float64
	mode        string
	min         float64
	max         float64
	step        float64
	phase       float64
	node        *server.Node
	nodeID      *ua.NodeID
}

type Simulator struct {
	log    *zap.Logger
	cfg    SimulatorConfig
	mu     sync.RWMutex
	srv    *server.Server
	ns     *server.NodeNameSpace
	cancel contextCancel
	points []*dataPoint
}

type contextCancel func()

func NewSimulator(log *zap.Logger, cfg SimulatorConfig) *Simulator {
	if cfg.Port == 0 {
		cfg.Port = 4840
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	return &Simulator{
		log: log,
		cfg: cfg,
	}
}

func (s *Simulator) Status() SimulatorStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := SimulatorStatus{
		Running:  s.srv != nil,
		Endpoint: fmt.Sprintf("opc.tcp://127.0.0.1:%d", s.cfg.Port),
		Port:     s.cfg.Port,
	}
	if s.ns != nil {
		status.Namespace = s.ns.ID()
	}
	for _, p := range s.points {
		if p.nodeID != nil {
			status.NodeIDs = append(status.NodeIDs, p.nodeID.String())
		}
	}
	return status
}

func (s *Simulator) UpdateConfig(raw string) error {
	if raw == "" {
		return nil
	}
	var cfg SimulatorConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return fmt.Errorf("parse simulator config: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return fmt.Errorf("cannot update config while simulator is running")
	}
	if cfg.Host != "" {
		s.cfg.Host = cfg.Host
	}
	if cfg.Port > 0 {
		s.cfg.Port = cfg.Port
	}
	return nil
}

func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		return fmt.Errorf("opc ua simulator already running")
	}

	opts := []server.Option{
		server.EnableSecurity("None", ua.MessageSecurityModeNone),
		server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		server.EndPoint(s.cfg.Host, s.cfg.Port),
		server.ServerName("InduGate OPC UA Simulator"),
		server.ProductName("InduGate Simulator"),
		server.ManufacturerName("InduGate"),
		server.SoftwareVersion("0.1.0"),
	}

	srv := server.New(opts...)
	ns := server.NewNodeNameSpace(srv, "InduGate")
	s.ns = ns

	objects := ns.Objects()
	if objects == nil {
		return fmt.Errorf("failed to create objects folder")
	}

	s.points = []*dataPoint{
		{name: "Temperature", unit: "°C", description: "Reactor temperature", mode: "sine", min: 20, max: 80, phase: 0},
		{name: "Pressure", unit: "bar", description: "Pipeline pressure", mode: "random", min: 1.0, max: 5.0},
		{name: "Flow", unit: "m³/h", description: "Coolant flow rate", mode: "sine", min: 10, max: 100, phase: math.Pi / 2},
		{name: "MotorSpeed", unit: "rpm", description: "Motor rotation speed", mode: "step", min: 0, max: 3000, step: 100},
		{name: "AlarmActive", unit: "", description: "Alarm status (0=normal, 1=alarm)", mode: "alarm", min: 0, max: 1},
	}

	for _, p := range s.points {
		p.nodeID = ua.NewStringNodeID(ns.ID(), p.name)
		p.node = s.createVariableNode(p)
		ns.AddNode(p.node)
		objects.AddRef(p.node, server.RefTypeIDOrganizes, true)
	}

	if err := srv.Start(context.Background()); err != nil {
		return fmt.Errorf("start opc ua simulator: %w", err)
	}

	stopCh := make(chan struct{})
	go s.runSimulation(stopCh)

	s.srv = srv
	s.cancel = func() {
		close(stopCh)
		srv.Close()
	}

	s.log.Info("opc ua simulator started",
		zap.String("endpoint", fmt.Sprintf("opc.tcp://127.0.0.1:%d", s.cfg.Port)),
	)
	return nil
}

func (s *Simulator) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.srv = nil
	s.ns = nil
	s.points = nil
	s.cancel = nil

	s.log.Info("opc ua simulator stopped")
	return nil
}

func (s *Simulator) createVariableNode(p *dataPoint) *server.Node {
	accessLevel := ua.AccessLevelTypeCurrentRead | ua.AccessLevelTypeCurrentWrite
	node := server.NewVariableNode(p.nodeID, p.name, func() *ua.DataValue {
		return server.DataValueFromValue(p.value)
	})
	node.SetAttribute(ua.AttributeIDAccessLevel, server.DataValueFromValue(byte(accessLevel)))
	node.SetAttribute(ua.AttributeIDUserAccessLevel, server.DataValueFromValue(byte(accessLevel)))
	node.SetAttribute(ua.AttributeIDDescription, server.DataValueFromValue(
		&ua.LocalizedText{EncodingMask: ua.LocalizedTextText, Text: p.description},
	))
	node.SetAttribute(ua.AttributeIDDataType, server.DataValueFromValue(
		ua.NewNumericExpandedNodeID(0, id.Double),
	))
	return node
}

func (s *Simulator) runSimulation(stopCh <-chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-stopCh:
			return
		case t := <-ticker.C:
			s.updatePoints(t.Sub(start))
		}
	}
}

func (s *Simulator) updatePoints(elapsed time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ns == nil {
		return
	}

	seconds := elapsed.Seconds()
	for _, p := range s.points {
		old := p.value
		switch p.mode {
		case "sine":
			mid := (p.max + p.min) / 2
			amp := (p.max - p.min) / 2
			p.value = mid + amp*math.Sin(seconds/10+p.phase)
		case "random":
			p.value = p.min + rand.Float64()*(p.max-p.min)
		case "step":
			steps := int(seconds) % int((p.max-p.min)/p.step+1)
			p.value = p.min + float64(steps)*p.step
			if p.value > p.max {
				p.value = p.max
			}
		case "alarm":
			if int(seconds)%30 < 3 {
				p.value = 1
			} else {
				p.value = 0
			}
		}

		if math.Abs(p.value-old) > 0.001 {
			s.ns.ChangeNotification(p.nodeID)
		}
	}
}
