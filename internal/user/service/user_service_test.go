package service_test

import (
	"context"
	"testing"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/internal/user/service"
	"github.com/narwhalmedia/narwhal/pkg/errors"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/utils"
	"github.com/narwhalmedia/narwhal/test/mocks"
	"github.com/narwhalmedia/narwhal/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), "testuser", user.Username)
	assert.Equal(suite.T(), "test@example.com", user.Email)
	assert.Equal(suite.T(), "Test User", user.DisplayName)
	assert.True(suite.T(), user.IsActive)
	assert.False(suite.T(), user.IsVerified)
	assert.Len(suite.T(), user.Roles, 1)
	assert.Equal(suite.T(), domain.RoleUser, user.Roles[0].Name)
}

func (suite *UserServiceTestSuite) TestCreateUser_UserExists() {
	// Arrange
	suite.mockRepo.On("UserExists", suite.ctx, "testuser", "test@example.com").Return(true, nil)
	
	// Act
	user, err := suite.userService.CreateUser(suite.ctx, "testuser", "test@example.com", "password123", "Test User")
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.True(suite.T(), errors.IsConflict(err))
}

func (suite *UserServiceTestSuite) TestCreateUser_MissingFields() {
	// Act
	user, err := suite.userService.CreateUser(suite.ctx, "", "", "", "")
	
	// Assert
	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), user)
	assert.True(suite.T(), errors.IsBadRequest(err))
}

func (suite *UserServiceTestSuite) TestGetUser_Success() {
	// Arrange
	expectedUser := testutil.CreateTestUser("testuser", "test@example.com")
	
	suite.mockRepo.On("GetUser", suite.ctx, expectedUser.ID).Return(expectedUser, nil)
	
	// Act
	user, err := suite.userService.GetUser(suite.ctx, expectedUser.ID)
	
	// Assert
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), user)
	assert.Equal(suite.T(), expectedUser.ID, user.ID)
	assert.Equal(suite.T(), expectedUser.Username, user.Username)
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
	assert.NoError(suite.T(), err1)
	assert.NoError(suite.T(), err2)
	assert.Equal(suite.T(), user1.ID, user2.ID)
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
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), updatedUser)
	assert.Equal(suite.T(), "Updated Name", updatedUser.DisplayName)
	assert.Equal(suite.T(), "https://example.com/avatar.jpg", updatedUser.Avatar)
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
	assert.NoError(suite.T(), err)
}

func (suite *UserServiceTestSuite) TestChangePassword_WrongOldPassword() {
	// Arrange
	user := testutil.CreateTestUser("testuser", "test@example.com")
	user.SetPassword("oldpassword")
	
	suite.mockRepo.On("GetUser", suite.ctx, user.ID).Return(user, nil)
	
	// Act
	err := suite.userService.ChangePassword(suite.ctx, user.ID, "wrongpassword", "newpassword")
	
	// Assert
	assert.Error(suite.T(), err)
	assert.True(suite.T(), errors.IsUnauthorized(err))
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
	assert.NoError(suite.T(), err)
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
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 2)
	assert.Equal(suite.T(), int64(2), total)
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
	assert.NoError(suite.T(), err)
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
	assert.NoError(suite.T(), err)
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
	assert.NoError(suite.T(), err)
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
	assert.NoError(suite.T(), err)
}

func TestUserServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserServiceTestSuite))
}