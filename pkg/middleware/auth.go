package middleware

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/narwhalmedia/narwhal/pkg/auth"
)

// AuthInterceptor creates a gRPC interceptor for JWT authentication.
func AuthInterceptor(jwtManager *auth.JWTManager, publicMethods map[string]bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check if method is public
		if publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Parse bearer token
		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return nil, status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// Add claims to context
		ctx = context.WithValue(ctx, "claims", claims)

		// Call handler
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor creates a gRPC stream interceptor for JWT authentication.
func StreamAuthInterceptor(jwtManager *auth.JWTManager, publicMethods map[string]bool) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Check if method is public
		if publicMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		// Parse bearer token
		authHeader := authHeaders[0]
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return status.Error(codes.Unauthenticated, "invalid authorization header format")
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate token
		claims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid token")
		}

		// Create wrapped stream with claims in context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ss.Context(), "claims", claims),
		}

		// Call handler
		return handler(srv, wrappedStream)
	}
}

// wrappedServerStream wraps a ServerStream with a custom context.
type wrappedServerStream struct {
	grpc.ServerStream

	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// PublicMethods returns a map of public gRPC methods that don't require authentication.
func PublicMethods() map[string]bool {
	return map[string]bool{
		"/narwhal.auth.v1.AuthService/Login":                        true,
		"/narwhal.auth.v1.AuthService/RefreshToken":                 true,
		"/narwhal.auth.v1.AuthService/CreateUser":                   true, // First user creation
		"/grpc.health.v1.Health/Check":                              true,
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo": true,
	}
}
