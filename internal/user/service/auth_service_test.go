package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/service"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/test/mocks"
	"github.com/narwhalmedia/narwhal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AuthServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	mockRepo    *mocks.MockRepository
	authService *service.AuthService
	jwtManager  *auth.JWTManager
	eventBus    *events.LocalEventBus
}

func (suite *AuthServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockRepo = new(mocks.MockRepository)
	suite.eventBus = events.NewLocalEventBus(logger.NewNoopLogger())

	// Create JWT manager with test configuration
	suite.jwtManager = auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	suite.authService = service.NewAuthService(
		suite.mockRepo,
		suite.jwtManager,
		suite.eventBus,
		logger.NewNoopLogger(),
	)
}

func (suite *AuthServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *AuthServiceTestSuite) TestLogin_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("password123")

	suite.mockRepo.On("GetUserByUsername", suite.ctx, "testuser").Return(user, nil)
	suite.mockRepo.On("CreateSession", suite.ctx, mock.AnythingOfType("*domain.Session")).Return(nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Act
	tokens, err := suite.authService.Login(suite.ctx, "testuser", "password123", "Test Device", "127.0.0.1", "Test/1.0")

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), tokens)
	assert.NotEmpty(suite.T(), tokens.AccessToken)
	assert.NotEmpty(suite.T(), tokens.RefreshToken)
	assert.Equal(suite.T(), "Bearer", tokens.TokenType)
	assert.Greater(suite.T(), tokens.ExpiresIn, 0)
}

func (suite *AuthServiceTestSuite) TestLogin_InvalidCredentials() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("password123")

	suite.mockRepo.On("GetUserByUsername", suite.ctx, "testuser").Return(user, nil)

	// Act
	tokens, err := suite.authService.Login(suite.ctx, "testuser", "wrongpassword", "Test Device", "127.0.0.1", "Test/1.0")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), tokens)
	assert.True(suite.T(), errors.IsUnauthorized(err))
}

func (suite *AuthServiceTestSuite) TestLogin_UserNotFound() {
	// Arrange
	suite.mockRepo.On("GetUserByUsername", suite.ctx, "nonexistent").Return(nil, errors.NotFound("user not found"))
	suite.mockRepo.On("GetUserByEmail", suite.ctx, "nonexistent").Return(nil, errors.NotFound("user not found"))

	// Act
	tokens, err := suite.authService.Login(suite.ctx, "nonexistent", "password123", "Test Device", "127.0.0.1", "Test/1.0")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), tokens)
	assert.True(suite.T(), errors.IsUnauthorized(err))
}

func (suite *AuthServiceTestSuite) TestLogin_InactiveUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("password123")
	user.IsActive = false

	suite.mockRepo.On("GetUserByUsername", suite.ctx, "testuser").Return(user, nil)

	// Act
	tokens, err := suite.authService.Login(suite.ctx, "testuser", "password123", "Test Device", "127.0.0.1", "Test/1.0")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), tokens)
	assert.True(suite.T(), errors.IsForbidden(err))
}

func (suite *AuthServiceTestSuite) TestRefreshToken_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	session := testutil.CreateTestSession(user.ID)

	suite.mockRepo.On("GetSessionByRefreshToken", suite.ctx, session.RefreshToken).Return(session, nil)
	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("UpdateSession", suite.ctx, mock.AnythingOfType("*domain.Session")).Return(nil)

	// Act
	tokens, err := suite.authService.RefreshToken(suite.ctx, session.RefreshToken)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), tokens)
	assert.NotEmpty(suite.T(), tokens.AccessToken)
	assert.Equal(suite.T(), session.RefreshToken, tokens.RefreshToken)
}

func (suite *AuthServiceTestSuite) TestRefreshToken_ExpiredSession() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	session := testutil.CreateTestSession(user.ID)
	session.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired

	suite.mockRepo.On("GetSessionByRefreshToken", suite.ctx, session.RefreshToken).Return(session, nil)
	suite.mockRepo.On("DeleteSession", suite.ctx, session.ID).Return(nil)

	// Act
	tokens, err := suite.authService.RefreshToken(suite.ctx, session.RefreshToken)

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), tokens)
	assert.True(suite.T(), errors.IsUnauthorized(err))
}

func (suite *AuthServiceTestSuite) TestLogout_Success() {
	// Arrange
	userID := uuid.New()
	sessionID := uuid.New()
	session := &domain.Session{
		ID:     sessionID,
		UserID: userID,
	}

	suite.mockRepo.On("GetSession", suite.ctx, sessionID).Return(session, nil)
	suite.mockRepo.On("DeleteSession", suite.ctx, sessionID).Return(nil)

	// Act
	err := suite.authService.Logout(suite.ctx, userID, sessionID.String())

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestLogout_SessionNotBelongToUser() {
	// Arrange
	userID := uuid.New()
	otherUserID := uuid.New()
	sessionID := uuid.New()
	session := &domain.Session{
		ID:     sessionID,
		UserID: otherUserID, // Different user
	}

	suite.mockRepo.On("GetSession", suite.ctx, sessionID).Return(session, nil)

	// Act
	err := suite.authService.Logout(suite.ctx, userID, sessionID.String())

	// Assert
	assert.Error(suite.T(), err)
	assert.True(suite.T(), errors.IsForbidden(err))
}

func (suite *AuthServiceTestSuite) TestLogoutAll_Success() {
	// Arrange
	userID := uuid.New()

	suite.mockRepo.On("DeleteUserSessions", suite.ctx, userID).Return(nil)

	// Act
	err := suite.authService.LogoutAll(suite.ctx, userID)

	// Assert
	assert.NoError(suite.T(), err)
}

func (suite *AuthServiceTestSuite) TestValidateToken_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	sessionID := uuid.New()

	// Generate a valid token
	tokens, _ := suite.jwtManager.GenerateTokenPair(user, sessionID)

	suite.mockRepo.On("GetSession", suite.ctx, sessionID).Return(&domain.Session{ID: sessionID}, nil)

	// Act
	claims, err := suite.authService.ValidateToken(suite.ctx, tokens.AccessToken)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), claims)
	assert.Equal(suite.T(), user.ID.String(), claims.UserID)
	assert.Equal(suite.T(), user.Username, claims.Username)
}

func (suite *AuthServiceTestSuite) TestValidateToken_InvalidToken() {
	// Act
	claims, err := suite.authService.ValidateToken(suite.ctx, "invalid-token")

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), claims)
	assert.True(suite.T(), errors.IsUnauthorized(err))
}

func (suite *AuthServiceTestSuite) TestGetUserSessions_Success() {
	// Arrange
	userID := uuid.New()
	sessions := []*domain.Session{
		testutil.CreateTestSession(userID),
		testutil.CreateTestSession(userID),
	}

	suite.mockRepo.On("ListUserSessions", suite.ctx, userID).Return(sessions, nil)

	// Act
	result, err := suite.authService.GetUserSessions(suite.ctx, userID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
}

func (suite *AuthServiceTestSuite) TestCleanupExpiredSessions_Success() {
	// Arrange
	suite.mockRepo.On("DeleteExpiredSessions", suite.ctx).Return(nil)

	// Act
	err := suite.authService.CleanupExpiredSessions(suite.ctx)

	// Assert
	assert.NoError(suite.T(), err)
}

func TestAuthServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceTestSuite))
}
