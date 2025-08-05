package auth

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// ContextKeyClaims is the context key for JWT claims
	ContextKeyClaims ContextKey = "claims"
	// ContextKeyUserID is the context key for user ID
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyRoles is the context key for user roles
	ContextKeyRoles ContextKey = "roles"
)

// AuthInterceptor provides authentication and authorization for gRPC
type AuthInterceptor struct {
	jwtManager *JWTManager
	rbac       RBACInterface
	enforcer   PolicyEnforcerInterface
}

// PolicyEnforcerInterface defines the interface for policy enforcement
type PolicyEnforcerInterface interface {
	Enforce(roles []string, resource, action string) error
	EnforceAny(roles []string, permissions ...Permission) error
	EnforceAll(roles []string, permissions ...Permission) error
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(jwtManager *JWTManager, rbac RBACInterface) *AuthInterceptor {
	var enforcer PolicyEnforcerInterface
	
	// Create appropriate enforcer based on RBAC type
	switch r := rbac.(type) {
	case *RBAC:
		enforcer = NewPolicyEnforcer(r)
	case *CasbinRBAC:
		enforcer = NewCasbinPolicyEnforcer(r)
	default:
		// Default to a generic enforcer
		enforcer = &GenericPolicyEnforcer{rbac: rbac}
	}
	
	return &AuthInterceptor{
		jwtManager: jwtManager,
		rbac:       rbac,
		enforcer:   enforcer,
	}
}

// GenericPolicyEnforcer provides a generic policy enforcer for any RBAC implementation
type GenericPolicyEnforcer struct {
	rbac RBACInterface
}

// Enforce checks if the given roles satisfy the permission requirement
func (p *GenericPolicyEnforcer) Enforce(roles []string, resource, action string) error {
	if !p.rbac.CheckPermissions(roles, resource, action) {
		return fmt.Errorf("permission denied: %s:%s", resource, action)
	}
	return nil
}

// EnforceAny checks if the given roles satisfy any of the permission requirements
func (p *GenericPolicyEnforcer) EnforceAny(roles []string, permissions ...Permission) error {
	for _, perm := range permissions {
		if p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return nil
		}
	}
	
	permStrs := []string{}
	for _, perm := range permissions {
		permStrs = append(permStrs, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
	}
	return fmt.Errorf("permission denied: requires any of [%s]", strings.Join(permStrs, ", "))
}

// EnforceAll checks if the given roles satisfy all permission requirements
func (p *GenericPolicyEnforcer) EnforceAll(roles []string, permissions ...Permission) error {
	for _, perm := range permissions {
		if !p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return fmt.Errorf("permission denied: %s:%s", perm.Resource, perm.Action)
		}
	}
	return nil
}

// UnaryServerInterceptor returns a gRPC unary interceptor for authentication
func (a *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip auth for certain methods (like login)
		if a.skipAuth(info.FullMethod) {
			return handler(ctx, req)
		}

		// Extract and validate token
		newCtx, err := a.authenticate(ctx)
		if err != nil {
			return nil, err
		}

		// Check authorization if required
		if err := a.authorize(newCtx, info.FullMethod); err != nil {
			return nil, err
		}

		return handler(newCtx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream interceptor for authentication
func (a *AuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip auth for certain methods
		if a.skipAuth(info.FullMethod) {
			return handler(srv, stream)
		}

		// Extract and validate token
		newCtx, err := a.authenticate(stream.Context())
		if err != nil {
			return err
		}

		// Check authorization if required
		if err := a.authorize(newCtx, info.FullMethod); err != nil {
			return err
		}

		// Create wrapped stream with new context
		wrappedStream := &wrappedServerStream{
			ServerStream: stream,
			ctx:          newCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// authenticate extracts and validates the JWT token from the request
func (a *AuthInterceptor) authenticate(ctx context.Context) (context.Context, error) {
	token, err := a.extractToken(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "missing token: %v", err)
	}

	claims, err := a.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	// Add claims to context
	ctx = context.WithValue(ctx, ContextKeyClaims, claims)
	ctx = context.WithValue(ctx, ContextKeyUserID, claims.UserID)
	ctx = context.WithValue(ctx, ContextKeyRoles, claims.Roles)

	return ctx, nil
}

// authorize checks if the user has permission to access the method
func (a *AuthInterceptor) authorize(ctx context.Context, method string) error {
	// Get required permissions for the method
	resource, action := a.getMethodPermissions(method)
	if resource == "" || action == "" {
		// No specific permissions required
		return nil
	}

	roles, ok := ctx.Value(ContextKeyRoles).([]string)
	if !ok {
		return status.Error(codes.Internal, "roles not found in context")
	}

	if err := a.enforcer.Enforce(roles, resource, action); err != nil {
		return status.Errorf(codes.PermissionDenied, "%v", err)
	}

	return nil
}

// extractToken extracts the token from the authorization header
func (a *AuthInterceptor) extractToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "metadata not provided")
	}

	values := md["authorization"]
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "authorization token not provided")
	}

	// Remove "Bearer " prefix
	token := values[0]
	if strings.HasPrefix(token, "Bearer ") {
		token = strings.TrimPrefix(token, "Bearer ")
	}

	return token, nil
}

