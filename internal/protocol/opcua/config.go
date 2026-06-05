package opcua

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	SecurityPolicy   string `json:"security_policy"`
	SecurityMode     string `json:"security_mode"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	CertFile         string `json:"cert_file"`
	KeyFile          string `json:"key_file"`
	RequestTimeoutMS int    `json:"request_timeout_ms"`
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{
		SecurityPolicy:   "None",
		SecurityMode:     "None",
		RequestTimeoutMS: 5000,
	}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse opcua config: %w", err)
	}
	return cfg, nil
}

func (c *Config) RequestTimeout() time.Duration {
	if c.RequestTimeoutMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}
