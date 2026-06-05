package bacnet

import "testing"

func TestParseObjectRef(t *testing.T) {
	obj, err := ParseObjectRef("analogInput:1")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Type != 0 || obj.Instance != 1 {
		t.Fatalf("unexpected object: %+v", obj)
	}

	obj, err = ParseObjectRef("bi:2")
	if err != nil || obj.Type != 3 || obj.Instance != 2 {
		t.Fatalf("unexpected binary input: %+v err=%v", obj, err)
	}
}

func TestEncodeObjectID(t *testing.T) {
	id := encodeObjectID(0, 1)
	if id != 1 {
		t.Fatalf("expected 1, got %d", id)
	}
}

func TestBuildReadPropertyPacket(t *testing.T) {
	obj := ObjectRef{Type: 0, Instance: 1, Property: propPresentValue}
	pkt := buildReadProperty(obj, 1)
	if len(pkt) < 16 || pkt[0] != 0x81 {
		t.Fatalf("invalid packet: % x", pkt)
	}
}
