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
	InfluxDB  InfluxDBConfig  `mapstructure:"influxdb"`
	History   HistoryConfig   `mapstructure:"history"`
	Alerts    AlertConfig     `mapstructure:"alerts"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Audit     AuditConfig     `mapstructure:"audit"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	MCP       MCPConfig       `mapstructure:"mcp"`
	Simulator SimulatorConfig `mapstructure:"simulator"`
	Web       WebConfig       `mapstructure:"web"`
	Bootstrap BootstrapConfig `mapstructure:"bootstrap"`
}

type HistoryConfig struct {
	RetentionDays        int `mapstructure:"retention_days"`
	CleanupIntervalHours int `mapstructure:"cleanup_interval_hours"`
}

type AlertConfig struct {
	WebhookURL   string `mapstructure:"webhook_url"`
	MQTTEnabled  bool   `mapstructure:"mqtt_enabled"`
	MQTTBroker   string `mapstructure:"mqtt_broker"`
	MQTTTopic    string `mapstructure:"mqtt_topic"`
	MQTTClientID string `mapstructure:"mqtt_client_id"`
}

type AuthConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	RBACEnabled      bool   `mapstructure:"rbac_enabled"`
	DeviceACLEnabled bool   `mapstructure:"device_acl_enabled"`
	APIToken         string `mapstructure:"api_token"`
	JWTSecret        string `mapstructure:"jwt_secret"`
	JWTExpireHours   int    `mapstructure:"jwt_expire_hours"`
	DefaultAdminUser string `mapstructure:"default_admin_user"`
	DefaultAdminPass string `mapstructure:"default_admin_password"`
}

type AuditConfig struct {
	Enabled              bool `mapstructure:"enabled"`
	RetentionDays        int  `mapstructure:"retention_days"`
	CleanupIntervalHours int  `mapstructure:"cleanup_interval_hours"`
}

type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

type InfluxDBConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
	Token   string `mapstructure:"token"`
	Org     string `mapstructure:"org"`
	Bucket  string `mapstructure:"bucket"`
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

type WebConfig struct {
	StaticDir string `mapstructure:"static_dir"`
}

type BootstrapConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	DevicesFile string `mapstructure:"devices_file"`
}

type SimulatorConfig struct {
	OPCUA  OPCUASimulatorConfig  `mapstructure:"opcua"`
	Modbus ModbusSimulatorConfig `mapstructure:"modbus"`
	MQTT   MQTTSimulatorConfig   `mapstructure:"mqtt"`
}

type OPCUASimulatorConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	AutoStart bool   `mapstructure:"auto_start"`
}

type ModbusSimulatorConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	AutoStart bool   `mapstructure:"auto_start"`
}

type MQTTSimulatorConfig struct {
	Host      string   `mapstructure:"host"`
	Port      int      `mapstructure:"port"`
	AutoStart bool     `mapstructure:"auto_start"`
	Topics    []string `mapstructure:"topics"`
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

	v.SetDefault("web.static_dir", "./web/dist")

	v.SetDefault("bootstrap.enabled", true)
	v.SetDefault("bootstrap.devices_file", "./configs/devices.yaml")

	v.SetDefault("influxdb.enabled", false)
	v.SetDefault("influxdb.url", "http://localhost:8086")
	v.SetDefault("influxdb.org", "indugate")
	v.SetDefault("influxdb.bucket", "telemetry")

	v.SetDefault("history.retention_days", 30)
	v.SetDefault("history.cleanup_interval_hours", 24)

	v.SetDefault("alerts.mqtt_topic", "indugate/alerts")
	v.SetDefault("alerts.mqtt_client_id", "indugate-alert-notifier")

	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.rbac_enabled", true)
	v.SetDefault("auth.device_acl_enabled", false)
	v.SetDefault("auth.jwt_expire_hours", 24)
	v.SetDefault("auth.default_admin_user", "admin")
	v.SetDefault("auth.default_admin_password", "admin123")

	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.retention_days", 90)
	v.SetDefault("audit.cleanup_interval_hours", 24)

	v.SetDefault("metrics.enabled", false)

	v.SetDefault("simulator.opcua.host", "0.0.0.0")
	v.SetDefault("simulator.opcua.port", 4840)
	v.SetDefault("simulator.opcua.auto_start", false)

	v.SetDefault("simulator.modbus.host", "0.0.0.0")
	v.SetDefault("simulator.modbus.port", 502)
	v.SetDefault("simulator.modbus.auto_start", false)

	v.SetDefault("simulator.mqtt.host", "0.0.0.0")
	v.SetDefault("simulator.mqtt.port", 1883)
	v.SetDefault("simulator.mqtt.auto_start", false)
	v.SetDefault("simulator.mqtt.topics", []string{
		"factory/device1/telemetry",
		"factory/device2/telemetry",
	})
}
