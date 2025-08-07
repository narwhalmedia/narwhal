package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/pkg/repository"
)

// GormRepository implements Repository using GORM.
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM repository.
func NewGormRepository(db *gorm.DB) Repository {
	return &GormRepository{db: db}
}

// BeginTx starts a new transaction.
func (r *GormRepository) BeginTx(ctx context.Context) (Repository, error) {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &GormRepository{db: tx}, nil
}

// Commit commits the transaction.
func (r *GormRepository) Commit() error {
	return r.db.Commit().Error
}

// Rollback rolls back the transaction.
func (r *GormRepository) Rollback() error {
	return r.db.Rollback().Error
}

// User operations

func (r *GormRepository) CreateUser(ctx context.Context, user *models.User) error {
	return repository.Create(ctx, r.db, user)
}

func (r *GormRepository) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	return repository.FindByID[models.User](ctx, r.db, id, "Roles.Permissions")
}

func (r *GormRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "username = ?", username).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormRepository) UpdateUser(ctx context.Context, user *models.User) error {
	return repository.Update(ctx, r.db, user)
}

func (r *GormRepository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.User](ctx, r.db, id)
}

func (r *GormRepository) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	return repository.List[models.User](ctx, r.db, limit, offset, "Roles")
}

func (r *GormRepository) CountUsers(ctx context.Context) (int64, error) {
	return repository.Count[models.User](ctx, r.db)
}

func (r *GormRepository) UserExists(ctx context.Context, username, email string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("username = ? OR email = ?", username, email).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}
	return count > 0, nil
}

// Role operations

func (r *GormRepository) CreateRole(ctx context.Context, role *models.Role) error {
	return repository.Create(ctx, r.db, role)
}

func (r *GormRepository) GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	return repository.FindByID[models.Role](ctx, r.db, id, "Permissions")
}

func (r *GormRepository) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *GormRepository) UpdateRole(ctx context.Context, role *models.Role) error {
	return repository.Update(ctx, r.db, role)
}

func (r *GormRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Role](ctx, r.db, id)
}

func (r *GormRepository) ListRoles(ctx context.Context) ([]*models.Role, error) {
	var roles []*models.Role
	if err := r.db.WithContext(ctx).Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	return roles, nil
}

func (r *GormRepository) AssignPermissionsToRole(
	ctx context.Context,
	roleID uuid.UUID,
	permissionIDs []uuid.UUID,
) error {
	var permissions []models.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return fmt.Errorf("failed to find permissions: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&models.Role{ID: roleID}).Association("Permissions").Append(&permissions); err != nil {
		return fmt.Errorf("failed to assign permissions: %w", err)
	}
	return nil
}

func (r *GormRepository) RemovePermissionsFromRole(
	ctx context.Context,
	roleID uuid.UUID,
	permissionIDs []uuid.UUID,
) error {
	var permissions []models.Permission
	if err := r.db.WithContext(ctx).Find(&permissions, "id IN ?", permissionIDs).Error; err != nil {
		return fmt.Errorf("failed to find permissions: %w", err)
	}

	if err := r.db.WithContext(ctx).Model(&models.Role{ID: roleID}).Association("Permissions").Delete(&permissions); err != nil {
		return fmt.Errorf("failed to remove permissions: %w", err)
	}
	return nil
}

// Permission operations

func (r *GormRepository) CreatePermission(ctx context.Context, permission *models.Permission) error {
	return repository.Create(ctx, r.db, permission)
}

func (r *GormRepository) GetPermission(ctx context.Context, id uuid.UUID) (*models.Permission, error) {
	return repository.FindByID[models.Permission](ctx, r.db, id)
}

func (r *GormRepository) GetPermissionByResourceAction(
	ctx context.Context,
	resource, action string,
) (*models.Permission, error) {
	return repository.FindOneBy[models.Permission](ctx, r.db, "resource = ? AND action = ?", resource, action)
}

func (r *GormRepository) UpdatePermission(ctx context.Context, permission *models.Permission) error {
	return repository.Update(ctx, r.db, permission)
}

func (r *GormRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Permission](ctx, r.db, id)
}

func (r *GormRepository) ListPermissions(ctx context.Context) ([]*models.Permission, error) {
	var permissions []*models.Permission
	if err := r.db.WithContext(ctx).Find(&permissions).Error; err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	return permissions, nil
}

// Session operations

func (r *GormRepository) CreateSession(ctx context.Context, session *models.Session) error {
	return repository.Create(ctx, r.db, session)
}

func (r *GormRepository) GetSession(ctx context.Context, id uuid.UUID) (*models.Session, error) {
	return repository.FindByID[models.Session](ctx, r.db, id)
}

func (r *GormRepository) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*models.Session, error) {
	return repository.FindOneBy[models.Session](ctx, r.db, "refresh_token = ?", refreshToken)
}

func (r *GormRepository) UpdateSession(ctx context.Context, session *models.Session) error {
	return repository.Update(ctx, r.db, session)
}

func (r *GormRepository) DeleteSession(ctx context.Context, id uuid.UUID) error {
	return repository.Delete[models.Session](ctx, r.db, id)
}

func (r *GormRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Session{}, "user_id = ?", userID).Error; err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *GormRepository) DeleteExpiredSessions(ctx context.Context) error {
	if err := r.db.WithContext(ctx).Delete(&models.Session{}, "expires_at < NOW()").Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

func (r *GormRepository) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*models.Session, error) {
	var sessions []*models.Session
	if err := r.db.WithContext(ctx).Find(&sessions, "user_id = ?", userID).Error; err != nil {
		return nil, fmt.Errorf("failed to list user sessions: %w", err)
	}
	return sessions, nil
}
