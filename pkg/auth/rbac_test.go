package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/models"
)

func TestRBAC_CheckPermission(t *testing.T) {
	rbac := auth.NewRBAC()

	tests := []struct {
		name       string
		role       string
		resource   string
		action     string
		shouldPass bool
	}{
		// Admin tests
		{"Admin can read library", models.RoleAdmin, models.ResourceLibrary, models.ActionRead, true},
		{"Admin can write library", models.RoleAdmin, models.ResourceLibrary, models.ActionWrite, true},
		{"Admin can delete library", models.RoleAdmin, models.ResourceLibrary, models.ActionDelete, true},
		{"Admin can admin library", models.RoleAdmin, models.ResourceLibrary, models.ActionAdmin, true},
		{"Admin can admin system", models.RoleAdmin, models.ResourceSystem, models.ActionAdmin, true},

		// User tests
		{"User can read library", models.RoleUser, models.ResourceLibrary, models.ActionRead, true},
		{"User cannot write library", models.RoleUser, models.ResourceLibrary, models.ActionWrite, false},
		{"User cannot delete library", models.RoleUser, models.ResourceLibrary, models.ActionDelete, false},
		{"User can read media", models.RoleUser, models.ResourceMedia, models.ActionRead, true},
		{"User can write media", models.RoleUser, models.ResourceMedia, models.ActionWrite, true},
		{"User cannot admin system", models.RoleUser, models.ResourceSystem, models.ActionAdmin, false},

		// Guest tests
		{"Guest can read library", models.RoleGuest, models.ResourceLibrary, models.ActionRead, true},
		{"Guest cannot write library", models.RoleGuest, models.ResourceLibrary, models.ActionWrite, false},
		{"Guest can read media", models.RoleGuest, models.ResourceMedia, models.ActionRead, true},
		{"Guest cannot write media", models.RoleGuest, models.ResourceMedia, models.ActionWrite, false},
		{"Guest cannot access user", models.RoleGuest, models.ResourceUser, models.ActionRead, false},

		// Unknown role
		{"Unknown role cannot access", "unknown", models.ResourceLibrary, models.ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rbac.CheckPermission(tt.role, tt.resource, tt.action)
			assert.Equal(t, tt.shouldPass, result)
		})
	}
}

func TestRBAC_CheckPermissions(t *testing.T) {
	rbac := auth.NewRBAC()

	tests := []struct {
		name       string
		roles      []string
		resource   string
		action     string
		shouldPass bool
	}{
		{
			"Multiple roles - admin passes",
			[]string{models.RoleUser, models.RoleAdmin},
			models.ResourceSystem,
			models.ActionAdmin,
			true,
		},
		{
			"Multiple roles - user can read",
			[]string{models.RoleGuest, models.RoleUser},
			models.ResourceMedia,
			models.ActionWrite,
			true,
		},
		{"Multiple roles - guest cannot", []string{models.RoleGuest}, models.ResourceMedia, models.ActionWrite, false},
		{"Empty roles", []string{}, models.ResourceLibrary, models.ActionRead, false},
		{"Unknown roles", []string{"unknown1", "unknown2"}, models.ResourceLibrary, models.ActionRead, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rbac.CheckPermissions(tt.roles, tt.resource, tt.action)
			assert.Equal(t, tt.shouldPass, result)
		})
	}
}

func TestRBAC_GetRolePermissions(t *testing.T) {
	rbac := auth.NewRBAC()

	// Test admin role
	adminPerms := rbac.GetRolePermissions(models.RoleAdmin)
	assert.NotNil(t, adminPerms)
	assert.NotEmpty(t, adminPerms)
	assert.Contains(t, adminPerms[models.ResourceSystem], models.ActionAdmin)

	// Test user role
	userPerms := rbac.GetRolePermissions(models.RoleUser)
	assert.NotNil(t, userPerms)
	assert.Contains(t, userPerms[models.ResourceLibrary], models.ActionRead)
	assert.NotContains(t, userPerms[models.ResourceSystem], models.ActionAdmin)

	// Test unknown role
	unknownPerms := rbac.GetRolePermissions("unknown")
	assert.Nil(t, unknownPerms)
}

