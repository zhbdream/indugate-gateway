package bacnet

import (
	"encoding/binary"
	"fmt"
)

func buildSubscribeCOV(object ObjectRef, processID uint32, lifetime uint32, invokeID byte) []byte {
	objID := encodeObjectID(object.Type, object.Instance)
	apdu := []byte{
		0x00, invokeID, 0x05,
		0x09, byte(processID & 0xff),
		0x1c,
		byte(objID >> 24), byte(objID >> 16), byte(objID >> 8), byte(objID),
	}
	if lifetime > 0 {
		if lifetime <= 255 {
			apdu = append(apdu, 0x39, byte(lifetime))
		} else {
			apdu = append(apdu, 0x3a, byte(lifetime>>8), byte(lifetime))
		}
	}
	return wrapBVLC(apdu)
}

func buildSubscribeCOVCancel(processID uint32, invokeID byte) []byte {
	apdu := []byte{
		0x00, invokeID, 0x06,
		0x09, byte(processID & 0xff),
	}
	return wrapBVLC(apdu)
}

func parseCOVNotification(data []byte) (ObjectRef, any, bool) {
	if len(data) < 10 {
		return ObjectRef{}, nil, false
	}
	apduOffset := 6
	if data[apduOffset] != 0x10 {
		return ObjectRef{}, nil, false
	}
	if len(data) <= apduOffset+1 || data[apduOffset+1] != 0x02 {
		return ObjectRef{}, nil, false
	}

	objID := uint32(0)
	for i := apduOffset + 2; i < len(data)-3; i++ {
		if data[i] == 0x1c || data[i] == 0x0c {
			objID = binary.BigEndian.Uint32(data[i+1 : i+5])
			break
		}
	}
	if objID == 0 {
		return ObjectRef{}, nil, false
	}
	object := ObjectRef{
		Type:     uint16(objID >> 22),
		Instance: objID & 0x3fffff,
		Property: propPresentValue,
	}
	value, err := parsePropertyValue(data)
	if err != nil {
		return object, nil, true
	}
	return object, value, true
}

func parseSimpleACK(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("short bacnet ack")
	}
	if data[6] == 0x20 {
		return nil
	}
	return nil
}
