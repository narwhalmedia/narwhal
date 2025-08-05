package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the database
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Username     string    `gorm:"uniqueIndex;not null"`
	Email        string    `gorm:"uniqueIndex;not null"`
	PasswordHash string    `gorm:"not null"`
	DisplayName  string
	Avatar       string
	IsActive     bool       `gorm:"default:true"`
	IsVerified   bool       `gorm:"default:false"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	// Embedded preferences
	PrefLanguage          string `gorm:"column:pref_language;default:'en'"`
	PrefTheme             string `gorm:"column:pref_theme;default:'dark'"`
	PrefTimeZone          string `gorm:"column:pref_time_zone;default:'UTC'"`
	PrefAutoPlayNext      bool   `gorm:"column:pref_auto_play_next;default:true"`
	PrefSubtitleLanguage  string `gorm:"column:pref_subtitle_language;default:'en'"`
	PrefPreferredQuality  string `gorm:"column:pref_preferred_quality;default:'auto'"`
	PrefEnableNotifications bool `gorm:"column:pref_enable_notifications;default:true"`

	// Relationships
	Roles    []Role    `gorm:"many2many:user_roles;"`
	Sessions []Session `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// Role represents a user role in the database
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `gorm:"uniqueIndex;not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Relationships
	Permissions []Permission `gorm:"many2many:role_permissions;"`
	Users       []User       `gorm:"many2many:user_roles;"`
}

// Permission represents a system permission in the database
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Resource    string    `gorm:"not null;index:idx_permission_resource_action"` // e.g., "library", "media", "user"
	Action      string    `gorm:"not null;index:idx_permission_resource_action"` // e.g., "read", "write", "delete"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Relationships
	Roles []Role `gorm:"many2many:role_permissions;"`
}

// Session represents an active user session in the database
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

	// Relationships
	User User `gorm:"foreignKey:UserID"`
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}

// RolePermission represents the many-to-many relationship between roles and permissions
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt    time.Time
}

// TableName customizations
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

func (UserRole) TableName() string {
	return "user_roles"
}

func (RolePermission) TableName() string {
	return "role_permissions"
}