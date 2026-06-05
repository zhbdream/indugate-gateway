package modbus

import (
	"testing"
)

func TestParseConfigDefaults(t *testing.T) {
	cfg, err := ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UnitID != 1 {
		t.Fatalf("expected unit id 1, got %d", cfg.UnitID)
	}
	if cfg.TimeoutMS != 3000 {
		t.Fatalf("expected 3000 timeout, got %d", cfg.TimeoutMS)
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		nodeID   string
		wantType RegisterType
		wantAddr uint16
	}{
		{"holding:0", RegisterHolding, 0},
		{"coil:10", RegisterCoil, 10},
		{"4x100", RegisterHolding, 100},
		{"1x5", RegisterDiscrete, 5},
	}

	for _, tt := range tests {
		addr, err := ParseAddress(tt.nodeID)
		if err != nil {
			t.Fatalf("parse %q: %v", tt.nodeID, err)
		}
		if addr.Type != tt.wantType || addr.Addr != tt.wantAddr {
			t.Fatalf("parse %q: got %+v, want type=%d addr=%d", tt.nodeID, addr, tt.wantType, tt.wantAddr)
		}
	}
}

func TestDriverNotConnected(t *testing.T) {
	d := NewDriver("127.0.0.1:502", nil)
	if d.IsConnected() {
		t.Fatal("expected disconnected")
	}
	_, err := d.Read(t.Context(), "holding:0")
	if err != ErrNotConnected {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}
