package auth_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/logger"
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
	suite.True(suite.rbac.CheckPermission(domain.RoleAdmin, domain.ResourceLibrary, domain.ActionRead))
	suite.True(suite.rbac.CheckPermission(domain.RoleAdmin, domain.ResourceLibrary, domain.ActionWrite))
	suite.True(suite.rbac.CheckPermission(domain.RoleAdmin, domain.ResourceLibrary, domain.ActionDelete))
	suite.True(suite.rbac.CheckPermission(domain.RoleAdmin, domain.ResourceLibrary, domain.ActionAdmin))

	// Test user role
	suite.True(suite.rbac.CheckPermission(domain.RoleUser, domain.ResourceLibrary, domain.ActionRead))
	suite.False(suite.rbac.CheckPermission(domain.RoleUser, domain.ResourceLibrary, domain.ActionDelete))
	suite.False(suite.rbac.CheckPermission(domain.RoleUser, domain.ResourceLibrary, domain.ActionAdmin))

	// Test guest role
	suite.True(suite.rbac.CheckPermission(domain.RoleGuest, domain.ResourceLibrary, domain.ActionRead))
	suite.False(suite.rbac.CheckPermission(domain.RoleGuest, domain.ResourceLibrary, domain.ActionWrite))
	suite.False(suite.rbac.CheckPermission(domain.RoleGuest, domain.ResourceUser, domain.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestCheckPermissions() {
	roles := []string{domain.RoleUser, domain.RoleGuest}

	// Should return true if any role has permission
	suite.True(suite.rbac.CheckPermissions(roles, domain.ResourceMedia, domain.ActionWrite))

	// Should return false if no role has permission
	suite.False(suite.rbac.CheckPermissions(roles, domain.ResourceLibrary, domain.ActionDelete))
}

func (suite *CasbinRBACTestSuite) TestGetRolePermissions() {
	// Test admin permissions
	adminPerms := suite.rbac.GetRolePermissions(domain.RoleAdmin)
	suite.NotEmpty(adminPerms)
	suite.Contains(adminPerms[domain.ResourceLibrary], domain.ActionRead)
	suite.Contains(adminPerms[domain.ResourceLibrary], domain.ActionWrite)
	suite.Contains(adminPerms[domain.ResourceLibrary], domain.ActionDelete)
	suite.Contains(adminPerms[domain.ResourceLibrary], domain.ActionAdmin)

	// Test user permissions
	userPerms := suite.rbac.GetRolePermissions(domain.RoleUser)
	suite.NotEmpty(userPerms)
	suite.Contains(userPerms[domain.ResourceLibrary], domain.ActionRead)
	suite.NotContains(userPerms[domain.ResourceLibrary], domain.ActionDelete)

	// Test non-existent role
	emptyPerms := suite.rbac.GetRolePermissions("non-existent")
	suite.Empty(emptyPerms)
}

func (suite *CasbinRBACTestSuite) TestAddRemovePermission() {
	testRole := "test-role"

	// Add permission
	suite.rbac.AddPermission(testRole, domain.ResourceMedia, domain.ActionRead)
	suite.True(suite.rbac.CheckPermission(testRole, domain.ResourceMedia, domain.ActionRead))

	// Add another permission
	suite.rbac.AddPermission(testRole, domain.ResourceMedia, domain.ActionWrite)
	suite.True(suite.rbac.CheckPermission(testRole, domain.ResourceMedia, domain.ActionWrite))

	// Remove permission
	suite.rbac.RemovePermission(testRole, domain.ResourceMedia, domain.ActionRead)
	suite.False(suite.rbac.CheckPermission(testRole, domain.ResourceMedia, domain.ActionRead))
	suite.True(suite.rbac.CheckPermission(testRole, domain.ResourceMedia, domain.ActionWrite))
}

func (suite *CasbinRBACTestSuite) TestUserRoleAssignment() {
	userID := "user123"

	// Assign role to user
	err := suite.rbac.AssignRole(userID, domain.RoleUser)
	suite.Require().NoError(err)

	// Check user permissions through role
	suite.True(suite.rbac.CheckUserPermission(userID, domain.ResourceLibrary, domain.ActionRead))
	suite.True(suite.rbac.CheckUserPermission(userID, domain.ResourceMedia, domain.ActionWrite))
	suite.False(suite.rbac.CheckUserPermission(userID, domain.ResourceLibrary, domain.ActionDelete))

	// Get user roles
	roles := suite.rbac.GetUserRoles(userID)
	suite.Contains(roles, domain.RoleUser)

	// Assign admin role
	err = suite.rbac.AssignRole(userID, domain.RoleAdmin)
	suite.Require().NoError(err)

	// Now user should have admin permissions
	suite.True(suite.rbac.CheckUserPermission(userID, domain.ResourceLibrary, domain.ActionDelete))

	// Remove user role
	err = suite.rbac.RemoveRole(userID, domain.RoleUser)
	suite.Require().NoError(err)

	// User should still have admin permissions
	suite.True(suite.rbac.CheckUserPermission(userID, domain.ResourceLibrary, domain.ActionDelete))

	// Remove admin role
	err = suite.rbac.RemoveRole(userID, domain.RoleAdmin)
	suite.Require().NoError(err)

	// User should have no permissions now
	suite.False(suite.rbac.CheckUserPermission(userID, domain.ResourceLibrary, domain.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestAddRemoveRole() {
	newRole := "moderator"
	permissions := []auth.Permission{
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
		{Resource: domain.ResourceMedia, Action: domain.ActionWrite},
		{Resource: domain.ResourceMedia, Action: domain.ActionDelete},
		{Resource: domain.ResourceUser, Action: domain.ActionRead},
	}

	// Add new role
	err := suite.rbac.AddRole(newRole, permissions)
	suite.Require().NoError(err)

	// Check role permissions
	suite.True(suite.rbac.CheckPermission(newRole, domain.ResourceMedia, domain.ActionRead))
	suite.True(suite.rbac.CheckPermission(newRole, domain.ResourceMedia, domain.ActionWrite))
	suite.True(suite.rbac.CheckPermission(newRole, domain.ResourceMedia, domain.ActionDelete))
	suite.True(suite.rbac.CheckPermission(newRole, domain.ResourceUser, domain.ActionRead))
	suite.False(suite.rbac.CheckPermission(newRole, domain.ResourceUser, domain.ActionWrite))

	// Get all roles
	roles := suite.rbac.GetAllRoles()
	suite.Contains(roles, newRole)
	suite.Contains(roles, domain.RoleAdmin)
	suite.Contains(roles, domain.RoleUser)
	suite.Contains(roles, domain.RoleGuest)

	// Remove role
	err = suite.rbac.DeleteRole(newRole)
	suite.Require().NoError(err)

	// Check role is removed
	suite.False(suite.rbac.CheckPermission(newRole, domain.ResourceMedia, domain.ActionRead))
}

func (suite *CasbinRBACTestSuite) TestPolicyEnforcer() {
	enforcer := auth.NewCasbinPolicyEnforcer(suite.rbac)

	// Test Enforce
	err := enforcer.Enforce([]string{domain.RoleAdmin}, domain.ResourceLibrary, domain.ActionDelete)
	suite.Require().NoError(err)

	err = enforcer.Enforce([]string{domain.RoleUser}, domain.ResourceLibrary, domain.ActionDelete)
	suite.Require().Error(err)

	// Test EnforceUser
	userID := "user456"
	suite.rbac.AssignRole(userID, domain.RoleUser)

	err = enforcer.EnforceUser(userID, domain.ResourceMedia, domain.ActionRead)
	suite.Require().NoError(err)

	err = enforcer.EnforceUser(userID, domain.ResourceLibrary, domain.ActionDelete)
	suite.Require().Error(err)

	// Test EnforceAny
	permissions := []auth.Permission{
		{Resource: domain.ResourceLibrary, Action: domain.ActionDelete},
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
	}

	err = enforcer.EnforceAny([]string{domain.RoleUser}, permissions...)
	suite.Require().NoError(err) // User can read media

	// Test EnforceAll
	permissions = []auth.Permission{
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
		{Resource: domain.ResourceMedia, Action: domain.ActionWrite},
	}

	err = enforcer.EnforceAll([]string{domain.RoleUser}, permissions...)
	suite.Require().NoError(err) // User has both permissions

	permissions = append(permissions, auth.Permission{Resource: domain.ResourceMedia, Action: domain.ActionDelete})
	err = enforcer.EnforceAll([]string{domain.RoleUser}, permissions...)
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
	err := enforcer.CheckOwnership(userID, resourceUserID, []string{domain.RoleUser}, ownership)
	suite.Require().NoError(err)

	// Non-owner without admin should not have access
	err = enforcer.CheckOwnership("other-user", resourceUserID, []string{domain.RoleUser}, ownership)
	suite.Require().Error(err)

	// Admin should have access when AllowAdmin is true
	err = enforcer.CheckOwnership(adminUserID, resourceUserID, []string{domain.RoleAdmin}, ownership)
	suite.Require().NoError(err)

	// Admin should not have access when AllowAdmin is false
	ownership.AllowAdmin = false
	err = enforcer.CheckOwnership(adminUserID, resourceUserID, []string{domain.RoleAdmin}, ownership)
	suite.Require().Error(err)
}

func TestCasbinRBACTestSuite(t *testing.T) {
	suite.Run(t, new(CasbinRBACTestSuite))
}
