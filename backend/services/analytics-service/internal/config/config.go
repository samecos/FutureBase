package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	ClickHouse ClickHouseConfig `yaml:"clickhouse"`
	Redis      RedisConfig      `yaml:"redis"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Analytics  AnalyticsConfig  `yaml:"analytics"`
	Reports    ReportsConfig    `yaml:"reports"`
	Logging    LoggingConfig    `yaml:"logging"`
}

type ServerConfig struct {
	HTTPPort int `yaml:"http_port"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	Schema   string `yaml:"schema"`
	SSLMode  string `yaml:"ssl_mode"`
}

type ClickHouseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Debug    bool   `yaml:"debug"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	GroupID string   `yaml:"group_id"`
}

type AnalyticsConfig struct {
	RetentionDays              int      `yaml:"retention_days"`
	BatchSize                  int      `yaml:"batch_size"`
	AggregationIntervalMinutes int      `yaml:"aggregation_interval_minutes"`
	Metrics                    []string `yaml:"metrics"`
}

type ReportsConfig struct {
	Enabled       bool   `yaml:"enabled"`
	TemplatesDir  string `yaml:"templates_dir"`
	DefaultFormat string `yaml:"default_format"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	overrideWithEnv(&cfg)
	return &cfg, nil
}

func overrideWithEnv(cfg *Config) {
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.Database.Host = host
	}
	if port := os.Getenv("DB_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Database.Port)
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.Database.Password = password
	}
	if chHost := os.Getenv("CLICKHOUSE_HOST"); chHost != "" {
		cfg.ClickHouse.Host = chHost
	}
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Schema)
}
