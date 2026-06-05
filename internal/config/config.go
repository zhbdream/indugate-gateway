package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Log       LogConfig       `mapstructure:"log"`
	Database  DatabaseConfig  `mapstructure:"database"`
	MCP       MCPConfig       `mapstructure:"mcp"`
	Simulator SimulatorConfig `mapstructure:"simulator"`
}

type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Mode         string `mapstructure:"mode"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	DSN             string `mapstructure:"dsn"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

type MCPConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BasePath string `mapstructure:"base_path"`
}

type SimulatorConfig struct {
	OPCUA OPCUASimulatorConfig `mapstructure:"opcua"`
}

type OPCUASimulatorConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	AutoStart bool   `mapstructure:"auto_start"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("INDUGATE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")
	v.SetDefault("server.read_timeout", 30)
	v.SetDefault("server.write_timeout", 30)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.output_path", "stdout")

	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "./data/indugate.db")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", 3600)

	v.SetDefault("mcp.enabled", true)
	v.SetDefault("mcp.base_path", "/mcp")

	v.SetDefault("simulator.opcua.host", "0.0.0.0")
	v.SetDefault("simulator.opcua.port", 4840)
	v.SetDefault("simulator.opcua.auto_start", false)
}
