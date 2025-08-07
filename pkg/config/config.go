package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// Config is the interface that all service configs must implement.
type Config interface {
	Validate() error
}

// BaseConfig contains common configuration for all services.
type BaseConfig struct {
	Service    ServiceConfig    `koanf:"service"`
	Database   DatabaseConfig   `koanf:"database"`
	Redis      RedisConfig      `koanf:"redis"`
	Logger     LoggerConfig     `koanf:"logger"`
	Metrics    MetricsConfig    `koanf:"metrics"`
	Tracing    TracingConfig    `koanf:"tracing"`
	Auth       AuthConfig       `koanf:"auth"`
	Pagination PaginationConfig `koanf:"pagination"`
}

// ServiceConfig contains service-specific metadata.
type ServiceConfig struct {
	Name        string `koanf:"name"`
	Version     string `koanf:"version"`
	Environment string `koanf:"environment"` // dev, staging, production
	Port        int    `koanf:"port"`
	GRPCPort    int    `koanf:"grpc_port"`
}

// AuthConfig contains authentication configuration shared across services.
type AuthConfig struct {
	JWTSecret            string        `koanf:"jwt_secret"`
	AccessTokenDuration  time.Duration `koanf:"access_token_duration"`
	RefreshTokenDuration time.Duration `koanf:"refresh_token_duration"`
	RBACType             string        `koanf:"rbac_type"` // "builtin" or "casbin"
	RBACModelPath        string        `koanf:"rbac_model_path"`
	RBACPolicyPath       string        `koanf:"rbac_policy_path"`
}

