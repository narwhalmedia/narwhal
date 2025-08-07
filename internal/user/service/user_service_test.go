package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/service"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/utils"
	"github.com/narwhalmedia/narwhal/test/mocks"
	"github.com/narwhalmedia/narwhal/test/testutil"
)

type UserServiceTestSuite struct {
	suite.Suite

	ctx         context.Context
	mockRepo    *mocks.MockRepository
	userService *service.UserService
	cache       *utils.InMemoryCache
	eventBus    *events.LocalEventBus
}

func (suite *UserServiceTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockRepo = new(mocks.MockRepository)
	suite.cache = utils.NewInMemoryCache()
	suite.eventBus = events.NewLocalEventBus(logger.NewNoopLogger())

	suite.userService = service.NewUserService(
		suite.mockRepo,
		suite.eventBus,
		suite.cache,
		logger.NewNoopLogger(),
	)
}

func (suite *UserServiceTestSuite) TearDownTest() {
	suite.mockRepo.AssertExpectations(suite.T())
}

func (suite *UserServiceTestSuite) TestCreateUser_Success() {
	// Arrange
	role := testutil.CreateTestRole(domain.RoleUser, "Default user role")

	suite.mockRepo.On("UserExists", suite.ctx, "testuser", "test@example.com").Return(false, nil)
	suite.mockRepo.On("GetRoleByName", suite.ctx, domain.RoleUser).Return(role, nil)
	suite.mockRepo.On("CreateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Act
	user, err := suite.userService.CreateUser(suite.ctx, "testuser", "test@example.com", "password123", "Test User")

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(user)
	suite.Equal("testuser", user.Username)
	suite.Equal("test@example.com", user.Email)
	suite.Equal("Test User", user.DisplayName)
	suite.True(user.IsActive)
	suite.False(user.IsVerified)
	suite.Len(user.Roles, 1)
	suite.Equal(domain.RoleUser, user.Roles[0].Name)
}

func (suite *UserServiceTestSuite) TestCreateUser_UserExists() {
	// Arrange
	suite.mockRepo.On("UserExists", suite.ctx, "testuser", "test@example.com").Return(true, nil)

	// Act
	user, err := suite.userService.CreateUser(suite.ctx, "testuser", "test@example.com", "password123", "Test User")

	// Assert
	suite.Require().Error(err)
	suite.Nil(user)
	suite.True(errors.IsConflict(err))
}

func (suite *UserServiceTestSuite) TestCreateUser_MissingFields() {
	// Act
	user, err := suite.userService.CreateUser(suite.ctx, "", "", "", "")

	// Assert
	suite.Require().Error(err)
	suite.Nil(user)
	suite.True(errors.IsBadRequest(err))
}

func (suite *UserServiceTestSuite) TestGetUser_Success() {
	// Arrange
	expectedUser := testutil.CreateTestUser("testuser", "test@example.com")

	suite.mockRepo.On("GetUser", suite.ctx, expectedUser.ID).Return(expectedUser, nil)

	// Act
	user, err := suite.userService.GetUser(suite.ctx, expectedUser.ID)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(user)
	suite.Equal(expectedUser.ID, user.ID)
	suite.Equal(expectedUser.Username, user.Username)
}

func (suite *UserServiceTestSuite) TestGetUser_Cached() {
	// Arrange
	expectedUser := testutil.CreateTestUser("testuser", "test@example.com")

	// First call - from repository
	suite.mockRepo.On("GetUser", suite.ctx, expectedUser.ID).Return(expectedUser, nil).Once()

	// Act - First call
	user1, err1 := suite.userService.GetUser(suite.ctx, expectedUser.ID)

	// Act - Second call (should use cache)
	user2, err2 := suite.userService.GetUser(suite.ctx, expectedUser.ID)

	// Assert
	suite.Require().NoError(err1)
	suite.Require().NoError(err2)
	suite.Equal(user1.ID, user2.ID)
	// Verify repo was only called once
	suite.mockRepo.AssertNumberOfCalls(suite.T(), "GetUser", 1)
}

func (suite *UserServiceTestSuite) TestUpdateUser_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	updates := map[string]interface{}{
		"display_name": "Updated Name",
		"avatar":       "https://example.com/avatar.jpg",
	}

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Act
	updatedUser, err := suite.userService.UpdateUser(suite.ctx, user.ID, updates)

	// Assert
	suite.Require().NoError(err)
	suite.NotNil(updatedUser)
	suite.Equal("Updated Name", updatedUser.DisplayName)
	suite.Equal("https://example.com/avatar.jpg", updatedUser.Avatar)
}

