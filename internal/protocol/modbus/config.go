package modbus

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	UnitID    uint8 `json:"unit_id"`
	TimeoutMS int   `json:"timeout_ms"`
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{
		UnitID:    1,
		TimeoutMS: 3000,
	}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse modbus config: %w", err)
	}
	return cfg, nil
}

func (c *Config) RequestTimeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 3 * time.Second
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
}
