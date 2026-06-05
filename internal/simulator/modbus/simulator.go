package modbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/tbrandon/mbserver"
	"go.uber.org/zap"
)

type SimulatorConfig struct {
	Host             string            `json:"host"`
	Port             int               `json:"port"`
	HoldingRegisters map[string]uint16 `json:"holding_registers,omitempty"`
	Coils            map[string]bool   `json:"coils,omitempty"`
}

func DefaultSimulatorConfig() SimulatorConfig {
	return SimulatorConfig{
		Host: "0.0.0.0",
		Port: 502,
	}
}

type SimulatorStatus struct {
	Running  bool     `json:"running"`
	Endpoint string   `json:"endpoint"`
	Port     int      `json:"port"`
	NodeIDs  []string `json:"node_ids,omitempty"`
}

type Simulator struct {
	log    *zap.Logger
	cfg    SimulatorConfig
	mu     sync.RWMutex
	server *mbserver.Server
	cancel context.CancelFunc
}

func NewSimulator(log *zap.Logger, cfg SimulatorConfig) *Simulator {
	if cfg.Port == 0 {
		cfg.Port = 502
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

	return SimulatorStatus{
		Running:  s.server != nil,
		Endpoint: fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
		Port:     s.cfg.Port,
		NodeIDs: []string{
			"holding:0", "holding:1", "holding:2", "holding:3", "coil:0",
		},
	}
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
	if s.server != nil {
		return fmt.Errorf("cannot update config while simulator is running")
	}
	if cfg.Host != "" {
		s.cfg.Host = cfg.Host
	}
	if cfg.Port > 0 {
		s.cfg.Port = cfg.Port
	}
	if cfg.HoldingRegisters != nil {
		s.cfg.HoldingRegisters = cfg.HoldingRegisters
	}
	if cfg.Coils != nil {
		s.cfg.Coils = cfg.Coils
	}
	return nil
}

func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return fmt.Errorf("modbus simulator already running")
	}

	serv := mbserver.NewServer()
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	if err := serv.ListenTCP(addr); err != nil {
		return fmt.Errorf("start modbus simulator: %w", err)
	}

	s.applyInitialValues(serv)

	ctx, cancel := context.WithCancel(context.Background())
	go s.runSimulation(ctx, serv)

	s.server = serv
	s.cancel = cancel

	s.log.Info("modbus simulator started", zap.String("endpoint", addr))
	return nil
}

func (s *Simulator) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}
	s.server.Close()
	s.server = nil
	s.cancel = nil

	s.log.Info("modbus simulator stopped")
	return nil
}

func (s *Simulator) runSimulation(ctx context.Context, serv *mbserver.Server) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.updateRegisters(serv, t.Sub(start))
		}
	}
}

func (s *Simulator) updateRegisters(serv *mbserver.Server, elapsed time.Duration) {
	seconds := elapsed.Seconds()

	temp := 50.0 + 30.0*math.Sin(seconds/10)
	pressure := 1.0 + rand.Float64()*4.0
	flow := 55.0 + 45.0*math.Sin(seconds/10+math.Pi/2)
	steps := int(seconds) % 31
	motorSpeed := uint16(steps * 100)
	if motorSpeed > 3000 {
		motorSpeed = 3000
	}

	serv.HoldingRegisters[0] = uint16(temp * 100)
	serv.HoldingRegisters[1] = uint16(pressure * 100)
	serv.HoldingRegisters[2] = uint16(flow * 100)
	serv.HoldingRegisters[3] = motorSpeed
	serv.InputRegisters[0] = serv.HoldingRegisters[0]

	if int(seconds)%30 < 3 {
		serv.Coils[0] = 1
		serv.DiscreteInputs[0] = 1
	} else {
		serv.Coils[0] = 0
		serv.DiscreteInputs[0] = 0
	}
}

func (s *Simulator) applyInitialValues(serv *mbserver.Server) {
	for key, val := range s.cfg.HoldingRegisters {
		addr := parseRegisterIndex(key)
		if addr >= 0 && addr < len(serv.HoldingRegisters) {
			serv.HoldingRegisters[addr] = val
		}
	}
	for key, val := range s.cfg.Coils {
		addr := parseRegisterIndex(key)
		if addr >= 0 && addr < len(serv.Coils) {
			if val {
				serv.Coils[addr] = 1
			} else {
				serv.Coils[addr] = 0
			}
		}
	}
}

func parseRegisterIndex(key string) int {
	key = strings.TrimSpace(key)
	if strings.Contains(key, ":") {
		parts := strings.SplitN(key, ":", 2)
		key = parts[1]
	}
	var addr int
	fmt.Sscanf(key, "%d", &addr)
	return addr
}
