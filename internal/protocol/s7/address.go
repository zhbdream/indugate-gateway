package s7

import (
	"fmt"
	"strconv"
	"strings"
)

type AreaType int

const (
	AreaDB AreaType = iota
	AreaM
	AreaI
	AreaQ
)

type DataKind string

const (
	KindUInt16 DataKind = "uint16"
	KindInt16  DataKind = "int16"
	KindInt32  DataKind = "dint"
	KindReal   DataKind = "real"
	KindBool   DataKind = "bool"
	KindByte   DataKind = "byte"
)

type Address struct {
	Area     AreaType
	DBNumber int
	Offset   int
	Bit      int
	Kind     DataKind
}

func (a Address) String() string {
	switch a.Area {
	case AreaDB:
		if a.Kind == KindBool {
			return fmt.Sprintf("db%d:%d.bool.%d", a.DBNumber, a.Offset, a.Bit)
		}
		if a.Kind != KindUInt16 {
			return fmt.Sprintf("db%d:%d.%s", a.DBNumber, a.Offset, a.Kind)
		}
		return fmt.Sprintf("db%d:%d", a.DBNumber, a.Offset)
	case AreaM:
		if a.Kind == KindByte {
			return fmt.Sprintf("mb%d", a.Offset)
		}
		return fmt.Sprintf("m%d.%d", a.Offset, a.Bit)
	case AreaI:
		return fmt.Sprintf("i%d.%d", a.Offset, a.Bit)
	case AreaQ:
		return fmt.Sprintf("q%d.%d", a.Offset, a.Bit)
	default:
		return "unknown"
	}
}

func (a Address) ReadSize() int {
	switch a.Kind {
	case KindReal, KindInt32:
		return 4
	case KindBool, KindByte:
		return 1
	default:
		return 2
	}
}

func (a Address) Writable() bool {
	switch a.Area {
	case AreaI:
		return false
	default:
		return true
	}
}

func ParseAddress(nodeID string) (Address, error) {
	nodeID = strings.TrimSpace(strings.ToLower(nodeID))
	if nodeID == "" {
		return Address{}, fmt.Errorf("empty node id")
	}

	if strings.HasPrefix(nodeID, "db") {
		return parseDBAddress(nodeID)
	}
	if strings.HasPrefix(nodeID, "mb") {
		offset, err := strconv.Atoi(strings.TrimPrefix(nodeID, "mb"))
		if err != nil {
			return Address{}, fmt.Errorf("invalid merker byte address %q", nodeID)
		}
		return Address{Area: AreaM, Offset: offset, Kind: KindByte}, nil
	}
	if len(nodeID) >= 2 && nodeID[0] == 'm' && nodeID[1] >= '0' && nodeID[1] <= '9' {
		return parseBitAddress(nodeID, AreaM)
	}
	if len(nodeID) >= 2 && nodeID[0] == 'i' {
		return parseBitAddress(nodeID, AreaI)
	}
	if len(nodeID) >= 2 && nodeID[0] == 'q' {
		return parseBitAddress(nodeID, AreaQ)
	}

	return Address{}, fmt.Errorf("invalid s7 node id %q, expected db1:0, m0.0, mb0, etc.", nodeID)
}

func parseDBAddress(nodeID string) (Address, error) {
	rest := strings.TrimPrefix(nodeID, "db")
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return Address{}, fmt.Errorf("invalid db address %q", nodeID)
	}
	dbNum, err := strconv.Atoi(parts[0])
	if err != nil || dbNum < 0 {
		return Address{}, fmt.Errorf("invalid db number in %q", nodeID)
	}

	offsetPart := parts[1]
	kind := KindUInt16
	bit := 0
	if strings.Contains(offsetPart, ".") {
		segments := strings.Split(offsetPart, ".")
		offset, err := strconv.Atoi(segments[0])
		if err != nil {
			return Address{}, fmt.Errorf("invalid db offset in %q", nodeID)
		}
		switch segments[1] {
		case "real", "dint", "int", "bool", "byte":
			kind = DataKind(segments[1])
		default:
			return Address{}, fmt.Errorf("unknown db data type in %q", nodeID)
		}
		if kind == KindBool {
			if len(segments) < 3 {
				return Address{}, fmt.Errorf("bool address requires bit index: db1:0.bool.0")
			}
			bit, err = strconv.Atoi(segments[2])
			if err != nil || bit < 0 || bit > 7 {
				return Address{}, fmt.Errorf("invalid bit index in %q", nodeID)
			}
		}
		return Address{Area: AreaDB, DBNumber: dbNum, Offset: offset, Kind: kind, Bit: bit}, nil
	}

	offset, err := strconv.Atoi(offsetPart)
	if err != nil {
		return Address{}, fmt.Errorf("invalid db offset in %q", nodeID)
	}
	return Address{Area: AreaDB, DBNumber: dbNum, Offset: offset, Kind: kind}, nil
}

func parseBitAddress(nodeID string, area AreaType) (Address, error) {
	parts := strings.Split(nodeID, ".")
	if len(parts) != 2 {
		return Address{}, fmt.Errorf("invalid bit address %q", nodeID)
	}
	byteOffset, err := strconv.Atoi(parts[0][1:])
	if err != nil {
		return Address{}, fmt.Errorf("invalid byte offset in %q", nodeID)
	}
	bit, err := strconv.Atoi(parts[1])
	if err != nil || bit < 0 || bit > 7 {
		return Address{}, fmt.Errorf("invalid bit index in %q", nodeID)
	}
	return Address{Area: area, Offset: byteOffset, Bit: bit, Kind: KindBool}, nil
}

func dataTypeFor(addr Address) string {
	return string(addr.Kind)
}
