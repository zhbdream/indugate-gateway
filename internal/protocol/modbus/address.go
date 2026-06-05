package modbus

import (
	"fmt"
	"strconv"
	"strings"
)

type RegisterType int

const (
	RegisterCoil RegisterType = iota
	RegisterDiscrete
	RegisterInput
	RegisterHolding
)

type Address struct {
	Type RegisterType
	Addr uint16
}

func ParseAddress(nodeID string) (Address, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return Address{}, fmt.Errorf("empty node id")
	}

	parts := strings.SplitN(nodeID, ":", 2)
	if len(parts) == 2 {
		addr, err := parseUint16(parts[1])
		if err != nil {
			return Address{}, fmt.Errorf("invalid address in %q: %w", nodeID, err)
		}
		switch strings.ToLower(parts[0]) {
		case "coil", "0x":
			return Address{Type: RegisterCoil, Addr: addr}, nil
		case "discrete", "1x":
			return Address{Type: RegisterDiscrete, Addr: addr}, nil
		case "input", "3x":
			return Address{Type: RegisterInput, Addr: addr}, nil
		case "holding", "4x":
			return Address{Type: RegisterHolding, Addr: addr}, nil
		}
	}

	if len(nodeID) >= 2 {
		prefix := strings.ToLower(nodeID[:2])
		addrStr := nodeID[2:]
		if addr, err := parseUint16(addrStr); err == nil {
			switch prefix {
			case "0x":
				return Address{Type: RegisterCoil, Addr: addr}, nil
			case "1x":
				return Address{Type: RegisterDiscrete, Addr: addr}, nil
			case "3x":
				return Address{Type: RegisterInput, Addr: addr}, nil
			case "4x":
				return Address{Type: RegisterHolding, Addr: addr}, nil
			}
		}
	}

	return Address{}, fmt.Errorf("invalid modbus node id %q, expected coil:0, holding:0, 4x0, etc.", nodeID)
}

func parseUint16(s string) (uint16, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty address")
	}
	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}

func (a Address) String() string {
	switch a.Type {
	case RegisterCoil:
		return fmt.Sprintf("coil:%d", a.Addr)
	case RegisterDiscrete:
		return fmt.Sprintf("discrete:%d", a.Addr)
	case RegisterInput:
		return fmt.Sprintf("input:%d", a.Addr)
	case RegisterHolding:
		return fmt.Sprintf("holding:%d", a.Addr)
	default:
		return fmt.Sprintf("unknown:%d", a.Addr)
	}
}

func dataTypeFor(addr Address) string {
	switch addr.Type {
	case RegisterCoil, RegisterDiscrete:
		return "bool"
	default:
		return "uint16"
	}
}
