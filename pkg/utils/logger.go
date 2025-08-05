package utils

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// ZapLogger wraps zap logger to implement our Logger interface
type ZapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// NewZapLogger creates a new zap logger
func NewZapLogger(development bool) (*ZapLogger, error) {
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
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

// Debug logs a debug message
func (l *ZapLogger) Debug(msg string, fields ...interfaces.Field) {
	l.logger.Debug(msg, convertFields(fields)...)
}

// Info logs an info message
func (l *ZapLogger) Info(msg string, fields ...interfaces.Field) {
	l.logger.Info(msg, convertFields(fields)...)
}

// Warn logs a warning message
func (l *ZapLogger) Warn(msg string, fields ...interfaces.Field) {
	l.logger.Warn(msg, convertFields(fields)...)
}

// Error logs an error message
func (l *ZapLogger) Error(msg string, fields ...interfaces.Field) {
	l.logger.Error(msg, convertFields(fields)...)
}

// Fatal logs a fatal message and exits
func (l *ZapLogger) Fatal(msg string, fields ...interfaces.Field) {
	l.logger.Fatal(msg, convertFields(fields)...)
}

// WithContext returns a logger with context
func (l *ZapLogger) WithContext(ctx context.Context) interfaces.Logger {
	// Extract any relevant context values and add as fields
	// For now, just return self
	return l
}

// WithFields returns a logger with additional fields
func (l *ZapLogger) WithFields(fields ...interfaces.Field) interfaces.Logger {
	newLogger := l.logger.With(convertFields(fields)...)
	return &ZapLogger{
		logger: newLogger,
		sugar:  newLogger.Sugar(),
	}
}

// Sync flushes any buffered log entries
func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}

// convertFields converts our Field type to zap.Field
func convertFields(fields []interfaces.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		switch v := field.Value.(type) {
		case string:
			zapFields[i] = zap.String(field.Key, v)
		case int:
			zapFields[i] = zap.Int(field.Key, v)
		case int64:
			zapFields[i] = zap.Int64(field.Key, v)
		case float64:
			zapFields[i] = zap.Float64(field.Key, v)
		case bool:
			zapFields[i] = zap.Bool(field.Key, v)
		case error:
			zapFields[i] = zap.Error(v)
		default:
			zapFields[i] = zap.Any(field.Key, v)
		}
	}
	return zapFields
}
