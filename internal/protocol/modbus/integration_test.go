package modbus

import (
	"testing"
	"time"

	modbussim "github.com/indugate/gateway/internal/simulator/modbus"
	"go.uber.org/zap"
)

func TestDriverWithSimulator(t *testing.T) {
	log := zap.NewNop()
	sim := modbussim.NewSimulator(log, modbussim.SimulatorConfig{
		Host: "127.0.0.1",
		Port: 1502,
	})
	if err := sim.Start(); err != nil {
		t.Fatal(err)
	}
	defer sim.Stop()

	time.Sleep(200 * time.Millisecond)

	driver := NewDriver("127.0.0.1:1502", &Config{UnitID: 1, TimeoutMS: 3000})
	if err := driver.Connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer driver.Disconnect(t.Context())

	val, err := driver.Read(t.Context(), "holding:0")
	if err != nil {
		t.Fatal(err)
	}
	if val.Value == nil {
		t.Fatal("expected holding register value")
	}

	nodes, err := driver.Browse(t.Context(), "", 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected register catalog")
	}
}
