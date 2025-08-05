package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/repository"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// AuthService handles authentication operations
type AuthService struct {
	repo       repository.Repository
	jwtManager *auth.JWTManager
	eventBus   interfaces.EventBus
	logger     interfaces.Logger
}

// NewAuthService creates a new authentication service
func NewAuthService(
	repo repository.Repository,
	jwtManager *auth.JWTManager,
	eventBus interfaces.EventBus,
	logger interfaces.Logger,
) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtManager: jwtManager,
		eventBus:   eventBus,
		logger:     logger,
	}
}

// Login authenticates a user and returns auth tokens
func (s *AuthService) Login(ctx context.Context, username, password, deviceInfo, ipAddress, userAgent string) (*domain.AuthTokens, error) {
	// Find user by username or email
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		// Try email if username fails
		user, err = s.repo.GetUserByEmail(ctx, username)
		if err != nil {
			return nil, errors.Unauthorized("invalid credentials")
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.Forbidden("account is disabled")
	}

	// Verify password
	if !user.CheckPassword(password) {
		return nil, errors.Unauthorized("invalid credentials")
	}

	// Generate refresh token
	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create session
	session := &domain.Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour), // 7 days
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate JWT tokens
	tokens, err := s.jwtManager.GenerateTokenPair(user, session.ID)
	if err != nil {
		// Rollback session creation
		s.repo.DeleteSession(ctx, session.ID)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update refresh token in response
	tokens.RefreshToken = refreshToken

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	s.repo.UpdateUser(ctx, user)

	// Publish login event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.logged_in", map[string]interface{}{
		"user_id":    user.ID,
		"username":   user.Username,
		"ip_address": ipAddress,
		"user_agent": userAgent,
	}))

	s.logger.Info("User logged in",
		interfaces.String("user_id", user.ID.String()),
		interfaces.String("username", user.Username))

	return tokens, nil
}

// RefreshToken generates new tokens using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthTokens, error) {
	// Find session by refresh token
	session, err := s.repo.GetSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.Unauthorized("invalid refresh token")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		s.repo.DeleteSession(ctx, session.ID)
		return nil, errors.Unauthorized("refresh token expired")
	}

	// Get user
	user, err := s.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, err
	}

	// Check if user is still active
	if !user.IsActive {
		s.repo.DeleteSession(ctx, session.ID)
		return nil, errors.Forbidden("account is disabled")
	}

	// Generate new tokens
	tokens, err := s.jwtManager.GenerateTokenPair(user, session.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Keep the same refresh token
	tokens.RefreshToken = refreshToken

	// Update session activity
	session.UpdatedAt = time.Now()
	s.repo.UpdateSession(ctx, session)

	return tokens, nil
}

// Logout invalidates a user's session
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, sessionID string) error {
	// Parse session ID
	sid, err := uuid.Parse(sessionID)
	if err != nil {
		return errors.BadRequest("invalid session ID")
	}

	// Get session to verify ownership
	session, err := s.repo.GetSession(ctx, sid)
	if err != nil {
		return err
	}

	// Verify session belongs to user
	if session.UserID != userID {
		return errors.Forbidden("session does not belong to user")
	}

	// Delete session
	if err := s.repo.DeleteSession(ctx, sid); err != nil {
		return err
	}

	// Publish logout event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.logged_out", map[string]interface{}{
		"user_id":    userID,
		"session_id": sessionID,
	}))

	s.logger.Info("User logged out",
		interfaces.String("user_id", userID.String()),
		interfaces.String("session_id", sessionID))

	return nil
}

// LogoutAll invalidates all of a user's sessions
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.repo.DeleteUserSessions(ctx, userID); err != nil {
		return err
	}

	// Publish event
	s.eventBus.PublishAsync(ctx, events.NewEvent("user.logged_out_all", map[string]interface{}{
		"user_id": userID,
	}))

	s.logger.Info("User logged out from all sessions",
		interfaces.String("user_id", userID.String()))

	return nil
}

// ValidateToken validates an access token and returns user info
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*auth.CustomClaims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, errors.Unauthorized("invalid token")
	}

	// Verify session still exists if session ID is present
	if claims.SessionID != "" {
		sessionID, err := uuid.Parse(claims.SessionID)
		if err == nil {
			if _, err := s.repo.GetSession(ctx, sessionID); err != nil {
				return nil, errors.Unauthorized("session not found")
			}
		}
	}

	return claims, nil
}

// GetUserSessions returns all active sessions for a user
func (s *AuthService) GetUserSessions(ctx context.Context, userID uuid.UUID) ([]*domain.Session, error) {
	return s.repo.ListUserSessions(ctx, userID)
}

// RevokeSession revokes a specific session
func (s *AuthService) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	// Get session to verify ownership
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	// Verify session belongs to user
	if session.UserID != userID {
		return errors.Forbidden("session does not belong to user")
	}

	// Delete session
	return s.repo.DeleteSession(ctx, sessionID)
}

// CleanupExpiredSessions removes expired sessions
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) error {
	if err := s.repo.DeleteExpiredSessions(ctx); err != nil {
		return err
	}

	s.logger.Info("Cleaned up expired sessions")
	return nil
}
