package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"gorm.io/gorm"
)

// GormRepository implements Repository using GORM
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM repository
func NewGormRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// BeginTx starts a new transaction
func (r *GormRepository) BeginTx(ctx context.Context) (Repository, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &GormRepository{db: tx}, nil
}

// Commit commits the transaction
func (r *GormRepository) Commit() error {
	return r.db.Commit().Error
}

// Rollback rolls back the transaction
func (r *GormRepository) Rollback() error {
	return r.db.Rollback().Error
}

// User operations

func (r *GormRepository) CreateUser(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		if errors.IsDuplicateError(err) {
			return errors.Conflict("username or email already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *GormRepository) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *GormRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "username = ?", username).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

func (r *GormRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "email = ?", email).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

func (r *GormRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("user not found")
	}
	return nil
}

func (r *GormRepository) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	var users []*domain.User
	if err := r.db.WithContext(ctx).
		Preload("Roles").
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (r *GormRepository) CountUsers(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (r *GormRepository) UserExists(ctx context.Context, username, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// Role operations

func (r *GormRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		if errors.IsDuplicateError(err) {
			return errors.Conflict("role name already exists")
		}
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *GormRepository) GetRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("role not found")
		}
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	return &role, nil
}

func (r *GormRepository) GetRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	var role domain.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("role not found")
		}
		return nil, fmt.Errorf("failed to get role by name: %w", err)
	}
	return &role, nil
}

func (r *GormRepository) UpdateRole(ctx context.Context, role *domain.Role) error {
	if err := r.db.WithContext(ctx).Save(role).Error; err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Role{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete role: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("role not found")
	}
	return nil
}

func (r *GormRepository) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	var roles []*domain.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (r *GormRepository) AssignPermissionsToRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	var permissions []domain.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return fmt.Errorf("failed to find permissions: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&domain.Role{ID: roleID}).Association("Permissions").Append(&permissions); err != nil {
		return fmt.Errorf("failed to assign permissions: %w", err)
	}
	return nil
}

func (r *GormRepository) RemovePermissionsFromRole(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	var permissions []domain.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return fmt.Errorf("failed to find permissions: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&domain.Role{ID: roleID}).Association("Permissions").Delete(&permissions); err != nil {
		return fmt.Errorf("failed to remove permissions: %w", err)
	}
	return nil
}

// Permission operations

func (r *GormRepository) CreatePermission(ctx context.Context, permission *domain.Permission) error {
	if err := r.db.WithContext(ctx).Create(permission).Error; err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}
	return nil
}

func (r *GormRepository) GetPermission(ctx context.Context, id uuid.UUID) (*domain.Permission, error) {
	var permission domain.Permission
	if err := r.db.WithContext(ctx).First(&permission, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return &permission, nil
}

func (r *GormRepository) GetPermissionByResourceAction(ctx context.Context, resource, action string) (*domain.Permission, error) {
	var permission domain.Permission
	if err := r.db.WithContext(ctx).First(&permission, "resource = ? AND action = ?", resource, action).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("permission not found")
		}
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}
	return &permission, nil
}

func (r *GormRepository) UpdatePermission(ctx context.Context, permission *domain.Permission) error {
	if err := r.db.WithContext(ctx).Save(permission).Error; err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}
	return nil
}

func (r *GormRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Permission{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete permission: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("permission not found")
	}
	return nil
}

func (r *GormRepository) ListPermissions(ctx context.Context) ([]*domain.Permission, error) {
	var permissions []*domain.Permission
	if err := r.db.WithContext(ctx).Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	return permissions, nil
}

// Session operations

func (r *GormRepository) CreateSession(ctx context.Context, session *domain.Session) error {
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *GormRepository) GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	var session domain.Session
	if err := r.db.WithContext(ctx).First(&session, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *GormRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	var session domain.Session
	if err := r.db.WithContext(ctx).First(&session, "refresh_token = ?", refreshToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NotFound("session not found")
		}
		return nil, fmt.Errorf("failed to get session by refresh token: %w", err)
	}
	return &session, nil
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *domain.Session) error {
	if err := r.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&domain.Session{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.NotFound("session not found")
	}
	return nil
}

func (r *GormRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Session{}, "user_id = ?", userID).Error; err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteExpiredSessions(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Delete(&domain.Session{}, "expires_at < NOW()").Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

func (r *GormRepository) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error) {
	var sessions []*domain.Session
	if err := r.db.WithContext(ctx).Find(&sessions, "user_id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("failed to list user sessions: %w", err)
	}
	return sessions, nil
}
