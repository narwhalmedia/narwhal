package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/auth"
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
		{"Admin can read library", domain.RoleAdmin, domain.ResourceLibrary, domain.ActionRead, true},
		{"Admin can write library", domain.RoleAdmin, domain.ResourceLibrary, domain.ActionWrite, true},
		{"Admin can delete library", domain.RoleAdmin, domain.ResourceLibrary, domain.ActionDelete, true},
		{"Admin can admin library", domain.RoleAdmin, domain.ResourceLibrary, domain.ActionAdmin, true},
		{"Admin can admin system", domain.RoleAdmin, domain.ResourceSystem, domain.ActionAdmin, true},

		// User tests
		{"User can read library", domain.RoleUser, domain.ResourceLibrary, domain.ActionRead, true},
		{"User cannot write library", domain.RoleUser, domain.ResourceLibrary, domain.ActionWrite, false},
		{"User cannot delete library", domain.RoleUser, domain.ResourceLibrary, domain.ActionDelete, false},
		{"User can read media", domain.RoleUser, domain.ResourceMedia, domain.ActionRead, true},
		{"User can write media", domain.RoleUser, domain.ResourceMedia, domain.ActionWrite, true},
		{"User cannot admin system", domain.RoleUser, domain.ResourceSystem, domain.ActionAdmin, false},

		// Guest tests
		{"Guest can read library", domain.RoleGuest, domain.ResourceLibrary, domain.ActionRead, true},
		{"Guest cannot write library", domain.RoleGuest, domain.ResourceLibrary, domain.ActionWrite, false},
		{"Guest can read media", domain.RoleGuest, domain.ResourceMedia, domain.ActionRead, true},
		{"Guest cannot write media", domain.RoleGuest, domain.ResourceMedia, domain.ActionWrite, false},
		{"Guest cannot access user", domain.RoleGuest, domain.ResourceUser, domain.ActionRead, false},

		// Unknown role
		{"Unknown role cannot access", "unknown", domain.ResourceLibrary, domain.ActionRead, false},
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
			[]string{domain.RoleUser, domain.RoleAdmin},
			domain.ResourceSystem,
			domain.ActionAdmin,
			true,
		},
		{
			"Multiple roles - user can read",
			[]string{domain.RoleGuest, domain.RoleUser},
			domain.ResourceMedia,
			domain.ActionWrite,
			true,
		},
		{"Multiple roles - guest cannot", []string{domain.RoleGuest}, domain.ResourceMedia, domain.ActionWrite, false},
		{"Empty roles", []string{}, domain.ResourceLibrary, domain.ActionRead, false},
		{"Unknown roles", []string{"unknown1", "unknown2"}, domain.ResourceLibrary, domain.ActionRead, false},
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
	adminPerms := rbac.GetRolePermissions(domain.RoleAdmin)
	assert.NotNil(t, adminPerms)
	assert.NotEmpty(t, adminPerms)
	assert.Contains(t, adminPerms[domain.ResourceSystem], domain.ActionAdmin)

	// Test user role
	userPerms := rbac.GetRolePermissions(domain.RoleUser)
	assert.NotNil(t, userPerms)
	assert.Contains(t, userPerms[domain.ResourceLibrary], domain.ActionRead)
	assert.NotContains(t, userPerms[domain.ResourceSystem], domain.ActionAdmin)

	// Test unknown role
	unknownPerms := rbac.GetRolePermissions("unknown")
	assert.Nil(t, unknownPerms)
}

func TestRBAC_AddRemovePermission(t *testing.T) {
	rbac := auth.NewRBAC()

	// Add permission to existing role
	rbac.AddPermission(domain.RoleGuest, domain.ResourceMedia, domain.ActionWrite)
	assert.True(t, rbac.CheckPermission(domain.RoleGuest, domain.ResourceMedia, domain.ActionWrite))

	// Add permission to new role
	rbac.AddPermission("custom", domain.ResourceLibrary, domain.ActionRead)
	assert.True(t, rbac.CheckPermission("custom", domain.ResourceLibrary, domain.ActionRead))

	// Remove permission
	rbac.RemovePermission(domain.RoleGuest, domain.ResourceMedia, domain.ActionWrite)
	assert.False(t, rbac.CheckPermission(domain.RoleGuest, domain.ResourceMedia, domain.ActionWrite))

	// Remove non-existent permission (should not panic)
	rbac.RemovePermission("nonexistent", domain.ResourceLibrary, domain.ActionRead)
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
		{"Admin allowed", []string{domain.RoleAdmin}, domain.ResourceSystem, domain.ActionAdmin, false},
		{"User denied system admin", []string{domain.RoleUser}, domain.ResourceSystem, domain.ActionAdmin, true},
		{"User allowed media read", []string{domain.RoleUser}, domain.ResourceMedia, domain.ActionRead, false},
		{"Empty roles denied", []string{}, domain.ResourceLibrary, domain.ActionRead, true},
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

	permissions := []auth.Permission{
		{Resource: domain.ResourceSystem, Action: domain.ActionAdmin},
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
	}

	// User has one of the permissions (media:read)
	err := enforcer.EnforceAny([]string{domain.RoleUser}, permissions...)
	require.NoError(t, err)

	// Guest has one of the permissions (media:read)
	err = enforcer.EnforceAny([]string{domain.RoleGuest}, permissions...)
	require.NoError(t, err)

	// Test with permissions guest doesn't have
	permissions2 := []auth.Permission{
		{Resource: domain.ResourceSystem, Action: domain.ActionAdmin},
		{Resource: domain.ResourceUser, Action: domain.ActionWrite},
	}
	err = enforcer.EnforceAny([]string{domain.RoleGuest}, permissions2...)
	require.Error(t, err)
}

func TestPolicyEnforcer_EnforceAll(t *testing.T) {
	rbac := auth.NewRBAC()
	enforcer := auth.NewPolicyEnforcer(rbac)

	permissions := []auth.Permission{
		{Resource: domain.ResourceLibrary, Action: domain.ActionRead},
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
	}

	// Admin has all permissions
	err := enforcer.EnforceAll([]string{domain.RoleAdmin}, permissions...)
	require.NoError(t, err)

	// User has all permissions
	err = enforcer.EnforceAll([]string{domain.RoleUser}, permissions...)
	require.NoError(t, err)

	// Guest doesn't have all permissions (missing streaming:read)
	permissions2 := []auth.Permission{
		{Resource: domain.ResourceMedia, Action: domain.ActionRead},
		{Resource: domain.ResourceStreaming, Action: domain.ActionRead},
		{Resource: domain.ResourceUser, Action: domain.ActionRead},
	}
	err = enforcer.EnforceAll([]string{domain.RoleGuest}, permissions2...)
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
	err := enforcer.CheckOwnership("user123", "user123", []string{domain.RoleUser}, ownership)
	require.NoError(t, err)

	// Non-owner cannot access
	err = enforcer.CheckOwnership("user123", "user456", []string{domain.RoleUser}, ownership)
	require.Error(t, err)

	// Admin can access when AllowAdmin is true
	err = enforcer.CheckOwnership("user123", "user456", []string{domain.RoleAdmin}, ownership)
	require.NoError(t, err)

	// Admin cannot access when AllowAdmin is false
	ownership.AllowAdmin = false
	err = enforcer.CheckOwnership("user123", "user456", []string{domain.RoleAdmin}, ownership)
	require.Error(t, err)
}
