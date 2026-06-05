package bacnet

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	DeviceID       uint32 `json:"device_id"`
	TimeoutMS      int    `json:"timeout_ms"`
	COVEnabled     bool   `json:"cov_enabled"`
	COVLifetimeSec int    `json:"cov_lifetime_sec"`
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{
		DeviceID:       1,
		TimeoutMS:      3000,
		COVLifetimeSec: 300,
	}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse bacnet config: %w", err)
	}
	return cfg, nil
}

func (c *Config) RequestTimeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 3 * time.Second
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

func parseTarget(address string) (host string, port int, err error) {
	port = 47808
	if address == "" {
		return "", port, fmt.Errorf("empty bacnet address")
	}
	for i := len(address) - 1; i >= 0; i-- {
		if address[i] == ':' {
			host = address[:i]
			if _, err := fmt.Sscanf(address[i+1:], "%d", &port); err != nil {
				return "", port, fmt.Errorf("invalid bacnet port")
			}
			return host, port, nil
		}
	}
	return address, port, nil
}
