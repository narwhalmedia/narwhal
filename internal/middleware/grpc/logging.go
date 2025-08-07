package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
)

// LoggingInterceptor creates a new unary logging interceptor.
func LoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Extract or generate request ID
		requestID := extractRequestID(ctx)
		if requestID == "" {
			requestID = uuid.New().String()
			ctx = context.WithValue(ctx, "request_id", requestID)
		}

		// Create request-scoped logger
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
		)

		// Log request
		reqLogger.Info("gRPC request started",
			zap.Any("request", req),
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Determine status code
		code := codes.OK
		if err != nil {
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			} else {
				code = codes.Unknown
			}
		}

		// Log response
		if err != nil {
			reqLogger.Error("gRPC request failed",
				zap.Error(err),
				zap.String("code", code.String()),
				zap.Duration("duration", duration),
			)
		} else {
			reqLogger.Info("gRPC request completed",
				zap.String("code", code.String()),
				zap.Duration("duration", duration),
				zap.Any("response", resp),
			)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor creates a new stream logging interceptor.
func StreamLoggingInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Extract or generate request ID
		requestID := extractRequestIDFromStream(ss.Context())
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Create request-scoped logger
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("method", info.FullMethod),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		)

		// Log stream start
		reqLogger.Info("gRPC stream started")

		// Wrap the stream with logging
		wrappedStream := &loggingServerStream{
			ServerStream: ss,
			logger:       reqLogger,
		}

		// Call handler
		err := handler(srv, wrappedStream)

		// Calculate duration
		duration := time.Since(start)

		// Log stream end
		if err != nil {
			reqLogger.Error("gRPC stream failed",
				zap.Error(err),
				zap.Duration("duration", duration),
			)
		} else {
			reqLogger.Info("gRPC stream completed",
				zap.Duration("duration", duration),
			)
		}

		return err
	}
}

// loggingServerStream wraps grpc.ServerStream to add logging.
type loggingServerStream struct {
	grpc.ServerStream

	logger *zap.Logger
}

func (s *loggingServerStream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err != nil {
		s.logger.Error("failed to send message", zap.Error(err))
	} else {
		s.logger.Debug("sent message", zap.Any("message", m))
	}
	return err
}

func (s *loggingServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err != nil {
		s.logger.Error("failed to receive message", zap.Error(err))
	} else {
		s.logger.Debug("received message", zap.Any("message", m))
	}
	return err
}

// extractRequestID extracts request ID from context metadata.
func extractRequestID(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get("x-request-id")
	if len(values) > 0 {
		return values[0]
	}

	return ""
}

// extractRequestIDFromStream extracts request ID from stream context.
func extractRequestIDFromStream(ctx context.Context) string {
	return extractRequestID(ctx)
}