// PaginationConfig contains pagination configuration.
type PaginationConfig struct {
	CursorEncryptionKey string        `koanf:"cursor_encryption_key"`
	MaxPageSize         int           `koanf:"max_page_size"`
	DefaultPageSize     int           `koanf:"default_page_size"`
	CursorExpiration    time.Duration `koanf:"cursor_expiration"`
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	Host            string        `koanf:"host"`
	Port            int           `koanf:"port"`
	User            string        `koanf:"user"`
	Password        string        `koanf:"password"`
	Database        string        `koanf:"database"`
	SSLMode         string        `koanf:"ssl_mode"`
	MaxConnections  int           `koanf:"max_connections"`
	MinConnections  int           `koanf:"min_connections"`
	MaxConnLifetime time.Duration `koanf:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `koanf:"max_conn_idle_time"`
}

// RedisConfig contains Redis connection settings.
type RedisConfig struct {
	Host         string        `koanf:"host"`
	Port         int           `koanf:"port"`
	Password     string        `koanf:"password"`
	DB           int           `koanf:"db"`
	MaxRetries   int           `koanf:"max_retries"`
	DialTimeout  time.Duration `koanf:"dial_timeout"`
	ReadTimeout  time.Duration `koanf:"read_timeout"`
	WriteTimeout time.Duration `koanf:"write_timeout"`
	PoolSize     int           `koanf:"pool_size"`
	MinIdleConns int           `koanf:"min_idle_conns"`
}

// LoggerConfig contains logging configuration.
type LoggerConfig struct {
	Level       string `koanf:"level"`  // debug, info, warn, error
	Format      string `koanf:"format"` // json, console
	Development bool   `koanf:"development"`
	OutputPath  string `koanf:"output_path"` // stdout, stderr, or file path
}

// MetricsConfig contains metrics configuration.
type MetricsConfig struct {
	Enabled  bool   `koanf:"enabled"`
	Path     string `koanf:"path"`     // /metrics
	Port     int    `koanf:"port"`     // separate port for metrics
	Interval int    `koanf:"interval"` // collection interval in seconds
}

// TracingConfig contains distributed tracing configuration.
type TracingConfig struct {
	Enabled      bool    `koanf:"enabled"`
	Provider     string  `koanf:"provider"` // jaeger, zipkin, otlp
	Endpoint     string  `koanf:"endpoint"`
	SamplingRate float64 `koanf:"sampling_rate"` // 0.0 to 1.0
}

// Manager handles configuration loading and parsing.
type Manager struct {
	k           *koanf.Koanf
	serviceName string
	configPaths []string
}

// NewManager creates a new configuration manager.
func NewManager(serviceName string) *Manager {
	return &Manager{
		k:           koanf.New("."),
		serviceName: serviceName,
		configPaths: getDefaultConfigPaths(serviceName),
	}
}

// LoadConfig loads configuration from all sources.
func (m *Manager) LoadConfig(cfg Config) error {
	// 1. Load defaults from struct tags
	if err := m.loadDefaults(cfg); err != nil {
		return fmt.Errorf("failed to load defaults: %w", err)
	}

	// 2. Load from config files (in order of precedence)
	for _, path := range m.configPaths {
		if err := m.loadFromFile(path); err != nil {
			// Skip if file doesn't exist, error on parse failures
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to load config from %s: %w", path, err)
			}
		}
	}

	// 3. Load from environment variables
	if err := m.loadFromEnv(); err != nil {
		return fmt.Errorf("failed to load from environment: %w", err)
	}

	// 4. Unmarshal into the config struct
	if err := m.k.Unmarshal("", cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 5. Validate the configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}

// Get returns a value for the given key.
func (m *Manager) Get(key string) interface{} {
	return m.k.Get(key)
}

// GetString returns a string value for the given key.
func (m *Manager) GetString(key string) string {
	return m.k.String(key)
}

// GetInt returns an int value for the given key.
func (m *Manager) GetInt(key string) int {
	return m.k.Int(key)
}

// GetBool returns a bool value for the given key.
func (m *Manager) GetBool(key string) bool {
	return m.k.Bool(key)
}

// loadDefaults loads default values from struct.
func (m *Manager) loadDefaults(cfg Config) error {
	return m.k.Load(structs.Provider(cfg, "koanf"), nil)
}

// loadFromFile loads configuration from a file.
func (m *Manager) loadFromFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	// Determine parser based on file extension
	var parser koanf.Parser
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".json":
		parser = json.Parser()
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	// Load the file
	return m.k.Load(file.Provider(path), parser)
}

// loadFromEnv loads configuration from environment variables.
func (m *Manager) loadFromEnv() error {
	// Convert service name to uppercase for env prefix
	prefix := strings.ToUpper(m.serviceName) + "_"

	// Load environment variables
	return m.k.Load(env.Provider(prefix, ".", func(s string) string {
		// Convert NARWHAL_DATABASE_HOST to database.host
		return strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(s, prefix), "_", "."))
	}), nil)
}

// getDefaultConfigPaths returns the default config paths to check.
func getDefaultConfigPaths(serviceName string) []string {
	paths := []string{
		// Current directory
		"config.yaml",
		"config.json",
		fmt.Sprintf("%s.yaml", serviceName),
		fmt.Sprintf("%s.json", serviceName),

		// Config directory
		"configs/config.yaml",
		"configs/config.json",
		fmt.Sprintf("configs/%s.yaml", serviceName),
		fmt.Sprintf("configs/%s.json", serviceName),

		// Environment-specific configs
		fmt.Sprintf("configs/%s.%s.yaml", serviceName, getEnvironment()),
		fmt.Sprintf("configs/%s.%s.json", serviceName, getEnvironment()),
	}

	// Add paths from CONFIG_PATH environment variable
	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		paths = append([]string{configPath}, paths...)
	}

	return paths
}

// getEnvironment returns the current environment.
func getEnvironment() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		return env
	}
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	return "dev"
}

// Validate validates the base configuration.
func (c *BaseConfig) Validate() error {
	if c.Service.Name == "" {
		return errors.New("service name is required")
	}
	if c.Service.Port <= 0 || c.Service.Port > 65535 {
		return fmt.Errorf("invalid service port: %d", c.Service.Port)
	}
	if c.Database.Host == "" {
		return errors.New("database host is required")
	}
	if c.Database.Port <= 0 || c.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Database.Port)
	}
	if c.Auth.JWTSecret == "" {
		return errors.New("JWT secret is required (set via LIBRARY_AUTH_JWT_SECRET env var or config)")
	}
	if c.Auth.AccessTokenDuration < time.Minute {
		return errors.New("access token duration must be at least 1 minute")
	}
	return nil
}

// GetDefaults returns default configuration values.
func GetDefaults() *BaseConfig {
	return &BaseConfig{
		Service: ServiceConfig{
			Environment: "dev",
			Port:        DefaultHTTPPort,
			GRPCPort:    DefaultGRPCPort,
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            DefaultPostgresPort,
			User:            "narwhal",
			Password:        "narwhal_dev",
			Database:        "narwhal_dev",
			SSLMode:         "disable",
			MaxConnections:  DefaultMaxConnections,
			MinConnections:  DefaultMinConnections,
			MaxConnLifetime: time.Hour,
			MaxConnIdleTime: DefaultMaxConnIdleTime,
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         DefaultRedisPort,
			DB:           0,
			MaxRetries:   DefaultMaxRetries,
			DialTimeout:  DefaultDialTimeout,
			ReadTimeout:  DefaultReadTimeout,
			WriteTimeout: DefaultWriteTimeout,
			PoolSize:     DefaultPoolSize,
			MinIdleConns: DefaultMinIdleConns,
		},
		Logger: LoggerConfig{
			Level:       "info",
			Format:      "json",
			Development: false,
			OutputPath:  "stdout",
		},
		Metrics: MetricsConfig{
			Enabled:  true,
			Path:     "/metrics",
			Port:     DefaultTelemetryPort,
			Interval: DefaultTelemetryInterval,
		},
		Tracing: TracingConfig{
			Enabled:      false,
			Provider:     "otlp",
			Endpoint:     "localhost:4317",
			SamplingRate: DefaultSamplingRate,
		},
		Auth: AuthConfig{
			JWTSecret:            "", // Must be set via env or config
			AccessTokenDuration:  DefaultAccessTokenDuration,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			RBACType:             "casbin",
			RBACModelPath:        "configs/rbac_model.conf",
			RBACPolicyPath:       "configs/rbac_policy.csv",
		},
		Pagination: PaginationConfig{
			CursorEncryptionKey: "", // Must be set via env or config
			MaxPageSize:         200,
			DefaultPageSize:     50,
			CursorExpiration:    24 * time.Hour,
		},
	}
}
