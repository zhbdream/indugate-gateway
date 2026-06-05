package mqtt

import (
	"testing"
	"time"

	mqttsim "github.com/indugate/gateway/internal/simulator/mqtt"
	"go.uber.org/zap"
)

func TestDriverWithSimulator(t *testing.T) {
	log := zap.NewNop()
	sim := mqttsim.NewSimulator(log, mqttsim.SimulatorConfig{
		Host: "127.0.0.1",
		Port: 11883,
		Topics: []string{
			"factory/device1/telemetry",
		},
	})
	if err := sim.Start(); err != nil {
		t.Fatal(err)
	}
	defer sim.Stop()

	time.Sleep(500 * time.Millisecond)

	driver := NewDriver("tcp://127.0.0.1:11883", &Config{
		ClientID: "test-client",
		QoS:      1,
		Topics:   []string{"factory/device1/telemetry"},
	})
	if err := driver.Connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer driver.Disconnect(t.Context())

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		val, err := driver.Read(t.Context(), "factory/device1/telemetry")
		if err == nil && val.Value != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("expected telemetry message on topic")
}
