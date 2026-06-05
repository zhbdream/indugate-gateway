package bacnet

import "testing"

func TestBuildSubscribeCOV(t *testing.T) {
	obj := ObjectRef{Type: 0, Instance: 1, Property: propPresentValue}
	pkt := buildSubscribeCOV(obj, 1, 300, 2)
	if len(pkt) < 16 || pkt[0] != 0x81 || pkt[8] != 0x05 {
		t.Fatalf("invalid subscribe cov packet: % x", pkt)
	}
}

func TestBuildSubscribeCOVCancel(t *testing.T) {
	pkt := buildSubscribeCOVCancel(1, 3)
	if len(pkt) < 10 || pkt[8] != 0x06 {
		t.Fatalf("invalid cancel cov packet: % x", pkt)
	}
}
