package logger

import (
	"context"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

type contextKey struct{}

var loggerKey = contextKey{}

// FromContext retrieves a logger from the context.
func FromContext(ctx context.Context) interfaces.Logger {
	if logger, ok := ctx.Value(loggerKey).(interfaces.Logger); ok {
		return logger
	}
	// Return a default logger if none is found
	return New()
}

// WithContext adds a logger to the context.
func WithContext(ctx context.Context, logger interfaces.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithFields adds fields to the logger in the context.
func WithFields(ctx context.Context, fields ...interfaces.Field) context.Context {
	logger := FromContext(ctx)
	return WithContext(ctx, logger.WithFields(fields...))
}
