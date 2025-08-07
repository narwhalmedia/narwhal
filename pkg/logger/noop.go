package logger

import (
	"context"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// NoopLogger is a logger that does nothing.
type NoopLogger struct{}

// NewNoop creates a new no-op logger.
func NewNoop() interfaces.Logger {
	return &NoopLogger{}
}

// NewNoopLogger creates a new no-op logger (alias for NewNoop).
func NewNoopLogger() interfaces.Logger {
	return NewNoop()
}

// Debug does nothing.
func (n *NoopLogger) Debug(msg string, fields ...interfaces.Field) {}

// Info does nothing.
func (n *NoopLogger) Info(msg string, fields ...interfaces.Field) {}

// Warn does nothing.
func (n *NoopLogger) Warn(msg string, fields ...interfaces.Field) {}

// Error does nothing.
func (n *NoopLogger) Error(msg string, fields ...interfaces.Field) {}

// Fatal does nothing (doesn't exit).
func (n *NoopLogger) Fatal(msg string, fields ...interfaces.Field) {}

// WithContext returns the same logger.
func (n *NoopLogger) WithContext(ctx context.Context) interfaces.Logger {
	return n
}

// WithFields returns the same logger.
func (n *NoopLogger) WithFields(fields ...interfaces.Field) interfaces.Logger {
	return n
}

// With returns the same logger (alias for WithFields).
func (n *NoopLogger) With(fields ...interfaces.Field) interfaces.Logger {
	return n
}
