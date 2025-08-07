package interfaces

import (
	"context"

	"github.com/narwhalmedia/narwhal/pkg/models"
)

// AuthService handles authentication operations.
type AuthService interface {
	// Authenticate validates user credentials and returns a token
	Authenticate(ctx context.Context, username, password string) (string, error)

	// ValidateToken validates a JWT token and returns claims
	ValidateToken(ctx context.Context, token string) (*TokenClaims, error)

	// RefreshToken refreshes an existing token
	RefreshToken(ctx context.Context, token string) (string, error)

	// RevokeToken revokes a token
	RevokeToken(ctx context.Context, token string) error

	// CreateUser creates a new user
	CreateUser(ctx context.Context, user *models.User, password string) error

	// UpdatePassword updates a user's password
	UpdatePassword(ctx context.Context, userID, oldPassword, newPassword string) error
}

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID    string   `json:"sub"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Role      string   `json:"role"`
	Scopes    []string `json:"scopes"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

// AuthorizationService handles authorization operations.
type AuthorizationService interface {
	// Authorize checks if a user can perform an action on a resource
	Authorize(ctx context.Context, userID, resource, action string) error

	// GetPermissions gets all permissions for a user
	GetPermissions(ctx context.Context, userID string) ([]Permission, error)

	// GrantPermission grants a permission to a user
	GrantPermission(ctx context.Context, userID string, permission Permission) error

	// RevokePermission revokes a permission from a user
	RevokePermission(ctx context.Context, userID string, permission Permission) error
}

// Permission represents a permission.
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope,omitempty"`
}
