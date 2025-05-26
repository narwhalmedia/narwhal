package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// Redis configuration
	Redis RedisConfig

	// NATS configuration
	NATS NATSConfig

	// Observability configuration
	Observability ObservabilityConfig

	// Storage configuration
	Storage StorageConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	GRPCPort     int
	HTTPPort     int
	Environment  string
	ServiceName  string
	LogLevel     string
	ShutdownTime time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string
	Port         int
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL          string
	ClusterID    string
	ClientID     string
	DurableName  string
	MaxReconnect int
	ReconnectWait time.Duration
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	TracingEnabled    bool
	TracingEndpoint   string
	MetricsEnabled    bool
	MetricsPort       int
	LogLevel          string
	LogFormat         string // json or text
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type      string // local, s3, minio
	LocalPath string
	S3Config  S3Config
}

// S3Config holds S3/MinIO configuration
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Region          string
	UseSSL          bool
}

// Load loads configuration from environment variables
func Load(serviceName string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort:     getEnvAsInt("GRPC_PORT", 9090),
			HTTPPort:     getEnvAsInt("HTTP_PORT", 8080),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ServiceName:  serviceName,
			LogLevel:     getEnv("LOG_LEVEL", "info"),
			ShutdownTime: getEnvAsDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:         getEnv("DB_HOST", "localhost"),
			Port:         getEnvAsInt("DB_PORT", 5432),
			User:         getEnv("DB_USER", "narwhal"),
			Password:     getEnv("DB_PASSWORD", "narwhal"),
			Database:     getEnv("DB_NAME", "narwhal"),
			SSLMode:      getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns: getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns: getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			MaxLifetime:  getEnvAsDuration("DB_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnvAsInt("REDIS_PORT", 6379),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 0),
			PoolSize:     getEnvAsInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvAsInt("REDIS_MIN_IDLE_CONNS", 5),
			MaxRetries:   getEnvAsInt("REDIS_MAX_RETRIES", 3),
		},
		NATS: NATSConfig{
			URL:           getEnv("NATS_URL", "nats://localhost:4222"),
			ClusterID:     getEnv("NATS_CLUSTER_ID", "narwhal-cluster"),
			ClientID:      fmt.Sprintf("%s-%s", serviceName, getEnv("HOSTNAME", "local")),
			DurableName:   fmt.Sprintf("%s-durable", serviceName),
			MaxReconnect:  getEnvAsInt("NATS_MAX_RECONNECT", 60),
			ReconnectWait: getEnvAsDuration("NATS_RECONNECT_WAIT", 2*time.Second),
		},
		Observability: ObservabilityConfig{
			TracingEnabled:  getEnvAsBool("TRACING_ENABLED", true),
			TracingEndpoint: getEnv("TRACING_ENDPOINT", "localhost:4317"),
			MetricsEnabled:  getEnvAsBool("METRICS_ENABLED", true),
			MetricsPort:     getEnvAsInt("METRICS_PORT", 9091),
			LogLevel:        getEnv("LOG_LEVEL", "info"),
			LogFormat:       getEnv("LOG_FORMAT", "json"),
		},
		Storage: StorageConfig{
			Type:      getEnv("STORAGE_TYPE", "local"),
			LocalPath: getEnv("STORAGE_LOCAL_PATH", "/var/narwhal/media"),
			S3Config: S3Config{
				Endpoint:        getEnv("S3_ENDPOINT", ""),
				AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
				Bucket:          getEnv("S3_BUCKET", "narwhal-media"),
				Region:          getEnv("S3_REGION", "us-east-1"),
				UseSSL:          getEnvAsBool("S3_USE_SSL", true),
			},
		},
	}

	return cfg, nil
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	if value, err := strconv.Atoi(strValue); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	if value, err := strconv.ParseBool(strValue); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	strValue := getEnv(key, "")
	if strValue == "" {
		return defaultValue
	}
	if value, err := time.ParseDuration(strValue); err == nil {
		return value
	}
	return defaultValue
}

// DSN returns the database connection string
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Database, d.SSLMode)
}