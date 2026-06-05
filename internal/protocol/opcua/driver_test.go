package opcua

import (
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SecurityPolicy != "None" {
		t.Fatalf("expected None policy, got %s", cfg.SecurityPolicy)
	}
	if cfg.RequestTimeoutMS != 5000 {
		t.Fatalf("expected 5000 timeout, got %d", cfg.RequestTimeoutMS)
	}
}

func TestParseConfigJSON(t *testing.T) {
	cfg, err := ParseConfig(`{"security_policy":"Basic256Sha256","security_mode":"Sign","request_timeout_ms":3000}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SecurityPolicy != "Basic256Sha256" {
		t.Fatalf("unexpected policy: %s", cfg.SecurityPolicy)
	}
	if cfg.RequestTimeout().Milliseconds() != 3000 {
		t.Fatalf("unexpected timeout: %v", cfg.RequestTimeout())
	}
}

func TestDriverNotConnected(t *testing.T) {
	d := NewDriver("opc.tcp://localhost:4840", nil)
	if d.IsConnected() {
		t.Fatal("expected disconnected")
	}
	_, err := d.Read(t.Context(), "ns=1;s=Temperature")
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}
