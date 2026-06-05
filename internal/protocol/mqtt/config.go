package mqtt

import (
	"encoding/json"
	"fmt"
	"time"
)

type WillConfig struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
	QoS     byte   `json:"qos"`
	Retain  bool   `json:"retain"`
}

type Config struct {
	ClientID         string      `json:"client_id"`
	QoS              byte        `json:"qos"`
	Topics           []string    `json:"topics"`
	Username         string      `json:"username"`
	Password         string      `json:"password"`
	KeepAliveSec     int         `json:"keep_alive_sec"`
	CleanSession     bool        `json:"clean_session"`
	RequestTimeoutMS int         `json:"request_timeout_ms"`
	Will             *WillConfig `json:"will"`
}

func ParseConfig(raw string) (*Config, error) {
	cfg := &Config{
		ClientID:         "indugate-gateway",
		QoS:              1,
		KeepAliveSec:     60,
		CleanSession:     true,
		RequestTimeoutMS: 5000,
	}
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil, fmt.Errorf("parse mqtt config: %w", err)
	}
	return cfg, nil
}

func (c *Config) RequestTimeout() time.Duration {
	if c.RequestTimeoutMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.RequestTimeoutMS) * time.Millisecond
}
