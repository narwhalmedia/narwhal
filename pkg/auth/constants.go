package auth

import "time"

const (
	// Token constants.
	TokenKeySize        = 32
	DefaultAccessTTL    = 15 * time.Minute
	RefreshTokenKeySize = 32

	// RBAC constants.
	MinimumPolicyParts  = 3
	RequiredPolicyParts = 4
)
