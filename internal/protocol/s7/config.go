package s7

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Config struct {
	Rack      int `json:"rack"`
	Slot      int `json:"slot"`
	TimeoutMS int `json:"timeout_ms"`
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{
		Rack:      0,
		Slot:      1,
		TimeoutMS: 5000,
	}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse s7 config: %w", err)
	}
	return cfg, nil
}

func (c *Config) RequestTimeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

func parseHost(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}
