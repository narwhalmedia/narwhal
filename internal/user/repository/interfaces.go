package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
)

// UserRepository defines methods for user operations.
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdateUser(ctx context.Context, user *domain.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, error)
	CountUsers(ctx context.Context) (int64, error)
	UserExists(ctx context.Context, username, email string) (bool, error)
}

// RoleRepository defines methods for role operations.
type RoleRepository interface {
	CreateRole(ctx context.Context, role *domain.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*domain.Role, error)
	GetRoleByName(ctx context.Context, name string) (*domain.Role, error)
	UpdateRole(ctx context.Context, role *domain.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context) ([]*domain.Role, error)
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}

// PermissionRepository defines methods for permission operations.
type PermissionRepository interface {
	CreatePermission(ctx context.Context, permission *domain.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*domain.Permission, error)
	GetPermissionByResourceAction(ctx context.Context, resource, action string) (*domain.Permission, error)
	UpdatePermission(ctx context.Context, permission *domain.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*domain.Permission, error)
}

// SessionRepository defines methods for session operations.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	UpdateSession(ctx context.Context, session *domain.Session) error
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) error
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error)
}

// Repository aggregates all user-related repositories.
type Repository interface {
	UserRepository
	RoleRepository
	PermissionRepository
	SessionRepository

	// Transaction support
	BeginTx(ctx context.Context) (Repository, error)
	Commit() error
	Rollback() error
}
