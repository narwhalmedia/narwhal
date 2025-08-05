package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/repository"
	"github.com/narwhalmedia/narwhal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type GormRepositoryTestSuite struct {
	suite.Suite
	container *testutil.PostgresContainer
	repo      repository.Repository
	ctx       context.Context
}

func (suite *GormRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.container = testutil.SetupPostgresContainer(suite.T())

	// Run migrations
	err := suite.container.MigrateModels(
		&domain.User{},
		&domain.Role{},
		&domain.Permission{},
		&domain.Session{},
	)
	require.NoError(suite.T(), err)
}

func (suite *GormRepositoryTestSuite) SetupTest() {
	// Create repository
	suite.repo = repository.NewGormRepository(suite.container.DB)

	// Clean tables before each test
	suite.container.TruncateTables("sessions", "user_roles", "role_permissions", "users", "roles", "permissions")
}

func (suite *GormRepositoryTestSuite) TestCreateUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")

	// Act
	err := suite.repo.CreateUser(suite.ctx, user)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotEqual(suite.T(), uuid.Nil, user.ID)

	// Verify user was created
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), user.Username, retrieved.Username)
	assert.Equal(suite.T(), user.Email, retrieved.Email)
}

func (suite *GormRepositoryTestSuite) TestCreateUser_DuplicateUsername() {
	// Arrange
	user1 := testutil.CreateTestUser("testuser", "test1@example.com")
	user2 := testutil.CreateTestUser("testuser", "test2@example.com") // Same username

	// Act
	err1 := suite.repo.CreateUser(suite.ctx, user1)
	err2 := suite.repo.CreateUser(suite.ctx, user2)

	// Assert
	assert.NoError(suite.T(), err1)
	assert.Error(suite.T(), err2)
}

func (suite *GormRepositoryTestSuite) TestGetUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved)
	assert.Equal(suite.T(), user.ID, retrieved.ID)
	assert.Equal(suite.T(), user.Username, retrieved.Username)
}

func (suite *GormRepositoryTestSuite) TestGetUser_NotFound() {
	// Act
	retrieved, err := suite.repo.GetUser(suite.ctx, uuid.New())

	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
	assert.ErrorIs(suite.T(), err, gorm.ErrRecordNotFound)
}

func (suite *GormRepositoryTestSuite) TestGetUserByUsername() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	retrieved, err := suite.repo.GetUserByUsername(suite.ctx, "testuser")

	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved)
	assert.Equal(suite.T(), user.Username, retrieved.Username)
}

func (suite *GormRepositoryTestSuite) TestUpdateUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	user.DisplayName = "Updated Name"
	user.Avatar = "https://example.com/avatar.jpg"
	err := suite.repo.UpdateUser(suite.ctx, user)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify update
	retrieved, _ := suite.repo.GetUser(suite.ctx, user.ID)
	assert.Equal(suite.T(), "Updated Name", retrieved.DisplayName)
	assert.Equal(suite.T(), "https://example.com/avatar.jpg", retrieved.Avatar)
}

func (suite *GormRepositoryTestSuite) TestDeleteUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	err := suite.repo.DeleteUser(suite.ctx, user.ID)

	// Assert
	assert.NoError(suite.T(), err)

	// Verify deletion
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)
}

func (suite *GormRepositoryTestSuite) TestListUsers() {
	// Arrange
	for i := 0; i < 5; i++ {
		user := testutil.CreateTestUser(
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("user%d@example.com", i),
		)
		suite.repo.CreateUser(suite.ctx, user)
	}

	// Act
	users, err := suite.repo.ListUsers(suite.ctx, 3, 0)

	// Assert
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), users, 3)

	// Test pagination
	users2, err := suite.repo.ListUsers(suite.ctx, 3, 3)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), users2, 2)
}

func (suite *GormRepositoryTestSuite) TestUserExists() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act & Assert
	exists, err := suite.repo.UserExists(suite.ctx, "testuser", "other@example.com")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	exists, err = suite.repo.UserExists(suite.ctx, "other", "test@example.com")
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exists)

	exists, err = suite.repo.UserExists(suite.ctx, "nonexistent", "nonexistent@example.com")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exists)
}

func (suite *GormRepositoryTestSuite) TestRoleOperations() {
	// Create role
	role := testutil.CreateTestRole(domain.RoleAdmin, "Administrator")
	err := suite.repo.CreateRole(suite.ctx, role)
	assert.NoError(suite.T(), err)

	// Get role
	retrieved, err := suite.repo.GetRole(suite.ctx, role.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), role.Name, retrieved.Name)

	// Get role by name
	retrieved, err = suite.repo.GetRoleByName(suite.ctx, domain.RoleAdmin)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), role.Name, retrieved.Name)

	// Update role
	role.Description = "Updated description"
	err = suite.repo.UpdateRole(suite.ctx, role)
	assert.NoError(suite.T(), err)

	// List roles
	roles, err := suite.repo.ListRoles(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), roles, 1)

	// Delete role
	err = suite.repo.DeleteRole(suite.ctx, role.ID)
	assert.NoError(suite.T(), err)
}

