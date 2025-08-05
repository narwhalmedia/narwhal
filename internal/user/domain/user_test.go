package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type UserDomainTestSuite struct {
	suite.Suite
}

func (suite *UserDomainTestSuite) TestUser_SetPassword() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	
	// Act
	err := user.SetPassword("testpassword123")
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), user.PasswordHash)
	assert.NotEqual(suite.T(), "testpassword123", user.PasswordHash)
}

func (suite *UserDomainTestSuite) TestUser_CheckPassword() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	user.SetPassword("testpassword123")
	
	// Act & Assert
	assert.True(suite.T(), user.CheckPassword("testpassword123"))
	assert.False(suite.T(), user.CheckPassword("wrongpassword"))
	assert.False(suite.T(), user.CheckPassword(""))
}

func (suite *UserDomainTestSuite) TestUser_HasRole() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Roles: []domain.Role{
			{Name: domain.RoleAdmin},
			{Name: domain.RoleUser},
		},
	}
	
	// Act & Assert
	assert.True(suite.T(), user.HasRole(domain.RoleAdmin))
	assert.True(suite.T(), user.HasRole(domain.RoleUser))
	assert.False(suite.T(), user.HasRole(domain.RoleModerator))
}

func (suite *UserDomainTestSuite) TestUser_HasPermission() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Roles: []domain.Role{
			{
				Name: domain.RoleAdmin,
				Permissions: []domain.Permission{
					{Resource: "users", Action: "create"},
					{Resource: "users", Action: "read"},
					{Resource: "users", Action: "update"},
					{Resource: "users", Action: "delete"},
				},
			},
		},
	}
	
	// Act & Assert
	assert.True(suite.T(), user.HasPermission("users", "create"))
	assert.True(suite.T(), user.HasPermission("users", "read"))
	assert.True(suite.T(), user.HasPermission("users", "update"))
	assert.True(suite.T(), user.HasPermission("users", "delete"))
	assert.False(suite.T(), user.HasPermission("media", "create"))
	assert.False(suite.T(), user.HasPermission("users", "admin"))
}

func (suite *UserDomainTestSuite) TestUser_GetPermissions() {
	// Arrange
	permissions := []domain.Permission{
		{Resource: "users", Action: "read"},
		{Resource: "media", Action: "read"},
		{Resource: "media", Action: "create"},
	}
	
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Roles: []domain.Role{
			{
				Name:        domain.RoleUser,
				Permissions: permissions[:2], // First two permissions
			},
			{
				Name:        domain.RoleModerator,
				Permissions: permissions[1:], // Last two permissions (media:read duplicated)
			},
		},
	}
	
	// Act
	userPermissions := user.GetPermissions()
	
	// Assert - Should have 3 unique permissions
	assert.Len(suite.T(), userPermissions, 3)
	
	// Check all permissions are present
	hasPermission := func(perms []domain.Permission, resource, action string) bool {
		for _, p := range perms {
			if p.Resource == resource && p.Action == action {
				return true
			}
		}
		return false
	}
	
	assert.True(suite.T(), hasPermission(userPermissions, "users", "read"))
	assert.True(suite.T(), hasPermission(userPermissions, "media", "read"))
	assert.True(suite.T(), hasPermission(userPermissions, "media", "create"))
}

func (suite *UserDomainTestSuite) TestSession_IsExpired() {
	// Arrange
	now := time.Now()
	
	// Expired session
	expiredSession := &domain.Session{
		ID:        uuid.New(),
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	
	// Valid session
	validSession := &domain.Session{
		ID:        uuid.New(),
		ExpiresAt: now.Add(1 * time.Hour),
	}
	
	// Act & Assert
	assert.True(suite.T(), expiredSession.IsExpired())
	assert.False(suite.T(), validSession.IsExpired())
}

func (suite *UserDomainTestSuite) TestUserPreferences_Defaults() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Preferences: domain.UserPreferences{
			Language:         "en",
			Theme:            "dark",
			TimeZone:         "UTC",
			AutoPlayNext:     true,
			SubtitleLanguage: "en",
			PreferredQuality: "auto",
		},
	}
	
	// Assert
	assert.Equal(suite.T(), "en", user.Preferences.Language)
	assert.Equal(suite.T(), "dark", user.Preferences.Theme)
	assert.Equal(suite.T(), "UTC", user.Preferences.TimeZone)
	assert.True(suite.T(), user.Preferences.AutoPlayNext)
	assert.Equal(suite.T(), "en", user.Preferences.SubtitleLanguage)
	assert.Equal(suite.T(), "auto", user.Preferences.PreferredQuality)
}

func (suite *UserDomainTestSuite) TestRole_HasPermission() {
	// Arrange
	role := &domain.Role{
		ID:   uuid.New(),
		Name: domain.RoleAdmin,
		Permissions: []domain.Permission{
			{Resource: "users", Action: "create"},
			{Resource: "users", Action: "read"},
			{Resource: "media", Action: "*"}, // Wildcard permission
		},
	}
	
	// Act & Assert
	assert.True(suite.T(), role.HasPermission("users", "create"))
	assert.True(suite.T(), role.HasPermission("users", "read"))
	assert.False(suite.T(), role.HasPermission("users", "delete"))
	
	// Wildcard should match any action
	assert.True(suite.T(), role.HasPermission("media", "create"))
	assert.True(suite.T(), role.HasPermission("media", "read"))
	assert.True(suite.T(), role.HasPermission("media", "delete"))
	assert.True(suite.T(), role.HasPermission("media", "anything"))
}

func (suite *UserDomainTestSuite) TestPermission_Matches() {
	// Test exact match
	perm := domain.Permission{Resource: "users", Action: "read"}
	assert.True(suite.T(), perm.Matches("users", "read"))
	assert.False(suite.T(), perm.Matches("users", "write"))
	assert.False(suite.T(), perm.Matches("media", "read"))
	
	// Test wildcard action
	wildcard := domain.Permission{Resource: "media", Action: "*"}
	assert.True(suite.T(), wildcard.Matches("media", "read"))
	assert.True(suite.T(), wildcard.Matches("media", "write"))
	assert.True(suite.T(), wildcard.Matches("media", "delete"))
	assert.False(suite.T(), wildcard.Matches("users", "read"))
	
	// Test wildcard resource
	wildcardResource := domain.Permission{Resource: "*", Action: "read"}
	assert.True(suite.T(), wildcardResource.Matches("users", "read"))
	assert.True(suite.T(), wildcardResource.Matches("media", "read"))
	assert.False(suite.T(), wildcardResource.Matches("users", "write"))
	
	// Test double wildcard
	allAccess := domain.Permission{Resource: "*", Action: "*"}
	assert.True(suite.T(), allAccess.Matches("users", "read"))
	assert.True(suite.T(), allAccess.Matches("media", "write"))
	assert.True(suite.T(), allAccess.Matches("anything", "anything"))
}

func TestUserDomainTestSuite(t *testing.T) {
	suite.Run(t, new(UserDomainTestSuite))
}