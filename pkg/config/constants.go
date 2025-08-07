package config

import "time"

const (
	// Server ports.
	DefaultHTTPPort = 8080
	DefaultGRPCPort = 9090

	// Database defaults.
	DefaultPostgresPort = 5432
	DefaultRedisPort    = 6379

	// Connection pool defaults.
	DefaultMaxConnections = 25
	DefaultMinConnections = 5
	DefaultMaxRetries     = 3
	DefaultPoolSize       = 10
	DefaultMinIdleConns   = 2

	// Timeout defaults.
	DefaultMaxConnIdleTime = 30 * time.Minute
	DefaultDialTimeout     = 5 * time.Second
	DefaultReadTimeout     = 3 * time.Second
	DefaultWriteTimeout    = 3 * time.Second

	// Telemetry defaults.
	DefaultTelemetryPort     = 2112
	DefaultTelemetryInterval = 10

	// Auth defaults.
	DefaultAccessTokenDuration = 15 * time.Minute
	DefaultSamplingRate        = 0.1
)
