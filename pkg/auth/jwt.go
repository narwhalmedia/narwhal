package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
)

// JWTManager handles JWT token operations
type JWTManager struct {
	accessSecret  string
	refreshSecret string
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(accessSecret, refreshSecret, issuer string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		issuer:        issuer,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

// CustomClaims extends jwt.RegisteredClaims with our custom fields
type CustomClaims struct {
	jwt.RegisteredClaims
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
	SessionID string   `json:"session_id,omitempty"`
}

// GenerateTokenPair generates both access and refresh tokens
func (j *JWTManager) GenerateTokenPair(user *domain.User, sessionID uuid.UUID) (*domain.AuthTokens, error) {
	// Extract role names
	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.Name
	}

	// Generate access token
	accessToken, err := j.generateToken(user, roleNames, sessionID.String(), domain.TokenTypeAccess, j.accessSecret, j.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := j.generateToken(user, roleNames, sessionID.String(), domain.TokenTypeRefresh, j.refreshSecret, j.refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().Add(j.accessTTL)

	return &domain.AuthTokens{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(j.accessTTL.Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// generateToken creates a JWT token with the specified parameters
func (j *JWTManager) generateToken(user *domain.User, roles []string, sessionID, tokenType, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		UserID:    user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		Roles:     roles,
		TokenType: tokenType,
		SessionID: sessionID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateAccessToken validates an access token and returns the claims
func (j *JWTManager) ValidateAccessToken(tokenString string) (*CustomClaims, error) {
	return j.validateToken(tokenString, j.accessSecret)
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (j *JWTManager) ValidateRefreshToken(tokenString string) (*CustomClaims, error) {
	return j.validateToken(tokenString, j.refreshSecret)
}

// validateToken parses and validates a JWT token
func (j *JWTManager) validateToken(tokenString, secret string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GenerateRefreshToken generates a cryptographically secure refresh token
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Config holds JWT configuration
type Config struct {
	AccessSecret  string
	RefreshSecret string
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

// DefaultConfig returns default JWT configuration
func DefaultConfig() *Config {
	return &Config{
		AccessSecret:  GenerateSecret(),
		RefreshSecret: GenerateSecret(),
		Issuer:        "narwhal",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour, // 7 days
	}
}

// GenerateSecret generates a random secret for JWT signing
func GenerateSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
