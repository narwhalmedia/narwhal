package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/repository"
	"github.com/narwhalmedia/narwhal/test/testutil"
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
	suite.Require().NoError(err)
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
	suite.Require().NoError(err)
	suite.NotEqual(uuid.Nil, user.ID)

	// Verify user was created
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	suite.Require().NoError(err)
	suite.Equal(user.Username, retrieved.Username)
	suite.Equal(user.Email, retrieved.Email)
}

func (suite *GormRepositoryTestSuite) TestCreateUser_DuplicateUsername() {
	// Arrange
	user1 := testutil.CreateTestUser("testuser", "test1@example.com")
	user2 := testutil.CreateTestUser("testuser", "test2@example.com") // Same username

	// Act
	err1 := suite.repo.CreateUser(suite.ctx, user1)
	err2 := suite.repo.CreateUser(suite.ctx, user2)

	// Assert
	suite.Require().NoError(err1)
	suite.Require().Error(err2)
}

func (suite *GormRepositoryTestSuite) TestGetUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(retrieved)
	suite.Equal(user.ID, retrieved.ID)
	suite.Equal(user.Username, retrieved.Username)
}

func (suite *GormRepositoryTestSuite) TestGetUser_NotFound() {
	// Act
	retrieved, err := suite.repo.GetUser(suite.ctx, uuid.New())

	// Assert
	suite.Require().Error(err)
	suite.Nil(retrieved)
	suite.ErrorIs(err, gorm.ErrRecordNotFound)
}

func (suite *GormRepositoryTestSuite) TestGetUserByUsername() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	retrieved, err := suite.repo.GetUserByUsername(suite.ctx, "testuser")

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(retrieved)
	suite.Equal(user.Username, retrieved.Username)
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
	suite.Require().NoError(err)

	// Verify update
	retrieved, _ := suite.repo.GetUser(suite.ctx, user.ID)
	suite.Equal("Updated Name", retrieved.DisplayName)
	suite.Equal("https://example.com/avatar.jpg", retrieved.Avatar)
}

func (suite *GormRepositoryTestSuite) TestDeleteUser() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act
	err := suite.repo.DeleteUser(suite.ctx, user.ID)

	// Assert
	suite.Require().NoError(err)

	// Verify deletion
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	suite.Require().Error(err)
	suite.Nil(retrieved)
}

func (suite *GormRepositoryTestSuite) TestListUsers() {
	// Arrange
	for i := range 5 {
		user := testutil.CreateTestUser(
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("user%d@example.com", i),
		)
		suite.repo.CreateUser(suite.ctx, user)
	}

	// Act
	users, err := suite.repo.ListUsers(suite.ctx, 3, 0)

	// Assert
	suite.Require().NoError(err)
	suite.Len(users, 3)

	// Test pagination
	users2, err := suite.repo.ListUsers(suite.ctx, 3, 3)
	suite.Require().NoError(err)
	suite.Len(users2, 2)
}

func (suite *GormRepositoryTestSuite) TestUserExists() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Act & Assert
	exists, err := suite.repo.UserExists(suite.ctx, "testuser", "other@example.com")
	suite.Require().NoError(err)
	suite.True(exists)

	exists, err = suite.repo.UserExists(suite.ctx, "other", "test@example.com")
	suite.Require().NoError(err)
	suite.True(exists)

	exists, err = suite.repo.UserExists(suite.ctx, "nonexistent", "nonexistent@example.com")
	suite.Require().NoError(err)
	suite.False(exists)
}

func (suite *GormRepositoryTestSuite) TestRoleOperations() {
	// Create role
	role := testutil.CreateTestRole(domain.RoleAdmin, "Administrator")
	err := suite.repo.CreateRole(suite.ctx, role)
	suite.Require().NoError(err)

	// Get role
	retrieved, err := suite.repo.GetRole(suite.ctx, role.ID)
	suite.Require().NoError(err)
	suite.Equal(role.Name, retrieved.Name)

	// Get role by name
	retrieved, err = suite.repo.GetRoleByName(suite.ctx, domain.RoleAdmin)
	suite.Require().NoError(err)
	suite.Equal(role.Name, retrieved.Name)

	// Update role
	role.Description = "Updated description"
	err = suite.repo.UpdateRole(suite.ctx, role)
	suite.Require().NoError(err)

	// List roles
	roles, err := suite.repo.ListRoles(suite.ctx)
	suite.Require().NoError(err)
	suite.Len(roles, 1)

	// Delete role
	err = suite.repo.DeleteRole(suite.ctx, role.ID)
	suite.Require().NoError(err)
}

