package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
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
	suite.Require().NoError(err)
	suite.NotEmpty(user.PasswordHash)
	suite.NotEqual("testpassword123", user.PasswordHash)
}

func (suite *UserDomainTestSuite) TestUser_CheckPassword() {
	// Arrange
	user := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
	}
	user.SetPassword("testpassword123")

	// Act & Assert
	suite.True(user.CheckPassword("testpassword123"))
	suite.False(user.CheckPassword("wrongpassword"))
	suite.False(user.CheckPassword(""))
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
	suite.True(user.HasRole(domain.RoleAdmin))
	suite.True(user.HasRole(domain.RoleUser))
	suite.False(user.HasRole(domain.RoleModerator))
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
	suite.True(user.HasPermission("users", "create"))
	suite.True(user.HasPermission("users", "read"))
	suite.True(user.HasPermission("users", "update"))
	suite.True(user.HasPermission("users", "delete"))
	suite.False(user.HasPermission("media", "create"))
	suite.False(user.HasPermission("users", "admin"))
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
	suite.Len(userPermissions, 3)

	// Check all permissions are present
	hasPermission := func(perms []domain.Permission, resource, action string) bool {
		for _, p := range perms {
			if p.Resource == resource && p.Action == action {
				return true
			}
		}
		return false
	}

	suite.True(hasPermission(userPermissions, "users", "read"))
	suite.True(hasPermission(userPermissions, "media", "read"))
	suite.True(hasPermission(userPermissions, "media", "create"))
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
	suite.True(expiredSession.IsExpired())
	suite.False(validSession.IsExpired())
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
	suite.Equal("en", user.Preferences.Language)
	suite.Equal("dark", user.Preferences.Theme)
	suite.Equal("UTC", user.Preferences.TimeZone)
	suite.True(user.Preferences.AutoPlayNext)
	suite.Equal("en", user.Preferences.SubtitleLanguage)
	suite.Equal("auto", user.Preferences.PreferredQuality)
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
	suite.True(role.HasPermission("users", "create"))
	suite.True(role.HasPermission("users", "read"))
	suite.False(role.HasPermission("users", "delete"))

	// Wildcard should match any action
	suite.True(role.HasPermission("media", "create"))
	suite.True(role.HasPermission("media", "read"))
	suite.True(role.HasPermission("media", "delete"))
	suite.True(role.HasPermission("media", "anything"))
}

func (suite *UserDomainTestSuite) TestPermission_Matches() {
	// Test exact match
	perm := domain.Permission{Resource: "users", Action: "read"}
	suite.True(perm.Matches("users", "read"))
	suite.False(perm.Matches("users", "write"))
	suite.False(perm.Matches("media", "read"))

	// Test wildcard action
	wildcard := domain.Permission{Resource: "media", Action: "*"}
	suite.True(wildcard.Matches("media", "read"))
	suite.True(wildcard.Matches("media", "write"))
	suite.True(wildcard.Matches("media", "delete"))
	suite.False(wildcard.Matches("users", "read"))

	// Test wildcard resource
	wildcardResource := domain.Permission{Resource: "*", Action: "read"}
	suite.True(wildcardResource.Matches("users", "read"))
	suite.True(wildcardResource.Matches("media", "read"))
	suite.False(wildcardResource.Matches("users", "write"))

	// Test double wildcard
	allAccess := domain.Permission{Resource: "*", Action: "*"}
	suite.True(allAccess.Matches("users", "read"))
	suite.True(allAccess.Matches("media", "write"))
	suite.True(allAccess.Matches("anything", "anything"))
}

func TestUserDomainTestSuite(t *testing.T) {
	suite.Run(t, new(UserDomainTestSuite))
}