// skipAuth returns true if the method should skip authentication
func (a *AuthInterceptor) skipAuth(method string) bool {
	// List of methods that don't require authentication
	skipMethods := []string{
		"/narwhal.user.v1.UserService/Login",
		"/narwhal.user.v1.UserService/Register",
		"/narwhal.user.v1.UserService/RefreshToken",
		"/narwhal.user.v1.UserService/ForgotPassword",
		"/narwhal.user.v1.UserService/ResetPassword",
		"/grpc.health.v1.Health/Check",
		"/grpc.health.v1.Health/Watch",
	}

	for _, m := range skipMethods {
		if method == m {
			return true
		}
	}

	return false
}

// getMethodPermissions returns the required resource and action for a gRPC method
func (a *AuthInterceptor) getMethodPermissions(method string) (resource, action string) {
	// Define permission requirements for each service method
	permissions := map[string]struct{ Resource, Action string }{
		// Library service
		"/narwhal.library.v1.LibraryService/CreateLibrary": {"library", "write"},
		"/narwhal.library.v1.LibraryService/UpdateLibrary": {"library", "write"},
		"/narwhal.library.v1.LibraryService/DeleteLibrary": {"library", "delete"},
		"/narwhal.library.v1.LibraryService/ScanLibrary":   {"library", "write"},
		"/narwhal.library.v1.LibraryService/GetLibrary":    {"library", "read"},
		"/narwhal.library.v1.LibraryService/ListLibraries": {"library", "read"},
		
		// Media operations
		"/narwhal.library.v1.LibraryService/GetMedia":      {"media", "read"},
		"/narwhal.library.v1.LibraryService/ListMedia":     {"media", "read"},
		"/narwhal.library.v1.LibraryService/SearchMedia":   {"media", "read"},
		"/narwhal.library.v1.LibraryService/UpdateMedia":   {"media", "write"},
		"/narwhal.library.v1.LibraryService/DeleteMedia":   {"media", "delete"},
		
		// User service
		"/narwhal.user.v1.UserService/GetUser":      {"user", "read"},
		"/narwhal.user.v1.UserService/ListUsers":    {"user", "read"},
		"/narwhal.user.v1.UserService/UpdateUser":   {"user", "write"},
		"/narwhal.user.v1.UserService/DeleteUser":   {"user", "delete"},
		"/narwhal.user.v1.UserService/CreateUser":   {"user", "admin"},
		"/narwhal.user.v1.UserService/AssignRole":   {"user", "admin"},
		"/narwhal.user.v1.UserService/RemoveRole":   {"user", "admin"},
		
		// System operations
		"/narwhal.user.v1.UserService/CreateRole":       {"system", "admin"},
		"/narwhal.user.v1.UserService/UpdateRole":       {"system", "admin"},
		"/narwhal.user.v1.UserService/DeleteRole":       {"system", "admin"},
		"/narwhal.user.v1.UserService/CreatePermission": {"system", "admin"},
		"/narwhal.user.v1.UserService/DeletePermission": {"system", "admin"},
	}

	if perm, ok := permissions[method]; ok {
		return perm.Resource, perm.Action
	}

	return "", ""
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// GetClaimsFromContext extracts JWT claims from context
func GetClaimsFromContext(ctx context.Context) (*CustomClaims, bool) {
	claims, ok := ctx.Value(ContextKeyClaims).(*CustomClaims)
	return claims, ok
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(ContextKeyUserID).(string)
	return userID, ok
}

// GetRolesFromContext extracts roles from context
func GetRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value(ContextKeyRoles).([]string)
	return roles, ok
}

// RequirePermission creates a function that checks for a specific permission
func (a *AuthInterceptor) RequirePermission(resource, action string) func(context.Context) error {
	return func(ctx context.Context) error {
		roles, ok := GetRolesFromContext(ctx)
		if !ok {
			return status.Error(codes.Internal, "roles not found in context")
		}

		return a.enforcer.Enforce(roles, resource, action)
	}
}

// RequireAnyPermission creates a function that checks for any of the given permissions
func (a *AuthInterceptor) RequireAnyPermission(permissions ...Permission) func(context.Context) error {
	return func(ctx context.Context) error {
		roles, ok := GetRolesFromContext(ctx)
		if !ok {
			return status.Error(codes.Internal, "roles not found in context")
		}

		return a.enforcer.EnforceAny(roles, permissions...)
	}
}

// RequireAllPermissions creates a function that checks for all of the given permissions
func (a *AuthInterceptor) RequireAllPermissions(permissions ...Permission) func(context.Context) error {
	return func(ctx context.Context) error {
		roles, ok := GetRolesFromContext(ctx)
		if !ok {
			return status.Error(codes.Internal, "roles not found in context")
		}

		return a.enforcer.EnforceAll(roles, permissions...)
	}
}