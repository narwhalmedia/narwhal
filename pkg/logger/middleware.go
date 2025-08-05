package logger

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor for logging
func UnaryServerInterceptor(logger interfaces.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Add logger to context
		ctx = WithContext(ctx, logger)

		// Log request
		logger.Info("gRPC request started",
			interfaces.String("method", info.FullMethod),
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Determine status
		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			} else {
				code = codes.Unknown
			}
		}

		// Log response
		fields := []interfaces.Field{
			interfaces.String("method", info.FullMethod),
			interfaces.Any("duration_ms", duration.Milliseconds()),
			interfaces.String("status", code.String()),
		}

		if err != nil {
			fields = append(fields, interfaces.Error(err))
			logger.Error("gRPC request failed", fields...)
		} else {
			logger.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for logging
func StreamServerInterceptor(logger interfaces.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Create wrapped stream with logger in context
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          WithContext(ss.Context(), logger),
		}

		// Log request
		logger.Info("gRPC stream started",
			interfaces.String("method", info.FullMethod),
		)

		// Call handler
		err := handler(srv, wrapped)

		// Calculate duration
		duration := time.Since(start)

		// Determine status
		code := codes.OK
		if err != nil {
			if s, ok := status.FromError(err); ok {
				code = s.Code()
			} else {
				code = codes.Unknown
			}
		}

		// Log response
		fields := []interfaces.Field{
			interfaces.String("method", info.FullMethod),
			interfaces.Any("duration_ms", duration.Milliseconds()),
			interfaces.String("status", code.String()),
		}

		if err != nil {
			fields = append(fields, interfaces.Error(err))
			logger.Error("gRPC stream failed", fields...)
		} else {
			logger.Info("gRPC stream completed", fields...)
		}

		return err
	}
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
