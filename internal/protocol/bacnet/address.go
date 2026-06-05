package bacnet

import (
	"fmt"
	"strconv"
	"strings"
)

const propPresentValue = 85

type ObjectRef struct {
	Type     uint16
	Instance uint32
	Property uint8
}

func (o ObjectRef) String() string {
	name, ok := objectTypeName(o.Type)
	if !ok {
		return fmt.Sprintf("%d:%d", o.Type, o.Instance)
	}
	return fmt.Sprintf("%s:%d", name, o.Instance)
}

func ParseObjectRef(nodeID string) (ObjectRef, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return ObjectRef{}, fmt.Errorf("empty node id")
	}

	parts := strings.Split(nodeID, ".")
	objPart := parts[0]
	prop := uint8(propPresentValue)
	if len(parts) > 1 {
		switch strings.ToLower(parts[1]) {
		case "presentvalue", "present_value", "value":
			prop = propPresentValue
		default:
			return ObjectRef{}, fmt.Errorf("unsupported property %q", parts[1])
		}
	}

	seg := strings.Split(objPart, ":")
	if len(seg) != 2 {
		return ObjectRef{}, fmt.Errorf("invalid bacnet node id %q, expected analogInput:1", nodeID)
	}

	objType, err := parseObjectType(seg[0])
	if err != nil {
		return ObjectRef{}, err
	}
	instance, err := strconv.ParseUint(seg[1], 10, 32)
	if err != nil {
		return ObjectRef{}, fmt.Errorf("invalid instance in %q", nodeID)
	}
	return ObjectRef{Type: objType, Instance: uint32(instance), Property: prop}, nil
}

func parseObjectType(name string) (uint16, error) {
	switch strings.ToLower(name) {
	case "analoginput", "ai", "0":
		return 0, nil
	case "analogoutput", "ao", "1":
		return 1, nil
	case "analogvalue", "av", "2":
		return 2, nil
	case "binaryinput", "bi", "3":
		return 3, nil
	case "binaryoutput", "bo", "4":
		return 4, nil
	case "binaryvalue", "bv", "5":
		return 5, nil
	case "multistateinput", "mi", "13":
		return 13, nil
	case "multistateoutput", "mo", "14":
		return 14, nil
	case "multistatevalue", "mv", "19":
		return 19, nil
	default:
		if n, err := strconv.ParseUint(name, 10, 16); err == nil {
			return uint16(n), nil
		}
		return 0, fmt.Errorf("unknown object type %q", name)
	}
}

func objectTypeName(t uint16) (string, bool) {
	switch t {
	case 0:
		return "analogInput", true
	case 1:
		return "analogOutput", true
	case 2:
		return "analogValue", true
	case 3:
		return "binaryInput", true
	case 4:
		return "binaryOutput", true
	case 5:
		return "binaryValue", true
	default:
		return "", false
	}
}

func dataTypeFor(obj ObjectRef) string {
	switch obj.Type {
	case 3, 4, 5:
		return "bool"
	case 13, 14, 19:
		return "uint32"
	default:
		return "float"
	}
}
