package mqtt

import (
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "indugate-gateway" {
		t.Fatalf("expected default client id, got %s", cfg.ClientID)
	}
	if cfg.QoS != 1 {
		t.Fatalf("expected qos 1, got %d", cfg.QoS)
	}
}

func TestNormalizeBroker(t *testing.T) {
	if got := normalizeBroker("127.0.0.1:1883"); got != "tcp://127.0.0.1:1883" {
		t.Fatalf("unexpected broker: %s", got)
	}
	if got := normalizeBroker("tcp://localhost:1883"); got != "tcp://localhost:1883" {
		t.Fatalf("unexpected broker: %s", got)
	}
}

func TestDriverNotConnected(t *testing.T) {
	d := NewDriver("tcp://127.0.0.1:1883", nil)
	if d.IsConnected() {
		t.Fatal("expected disconnected")
	}
	_, err := d.Read(t.Context(), "factory/device1/telemetry")
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}
