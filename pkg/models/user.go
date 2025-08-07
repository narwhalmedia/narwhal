package models

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user in the database.
type User struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string         `json:"username" gorm:"uniqueIndex;not null"`
	Email        string         `json:"email" gorm:"uniqueIndex;not null"`
	PasswordHash string         `json:"-" gorm:"not null"`
	DisplayName  string         `json:"display_name"`
	Avatar       string         `json:"avatar"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	IsVerified   bool           `json:"is_verified" gorm:"default:false"`
	LastLoginAt  *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Preferences
	PrefLanguage            string `json:"pref_language" gorm:"default:'en'"`
	PrefTheme               string `json:"pref_theme" gorm:"default:'dark'"`
	PrefTimeZone            string `json:"pref_time_zone" gorm:"default:'UTC'"`
	PrefAutoPlayNext        bool   `json:"pref_auto_play_next" gorm:"default:true"`
	PrefSubtitleLanguage    string `json:"pref_subtitle_language" gorm:"default:'en'"`
	PrefPreferredQuality    string `json:"pref_preferred_quality" gorm:"default:'auto'"`
	PrefEnableNotifications bool   `json:"pref_enable_notifications" gorm:"default:true"`

	// Relationships
	Roles    []Role    `json:"roles,omitempty" gorm:"many2many:user_roles;"`
	Sessions []Session `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// Role represents a user role in the database.
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Permissions []Permission `json:"permissions,omitempty" gorm:"many2many:role_permissions;"`
	Users       []User       `json:"-" gorm:"many2many:user_roles;"`
}

// Permission represents a system permission in the database.
type Permission struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Resource    string    `json:"resource" gorm:"not null;index:idx_permission_resource_action"` // e.g., "library", "media", "user"
	Action      string    `json:"action" gorm:"not null;index:idx_permission_resource_action"` // e.g., "read", "write", "delete"
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Relationships
	Roles []Role `json:"-" gorm:"many2many:role_permissions;"`
}

// Session represents an active user session in the database.
type Session struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID `json:"user_id" gorm:"not null;index"`
	RefreshToken string    `json:"-" gorm:"uniqueIndex;not null"`
	DeviceInfo   string    `json:"device_info,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"not null;index"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID"`
}

// TableName customizations.
func (User) TableName() string {
	return "users"
}

func (Role) TableName() string {
	return "roles"
}

func (Permission) TableName() string {
	return "permissions"
}

func (Session) TableName() string {
	return "sessions"
}

// WatchHistory represents a user's watch history for a media item.
type WatchHistory struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	MediaID     uuid.UUID  `json:"media_id" db:"media_id"`
	EpisodeID   *uuid.UUID `json:"episode_id,omitempty" db:"episode_id"`
	Position    int        `json:"position" db:"position"` // in seconds
	Duration    int        `json:"duration" db:"duration"` // total duration
	Completed   bool       `json:"completed" db:"completed"`
	LastWatched time.Time  `json:"last_watched" db:"last_watched"`
}

// UserProfile represents a user's profile within an account.
type UserProfile struct {
	ID           uuid.UUID `json:"id" db:"id"`
	UserID       uuid.UUID `json:"user_id" db:"user_id"`
	Name         string    `json:"name" db:"name"`
	Avatar       string    `json:"avatar,omitempty" db:"avatar"`
	IsKid        bool      `json:"is_kid" db:"is_kid"`
	PIN          string    `json:"-" db:"pin"` // Optional PIN for profile
	Restrictions []string  `json:"restrictions,omitempty"`
	Created      time.Time `json:"created" db:"created"`
}

// SetPassword hashes and sets the user's password.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the user's password.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HasRole checks if the user has a specific role.
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// HasPermission checks if the user has a specific permission.
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

// GetPermissions returns all unique permissions for the user.
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

// Token types.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// TokenClaims represents JWT token claims.
type TokenClaims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	TokenType string   `json:"token_type"`
	SessionID string   `json:"session_id,omitempty"`
}

// AuthTokens represents a pair of access and refresh tokens.
type AuthTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Default roles.
const (
	RoleAdmin     = "admin"
	RoleUser      = "user"
	RoleGuest     = "guest"
	RoleModerator = "moderator"
)

// Resource types for permissions.
const (
	ResourceLibrary     = "library"
	ResourceMedia       = "media"
	ResourceUser        = "user"
	ResourceTranscoding = "transcoding"
	ResourceStreaming   = "streaming"
	ResourceAcquisition = "acquisition"
	ResourceAnalytics   = "analytics"
	ResourceSystem      = "system"
)

// Action types for permissions.
const (
	ActionRead   = "read"
	ActionWrite  = "write"
	ActionDelete = "delete"
	ActionAdmin  = "admin"
)

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// HasPermission checks if the role has a specific permission.
func (r *Role) HasPermission(resource, action string) bool {
	for _, perm := range r.Permissions {
		if perm.Matches(resource, action) {
			return true
		}
	}
	return false
}

// Matches checks if the permission matches the given resource and action.
func (p *Permission) Matches(resource, action string) bool {
	// Support wildcards
	resourceMatch := p.Resource == "*" || p.Resource == resource
	actionMatch := p.Action == "*" || p.Action == action
	return resourceMatch && actionMatch
}
