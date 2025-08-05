package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration
type Config struct {
	Level       string   `json:"level" yaml:"level"`
	Development bool     `json:"development" yaml:"development"`
	Encoding    string   `json:"encoding" yaml:"encoding"` // json or console
	OutputPaths []string `json:"output_paths" yaml:"output_paths"`
	ErrorPaths  []string `json:"error_paths" yaml:"error_paths"`
	
	// Additional fields to include in all logs
	InitialFields map[string]interface{} `json:"initial_fields" yaml:"initial_fields"`
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:       "info",
		Development: false,
		Encoding:    "json",
		OutputPaths: []string{"stdout"},
		ErrorPaths:  []string{"stderr"},
	}
}

// DevelopmentConfig returns development logger configuration
func DevelopmentConfig() *Config {
	return &Config{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths: []string{"stdout"},
		ErrorPaths:  []string{"stderr"},
	}
}

// Build creates a logger from the configuration
func (c *Config) Build() (*ZapLogger, error) {
	// Parse log level
	level, err := zapcore.ParseLevel(c.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// Create zap config
	var zapConfig zap.Config
	
	if c.Development {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		zapConfig.EncoderConfig.MessageKey = "message"
		zapConfig.EncoderConfig.LevelKey = "level"
		zapConfig.EncoderConfig.CallerKey = "caller"
		zapConfig.EncoderConfig.StacktraceKey = "stacktrace"
	}

	// Apply configuration
	zapConfig.Level = zap.NewAtomicLevelAt(level)
	zapConfig.Encoding = c.Encoding
	zapConfig.OutputPaths = c.OutputPaths
	zapConfig.ErrorOutputPaths = c.ErrorPaths
	zapConfig.Development = c.Development

	// Build logger
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	// Add initial fields if any
	if len(c.InitialFields) > 0 {
		fields := make([]zap.Field, 0, len(c.InitialFields))
		for k, v := range c.InitialFields {
			fields = append(fields, zap.Any(k, v))
		}
		logger = logger.With(fields...)
	}

	return &ZapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// NewFromConfig creates a new logger from configuration
func NewFromConfig(cfg *Config) (*ZapLogger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return cfg.Build()
}