func TestRBAC_AddRemovePermission(t *testing.T) {
	rbac := auth.NewRBAC()

	// Add permission to existing role
	rbac.AddPermission(models.RoleGuest, models.ResourceMedia, models.ActionWrite)
	assert.True(t, rbac.CheckPermission(models.RoleGuest, models.ResourceMedia, models.ActionWrite))

	// Add permission to new role
	rbac.AddPermission("custom", models.ResourceLibrary, models.ActionRead)
	assert.True(t, rbac.CheckPermission("custom", models.ResourceLibrary, models.ActionRead))

	// Remove permission
	rbac.RemovePermission(models.RoleGuest, models.ResourceMedia, models.ActionWrite)
	assert.False(t, rbac.CheckPermission(models.RoleGuest, models.ResourceMedia, models.ActionWrite))

	// Remove non-existent permission (should not panic)
	rbac.RemovePermission("nonexistent", models.ResourceLibrary, models.ActionRead)
}

func TestPolicyEnforcer_Enforce(t *testing.T) {
	rbac := auth.NewRBAC()
	enforcer := auth.NewPolicyEnforcer(rbac)

	tests := []struct {
		name      string
		roles     []string
		resource  string
		action    string
		shouldErr bool
	}{
		{"Admin allowed", []string{models.RoleAdmin}, models.ResourceSystem, models.ActionAdmin, false},
		{"User denied system admin", []string{models.RoleUser}, models.ResourceSystem, models.ActionAdmin, true},
		{"User allowed media read", []string{models.RoleUser}, models.ResourceMedia, models.ActionRead, false},
		{"Empty roles denied", []string{}, models.ResourceLibrary, models.ActionRead, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := enforcer.Enforce(tt.roles, tt.resource, tt.action)
			if tt.shouldErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "permission denied")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPolicyEnforcer_EnforceAny(t *testing.T) {
	rbac := auth.NewRBAC()
	enforcer := auth.NewPolicyEnforcer(rbac)

	permissions := []models.Permission{
		{Resource: models.ResourceSystem, Action: models.ActionAdmin},
		{Resource: models.ResourceMedia, Action: models.ActionRead},
	}

	// User has one of the permissions (media:read)
	err := enforcer.EnforceAny([]string{models.RoleUser}, permissions...)
	require.NoError(t, err)

	// Guest has one of the permissions (media:read)
	err = enforcer.EnforceAny([]string{models.RoleGuest}, permissions...)
	require.NoError(t, err)

	// Test with permissions guest doesn't have
	permissions2 := []models.Permission{
		{Resource: models.ResourceSystem, Action: models.ActionAdmin},
		{Resource: models.ResourceUser, Action: models.ActionWrite},
	}
	err = enforcer.EnforceAny([]string{models.RoleGuest}, permissions2...)
	require.Error(t, err)
}

func TestPolicyEnforcer_EnforceAll(t *testing.T) {
	rbac := auth.NewRBAC()
	enforcer := auth.NewPolicyEnforcer(rbac)

	permissions := []models.Permission{
		{Resource: models.ResourceLibrary, Action: models.ActionRead},
		{Resource: models.ResourceMedia, Action: models.ActionRead},
	}

	// Admin has all permissions
	err := enforcer.EnforceAll([]string{models.RoleAdmin}, permissions...)
	require.NoError(t, err)

	// User has all permissions
	err = enforcer.EnforceAll([]string{models.RoleUser}, permissions...)
	require.NoError(t, err)

	// Guest doesn't have all permissions (missing streaming:read)
	permissions2 := []models.Permission{
		{Resource: models.ResourceMedia, Action: models.ActionRead},
		{Resource: models.ResourceStreaming, Action: models.ActionRead},
		{Resource: models.ResourceUser, Action: models.ActionRead},
	}
	err = enforcer.EnforceAll([]string{models.RoleGuest}, permissions2...)
	require.Error(t, err)
}

func TestPolicyEnforcer_CheckOwnership(t *testing.T) {
	rbac := auth.NewRBAC()
	enforcer := auth.NewPolicyEnforcer(rbac)

	ownership := auth.ResourceOwnership{
		UserIDField: "user_id",
		AllowAdmin:  true,
	}

	// Owner can access
	err := enforcer.CheckOwnership("user123", "user123", []string{models.RoleUser}, ownership)
	require.NoError(t, err)

	// Non-owner cannot access
	err = enforcer.CheckOwnership("user123", "user456", []string{models.RoleUser}, ownership)
	require.Error(t, err)

	// Admin can access when AllowAdmin is true
	err = enforcer.CheckOwnership("user123", "user456", []string{models.RoleAdmin}, ownership)
	require.NoError(t, err)

	// Admin cannot access when AllowAdmin is false
	ownership.AllowAdmin = false
	err = enforcer.CheckOwnership("user123", "user456", []string{models.RoleAdmin}, ownership)
	require.Error(t, err)
}
