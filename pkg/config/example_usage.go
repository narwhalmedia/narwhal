package config

// Example usage in a service main.go:
/*

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/narwhalmedia/narwhal/pkg/config"
	"github.com/narwhalmedia/narwhal/pkg/database"
)

func main() {
	// 1. Load configuration
	cfg := config.MustLoadServiceConfig("library", config.GetDefaultLibraryConfig())

	// 2. Initialize logger based on config
	logger := initLogger(cfg.Logger)

	// 3. Connect to database
	db, err := database.NewGormDB(cfg.Database.ToDatabaseConfig())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 4. Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// 5. Initialize service components using config
	// ... service initialization ...

	// 6. Start servers
	go startHTTPServer(config.GetListenAddress(&cfg.Service))
	go startGRPCServer(config.GetGRPCListenAddress(&cfg.Service))

	// 7. Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics)
	}

	// Wait for shutdown signal
	waitForShutdown()
}

// Environment variable override examples:
// LIBRARY_DATABASE_HOST=postgres.prod.example.com
// LIBRARY_DATABASE_PASSWORD=secret-password
// LIBRARY_AUTH_JWT_SECRET=production-secret
// LIBRARY_SERVICE_ENVIRONMENT=production
// LIBRARY_LOGGER_LEVEL=info
// LIBRARY_METRICS_ENABLED=true

*/

// Configuration file hierarchy:
// 1. Environment variables (highest priority)
// 2. Service-specific environment file: configs/library.production.yaml
// 3. Service-specific file: configs/library.yaml
// 4. General config file: configs/config.yaml
// 5. Default values from struct (lowest priority)

// Testing configuration:
/*

func TestServiceWithConfig(t *testing.T) {
	// Create test configuration
	cfg := &config.LibraryConfig{
		BaseConfig: config.BaseConfig{
			Service: config.ServiceConfig{
				Name: "library-test",
				Port: 0, // Use random port
			},
			Database: config.DatabaseConfig{
				// Use test container
			},
		},
		Library: config.LibrarySettings{
			ScanInterval: time.Minute,
			// ... test settings
		},
	}

	// Use the test config
	service := NewService(cfg)
	// ... run tests
}

*/
