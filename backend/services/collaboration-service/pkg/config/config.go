package config

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Service  ServiceConfig  `mapstructure:"service"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Server   ServerConfig   `mapstructure:"server"`
	Log      LogConfig      `mapstructure:"log"`
	Session  SessionConfig  `mapstructure:"session"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL             string        `mapstructure:"url"`
	ReconnectWait   time.Duration `mapstructure:"reconnect_wait"`
	MaxReconnects   int           `mapstructure:"max_reconnects"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout"`
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers         []string      `mapstructure:"brokers"`
	Topic           string        `mapstructure:"topic"`
	ConsumerGroup   string        `mapstructure:"consumer_group"`
}

// ServerConfig holds gRPC and HTTP server configuration
type ServerConfig struct {
	GRPCPort            int           `mapstructure:"grpc_port"`
	HTTPPort            int           `mapstructure:"http_port"`
	WebSocketPort       int           `mapstructure:"websocket_port"`
	ReadTimeout         time.Duration `mapstructure:"read_timeout"`
	WriteTimeout        time.Duration `mapstructure:"write_timeout"`
	MaxConnectionIdle   time.Duration `mapstructure:"max_connection_idle"`
	MaxConnectionAge    time.Duration `mapstructure:"max_connection_age"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// SessionConfig holds session configuration
type SessionConfig struct {
	TTL              time.Duration `mapstructure:"ttl"`
	CleanupInterval  time.Duration `mapstructure:"cleanup_interval"`
	MaxParticipants  int           `mapstructure:"max_participants"`
	RateLimitPerSec  int           `mapstructure:"rate_limit_per_sec"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(path string) (*Config, error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Set defaults
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Println("Config file not found, using environment variables and defaults")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Service defaults
	viper.SetDefault("service.name", "collaboration-service")
	viper.SetDefault("service.version", "1.0.0")
	viper.SetDefault("service.environment", "development")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.dbname", "archdesign_platform")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 100)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.reconnect_wait", "1s")
	viper.SetDefault("nats.max_reconnects", 10)
	viper.SetDefault("nats.connect_timeout", "5s")

	// Kafka defaults
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "collaboration-events")
	viper.SetDefault("kafka.consumer_group", "collaboration-service")

	// Server defaults
	viper.SetDefault("server.grpc_port", 50051)
	viper.SetDefault("server.http_port", 8081)
	viper.SetDefault("server.websocket_port", 8082)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.max_connection_idle", "15m")
	viper.SetDefault("server.max_connection_age", "1h")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	// Session defaults
	viper.SetDefault("session.ttl", "24h")
	viper.SetDefault("session.cleanup_interval", "5m")
	viper.SetDefault("session.max_participants", 100)
	viper.SetDefault("session.rate_limit_per_sec", 100)
}

// DatabaseDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// RedisAddr returns the Redis connection address
func (c *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
