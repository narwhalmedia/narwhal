package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/narwhalmedia/narwhal/pkg/models"
)

// UserRepository defines methods for user operations.
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error)
	CountUsers(ctx context.Context) (int64, error)
	UserExists(ctx context.Context, username, email string) (bool, error)
}

// RoleRepository defines methods for role operations.
type RoleRepository interface {
	CreateRole(ctx context.Context, role *models.Role) error
	GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	UpdateRole(ctx context.Context, role *models.Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ListRoles(ctx context.Context) ([]*models.Role, error)
	AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
	RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error
}

// PermissionRepository defines methods for permission operations.
type PermissionRepository interface {
	CreatePermission(ctx context.Context, permission *models.Permission) error
	GetPermission(ctx context.Context, id uuid.UUID) (*models.Permission, error)
	GetPermissionByResourceAction(ctx context.Context, resource, action string) (*models.Permission, error)
	UpdatePermission(ctx context.Context, permission *models.Permission) error
	DeletePermission(ctx context.Context, id uuid.UUID) error
	ListPermissions(ctx context.Context) ([]*models.Permission, error)
}

// SessionRepository defines methods for session operations.
type SessionRepository interface {
	CreateSession(ctx context.Context, session *models.Session) error
	GetSession(ctx context.Context, id uuid.UUID) (*models.Session, error)
	GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error)
	UpdateSession(ctx context.Context, session *models.Session) error
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context) error
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error)
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
