package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/narwhalmedia/narwhal/internal/user/domain"
)

// RBAC provides role-based access control functionality
type RBAC struct {
	permissions map[string]map[string][]string // role -> resource -> actions
}

// NewRBAC creates a new RBAC instance with default permissions
func NewRBAC() *RBAC {
	rbac := &RBAC{
		permissions: make(map[string]map[string][]string),
	}
	rbac.initializeDefaultPermissions()
	return rbac
}

// initializeDefaultPermissions sets up the default permission structure
func (r *RBAC) initializeDefaultPermissions() {
	// Admin role - full access to everything
	r.permissions[domain.RoleAdmin] = map[string][]string{
		domain.ResourceLibrary:     {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceMedia:       {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceUser:        {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceTranscoding: {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceStreaming:   {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceAcquisition: {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceAnalytics:   {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
		domain.ResourceSystem:      {domain.ActionRead, domain.ActionWrite, domain.ActionDelete, domain.ActionAdmin},
	}

	// User role - standard user permissions
	r.permissions[domain.RoleUser] = map[string][]string{
		domain.ResourceLibrary:     {domain.ActionRead},
		domain.ResourceMedia:       {domain.ActionRead, domain.ActionWrite},
		domain.ResourceUser:        {domain.ActionRead, domain.ActionWrite}, // Can read/update own profile
		domain.ResourceTranscoding: {domain.ActionRead},
		domain.ResourceStreaming:   {domain.ActionRead},
		domain.ResourceAcquisition: {domain.ActionRead},
		domain.ResourceAnalytics:   {domain.ActionRead}, // Can view own analytics
		domain.ResourceSystem:      {},                  // No system access
	}

	// Guest role - minimal permissions
	r.permissions[domain.RoleGuest] = map[string][]string{
		domain.ResourceLibrary:     {domain.ActionRead},
		domain.ResourceMedia:       {domain.ActionRead},
		domain.ResourceUser:        {},
		domain.ResourceTranscoding: {},
		domain.ResourceStreaming:   {domain.ActionRead},
		domain.ResourceAcquisition: {},
		domain.ResourceAnalytics:   {},
		domain.ResourceSystem:      {},
	}
}

// CheckPermission checks if a role has permission to perform an action on a resource
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

// CheckPermissions checks if any of the roles have permission to perform an action on a resource
func (r *RBAC) CheckPermissions(roles []string, resource, action string) bool {
	for _, role := range roles {
		if r.CheckPermission(role, resource, action) {
			return true
		}
	}
	return false
}

// GetRolePermissions returns all permissions for a role
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

// AddPermission adds a permission to a role
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

// RemovePermission removes a permission from a role
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

// Middleware provides context-aware permission checking
type Middleware interface {
	RequirePermission(resource, action string) func(context.Context) error
	RequireAnyPermission(permissions ...Permission) func(context.Context) error
	RequireAllPermissions(permissions ...Permission) func(context.Context) error
}

// Permission represents a resource-action pair
type Permission struct {
	Resource string
	Action   string
}

// PolicyEnforcer provides policy-based access control
type PolicyEnforcer struct {
	rbac *RBAC
}

// NewPolicyEnforcer creates a new policy enforcer
func NewPolicyEnforcer(rbac *RBAC) *PolicyEnforcer {
	return &PolicyEnforcer{rbac: rbac}
}

// Enforce checks if the given roles satisfy the permission requirement
func (p *PolicyEnforcer) Enforce(roles []string, resource, action string) error {
	if !p.rbac.CheckPermissions(roles, resource, action) {
		return fmt.Errorf("permission denied: %s:%s", resource, action)
	}
	return nil
}

// EnforceAny checks if the given roles satisfy any of the permission requirements
func (p *PolicyEnforcer) EnforceAny(roles []string, permissions ...Permission) error {
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

// EnforceAll checks if the given roles satisfy all permission requirements
func (p *PolicyEnforcer) EnforceAll(roles []string, permissions ...Permission) error {
	for _, perm := range permissions {
		if !p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return fmt.Errorf("permission denied: %s:%s", perm.Resource, perm.Action)
		}
	}
	return nil
}

// ResourceOwnership defines ownership rules for resources
type ResourceOwnership struct {
	UserIDField string // Field name containing the owner's user ID
	AllowAdmin  bool   // Whether admins can bypass ownership checks
}

// CheckOwnership verifies if a user owns a resource
func (p *PolicyEnforcer) CheckOwnership(userID string, resourceUserID string, roles []string, ownership ResourceOwnership) error {
	// Check if user is the owner
	if userID == resourceUserID {
		return nil
	}

	// Check if admin access is allowed and user has admin role
	if ownership.AllowAdmin {
		for _, role := range roles {
			if role == domain.RoleAdmin {
				return nil
			}
		}
	}

	return fmt.Errorf("permission denied: not the resource owner")
}

// DefaultPermissions returns the default permission set for initial setup
func DefaultPermissions() []domain.Permission {
	return []domain.Permission{
		// Library permissions
		{Resource: domain.ResourceLibrary, Action: domain.ActionRead, Description: "View libraries"},
		{Resource: domain.ResourceLibrary, Action: domain.ActionWrite, Description: "Create/update libraries"},
		{Resource: domain.ResourceLibrary, Action: domain.ActionDelete, Description: "Delete libraries"},
		{Resource: domain.ResourceLibrary, Action: domain.ActionAdmin, Description: "Manage library settings"},

		// Media permissions
		{Resource: domain.ResourceMedia, Action: domain.ActionRead, Description: "View media"},
		{Resource: domain.ResourceMedia, Action: domain.ActionWrite, Description: "Add/update media"},
		{Resource: domain.ResourceMedia, Action: domain.ActionDelete, Description: "Delete media"},
		{Resource: domain.ResourceMedia, Action: domain.ActionAdmin, Description: "Manage media settings"},

		// User permissions
		{Resource: domain.ResourceUser, Action: domain.ActionRead, Description: "View users"},
		{Resource: domain.ResourceUser, Action: domain.ActionWrite, Description: "Create/update users"},
		{Resource: domain.ResourceUser, Action: domain.ActionDelete, Description: "Delete users"},
		{Resource: domain.ResourceUser, Action: domain.ActionAdmin, Description: "Manage user settings"},

		// Transcoding permissions
		{Resource: domain.ResourceTranscoding, Action: domain.ActionRead, Description: "View transcoding jobs"},
		{Resource: domain.ResourceTranscoding, Action: domain.ActionWrite, Description: "Create transcoding jobs"},
		{Resource: domain.ResourceTranscoding, Action: domain.ActionDelete, Description: "Cancel transcoding jobs"},
		{Resource: domain.ResourceTranscoding, Action: domain.ActionAdmin, Description: "Manage transcoding settings"},

		// Streaming permissions
		{Resource: domain.ResourceStreaming, Action: domain.ActionRead, Description: "Stream media"},
		{Resource: domain.ResourceStreaming, Action: domain.ActionWrite, Description: "Manage streams"},
		{Resource: domain.ResourceStreaming, Action: domain.ActionDelete, Description: "Terminate streams"},
		{Resource: domain.ResourceStreaming, Action: domain.ActionAdmin, Description: "Manage streaming settings"},

		// Acquisition permissions
		{Resource: domain.ResourceAcquisition, Action: domain.ActionRead, Description: "View downloads"},
		{Resource: domain.ResourceAcquisition, Action: domain.ActionWrite, Description: "Add downloads"},
		{Resource: domain.ResourceAcquisition, Action: domain.ActionDelete, Description: "Remove downloads"},
		{Resource: domain.ResourceAcquisition, Action: domain.ActionAdmin, Description: "Manage acquisition settings"},

		// Analytics permissions
		{Resource: domain.ResourceAnalytics, Action: domain.ActionRead, Description: "View analytics"},
		{Resource: domain.ResourceAnalytics, Action: domain.ActionWrite, Description: "Create reports"},
		{Resource: domain.ResourceAnalytics, Action: domain.ActionDelete, Description: "Delete reports"},
		{Resource: domain.ResourceAnalytics, Action: domain.ActionAdmin, Description: "Manage analytics settings"},

		// System permissions
		{Resource: domain.ResourceSystem, Action: domain.ActionRead, Description: "View system status"},
		{Resource: domain.ResourceSystem, Action: domain.ActionWrite, Description: "Modify system settings"},
		{Resource: domain.ResourceSystem, Action: domain.ActionDelete, Description: "Delete system data"},
		{Resource: domain.ResourceSystem, Action: domain.ActionAdmin, Description: "Full system administration"},
	}
}

// DefaultRoles returns the default roles for initial setup
func DefaultRoles() []domain.Role {
	return []domain.Role{
		{
			Name:        domain.RoleAdmin,
			Description: "System administrator with full access",
		},
		{
			Name:        domain.RoleUser,
			Description: "Standard user with content access",
		},
		{
			Name:        domain.RoleGuest,
			Description: "Guest user with limited read-only access",
		},
	}
}
