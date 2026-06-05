package s7

import "testing"

func TestParseAddress(t *testing.T) {
	tests := []struct {
		input    string
		wantArea AreaType
		wantDB   int
		wantOff  int
		wantKind DataKind
		wantBit  int
	}{
		{"db1:0", AreaDB, 1, 0, KindUInt16, 0},
		{"DB1:4.REAL", AreaDB, 1, 4, KindReal, 0},
		{"db2:8.dint", AreaDB, 2, 8, KindInt32, 0},
		{"db1:12.bool.3", AreaDB, 1, 12, KindBool, 3},
		{"m0.0", AreaM, 0, 0, KindBool, 0},
		{"i1.5", AreaI, 0, 1, KindBool, 5},
		{"q2.7", AreaQ, 0, 2, KindBool, 7},
		{"mb10", AreaM, 0, 10, KindByte, 0},
	}

	for _, tc := range tests {
		addr, err := ParseAddress(tc.input)
		if err != nil {
			t.Fatalf("ParseAddress(%q): %v", tc.input, err)
		}
		if addr.Area != tc.wantArea || addr.DBNumber != tc.wantDB || addr.Offset != tc.wantOff || addr.Kind != tc.wantKind || addr.Bit != tc.wantBit {
			t.Fatalf("ParseAddress(%q) = %+v, want area=%d db=%d off=%d kind=%s bit=%d",
				tc.input, addr, tc.wantArea, tc.wantDB, tc.wantOff, tc.wantKind, tc.wantBit)
		}
	}
}

func TestParseAddressInvalid(t *testing.T) {
	if _, err := ParseAddress(""); err == nil {
		t.Fatal("expected error for empty node id")
	}
	if _, err := ParseAddress("invalid"); err == nil {
		t.Fatal("expected error for invalid node id")
	}
}

func TestParseConfig(t *testing.T) {
	cfg, err := ParseConfig(`{"rack":0,"slot":2,"timeout_ms":3000}`)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Slot != 2 || cfg.TimeoutMS != 3000 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParseHost(t *testing.T) {
	if got := parseHost("192.168.0.10:102"); got != "192.168.0.10" {
		t.Fatalf("parseHost = %q", got)
	}
	if got := parseHost("192.168.0.10"); got != "192.168.0.10" {
		t.Fatalf("parseHost = %q", got)
	}
}
