package bacnet

import (
	"encoding/binary"
	"fmt"
	"math"
)

func encodeObjectID(objectType uint16, instance uint32) uint32 {
	return (uint32(objectType) << 22) | (instance & 0x3fffff)
}

func buildReadProperty(object ObjectRef, invokeID byte) []byte {
	objID := encodeObjectID(object.Type, object.Instance)
	apdu := []byte{
		0x00, invokeID, 0x0c,
		0x0c,
		byte(objID >> 24), byte(objID >> 16), byte(objID >> 8), byte(objID),
		0x19, object.Property,
	}
	return wrapBVLC(apdu)
}

func buildWritePropertyPresentValue(object ObjectRef, invokeID byte, payload []byte) []byte {
	objID := encodeObjectID(object.Type, object.Instance)
	apdu := []byte{
		0x00, invokeID, 0x0f,
		0x0c,
		byte(objID >> 24), byte(objID >> 16), byte(objID >> 8), byte(objID),
		0x19, propPresentValue,
		0x3e,
	}
	apdu = append(apdu, payload...)
	apdu = append(apdu, 0x3f)
	return wrapBVLC(apdu)
}

func wrapBVLC(npduApdu []byte) []byte {
	npdu := append([]byte{0x01, 0x04}, npduApdu...)
	length := 4 + len(npdu)
	packet := make([]byte, length)
	packet[0] = 0x81
	packet[1] = 0x0a
	binary.BigEndian.PutUint16(packet[2:4], uint16(length))
	copy(packet[4:], npdu)
	return packet
}

func parsePropertyValue(data []byte) (any, error) {
	for i := 0; i < len(data)-1; i++ {
		switch data[i] {
		case 0x44:
			if i+4 < len(data) {
				bits := binary.BigEndian.Uint32(data[i+1 : i+5])
				return math.Float32frombits(bits), nil
			}
		case 0x91:
			if i+1 < len(data) {
				return data[i+1] != 0, nil
			}
		case 0x21:
			if i+1 < len(data) {
				return data[i+1], nil
			}
		}
	}
	return nil, fmt.Errorf("unable to parse bacnet property value")
}

func encodePresentValue(object ObjectRef, value any) ([]byte, error) {
	switch object.Type {
	case 3, 4, 5:
		b, err := toBool(value)
		if err != nil {
			return nil, err
		}
		v := byte(0)
		if b {
			v = 1
		}
		return []byte{0x91, v}, nil
	default:
		f, err := toFloat32(value)
		if err != nil {
			return nil, err
		}
		buf := make([]byte, 5)
		buf[0] = 0x44
		binary.BigEndian.PutUint32(buf[1:], math.Float32bits(f))
		return buf, nil
	}
}

func toBool(value any) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	case string:
		switch v {
		case "true", "1", "active", "on":
			return true, nil
		case "false", "0", "inactive", "off":
			return false, nil
		}
	}
	return false, fmt.Errorf("invalid bool value %T", value)
}

func toFloat32(value any) (float32, error) {
	switch v := value.(type) {
	case float32:
		return v, nil
	case float64:
		return float32(v), nil
	case int:
		return float32(v), nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("invalid float value %T", value)
	}
}
