package auth_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

type CasbinRBACTestSuite struct {
	suite.Suite

	rbac *auth.CasbinRBAC
}

func (suite *CasbinRBACTestSuite) SetupTest() {
	// Create RBAC with embedded model
	modelText := `[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act`

	var err error
	suite.rbac, err = auth.NewCasbinRBACFromString(modelText, "", logger.NewNoop())
	suite.Require().NoError(err)

	// Initialize default policies
	err = auth.InitializeDefaultPolicies(suite.rbac)
	suite.Require().NoError(err)
}

func (suite *CasbinRBACTestSuite) TestCheckPermission() {
	// Test admin role
	suite.True(suite.rbac.CheckPermission(models.RoleAdmin, models.ResourceLibrary, models.ActionRead))
	suite.True(suite.rbac.CheckPermission(models.RoleAdmin, models.ResourceLibrary, models.ActionWrite))
	suite.True(suite.rbac.CheckPermission(models.RoleAdmin, models.ResourceLibrary, models.ActionDelete))
	suite.True(suite.rbac.CheckPermission(models.RoleAdmin, models.ResourceLibrary, models.ActionAdmin))

	// Test user role
	suite.True(suite.rbac.CheckPermission(models.RoleUser, models.ResourceLibrary, models.ActionRead))
	suite.False(suite.rbac.CheckPermission(models.RoleUser, models.ResourceLibrary, models.ActionDelete))
	suite.False(suite.rbac.CheckPermission(models.RoleUser, models.ResourceLibrary, models.ActionAdmin))

	// Test guest role
	suite.True(suite.rbac.CheckPermission(models.RoleGuest, models.ResourceLibrary, models.ActionRead))
	suite.False(suite.rbac.CheckPermission(models.RoleGuest, models.ResourceLibrary, models.ActionWrite))
	suite.False(suite.rbac.CheckPermission(models.RoleGuest, models.ResourceUser, models.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestCheckPermissions() {
	roles := []string{models.RoleUser, models.RoleGuest}

	// Should return true if any role has permission
	suite.True(suite.rbac.CheckPermissions(roles, models.ResourceMedia, models.ActionWrite))

	// Should return false if no role has permission
	suite.False(suite.rbac.CheckPermissions(roles, models.ResourceLibrary, models.ActionDelete))
}

func (suite *CasbinRBACTestSuite) TestGetRolePermissions() {
	// Test admin permissions
	adminPerms := suite.rbac.GetRolePermissions(models.RoleAdmin)
	suite.NotEmpty(adminPerms)
	suite.Contains(adminPerms[models.ResourceLibrary], models.ActionRead)
	suite.Contains(adminPerms[models.ResourceLibrary], models.ActionWrite)
	suite.Contains(adminPerms[models.ResourceLibrary], models.ActionDelete)
	suite.Contains(adminPerms[models.ResourceLibrary], models.ActionAdmin)

	// Test user permissions
	userPerms := suite.rbac.GetRolePermissions(models.RoleUser)
	suite.NotEmpty(userPerms)
	suite.Contains(userPerms[models.ResourceLibrary], models.ActionRead)
	suite.NotContains(userPerms[models.ResourceLibrary], models.ActionDelete)

	// Test non-existent role
	emptyPerms := suite.rbac.GetRolePermissions("non-existent")
	suite.Empty(emptyPerms)
}

func (suite *CasbinRBACTestSuite) TestAddRemovePermission() {
	testRole := "test-role"

	// Add permission
	suite.rbac.AddPermission(testRole, models.ResourceMedia, models.ActionRead)
	suite.True(suite.rbac.CheckPermission(testRole, models.ResourceMedia, models.ActionRead))

	// Add another permission
	suite.rbac.AddPermission(testRole, models.ResourceMedia, models.ActionWrite)
	suite.True(suite.rbac.CheckPermission(testRole, models.ResourceMedia, models.ActionWrite))

	// Remove permission
	suite.rbac.RemovePermission(testRole, models.ResourceMedia, models.ActionRead)
	suite.False(suite.rbac.CheckPermission(testRole, models.ResourceMedia, models.ActionRead))
	suite.True(suite.rbac.CheckPermission(testRole, models.ResourceMedia, models.ActionWrite))
}

func (suite *CasbinRBACTestSuite) TestUserRoleAssignment() {
	userID := "user123"

	// Assign role to user
	err := suite.rbac.AssignRole(userID, models.RoleUser)
	suite.Require().NoError(err)

	// Check user permissions through role
	suite.True(suite.rbac.CheckUserPermission(userID, models.ResourceLibrary, models.ActionRead))
	suite.True(suite.rbac.CheckUserPermission(userID, models.ResourceMedia, models.ActionWrite))
	suite.False(suite.rbac.CheckUserPermission(userID, models.ResourceLibrary, models.ActionDelete))

	// Get user roles
	roles := suite.rbac.GetUserRoles(userID)
	suite.Contains(roles, models.RoleUser)

	// Assign admin role
	err = suite.rbac.AssignRole(userID, models.RoleAdmin)
	suite.Require().NoError(err)

	// Now user should have admin permissions
	suite.True(suite.rbac.CheckUserPermission(userID, models.ResourceLibrary, models.ActionDelete))

	// Remove user role
	err = suite.rbac.RemoveRole(userID, models.RoleUser)
	suite.Require().NoError(err)

	// User should still have admin permissions
	suite.True(suite.rbac.CheckUserPermission(userID, models.ResourceLibrary, models.ActionDelete))

	// Remove admin role
	err = suite.rbac.RemoveRole(userID, models.RoleAdmin)
	suite.Require().NoError(err)

	// User should have no permissions now
	suite.False(suite.rbac.CheckUserPermission(userID, models.ResourceLibrary, models.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestAddRemoveRole() {
	newRole := "moderator"
	permissions := []models.Permission{
		{Resource: models.ResourceMedia, Action: models.ActionRead},
		{Resource: models.ResourceMedia, Action: models.ActionWrite},
		{Resource: models.ResourceMedia, Action: models.ActionDelete},
		{Resource: models.ResourceUser, Action: models.ActionRead},
	}

	// Add new role
	err := suite.rbac.AddRole(newRole, permissions)
	suite.Require().NoError(err)

	// Check role permissions
	suite.True(suite.rbac.CheckPermission(newRole, models.ResourceMedia, models.ActionRead))
	suite.True(suite.rbac.CheckPermission(newRole, models.ResourceMedia, models.ActionWrite))
	suite.True(suite.rbac.CheckPermission(newRole, models.ResourceMedia, models.ActionDelete))
	suite.True(suite.rbac.CheckPermission(newRole, models.ResourceUser, models.ActionRead))
	suite.False(suite.rbac.CheckPermission(newRole, models.ResourceUser, models.ActionWrite))

	// Get all roles
	roles := suite.rbac.GetAllRoles()
	suite.Contains(roles, newRole)
	suite.Contains(roles, models.RoleAdmin)
	suite.Contains(roles, models.RoleUser)
	suite.Contains(roles, models.RoleGuest)

	// Remove role
	err = suite.rbac.DeleteRole(newRole)
	suite.Require().NoError(err)

	// Check role is removed
	suite.False(suite.rbac.CheckPermission(newRole, models.ResourceMedia, models.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestPolicyEnforcer() {
	enforcer := auth.NewCasbinPolicyEnforcer(suite.rbac)

	// Test Enforce
	err := enforcer.Enforce([]string{models.RoleAdmin}, models.ResourceLibrary, models.ActionDelete)
	suite.Require().NoError(err)

	err = enforcer.Enforce([]string{models.RoleUser}, models.ResourceLibrary, models.ActionDelete)
	suite.Require().Error(err)

	// Test EnforceUser
	userID := "user456"
	suite.rbac.AssignRole(userID, models.RoleUser)

	err = enforcer.EnforceUser(userID, models.ResourceMedia, models.ActionRead)
	suite.Require().NoError(err)

	err = enforcer.EnforceUser(userID, models.ResourceLibrary, models.ActionDelete)
	suite.Require().Error(err)

	// Test EnforceAny
	permissions := []models.Permission{
		{Resource: models.ResourceLibrary, Action: models.ActionDelete},
		{Resource: models.ResourceMedia, Action: models.ActionRead},
	}

	err = enforcer.EnforceAny([]string{models.RoleUser}, permissions...)
	suite.Require().NoError(err) // User can read media

	// Test EnforceAll
	permissions = []models.Permission{
		{Resource: models.ResourceMedia, Action: models.ActionRead},
		{Resource: models.ResourceMedia, Action: models.ActionWrite},
	}

	err = enforcer.EnforceAll([]string{models.RoleUser}, permissions...)
	suite.Require().NoError(err) // User has both permissions

	permissions = append(permissions, models.Permission{Resource: models.ResourceMedia, Action: models.ActionDelete})
	err = enforcer.EnforceAll([]string{models.RoleUser}, permissions...)
	suite.Require().Error(err) // User doesn't have delete permission
}

func (suite *CasbinRBACTestSuite) TestCheckOwnership() {
	enforcer := auth.NewCasbinPolicyEnforcer(suite.rbac)

	userID := "user123"
	resourceUserID := "user123"
	adminUserID := "admin456"

	ownership := auth.ResourceOwnership{
		UserIDField: "user_id",
		AllowAdmin:  true,
	}

	// Owner should have access
	err := enforcer.CheckOwnership(userID, resourceUserID, []string{models.RoleUser}, ownership)
	suite.Require().NoError(err)

	// Non-owner without admin should not have access
	err = enforcer.CheckOwnership("other-user", resourceUserID, []string{models.RoleUser}, ownership)
	suite.Require().Error(err)

	// Admin should have access when AllowAdmin is true
	err = enforcer.CheckOwnership(adminUserID, resourceUserID, []string{models.RoleAdmin}, ownership)
	suite.Require().NoError(err)

	// Admin should not have access when AllowAdmin is false
	ownership.AllowAdmin = false
	err = enforcer.CheckOwnership(adminUserID, resourceUserID, []string{models.RoleAdmin}, ownership)
	suite.Require().Error(err)
}

func TestCasbinRBACTestSuite(t *testing.T) {
	suite.Run(t, new(CasbinRBACTestSuite))
}
