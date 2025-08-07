package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/narwhalmedia/narwhal/internal/user/constants"
	"github.com/narwhalmedia/narwhal/internal/user/repository"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// UserService handles user management operations.
type UserService struct {
	repo     repository.Repository
	eventBus interfaces.EventBus
	cache    interfaces.Cache
	logger   interfaces.Logger
}

// NewUserService creates a new user service.
func NewUserService(
	repo repository.Repository,
	eventBus interfaces.EventBus,
	cache interfaces.Cache,
	logger interfaces.Logger,
) *UserService {
	return &UserService{
		repo:     repo,
		eventBus: eventBus,
		cache:    cache,
		logger:   logger,
	}
}

// CreateUser creates a new user.
func (s *UserService) CreateUser(
	ctx context.Context,
	username, email, password, displayName string,
) (*models.User, error) {
	// Validate input
	if username == "" || email == "" || password == "" {
		return nil, errors.BadRequest("username, email, and password are required")
	}

	// Normalize username and email
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if user exists
	exists, err := s.repo.UserExists(ctx, username, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.Conflict("username or email already exists")
	}

	// Create user
	user := &models.User{
		ID:          uuid.New(),
		Username:    username,
		Email:       email,
		DisplayName: displayName,
		IsActive:    true,
		IsVerified:  false,
	}

	// Hash password
	if err := user.SetPassword(password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Assign default user role
	defaultRole, err := s.repo.GetRoleByName(ctx, models.RoleUser)
	if err != nil {
		// Create default role if it doesn't exist
		defaultRole = &models.Role{
			ID:          uuid.New(),
			Name:        models.RoleUser,
			Description: "Default user role",
		}
		if err := s.repo.CreateRole(ctx, defaultRole); err != nil {
			return nil, fmt.Errorf("failed to create default role: %w", err)
		}
	}
	user.Roles = []models.Role{*defaultRole}

	// Create user
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.created", map[string]interface{}{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
	}))

	s.logger.Info("User created",
		interfaces.String("user_id", user.ID.String()),
		interfaces.String("username", user.Username))

	return user, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("user:%s", id.String())
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		if user, ok := cached.(*models.User); ok {
			return user, nil
		}
	}

	// Get from repository
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	_ = s.cache.Set(ctx, cacheKey, user, constants.CacheTTL)
	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.repo.GetUserByUsername(ctx, strings.ToLower(username))
}

// UpdateUser updates a user's information.
func (s *UserService) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	updates map[string]interface{},
) (*models.User, error) {
	// Get existing user
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if displayName, ok := updates["display_name"].(string); ok {
		user.DisplayName = displayName
	}
	if avatar, ok := updates["avatar"].(string); ok {
		user.Avatar = avatar
	}
	if prefs, ok := updates["preferences"].(models.User); ok {
		// This is tricky because preferences are flattened.
		// A better approach would be to pass a dedicated preferences object.
		// For now, let's assume the caller passes the correct fields.
		if prefs.PrefLanguage != "" {
			user.PrefLanguage = prefs.PrefLanguage
		}
		if prefs.PrefTheme != "" {
			user.PrefTheme = prefs.PrefTheme
		}
		// ... and so on for other preferences
	}

	// Update user
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s", id.String()))

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.updated", map[string]interface{}{
		"user_id": user.ID,
		"updates": updates,
	}))

	return user, nil
}

// ChangePassword changes a user's password.
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error {
	// Get user
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Verify old password
	if !user.CheckPassword(oldPassword) {
		return errors.Unauthorized("incorrect password")
	}

	// Set new password
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate all sessions to force re-login
	_ = s.repo.DeleteUserSessions(ctx, userID)
	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.password_changed", map[string]interface{}{
		"user_id": userID,
	}))

	s.logger.Info("User password changed",
		interfaces.String("user_id", userID.String()))

	return nil
}

// DeleteUser deletes a user.
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Get user to verify existence
	user, err := s.repo.GetUser(ctx, id)
	if err != nil {
		return err
	}

	// Delete all user sessions
	_ = s.repo.DeleteUserSessions(ctx, id)
	// Delete user
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s", id.String()))

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.deleted", map[string]interface{}{
		"user_id":  id,
		"username": user.Username,
	}))

	s.logger.Info("User deleted",
		interfaces.String("user_id", id.String()),
		interfaces.String("username", user.Username))

	return nil
}

// ListUsers lists all users with pagination.
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > constants.MaxPageSize {
		limit = 200
	}

	users, err := s.repo.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// AssignRole assigns a role to a user.
func (s *UserService) AssignRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	// Get user
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Check if user already has the role
	if user.HasRole(roleName) {
		return nil // Already has the role
	}

	// Get role
	role, err := s.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		return err
	}

	// Add role to user
	user.Roles = append(user.Roles, *role)

	// Update user
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s", userID.String()))

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.role_assigned", map[string]interface{}{
		"user_id":   userID,
		"role_name": roleName,
	}))

	s.logger.Info("Role assigned to user",
		interfaces.String("user_id", userID.String()),
		interfaces.String("role", roleName))

	return nil
}

// RemoveRole removes a role from a user.
func (s *UserService) RemoveRole(ctx context.Context, userID uuid.UUID, roleName string) error {
	// Get user
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Remove role
	newRoles := make([]models.Role, 0, len(user.Roles))
	removed := false
	for _, role := range user.Roles {
		if role.Name != roleName {
			newRoles = append(newRoles, role)
		} else {
			removed = true
		}
	}

	if !removed {
		return nil // User doesn't have the role
	}

	user.Roles = newRoles

	// Update user
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s", userID.String()))

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.role_removed", map[string]interface{}{
		"user_id":   userID,
		"role_name": roleName,
	}))

	s.logger.Info("Role removed from user",
		interfaces.String("user_id", userID.String()),
		interfaces.String("role", roleName))

	return nil
}

// SetUserActive activates or deactivates a user.
func (s *UserService) SetUserActive(ctx context.Context, userID uuid.UUID, active bool) error {
	// Get user
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	// Update status
	user.IsActive = active

	// Update user
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return err
	}

	// If deactivating, delete all sessions
	if !active {
		_ = s.repo.DeleteUserSessions(ctx, userID)
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("user:%s", userID.String()))

	// Publish event
	eventType := "user.activated"
	if !active {
		eventType = "user.deactivated"
	}

	s.eventBus.PublishAsync(ctx, events.NewEvent(eventType, map[string]interface{}{
		"user_id": userID,
	}))

	s.logger.Info("User status changed",
		interfaces.String("user_id", userID.String()),
		interfaces.Bool("active", active))

	return nil
}
