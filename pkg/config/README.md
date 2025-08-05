# Narwhal Configuration System

The Narwhal configuration system uses [Koanf](https://github.com/knadh/koanf) to provide a flexible, hierarchical configuration management solution for all microservices.

## Features

- **Multiple configuration sources** with clear precedence
- **Environment variable overrides** for production deployments
- **Type-safe configuration** with validation
- **Service-specific configurations** extending a common base
- **Hot-reloading support** (can be added via Koanf's watch feature)
- **Multiple format support**: YAML and JSON

## Configuration Hierarchy

Configuration is loaded in the following order (highest to lowest priority):

1. **Environment variables** - Override any file-based config
2. **Environment-specific config** - e.g., `library.production.yaml`
3. **Service-specific config** - e.g., `library.yaml`
4. **General config** - `config.yaml`
5. **Default values** - Hardcoded in the application

## Usage

### Basic Usage

```go
// In your service's main.go
package main

import (
    "github.com/narwhalmedia/narwhal/pkg/config"
)

func main() {
    // Load configuration for library service
    cfg := config.MustLoadServiceConfig("library", config.GetDefaultLibraryConfig())
    
    // Access configuration values
    dbHost := cfg.Database.Host
    scanInterval := cfg.Library.ScanInterval
}
```

### Environment Variables

Environment variables follow the pattern: `<SERVICE_NAME>_<SECTION>_<KEY>`

Examples:
```bash
# Database configuration
LIBRARY_DATABASE_HOST=postgres.example.com
LIBRARY_DATABASE_PASSWORD=secret
LIBRARY_DATABASE_SSL_MODE=require

# Service configuration
LIBRARY_SERVICE_PORT=8080
LIBRARY_SERVICE_ENVIRONMENT=production

# Library-specific configuration
LIBRARY_LIBRARY_SCAN_INTERVAL=1h
LIBRARY_LIBRARY_MAX_CONCURRENT_SCAN=10
```

### Configuration Files

Place configuration files in the `configs/` directory:

```
configs/
├── library.dev.yaml          # Development config for library service
├── library.production.yaml   # Production config for library service
├── user.dev.yaml            # Development config for user service
└── user.production.yaml     # Production config for user service
```

### Custom Configuration Path

Set a custom configuration file path:
```bash
CONFIG_PATH=/etc/narwhal/library.yaml ./library-service
```

## Service Configurations

### Library Service

```go
type LibraryConfig struct {
    BaseConfig `koanf:",squash"`
    Library    LibrarySettings `koanf:"library"`
}
```

Key settings:
- `scan_interval`: How often to scan libraries
- `max_concurrent_scan`: Maximum concurrent library scans
- `file_extensions`: Supported file extensions
- `ignore_patterns`: Patterns to ignore during scanning

### User Service

```go
type UserConfig struct {
    BaseConfig `koanf:",squash"`
    Auth       AuthSettings `koanf:"auth"`
}
```

Key settings:
- `jwt_secret`: Secret for JWT signing (required)
- `jwt_access_expiry`: Access token expiration
- `bcrypt_cost`: Password hashing cost
- `session_timeout`: User session timeout

### Streaming Service

```go
type StreamingConfig struct {
    BaseConfig `koanf:",squash"`
    Streaming  StreamingSettings `koanf:"streaming"`
}
```

Key settings:
- `transcoding_profiles`: Available transcoding profiles
- `segment_duration`: HLS/DASH segment duration
- `hardware_accel`: Hardware acceleration type

### Acquisition Service

```go
type AcquisitionConfig struct {
    BaseConfig  `koanf:",squash"`
    Acquisition AcquisitionSettings `koanf:"acquisition"`
}
```

Key settings:
- `indexers`: Configured indexers
- `max_active_downloads`: Concurrent download limit
- `preferred_quality`: Quality preferences

## Common Configuration

All services share these common configurations:

### Database
```yaml
database:
  host: localhost
  port: 5432
  user: narwhal
  password: narwhal_dev
  database: narwhal_dev
  ssl_mode: disable
  max_connections: 25
```

### Redis
```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10
```

### Logger
```yaml
logger:
  level: info          # debug, info, warn, error
  format: json         # json, console
  development: false
  output_path: stdout  # stdout, stderr, or file path
```

### Metrics
```yaml
metrics:
  enabled: true
  path: /metrics
  port: 2112
  interval: 10
```

### Tracing
```yaml
tracing:
  enabled: true
  provider: otlp       # jaeger, zipkin, otlp
  endpoint: localhost:4317
  sampling_rate: 0.1
```

## Validation

All configurations are validated on load. Implement the `Validate()` method for custom validation:

```go
func (c *LibraryConfig) Validate() error {
    if err := c.BaseConfig.Validate(); err != nil {
        return err
    }
    if c.Library.ScanInterval < time.Minute {
        return fmt.Errorf("scan interval must be at least 1 minute")
    }
    return nil
}
```

## Testing

For testing, create configuration programmatically:

```go
func TestServiceWithConfig(t *testing.T) {
    cfg := &config.LibraryConfig{
        BaseConfig: config.BaseConfig{
            Service: config.ServiceConfig{
                Name: "library-test",
                Port: 0, // Random port
            },
        },
        Library: config.LibrarySettings{
            ScanInterval: time.Minute,
        },
    }
    
    service := NewService(cfg)
    // Run tests...
}
```

## Best Practices

1. **Never commit secrets** - Use environment variables for sensitive data
2. **Use environment-specific files** - Separate dev/staging/production configs
3. **Validate early** - Fail fast on invalid configuration
4. **Document defaults** - Make default values clear and sensible
5. **Keep it DRY** - Use the base configuration for common settings
6. **Version configs** - Track configuration changes in version control

## Adding a New Service

1. Create service-specific config struct:
```go
type MyServiceConfig struct {
    BaseConfig `koanf:",squash"`
    MyService  MyServiceSettings `koanf:"myservice"`
}
```

2. Add validation:
```go
func (c *MyServiceConfig) Validate() error {
    // Validation logic
}
```

3. Create default configuration:
```go
func GetDefaultMyServiceConfig() *MyServiceConfig {
    // Return default config
}
```

4. Create config files in `configs/` directory

5. Use in your service:
```go
cfg := config.MustLoadServiceConfig("myservice", config.GetDefaultMyServiceConfig())
```