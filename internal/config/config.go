// Package config loads and validates application configuration from
// config.yaml (via Viper) with environment variable overrides.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root application configuration.
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Telegram  TelegramConfig  `mapstructure:"telegram"`
	Monitor   MonitorConfig   `mapstructure:"monitor"`
	SpeedTest SpeedTestConfig `mapstructure:"speedtest"`
	Retention RetentionConfig `mapstructure:"retention"`
	Logging   LoggingConfig   `mapstructure:"logging"`
}

// ServerConfig configures the HTTP API server.
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
}

// DatabaseConfig configures the embedded bbolt database.
type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

// TelegramConfig configures Telegram bot notifications.
type TelegramConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	BotToken string `mapstructure:"bot_token"`
	ChatID   string `mapstructure:"chat_id"`
}

// MonitorConfig configures the connectivity monitor.
type MonitorConfig struct {
	IntervalSeconds  int    `mapstructure:"interval_seconds"`
	HTTPCheckURL     string `mapstructure:"http_check_url"`
	PingHost         string `mapstructure:"ping_host"`
	DNSHost          string `mapstructure:"dns_host"`
	TimeoutSeconds   int    `mapstructure:"timeout_seconds"`
	FailureThreshold int    `mapstructure:"failure_threshold"`
}

// SpeedTestConfig configures periodic speed tests.
type SpeedTestConfig struct {
	IntervalHours int    `mapstructure:"interval_hours"`
	Provider      string `mapstructure:"provider"`
}

// RetentionConfig configures data retention/cleanup.
type RetentionConfig struct {
	RawDataDays     int           `mapstructure:"raw_data_days"`
	CleanupInterval time.Duration `mapstructure:"cleanup_interval"`
}

// LoggingConfig configures the Zap logger.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	Path  string `mapstructure:"path"`
}

// Load reads configuration from the given path (directory or file) and env vars.
// Environment variables use the prefix NETWATCH and "_" as the nested separator,
// e.g. NETWATCH_SERVER_PORT overrides server.port.
func Load(configPath string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	setDefaults(v)

	v.SetEnvPrefix("NETWATCH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("server.cors_origins", []string{"*"})

	v.SetDefault("database.path", "./data/netwatch.db")

	v.SetDefault("telegram.enabled", false)

	v.SetDefault("monitor.interval_seconds", 15)
	v.SetDefault("monitor.http_check_url", "https://google.com")
	v.SetDefault("monitor.ping_host", "8.8.8.8")
	v.SetDefault("monitor.dns_host", "google.com")
	v.SetDefault("monitor.timeout_seconds", 5)
	v.SetDefault("monitor.failure_threshold", 2)

	v.SetDefault("speedtest.interval_hours", 6)
	v.SetDefault("speedtest.provider", "ookla")

	v.SetDefault("retention.raw_data_days", 90)
	v.SetDefault("retention.cleanup_interval", "24h")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.path", "")
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Database.Path == "" {
		return fmt.Errorf("database.path must not be empty")
	}
	if c.Monitor.IntervalSeconds <= 0 {
		return fmt.Errorf("monitor.interval_seconds must be positive")
	}
	if c.SpeedTest.IntervalHours <= 0 {
		return fmt.Errorf("speedtest.interval_hours must be positive")
	}
	if c.Retention.RawDataDays <= 0 {
		return fmt.Errorf("retention.raw_data_days must be positive")
	}
	return nil
}
