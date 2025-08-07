package auth_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/test/testutil"
)

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	// Setup
	jwtManager := auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.Roles = []domain.Role{
		*testutil.CreateTestRole(domain.RoleUser, "User role"),
	}
	sessionID := uuid.New()

	// Test
	tokens, err := jwtManager.GenerateTokenPair(user, sessionID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, tokens)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.Equal(t, int(15*time.Minute.Seconds()), tokens.ExpiresIn)
	assert.True(t, tokens.ExpiresAt.After(time.Now()))
}

func TestJWTManager_ValidateAccessToken_Success(t *testing.T) {
	// Setup
	jwtManager := auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.Roles = []domain.Role{
		*testutil.CreateTestRole(domain.RoleUser, "User role"),
		*testutil.CreateTestRole(domain.RoleAdmin, "Admin role"),
	}
	sessionID := uuid.New()

	tokens, err := jwtManager.GenerateTokenPair(user, sessionID)
	require.NoError(t, err)

	// Test
	claims, err := jwtManager.ValidateAccessToken(tokens.AccessToken)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, user.ID.String(), claims.UserID)
	assert.Equal(t, user.Username, claims.Username)
	assert.Equal(t, user.Email, claims.Email)
	assert.Equal(t, []string{domain.RoleUser, domain.RoleAdmin}, claims.Roles)
	assert.Equal(t, domain.TokenTypeAccess, claims.TokenType)
	assert.Equal(t, sessionID.String(), claims.SessionID)
	assert.Equal(t, "test-issuer", claims.Issuer)
}

func TestJWTManager_ValidateAccessToken_InvalidToken(t *testing.T) {
	// Setup
	jwtManager := auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	// Test
	claims, err := jwtManager.ValidateAccessToken("invalid.token.here")

	// Assert
	require.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Setup - create manager with very short TTL
	jwtManager := auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		1*time.Millisecond, // Very short TTL
		7*24*time.Hour,
	)

	user := testutil.CreateTestUser("testuser", "test@example.com")
	sessionID := uuid.New()

	tokens, err := jwtManager.GenerateTokenPair(user, sessionID)
	require.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Test
	claims, err := jwtManager.ValidateAccessToken(tokens.AccessToken)

	// Assert
	require.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_ValidateAccessToken_WrongSecret(t *testing.T) {
	// Setup - create two managers with different secrets
	jwtManager1 := auth.NewJWTManager(
		"secret1",
		"refresh-secret1",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	jwtManager2 := auth.NewJWTManager(
		"secret2", // Different secret
		"refresh-secret2",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	user := testutil.CreateTestUser("testuser", "test@example.com")
	sessionID := uuid.New()

	// Generate token with first manager
	tokens, err := jwtManager1.GenerateTokenPair(user, sessionID)
	require.NoError(t, err)

	// Try to validate with second manager (different secret)
	claims, err := jwtManager2.ValidateAccessToken(tokens.AccessToken)

	// Assert
	require.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTManager_ValidateRefreshToken(t *testing.T) {
	// Setup
	jwtManager := auth.NewJWTManager(
		"test-access-secret",
		"test-refresh-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
	)

	user := testutil.CreateTestUser("testuser", "test@example.com")
	sessionID := uuid.New()

	_, err := jwtManager.GenerateTokenPair(user, sessionID)
	require.NoError(t, err)

	// Test - Note: The refresh token returned by GenerateTokenPair is a random string,
	// not a JWT. So we need to test with the actual JWT refresh token
	// For this test, we'll validate that the method exists and works with proper JWT

	// Create a proper JWT refresh token for testing
	claims := &auth.CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "test-issuer",
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
		UserID:    user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Roles:     []string{domain.RoleUser},
		TokenType: domain.TokenTypeRefresh,
		SessionID: sessionID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshToken, err := token.SignedString([]byte("test-refresh-secret"))
	require.NoError(t, err)

	// Validate the refresh token
	validatedClaims, err := jwtManager.ValidateRefreshToken(refreshToken)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, validatedClaims)
	assert.Equal(t, user.ID.String(), validatedClaims.UserID)
	assert.Equal(t, domain.TokenTypeRefresh, validatedClaims.TokenType)
}

func TestGenerateRefreshToken(t *testing.T) {
	// Test
	token1, err1 := auth.GenerateRefreshToken()
	token2, err2 := auth.GenerateRefreshToken()

	// Assert
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEmpty(t, token1)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should be unique

	// Verify it's base64 encoded
	assert.Regexp(t, `^[A-Za-z0-9+/\-_=]+$`, token1)
}

func TestGenerateSecret(t *testing.T) {
	// Test
	secret1 := auth.GenerateSecret()
	secret2 := auth.GenerateSecret()

	// Assert
	assert.NotEmpty(t, secret1)
	assert.NotEmpty(t, secret2)
	assert.NotEqual(t, secret1, secret2) // Should be unique

	// Verify it's base64 encoded
	assert.Regexp(t, `^[A-Za-z0-9+/=]+$`, secret1)
}

func TestDefaultConfig(t *testing.T) {
	// Test
	config := auth.DefaultConfig()

	// Assert
	assert.NotNil(t, config)
	assert.NotEmpty(t, config.AccessSecret)
	assert.NotEmpty(t, config.RefreshSecret)
	assert.NotEqual(t, config.AccessSecret, config.RefreshSecret)
	assert.Equal(t, "narwhal", config.Issuer)
	assert.Equal(t, 15*time.Minute, config.AccessTTL)
	assert.Equal(t, 7*24*time.Hour, config.RefreshTTL)
}
