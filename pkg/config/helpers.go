package config

import (
	"fmt"
	"os"

	"github.com/narwhalmedia/narwhal/pkg/database"
	"gorm.io/gorm/logger"
)

// LoadServiceConfig is a generic helper to load service configuration
func LoadServiceConfig[T Config](serviceName string, cfg T) error {
	manager := NewManager(serviceName)
	return manager.LoadConfig(cfg)
}

// ToDatabaseConfig converts config to database package config
func (c DatabaseConfig) ToDatabaseConfig() *database.PostgresConfig {
	logLevel := logger.Info
	if c.SSLMode == "" {
		c.SSLMode = "disable"
	}

	return &database.PostgresConfig{
		Host:            c.Host,
		Port:            c.Port,
		User:            c.User,
		Password:        c.Password,
		Database:        c.Database,
		SSLMode:         c.SSLMode,
		MaxConnections:  c.MaxConnections,
		MinConnections:  c.MinConnections,
		MaxConnLifetime: c.MaxConnLifetime,
		MaxConnIdleTime: c.MaxConnIdleTime,
		LogLevel:        logLevel,
	}
}

// GetServiceVersion returns the service version from config or git
func GetServiceVersion(cfg *ServiceConfig) string {
	if cfg.Version != "" {
		return cfg.Version
	}

	// Try to get from environment
	if version := os.Getenv("SERVICE_VERSION"); version != "" {
		return version
	}

	// Default to dev
	return "dev"
}

// IsProduction returns true if running in production environment
func IsProduction(cfg *ServiceConfig) bool {
	return cfg.Environment == "production" || cfg.Environment == "prod"
}

// IsDevelopment returns true if running in development environment
func IsDevelopment(cfg *ServiceConfig) bool {
	return cfg.Environment == "development" || cfg.Environment == "dev"
}

// GetListenAddress returns the formatted listen address for HTTP server
func GetListenAddress(cfg *ServiceConfig) string {
	return fmt.Sprintf(":%d", cfg.Port)
}

// GetGRPCListenAddress returns the formatted listen address for gRPC server
func GetGRPCListenAddress(cfg *ServiceConfig) string {
	return fmt.Sprintf(":%d", cfg.GRPCPort)
}

// MustLoadServiceConfig loads config and panics on error (for main functions)
func MustLoadServiceConfig[T Config](serviceName string, cfg T) T {
	if err := LoadServiceConfig(serviceName, cfg); err != nil {
		panic(fmt.Sprintf("failed to load %s config: %v", serviceName, err))
	}
	return cfg
}

// PrintConfig prints the loaded configuration (for debugging)
func PrintConfig(cfg Config) {
	fmt.Printf("Loaded configuration:\n%+v\n", cfg)
}
