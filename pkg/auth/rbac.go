package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/narwhalmedia/narwhal/pkg/models"
)

// RBAC provides role-based access control functionality.
type RBAC struct {
	permissions map[string]map[string][]string // role -> resource -> actions
}

// NewRBAC creates a new RBAC instance with default permissions.
func NewRBAC() *RBAC {
	rbac := &RBAC{
		permissions: make(map[string]map[string][]string),
	}
	rbac.initializeDefaultPermissions()
	return rbac
}

// initializeDefaultPermissions sets up the default permission structure.
func (r *RBAC) initializeDefaultPermissions() {
	// Admin role - full access to everything
	r.permissions[models.RoleAdmin] = map[string][]string{
		models.ResourceLibrary:     {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceMedia:       {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceUser:        {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceTranscoding: {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceStreaming:   {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceAcquisition: {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceAnalytics:   {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
		models.ResourceSystem:      {models.ActionRead, models.ActionWrite, models.ActionDelete, models.ActionAdmin},
	}

	// User role - standard user permissions
	r.permissions[models.RoleUser] = map[string][]string{
		models.ResourceLibrary:     {models.ActionRead},
		models.ResourceMedia:       {models.ActionRead, models.ActionWrite},
		models.ResourceUser:        {models.ActionRead, models.ActionWrite}, // Can read/update own profile
		models.ResourceTranscoding: {models.ActionRead},
		models.ResourceStreaming:   {models.ActionRead},
		models.ResourceAcquisition: {models.ActionRead},
		models.ResourceAnalytics:   {models.ActionRead}, // Can view own analytics
		models.ResourceSystem:      {},                  // No system access
	}

	// Guest role - minimal permissions
	r.permissions[models.RoleGuest] = map[string][]string{
		models.ResourceLibrary:     {models.ActionRead},
		models.ResourceMedia:       {models.ActionRead},
		models.ResourceUser:        {},
		models.ResourceTranscoding: {},
		models.ResourceStreaming:   {models.ActionRead},
		models.ResourceAcquisition: {},
		models.ResourceAnalytics:   {},
		models.ResourceSystem:      {},
	}
}

// CheckPermission checks if a role has permission to perform an action on a resource.
func (r *RBAC) CheckPermission(role, resource, action string) bool {
	if resourcePerms, ok := r.permissions[role]; ok {
		if actions, ok := resourcePerms[resource]; ok {
			for _, a := range actions {
				if a == action {
					return true
				}
			}
		}
	}
	return false
}

// CheckPermissions checks if any of the roles have permission to perform an action on a resource.
func (r *RBAC) CheckPermissions(roles []string, resource, action string) bool {
	for _, role := range roles {
		if r.CheckPermission(role, resource, action) {
			return true
		}
	}
	return false
}

// GetRolePermissions returns all permissions for a role.
func (r *RBAC) GetRolePermissions(role string) map[string][]string {
	if perms, ok := r.permissions[role]; ok {
		// Return a copy to prevent modification
		result := make(map[string][]string)
		for resource, actions := range perms {
			result[resource] = append([]string{}, actions...)
		}
		return result
	}
	return nil
}

// AddPermission adds a permission to a role.
func (r *RBAC) AddPermission(role, resource, action string) {
	if _, ok := r.permissions[role]; !ok {
		r.permissions[role] = make(map[string][]string)
	}

	if _, ok := r.permissions[role][resource]; !ok {
		r.permissions[role][resource] = []string{}
	}

	// Check if action already exists
	for _, a := range r.permissions[role][resource] {
		if a == action {
			return
		}
	}

	r.permissions[role][resource] = append(r.permissions[role][resource], action)
}

// RemovePermission removes a permission from a role.
func (r *RBAC) RemovePermission(role, resource, action string) {
	if resourcePerms, ok := r.permissions[role]; ok {
		if actions, ok := resourcePerms[resource]; ok {
			newActions := []string{}
			for _, a := range actions {
				if a != action {
					newActions = append(newActions, a)
				}
			}
			r.permissions[role][resource] = newActions
		}
	}
}

// Middleware provides context-aware permission checking.
type Middleware interface {
	RequirePermission(resource, action string) func(context.Context) error
	RequireAnyPermission(permissions ...models.Permission) func(context.Context) error
	RequireAllPermissions(permissions ...models.Permission) func(context.Context) error
}

// PolicyEnforcer provides policy-based access control.
type PolicyEnforcer struct {
	rbac *RBAC
}

// NewPolicyEnforcer creates a new policy enforcer.
func NewPolicyEnforcer(rbac *RBAC) *PolicyEnforcer {
	return &PolicyEnforcer{rbac: rbac}
}

// Enforce checks if the given roles satisfy the permission requirement.
func (p *PolicyEnforcer) Enforce(roles []string, resource, action string) error {
	if !p.rbac.CheckPermissions(roles, resource, action) {
		return fmt.Errorf("permission denied: %s:%s", resource, action)
	}
	return nil
}

// EnforceAny checks if the given roles satisfy any of the permission requirements.
func (p *PolicyEnforcer) EnforceAny(roles []string, permissions ...models.Permission) error {
	for _, perm := range permissions {
		if p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return nil
		}
	}

	permStrs := []string{}
	for _, perm := range permissions {
		permStrs = append(permStrs, fmt.Sprintf("%s:%s", perm.Resource, perm.Action))
	}
	return fmt.Errorf("permission denied: requires any of [%s]", strings.Join(permStrs, ", "))
}

// EnforceAll checks if the given roles satisfy all permission requirements.
func (p *PolicyEnforcer) EnforceAll(roles []string, permissions ...models.Permission) error {
	for _, perm := range permissions {
		if !p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return fmt.Errorf("permission denied: %s:%s", perm.Resource, perm.Action)
		}
	}
	return nil
}

// ResourceOwnership defines ownership rules for resources.
type ResourceOwnership struct {
	UserIDField string // Field name containing the owner's user ID
	AllowAdmin  bool   // Whether admins can bypass ownership checks
}

// CheckOwnership verifies if a user owns a resource.
func (p *PolicyEnforcer) CheckOwnership(
	userID string,
	resourceUserID string,
	roles []string,
	ownership ResourceOwnership,
) error {
	// Check if user is the owner
	if userID == resourceUserID {
		return nil
	}

	// Check if admin access is allowed and user has admin role
	if ownership.AllowAdmin {
		for _, role := range roles {
			if role == models.RoleAdmin {
				return nil
			}
		}
	}

	return errors.New("permission denied: not the resource owner")
}

// DefaultPermissions returns the default permission set for initial setup.
func DefaultPermissions() []models.Permission {
	return []models.Permission{
		// Library permissions
		{Resource: models.ResourceLibrary, Action: models.ActionRead, Description: "View libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionWrite, Description: "Create/update libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionDelete, Description: "Delete libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionAdmin, Description: "Manage library settings"},

		// Media permissions
		{Resource: models.ResourceMedia, Action: models.ActionRead, Description: "View media"},
		{Resource: models.ResourceMedia, Action: models.ActionWrite, Description: "Add/update media"},
		{Resource: models.ResourceMedia, Action: models.ActionDelete, Description: "Delete media"},
		{Resource: models.ResourceMedia, Action: models.ActionAdmin, Description: "Manage media settings"},

		// User permissions
		{Resource: models.ResourceUser, Action: models.ActionRead, Description: "View users"},
		{Resource: models.ResourceUser, Action: models.ActionWrite, Description: "Create/update users"},
		{Resource: models.ResourceUser, Action: models.ActionDelete, Description: "Delete users"},
		{Resource: models.ResourceUser, Action: models.ActionAdmin, Description: "Manage user settings"},

		// Transcoding permissions
		{Resource: models.ResourceTranscoding, Action: models.ActionRead, Description: "View transcoding jobs"},
		{Resource: models.ResourceTranscoding, Action: models.ActionWrite, Description: "Create transcoding jobs"},
		{Resource: models.ResourceTranscoding, Action: models.ActionDelete, Description: "Cancel transcoding jobs"},
		{Resource: models.ResourceTranscoding, Action: models.ActionAdmin, Description: "Manage transcoding settings"},

		// Streaming permissions
		{Resource: models.ResourceStreaming, Action: models.ActionRead, Description: "Stream media"},
		{Resource: models.ResourceStreaming, Action: models.ActionWrite, Description: "Manage streams"},
		{Resource: models.ResourceStreaming, Action: models.ActionDelete, Description: "Terminate streams"},
		{Resource: models.ResourceStreaming, Action: models.ActionAdmin, Description: "Manage streaming settings"},

		// Acquisition permissions
		{Resource: models.ResourceAcquisition, Action: models.ActionRead, Description: "View downloads"},
		{Resource: models.ResourceAcquisition, Action: models.ActionWrite, Description: "Add downloads"},
		{Resource: models.ResourceAcquisition, Action: models.ActionDelete, Description: "Remove downloads"},
		{Resource: models.ResourceAcquisition, Action: models.ActionAdmin, Description: "Manage acquisition settings"},

		// Analytics permissions
		{Resource: models.ResourceAnalytics, Action: models.ActionRead, Description: "View analytics"},
		{Resource: models.ResourceAnalytics, Action: models.ActionWrite, Description: "Create reports"},
		{Resource: models.ResourceAnalytics, Action: models.ActionDelete, Description: "Delete reports"},
		{Resource: models.ResourceAnalytics, Action: models.ActionAdmin, Description: "Manage analytics settings"},

		// System permissions
		{Resource: models.ResourceSystem, Action: models.ActionRead, Description: "View system status"},
		{Resource: models.ResourceSystem, Action: models.ActionWrite, Description: "Modify system settings"},
		{Resource: models.ResourceSystem, Action: models.ActionDelete, Description: "Delete system data"},
		{Resource: models.ResourceSystem, Action: models.ActionAdmin, Description: "Full system administration"},
	}
}

// DefaultRoles returns the default roles for initial setup.
func DefaultRoles() []models.Role {
	return []models.Role{
		{
			Name:        models.RoleAdmin,
			Description: "System administrator with full access",
		},
		{
			Name:        models.RoleUser,
			Description: "Standard user with content access",
		},
		{
			Name:        models.RoleGuest,
			Description: "Guest user with limited read-only access",
		},
	}
}