func (suite *GormRepositoryTestSuite) TestPermissionOperations() {
	// Create permission
	perm := testutil.CreateTestPermission(domain.ResourceLibrary, domain.ActionRead, "Read libraries")
	err := suite.repo.CreatePermission(suite.ctx, perm)
	suite.Require().NoError(err)

	// Get permission
	retrieved, err := suite.repo.GetPermission(suite.ctx, perm.ID)
	suite.Require().NoError(err)
	suite.Equal(perm.Resource, retrieved.Resource)

	// Get permission by resource/action
	retrieved, err = suite.repo.GetPermissionByResourceAction(suite.ctx, domain.ResourceLibrary, domain.ActionRead)
	suite.Require().NoError(err)
	suite.Equal(perm.Resource, retrieved.Resource)
	suite.Equal(perm.Action, retrieved.Action)

	// List permissions
	perms, err := suite.repo.ListPermissions(suite.ctx)
	suite.Require().NoError(err)
	suite.Len(perms, 1)
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
	suite.Require().NoError(err)

	// Verify permissions were assigned
	retrieved, _ := suite.repo.GetRole(suite.ctx, role.ID)
	suite.Len(retrieved.Permissions, 2)

	// Remove one permission
	err = suite.repo.RemovePermissionsFromRole(suite.ctx, role.ID, []uuid.UUID{perm1.ID})
	suite.Require().NoError(err)

	// Verify permission was removed
	retrieved, _ = suite.repo.GetRole(suite.ctx, role.ID)
	suite.Len(retrieved.Permissions, 1)
	suite.Equal(perm2.ID, retrieved.Permissions[0].ID)
}

func (suite *GormRepositoryTestSuite) TestSessionOperations() {
	// Create user first
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	// Create session
	session := testutil.CreateTestSession(user.ID)
	err := suite.repo.CreateSession(suite.ctx, session)
	suite.Require().NoError(err)

	// Get session
	retrieved, err := suite.repo.GetSession(suite.ctx, session.ID)
	suite.Require().NoError(err)
	suite.Equal(session.RefreshToken, retrieved.RefreshToken)

	// Get session by refresh token
	retrieved, err = suite.repo.GetSessionByRefreshToken(suite.ctx, session.RefreshToken)
	suite.Require().NoError(err)
	suite.Equal(session.ID, retrieved.ID)

	// List user sessions
	sessions, err := suite.repo.ListUserSessions(suite.ctx, user.ID)
	suite.Require().NoError(err)
	suite.Len(sessions, 1)

	// Delete session
	err = suite.repo.DeleteSession(suite.ctx, session.ID)
	suite.Require().NoError(err)

	// Verify deletion
	_, err = suite.repo.GetSession(suite.ctx, session.ID)
	suite.Require().Error(err)
}

func (suite *GormRepositoryTestSuite) TestDeleteUserSessions() {
	// Create user and multiple sessions
	user := testutil.CreateTestUser("testuser", "test@example.com")
	suite.repo.CreateUser(suite.ctx, user)

	for range 3 {
		session := testutil.CreateTestSession(user.ID)
		suite.repo.CreateSession(suite.ctx, session)
	}

	// Verify sessions exist
	sessions, _ := suite.repo.ListUserSessions(suite.ctx, user.ID)
	suite.Len(sessions, 3)

	// Delete all user sessions
	err := suite.repo.DeleteUserSessions(suite.ctx, user.ID)
	suite.Require().NoError(err)

	// Verify all sessions deleted
	sessions, _ = suite.repo.ListUserSessions(suite.ctx, user.ID)
	suite.Empty(sessions)
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
	suite.Require().NoError(err)

	// Verify only valid session remains
	sessions, _ := suite.repo.ListUserSessions(suite.ctx, user.ID)
	suite.Len(sessions, 1)
	suite.Equal(validSession.ID, sessions[0].ID)
}

func (suite *GormRepositoryTestSuite) TestTransaction() {
	// Start transaction
	tx, err := suite.repo.BeginTx(suite.ctx)
	suite.Require().NoError(err)

	// Create user in transaction
	user := testutil.CreateTestUser("txuser", "tx@example.com")
	err = tx.CreateUser(suite.ctx, user)
	suite.Require().NoError(err)

	// Rollback transaction
	err = tx.Rollback()
	suite.Require().NoError(err)

	// Verify user was not created
	retrieved, err := suite.repo.GetUser(suite.ctx, user.ID)
	suite.Require().Error(err)
	suite.Nil(retrieved)

	// Test commit
	tx2, _ := suite.repo.BeginTx(suite.ctx)
	user2 := testutil.CreateTestUser("txuser2", "tx2@example.com")
	tx2.CreateUser(suite.ctx, user2)
	err = tx2.Commit()
	suite.Require().NoError(err)

	// Verify user was created
	retrieved, err = suite.repo.GetUser(suite.ctx, user2.ID)
	suite.Require().NoError(err)
	suite.NotNil(retrieved)
}

func TestGormRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(GormRepositoryTestSuite))
}
