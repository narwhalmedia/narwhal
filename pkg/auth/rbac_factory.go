package auth

import (
	"fmt"
	"os"

	"github.com/narwhalmedia/narwhal/pkg/interfaces"
)

// RBACInterface defines the common interface for RBAC implementations.
type RBACInterface interface {
	CheckPermission(role, resource, action string) bool
	CheckPermissions(roles []string, resource, action string) bool
	GetRolePermissions(role string) map[string][]string
	AddPermission(role, resource, action string)
	RemovePermission(role, resource, action string)
}

// RBACType defines the type of RBAC implementation.
type RBACType string

const (
	// RBACTypeBuiltin uses the built-in custom RBAC implementation.
	RBACTypeBuiltin RBACType = "builtin"
	// RBACTypeCasbin uses Casbin for RBAC.
	RBACTypeCasbin RBACType = "casbin"
)

// RBACConfig holds configuration for RBAC.
type RBACConfig struct {
	Type             RBACType
	CasbinModelPath  string
	CasbinPolicyPath string
	Logger           interfaces.Logger
}

// NewRBACFromConfig creates an RBAC instance based on configuration.
func NewRBACFromConfig(config RBACConfig) (RBACInterface, error) {
	switch config.Type {
	case RBACTypeBuiltin:
		return NewRBAC(), nil

	case RBACTypeCasbin:
		// Check if files exist
		if config.CasbinModelPath != "" && config.CasbinPolicyPath != "" {
			if _, err := os.Stat(config.CasbinModelPath); err == nil {
				if _, err := os.Stat(config.CasbinPolicyPath); err == nil {
					return NewCasbinRBAC(config.CasbinModelPath, config.CasbinPolicyPath, config.Logger)
				}
			}
		}

		// Use embedded default configuration
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

		rbac, err := NewCasbinRBACFromString(modelText, "", config.Logger)
		if err != nil {
			return nil, err
		}

		// Initialize default policies
		if err := InitializeDefaultPolicies(rbac); err != nil {
			return nil, fmt.Errorf("failed to initialize default policies: %w", err)
		}

		return rbac, nil

	default:
		return nil, fmt.Errorf("unknown RBAC type: %s", config.Type)
	}
}

// GetRBACType returns the RBAC type from environment or default.
func GetRBACType() RBACType {
	rbacType := os.Getenv("RBAC_TYPE")
	switch rbacType {
	case "casbin":
		return RBACTypeCasbin
	case "builtin":
		return RBACTypeBuiltin
	default:
		// Default to Casbin
		return RBACTypeCasbin
	}
}
