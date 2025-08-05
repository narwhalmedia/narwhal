package domain

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string    `gorm:"uniqueIndex;not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	DisplayName  string
	Avatar       string
	Roles        []Role     `gorm:"many2many:user_roles;"`
	Sessions     []Session  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Preferences  UserPreferences `gorm:"embedded;embeddedPrefix:pref_"`
	IsActive     bool       `gorm:"default:true"`
	IsVerified   bool       `gorm:"default:false"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Role represents a user role
type Role struct {
	ID          uuid.UUID    `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string       `gorm:"uniqueIndex;not null"`
	Description string
	Permissions []Permission `gorm:"many2many:role_permissions;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Permission represents a system permission
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Resource    string    `gorm:"not null"` // e.g., "library", "media", "user"
	Action      string    `gorm:"not null"` // e.g., "read", "write", "delete"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Session represents an active user session
type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID `gorm:"not null;index"`
	RefreshToken string    `gorm:"uniqueIndex;not null"`
	DeviceInfo   string
	IPAddress    string
	UserAgent    string
	ExpiresAt    time.Time `gorm:"not null;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// UserPreferences represents user-specific preferences
type UserPreferences struct {
	Language          string `gorm:"default:'en'"`
	Theme             string `gorm:"default:'dark'"`
	TimeZone          string `gorm:"default:'UTC'"`
	AutoPlayNext      bool   `gorm:"default:true"`
	SubtitleLanguage  string `gorm:"default:'en'"`
	PreferredQuality  string `gorm:"default:'auto'"`
	EnableNotifications bool `gorm:"default:true"`
}

// SetPassword hashes and sets the user's password
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the user's password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HasRole checks if the user has a specific role
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(resource, action string) bool {
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			if perm.Matches(resource, action) {
				return true
			}
		}
	}
	return false
}

// GetPermissions returns all unique permissions for the user
func (u *User) GetPermissions() []Permission {
	permMap := make(map[string]Permission)
	
	for _, role := range u.Roles {
		for _, perm := range role.Permissions {
			key := perm.Resource + ":" + perm.Action
			if _, exists := permMap[key]; !exists {
				permMap[key] = perm
			}
		}
	}
	
	perms := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		perms = append(perms, perm)
	}
	
	return perms
}

// Token types
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	TokenType   string   `json:"token_type"`
	SessionID   string   `json:"session_id,omitempty"`
}

// AuthTokens represents a pair of access and refresh tokens
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Default roles
const (
	RoleAdmin     = "admin"
	RoleUser      = "user"
	RoleGuest     = "guest"
	RoleModerator = "moderator"
)

// Resource types for permissions
const (
	ResourceLibrary      = "library"
	ResourceMedia        = "media"
	ResourceUser         = "user"
	ResourceTranscoding  = "transcoding"
	ResourceStreaming    = "streaming"
	ResourceAcquisition  = "acquisition"
	ResourceAnalytics    = "analytics"
	ResourceSystem       = "system"
)

// Action types for permissions
const (
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// HasPermission checks if the role has a specific permission
func (r *Role) HasPermission(resource, action string) bool {
	for _, perm := range r.Permissions {
		if perm.Matches(resource, action) {
			return true
		}
	}
	return false
}

// Matches checks if the permission matches the given resource and action
func (p *Permission) Matches(resource, action string) bool {
	// Support wildcards
	resourceMatch := p.Resource == "*" || p.Resource == resource
	actionMatch := p.Action == "*" || p.Action == action
	return resourceMatch && actionMatch
}