package auth

import (
	"fmt"
	"strings"
	"sync"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/narwhalmedia/narwhal/internal/user/domain"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// CasbinRBAC provides Casbin-based role-based access control
type CasbinRBAC struct {
	enforcer *casbin.Enforcer
	logger   interfaces.Logger
	mu       sync.RWMutex
}

// NewCasbinRBAC creates a new Casbin-based RBAC instance
func NewCasbinRBAC(modelPath, policyPath string, logger interfaces.Logger) (*CasbinRBAC, error) {
	// Load model from file
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create enforcer with file adapter
	enforcer, err := casbin.NewEnforcer(m, policyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Enable auto-save
	enforcer.EnableAutoSave(true)

	return &CasbinRBAC{
		enforcer: enforcer,
		logger:   logger,
	}, nil
}

// NewCasbinRBACFromString creates a new Casbin-based RBAC instance from string configs
func NewCasbinRBACFromString(modelText, policyText string, logger interfaces.Logger) (*CasbinRBAC, error) {
	// Load model from string
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create enforcer
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Load policies from CSV string if provided
	if policyText != "" {
		// Create a string adapter
		lines := strings.Split(policyText, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			
			// Parse CSV line
			parts := strings.Split(line, ",")
			if len(parts) < 4 {
				continue
			}
			
			// Clean up parts
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			
			// Add policy
			if parts[0] == "p" {
				enforcer.AddPolicy(parts[1], parts[2], parts[3])
			} else if parts[0] == "g" && len(parts) >= 3 {
				enforcer.AddGroupingPolicy(parts[1], parts[2])
			}
		}
	}

	// Enable auto-save
	enforcer.EnableAutoSave(true)

	return &CasbinRBAC{
		enforcer: enforcer,
		logger:   logger,
	}, nil
}

// CheckPermission checks if a role has permission to perform an action on a resource
func (r *CasbinRBAC) CheckPermission(role, resource, action string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allowed, err := r.enforcer.Enforce(role, resource, action)
	if err != nil {
		r.logger.Error("Failed to check permission", 
			interfaces.Error(err),
			interfaces.String("role", role),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
		return false
	}

	return allowed
}

// CheckPermissions checks if any of the roles have permission to perform an action on a resource
func (r *CasbinRBAC) CheckPermissions(roles []string, resource, action string) bool {
	for _, role := range roles {
		if r.CheckPermission(role, resource, action) {
			return true
		}
	}
	return false
}

// GetRolePermissions returns all permissions for a role
func (r *CasbinRBAC) GetRolePermissions(role string) map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	permissions := make(map[string][]string)
	
	// Get all policies for the role
	policies, _ := r.enforcer.GetFilteredPolicy(0, role)
	
	for _, policy := range policies {
		if len(policy) >= 3 {
			resource := policy[1]
			action := policy[2]
			
			if _, exists := permissions[resource]; !exists {
				permissions[resource] = []string{}
			}
			
			// Check if action already exists
			found := false
			for _, a := range permissions[resource] {
				if a == action {
					found = true
					break
				}
			}
			
			if !found {
				permissions[resource] = append(permissions[resource], action)
			}
		}
	}
	
	return permissions
}

// AddPermission adds a permission to a role
func (r *CasbinRBAC) AddPermission(role, resource, action string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	added, err := r.enforcer.AddPolicy(role, resource, action)
	if err != nil {
		r.logger.Error("Failed to add permission", 
			interfaces.Error(err),
			interfaces.String("role", role),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
		return
	}

	if added {
		r.logger.Info("Permission added",
			interfaces.String("role", role),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
	}
}

// RemovePermission removes a permission from a role
func (r *CasbinRBAC) RemovePermission(role, resource, action string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed, err := r.enforcer.RemovePolicy(role, resource, action)
	if err != nil {
		r.logger.Error("Failed to remove permission", 
			interfaces.Error(err),
			interfaces.String("role", role),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
		return
	}

	if removed {
		r.logger.Info("Permission removed",
			interfaces.String("role", role),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
	}
}

// AssignRole assigns a role to a user
func (r *CasbinRBAC) AssignRole(userID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	added, err := r.enforcer.AddGroupingPolicy(userID, role)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	if added {
		r.logger.Info("Role assigned",
			interfaces.String("userID", userID),
			interfaces.String("role", role))
	}

	return nil
}

// RemoveRole removes a role from a user
func (r *CasbinRBAC) RemoveRole(userID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed, err := r.enforcer.RemoveGroupingPolicy(userID, role)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	if removed {
		r.logger.Info("Role removed",
			interfaces.String("userID", userID),
			interfaces.String("role", role))
	}

	return nil
}

// GetUserRoles returns all roles assigned to a user
func (r *CasbinRBAC) GetUserRoles(userID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles, err := r.enforcer.GetRolesForUser(userID)
	if err != nil {
		r.logger.Error("Failed to get user roles", 
			interfaces.Error(err),
			interfaces.String("userID", userID))
		return nil
	}

	return roles
}

// CheckUserPermission checks if a user has permission to perform an action on a resource
func (r *CasbinRBAC) CheckUserPermission(userID, resource, action string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allowed, err := r.enforcer.Enforce(userID, resource, action)
	if err != nil {
		r.logger.Error("Failed to check user permission", 
			interfaces.Error(err),
			interfaces.String("userID", userID),
			interfaces.String("resource", resource),
			interfaces.String("action", action))
		return false
	}

	return allowed
}

// AddRole creates a new role with permissions
func (r *CasbinRBAC) AddRole(role string, permissions []Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add each permission for the role
	for _, perm := range permissions {
		if _, err := r.enforcer.AddPolicy(role, perm.Resource, perm.Action); err != nil {
			return fmt.Errorf("failed to add permission %s:%s to role %s: %w", 
				perm.Resource, perm.Action, role, err)
		}
	}

	r.logger.Info("Role created",
		interfaces.String("role", role),
		interfaces.Int("permissions", len(permissions)))

	return nil
}

// DeleteRole removes a role and all its permissions
func (r *CasbinRBAC) DeleteRole(role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove all policies for the role
	removed, err := r.enforcer.RemoveFilteredPolicy(0, role)
	if err != nil {
		return fmt.Errorf("failed to remove role policies: %w", err)
	}

	// Remove all grouping policies (user assignments) for the role
	removedGroups, err := r.enforcer.RemoveFilteredGroupingPolicy(1, role)
	if err != nil {
		return fmt.Errorf("failed to remove role assignments: %w", err)
	}

	r.logger.Info("Role removed",
		interfaces.String("role", role),
		interfaces.Bool("policies_removed", removed),
		interfaces.Bool("assignments_removed", removedGroups))

	return nil
}

// GetAllRoles returns all defined roles
func (r *CasbinRBAC) GetAllRoles() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all unique subjects from policies
	roleMap := make(map[string]bool)
	policies, _ := r.enforcer.GetPolicy()
	
	for _, policy := range policies {
		if len(policy) > 0 {
			roleMap[policy[0]] = true
		}
	}

	// Convert map to slice
	roles := make([]string, 0, len(roleMap))
	for role := range roleMap {
		roles = append(roles, role)
	}

	return roles
}

// SavePolicy saves the current policy to file
func (r *CasbinRBAC) SavePolicy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.enforcer.SavePolicy()
}

// LoadPolicy reloads the policy from file
func (r *CasbinRBAC) LoadPolicy() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.enforcer.LoadPolicy()
}


// CasbinPolicyEnforcer provides policy-based access control using Casbin
type CasbinPolicyEnforcer struct {
	rbac *CasbinRBAC
}

// NewCasbinPolicyEnforcer creates a new Casbin policy enforcer
func NewCasbinPolicyEnforcer(rbac *CasbinRBAC) *CasbinPolicyEnforcer {
	return &CasbinPolicyEnforcer{rbac: rbac}
}

// Enforce checks if the given roles satisfy the permission requirement
func (p *CasbinPolicyEnforcer) Enforce(roles []string, resource, action string) error {
	if !p.rbac.CheckPermissions(roles, resource, action) {
		return fmt.Errorf("permission denied: %s:%s", resource, action)
	}
	return nil
}

// EnforceUser checks if the given user satisfies the permission requirement
func (p *CasbinPolicyEnforcer) EnforceUser(userID, resource, action string) error {
	if !p.rbac.CheckUserPermission(userID, resource, action) {
		return fmt.Errorf("permission denied: %s:%s for user %s", resource, action, userID)
	}
	return nil
}

// EnforceAny checks if the given roles satisfy any of the permission requirements
func (p *CasbinPolicyEnforcer) EnforceAny(roles []string, permissions ...Permission) error {
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
func (p *CasbinPolicyEnforcer) EnforceAll(roles []string, permissions ...Permission) error {
	for _, perm := range permissions {
		if !p.rbac.CheckPermissions(roles, perm.Resource, perm.Action) {
			return fmt.Errorf("permission denied: %s:%s", perm.Resource, perm.Action)
		}
	}
	return nil
}

// CheckOwnership verifies if a user owns a resource
func (p *CasbinPolicyEnforcer) CheckOwnership(userID string, resourceUserID string, roles []string, ownership ResourceOwnership) error {
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

// InitializeDefaultPolicies sets up the default policies for the system
func InitializeDefaultPolicies(rbac *CasbinRBAC) error {
	// Define default roles and their permissions
	defaultPolicies := map[string][]Permission{
		domain.RoleAdmin: {
			// Admin has full access to everything
			{Resource: domain.ResourceLibrary, Action: domain.ActionRead},
			{Resource: domain.ResourceLibrary, Action: domain.ActionWrite},
			{Resource: domain.ResourceLibrary, Action: domain.ActionDelete},
			{Resource: domain.ResourceLibrary, Action: domain.ActionAdmin},
			{Resource: domain.ResourceMedia, Action: domain.ActionRead},
			{Resource: domain.ResourceMedia, Action: domain.ActionWrite},
			{Resource: domain.ResourceMedia, Action: domain.ActionDelete},
			{Resource: domain.ResourceMedia, Action: domain.ActionAdmin},
			{Resource: domain.ResourceUser, Action: domain.ActionRead},
			{Resource: domain.ResourceUser, Action: domain.ActionWrite},
			{Resource: domain.ResourceUser, Action: domain.ActionDelete},
			{Resource: domain.ResourceUser, Action: domain.ActionAdmin},
			{Resource: domain.ResourceTranscoding, Action: domain.ActionRead},
			{Resource: domain.ResourceTranscoding, Action: domain.ActionWrite},
			{Resource: domain.ResourceTranscoding, Action: domain.ActionDelete},
			{Resource: domain.ResourceTranscoding, Action: domain.ActionAdmin},
			{Resource: domain.ResourceStreaming, Action: domain.ActionRead},
			{Resource: domain.ResourceStreaming, Action: domain.ActionWrite},
			{Resource: domain.ResourceStreaming, Action: domain.ActionDelete},
			{Resource: domain.ResourceStreaming, Action: domain.ActionAdmin},
			{Resource: domain.ResourceAcquisition, Action: domain.ActionRead},
			{Resource: domain.ResourceAcquisition, Action: domain.ActionWrite},
			{Resource: domain.ResourceAcquisition, Action: domain.ActionDelete},
			{Resource: domain.ResourceAcquisition, Action: domain.ActionAdmin},
			{Resource: domain.ResourceAnalytics, Action: domain.ActionRead},
			{Resource: domain.ResourceAnalytics, Action: domain.ActionWrite},
			{Resource: domain.ResourceAnalytics, Action: domain.ActionDelete},
			{Resource: domain.ResourceAnalytics, Action: domain.ActionAdmin},
			{Resource: domain.ResourceSystem, Action: domain.ActionRead},
			{Resource: domain.ResourceSystem, Action: domain.ActionWrite},
			{Resource: domain.ResourceSystem, Action: domain.ActionDelete},
			{Resource: domain.ResourceSystem, Action: domain.ActionAdmin},
		},
		domain.RoleUser: {
			// Standard user permissions
			{Resource: domain.ResourceLibrary, Action: domain.ActionRead},
			{Resource: domain.ResourceMedia, Action: domain.ActionRead},
			{Resource: domain.ResourceMedia, Action: domain.ActionWrite},
			{Resource: domain.ResourceUser, Action: domain.ActionRead},
			{Resource: domain.ResourceUser, Action: domain.ActionWrite},
			{Resource: domain.ResourceTranscoding, Action: domain.ActionRead},
			{Resource: domain.ResourceStreaming, Action: domain.ActionRead},
			{Resource: domain.ResourceAcquisition, Action: domain.ActionRead},
			{Resource: domain.ResourceAnalytics, Action: domain.ActionRead},
		},
		domain.RoleGuest: {
			// Guest permissions
			{Resource: domain.ResourceLibrary, Action: domain.ActionRead},
			{Resource: domain.ResourceMedia, Action: domain.ActionRead},
			{Resource: domain.ResourceStreaming, Action: domain.ActionRead},
		},
	}

	// Add each role with its permissions
	for role, permissions := range defaultPolicies {
		if err := rbac.AddRole(role, permissions); err != nil {
			return fmt.Errorf("failed to initialize role %s: %w", role, err)
		}
	}

	return nil
}