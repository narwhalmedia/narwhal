package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new logger instance based on configuration.
func New(serviceName, environment, logLevel, logFormat string) (*zap.Logger, error) {
	var config zap.Config

	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Set log level
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		return nil, err
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Set encoding
	if logFormat == "json" {
		config.Encoding = "json"
	} else {
		config.Encoding = "console"
	}

	// Add service name to all logs
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
		"env":     environment,
	}

	// Configure output paths
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Add caller info
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"

	// Use ISO8601 time format
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	// Add hostname if available
	if hostname, err := os.Hostname(); err == nil {
		logger = logger.With(zap.String("hostname", hostname))
	}

	return logger, nil
}

// WithContext creates a logger with request context fields.
func WithContext(logger *zap.Logger, requestID, userID string) *zap.Logger {
	fields := []zap.Field{}

	if requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if userID != "" {
		fields = append(fields, zap.String("user_id", userID))
	}

	if len(fields) > 0 {
		return logger.With(fields...)
	}

	return logger
}

// WithTracing adds trace and span IDs to the logger.
func WithTracing(logger *zap.Logger, traceID, spanID string) *zap.Logger {
	fields := []zap.Field{}

	if traceID != "" {
		fields = append(fields, zap.String("trace_id", traceID))
	}

	if spanID != "" {
		fields = append(fields, zap.String("span_id", spanID))
	}

	if len(fields) > 0 {
		return logger.With(fields...)
	}

	return logger
}
