package service

import "testing"

func TestDeriveAuditAction(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{"POST", "/api/v1/auth/login", "auth.login"},
		{"POST", "/api/v1/devices/1/connect", "device.connect"},
		{"POST", "/api/v1/devices/1/data/holding:0", "device.write"},
		{"POST", "/api/v1/alerts/events/3/acknowledge", "alert.event.acknowledge"},
		{"PUT", "/api/v1/simulators/modbus/config", "simulator.config"},
	}

	for _, tc := range cases {
		got := DeriveAuditAction(tc.method, tc.path)
		if got != tc.want {
			t.Fatalf("DeriveAuditAction(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}