func (suite *GormRepositoryTestSuite) TestPermissionOperations() {
	// Create permission
	perm := testutil.CreateTestPermission(domain.ResourceLibrary, domain.ActionRead, "Read libraries")
	err := suite.repo.CreatePermission(suite.ctx, perm)
	assert.NoError(suite.T(), err)

	// Get permission
	retrieved, err := suite.repo.GetPermission(suite.ctx, perm.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), perm.Resource, retrieved.Resource)

	// Get permission by resource/action
	retrieved, err = suite.repo.GetPermissionByResourceAction(suite.ctx, domain.ResourceLibrary, domain.ActionRead)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), perm.Resource, retrieved.Resource)
	assert.Equal(suite.T(), perm.Action, retrieved.Action)

	// List permissions
	perms, err := suite.repo.ListPermissions(suite.ctx)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), perms, 1)
}

func (suite *GormRepositoryTestSuite) TestRolePermissionAssociation() {
	// Create role and permissions
	role := testutil.CreateTestRole(domain.RoleUser, "User role")
	perm1 := testutil.CreateTestPermission(domain.ResourceLibrary, domain.ActionRead, "Read libraries")
	perm2 := testutil.CreateTestPermission(domain.ResourceMedia, domain.ActionRead, "Read media")

	suite.repo.CreateRole(suite.ctx, role)
	suite.repo.CreatePermission(suite.ctx, perm1)
	suite.repo.CreatePermission(suite.ctx, perm2)

	// Assign permissions to role
	err := suite.repo.AssignPermissionsToRole(suite.ctx, role.ID, []uuid.UUID{perm1.ID, perm2.ID})
	assert.NoError(suite.T(), err)

	// Verify permissions were assigned
	retrieved, _ := suite.repo.GetRole(suite.ctx, role.ID)
	assert.Len(suite.T(), retrieved.Permissions, 2)

	// Remove one permission
	err = suite.repo.RemovePermissionsFromRole(suite.ctx, role.ID, []uuid.UUID{perm1.ID})
	assert.NoError(suite.T(), err)

	// Verify permission was removed
	retrieved, _ = suite.repo.GetRole(suite.ctx, role.ID)
	assert.Len(suite.T(), retrieved.Permissions, 1)
	assert.Equal(suite.T(), perm2.ID, retrieved.Permissions[0].ID)
}

func (suite *GormRepositoryTestSuite) TestSessionOperations() {
	// Create user first
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Create session
	session := testutil.CreateTestSession(user.ID)
	err := suite.repo.CreateSession(suite.ctx, session)
	assert.NoError(suite.T(), err)

	// Get session
	retrieved, err := suite.repo.GetSession(suite.ctx, session.ID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), session.RefreshToken, retrieved.RefreshToken)

	// Get session by refresh token
	retrieved, err = suite.repo.GetSessionByRefreshToken(suite.ctx, session.RefreshToken)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), session.ID, retrieved.ID)

	// List user sessions
	sessions, err := suite.repo.ListUserSessions(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), sessions, 1)

	// Delete session
	err = suite.repo.DeleteSession(suite.ctx, session.ID)
	assert.NoError(suite.T(), err)

	// Verify deletion
	_, err = suite.repo.GetSession(suite.ctx, session.ID)
	assert.Error(suite.T(), err)
}

func (suite *GormRepositoryTestSuite) TestDeleteUserSessions() {
	// Create user and multiple sessions
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	for i := 0; i < 3; i++ {
		session := testutil.CreateTestSession(user.ID)
		suite.repo.CreateSession(suite.ctx, session)
	}

	// Verify sessions exist
	sessions, _ := suite.repo.ListUserSessions(suite.ctx, user.ID)
	assert.Len(suite.T(), sessions, 3)

	// Delete all user sessions
	err := suite.repo.DeleteUserSessions(suite.ctx, user.ID)
	assert.NoError(suite.T(), err)

	// Verify all sessions deleted
	sessions, _ = suite.repo.ListUserSessions(suite.ctx, user.ID)
	assert.Len(suite.T(), sessions, 0)
}

func (suite *GormRepositoryTestSuite) TestDeleteExpiredSessions() {
	// Create user
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Create expired and valid sessions
	expiredSession := testutil.CreateTestSession(user.ID)
	expiredSession.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired
	suite.repo.CreateSession(suite.ctx, expiredSession)

	validSession := testutil.CreateTestSession(user.ID)
	suite.repo.CreateSession(suite.ctx, validSession)

	// Delete expired sessions
	err := suite.repo.DeleteExpiredSessions(suite.ctx)
	assert.NoError(suite.T(), err)

	// Verify only valid session remains
	sessions, _ := suite.repo.ListUserSessions(suite.ctx, user.ID)
	assert.Len(suite.T(), sessions, 1)
	assert.Equal(suite.T(), validSession.ID, sessions[0].ID)
}

func (suite *GormRepositoryTestSuite) TestTransaction() {
	// Start transaction
	tx, err := suite.repo.BeginTx(suite.ctx)
	assert.NoError(suite.T(), err)

	// Create user in transaction
	user := testutil.CreateTestUser("txuser", "tx@example.com")
	err = tx.CreateUser(suite.ctx, user)
	assert.NoError(suite.T(), err)

	// Rollback transaction
	err = tx.Rollback()
	assert.NoError(suite.T(), err)

	// Verify user was not created
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), retrieved)

	// Test commit
	tx2, _ := suite.repo.BeginTx(suite.ctx)
	user2 := testutil.CreateTestUser("txuser2", "tx2@example.com")
	tx2.CreateUser(suite.ctx, user2)
	err = tx2.Commit()
	assert.NoError(suite.T(), err)

	// Verify user was created
	retrieved, err = suite.repo.GetUser(suite.ctx, user2.ID)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), retrieved)
}

func TestGormRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(GormRepositoryTestSuite))
}
