package opcua

import (
	"testing"
	"time"

	opcuasim "github.com/indugate/gateway/internal/simulator/opcua"
	"go.uber.org/zap"
)

func TestDriverWithSimulator(t *testing.T) {
	log := zap.NewNop()
	sim := opcuasim.NewSimulator(log, opcuasim.SimulatorConfig{
		Host: "127.0.0.1",
		Port: 14840,
	})
	if err := sim.Start(); err != nil {
		t.Fatal(err)
	}
	defer sim.Stop()

	time.Sleep(500 * time.Millisecond)

	status := sim.Status()
	if !status.Running {
		t.Fatal("simulator not running")
	}

	driver := NewDriver(status.Endpoint, &Config{RequestTimeoutMS: 5000})
	if err := driver.Connect(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer driver.Disconnect(t.Context())

	nodes, err := driver.Browse(t.Context(), "i=85", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected browse nodes")
	}

	nodeID := status.NodeIDs[0]
	if nodeID == "" {
		t.Fatal("simulator did not expose node ids")
	}

	val, err := driver.Read(t.Context(), nodeID)
	if err != nil {
		t.Fatal(err)
	}
	if val.Value == nil {
		t.Fatal("expected node value")
	}
}
