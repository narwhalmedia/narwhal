package logger

import (
	"context"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// ZapLogger wraps zap logger to implement the Logger interface.
type ZapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// New creates a new logger based on environment.
func New() interfaces.Logger {
	env := os.Getenv("ENVIRONMENT")
	development := env == "" || env == "development"

	logger, err := NewZapLogger(development)
	if err != nil {
		panic(err)
	}

	return logger
}

// NewZapLogger creates a new zap logger with the specified configuration.
func NewZapLogger(development bool) (*ZapLogger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.MessageKey = "message"
		config.EncoderConfig.LevelKey = "level"
		config.OutputPaths = []string{"stdout"}
		config.ErrorOutputPaths = []string{"stderr"}
	}

	// Set log level from environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(logLevel)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &ZapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// Debug logs a debug message.
func (l *ZapLogger) Debug(msg string, fields ...interfaces.Field) {
	l.logger.Debug(msg, convertFields(fields)...)
}

// Info logs an info message.
func (l *ZapLogger) Info(msg string, fields ...interfaces.Field) {
	l.logger.Info(msg, convertFields(fields)...)
}

// Warn logs a warning message.
func (l *ZapLogger) Warn(msg string, fields ...interfaces.Field) {
	l.logger.Warn(msg, convertFields(fields)...)
}

// Error logs an error message.
func (l *ZapLogger) Error(msg string, fields ...interfaces.Field) {
	l.logger.Error(msg, convertFields(fields)...)
}

// Fatal logs a fatal message and exits.
func (l *ZapLogger) Fatal(msg string, fields ...interfaces.Field) {
	l.logger.Fatal(msg, convertFields(fields)...)
}

// WithContext returns a logger with context (context is not used in zap).
func (l *ZapLogger) WithContext(ctx context.Context) interfaces.Logger {
	// Zap doesn't use context, so just return self
	return l
}

// WithFields returns a logger with additional fields.
func (l *ZapLogger) WithFields(fields ...interfaces.Field) interfaces.Logger {
	return &ZapLogger{
		logger: l.logger.With(convertFields(fields)...),
		sugar:  l.logger.With(convertFields(fields)...).Sugar(),
	}
}

// With creates a child logger with additional fields (alias for WithFields).
func (l *ZapLogger) With(fields ...interfaces.Field) interfaces.Logger {
	return l.WithFields(fields...)
}

// Sync flushes any buffered log entries.
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// convertFields converts our custom fields to zap fields.
func convertFields(fields []interfaces.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}

// Helper functions for common field types

// String creates a string field.
func String(key, value string) interfaces.Field {
	return interfaces.String(key, value)
}

// Int creates an int field.
func Int(key string, value int) interfaces.Field {
	return interfaces.Int(key, value)
}

// Bool creates a bool field.
func Bool(key string, value bool) interfaces.Field {
	return interfaces.Bool(key, value)
}

// Error creates an error field.
func Error(err error) interfaces.Field {
	return interfaces.Error(err)
}

// Any creates a field with any value.
func Any(key string, value interface{}) interfaces.Field {
	return interfaces.Any(key, value)
}
