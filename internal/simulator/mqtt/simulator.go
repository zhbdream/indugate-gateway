package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"go.uber.org/zap"
)

type SimulatorConfig struct {
	Host   string   `json:"host"`
	Port   int      `json:"port"`
	Topics []string `json:"topics"`
}

func DefaultSimulatorConfig() SimulatorConfig {
	return SimulatorConfig{
		Host: "0.0.0.0",
		Port: 1883,
		Topics: []string{
			"factory/device1/telemetry",
			"factory/device2/telemetry",
		},
	}
}

type SimulatorStatus struct {
	Running  bool     `json:"running"`
	Endpoint string   `json:"endpoint"`
	Port     int      `json:"port"`
	Topics   []string `json:"topics,omitempty"`
}

type Simulator struct {
	log    *zap.Logger
	cfg    SimulatorConfig
	mu     sync.RWMutex
	server *mqtt.Server
	client pahomqtt.Client
	cancel context.CancelFunc
}

func NewSimulator(log *zap.Logger, cfg SimulatorConfig) *Simulator {
	if cfg.Port == 0 {
		cfg.Port = 1883
	}
	if cfg.Host == "" {
		cfg.Host = "0.0.0.0"
	}
	if len(cfg.Topics) == 0 {
		cfg.Topics = DefaultSimulatorConfig().Topics
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
		Endpoint: fmt.Sprintf("tcp://127.0.0.1:%d", s.cfg.Port),
		Port:     s.cfg.Port,
		Topics:   append([]string(nil), s.cfg.Topics...),
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
	if len(cfg.Topics) > 0 {
		s.cfg.Topics = cfg.Topics
	}
	return nil
}

func (s *Simulator) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		return fmt.Errorf("mqtt simulator already running")
	}

	server := mqtt.New(nil)
	if err := server.AddHook(new(auth.AllowHook), nil); err != nil {
		return fmt.Errorf("start mqtt simulator: add auth hook: %w", err)
	}

	tcp := listeners.NewTCP(listeners.Config{
		ID:      "tcp1",
		Address: fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port),
	})
	if err := server.AddListener(tcp); err != nil {
		return fmt.Errorf("start mqtt simulator: add listener: %w", err)
	}

	go func() {
		if err := server.Serve(); err != nil {
			s.log.Error("mqtt simulator serve error", zap.Error(err))
		}
	}()

	broker := fmt.Sprintf("tcp://127.0.0.1:%d", s.cfg.Port)
	opts := pahomqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("indugate-mqtt-simulator").
		SetConnectTimeout(5 * time.Second)
	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		server.Close()
		return fmt.Errorf("start mqtt simulator: connect publisher timeout")
	}
	if err := token.Error(); err != nil {
		server.Close()
		return fmt.Errorf("start mqtt simulator: connect publisher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go s.runSimulation(ctx, client)

	s.server = server
	s.client = client
	s.cancel = cancel

	s.log.Info("mqtt simulator started", zap.String("endpoint", broker))
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
	if s.client != nil {
		s.client.Disconnect(250)
		s.client = nil
	}
	s.server.Close()
	s.server = nil
	s.cancel = nil

	s.log.Info("mqtt simulator stopped")
	return nil
}

func (s *Simulator) runSimulation(ctx context.Context, client pahomqtt.Client) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	start := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.publishTelemetry(client, t.Sub(start))
		}
	}
}

func (s *Simulator) publishTelemetry(client pahomqtt.Client, elapsed time.Duration) {
	seconds := elapsed.Seconds()
	for i, topic := range s.cfg.Topics {
		temp := 50.0 + 30.0*math.Sin(seconds/10+float64(i))
		pressure := 1.0 + rand.Float64()*4.0
		flow := 55.0 + 45.0*math.Sin(seconds/10+math.Pi/2+float64(i))

		payload, _ := json.Marshal(map[string]any{
			"temperature": temp,
			"pressure":    pressure,
			"flow":        flow,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		})
		token := client.Publish(topic, 1, false, payload)
		token.Wait()
	}
}