func (suite *UserServiceTestSuite) TestChangePassword_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("oldpassword")

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	suite.mockRepo.On("DeleteUserSessions", suite.ctx, user.ID).Return(nil)

	// Act
	err := suite.userService.ChangePassword(suite.ctx, user.ID, "oldpassword", "newpassword")

	// Assert
	suite.Require().NoError(err)
}

func (suite *UserServiceTestSuite) TestChangePassword_WrongOldPassword() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("oldpassword")

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)

	// Act
	err := suite.userService.ChangePassword(suite.ctx, user.ID, "wrongpassword", "newpassword")

	// Assert
	suite.Require().Error(err)
	suite.True(errors.IsUnauthorized(err))
}

func (suite *UserServiceTestSuite) TestDeleteUser_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("DeleteUserSessions", suite.ctx, user.ID).Return(nil)
	suite.mockRepo.On("DeleteUser", suite.ctx, user.ID).Return(nil)

	// Act
	err := suite.userService.DeleteUser(suite.ctx, user.ID)

	// Assert
	suite.Require().NoError(err)
}

func (suite *UserServiceTestSuite) TestListUsers_Success() {
	// Arrange
	users := []*domain.User{
		testutil.CreateTestUser("user1", "user1@example.com"),
		testutil.CreateTestUser("user2", "user2@example.com"),
	}

	suite.mockRepo.On("ListUsers", suite.ctx, 50, 0).Return(users, nil)
	suite.mockRepo.On("CountUsers", suite.ctx).Return(int64(2), nil)

	// Act
	result, total, err := suite.userService.ListUsers(suite.ctx, 0, 0)

	// Assert
	suite.Require().NoError(err)
	suite.Len(result, 2)
	suite.Equal(int64(2), total)
}

func (suite *UserServiceTestSuite) TestAssignRole_Success() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.Roles = []domain.Role{*testutil.CreateTestRole(domain.RoleUser, "User role")}

	adminRole := testutil.CreateTestRole(domain.RoleAdmin, "Admin role")

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("GetRoleByName", suite.ctx, domain.RoleAdmin).Return(adminRole, nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Act
	err := suite.userService.AssignRole(suite.ctx, user.ID, domain.RoleAdmin)

	// Assert
	suite.Require().NoError(err)
}

func (suite *UserServiceTestSuite) TestAssignRole_AlreadyHasRole() {
	// Arrange
	adminRole := testutil.CreateTestRole(domain.RoleAdmin, "Admin role")
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.Roles = []domain.Role{*adminRole}

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)

	// Act
	err := suite.userService.AssignRole(suite.ctx, user.ID, domain.RoleAdmin)

	// Assert
	suite.Require().NoError(err)
	// Should not call GetRoleByName or UpdateUser since user already has the role
	suite.mockRepo.AssertNotCalled(suite.T(), "GetRoleByName")
	suite.mockRepo.AssertNotCalled(suite.T(), "UpdateUser")
}

func (suite *UserServiceTestSuite) TestRemoveRole_Success() {
	// Arrange
	userRole := testutil.CreateTestRole(domain.RoleUser, "User role")
	adminRole := testutil.CreateTestRole(domain.RoleAdmin, "Admin role")

	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.Roles = []domain.Role{*userRole, *adminRole}

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)

	// Act
	err := suite.userService.RemoveRole(suite.ctx, user.ID, domain.RoleAdmin)

	// Assert
	suite.Require().NoError(err)
}

func (suite *UserServiceTestSuite) TestSetUserActive_Deactivate() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.IsActive = true

	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	suite.mockRepo.On("UpdateUser", suite.ctx, mock.AnythingOfType("*domain.User")).Return(nil)
	suite.mockRepo.On("DeleteUserSessions", suite.ctx, user.ID).Return(nil)

	// Act
	err := suite.userService.SetUserActive(suite.ctx, user.ID, false)

	// Assert
	suite.Require().NoError(err)
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}
