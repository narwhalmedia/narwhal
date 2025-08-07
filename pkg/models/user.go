package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents the role of a user.
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
	UserRoleGuest UserRole = "guest"
)

// User represents a user in the system.
type User struct {
	ID           uuid.UUID   `json:"id"                   db:"id"`
	Username     string      `json:"username"             db:"username"`
	Email        string      `json:"email"                db:"email"`
	PasswordHash string      `json:"-"                    db:"password_hash"`
	Role         UserRole    `json:"role"                 db:"role"`
	Active       bool        `json:"active"               db:"active"`
	Preferences  Preferences `json:"preferences"`
	Created      time.Time   `json:"created"              db:"created"`
	Updated      time.Time   `json:"updated"              db:"updated"`
	LastLogin    *time.Time  `json:"last_login,omitempty" db:"last_login"`
}

// Preferences contains user preferences.
type Preferences struct {
	Language         string `json:"language"`
	Theme            string `json:"theme"`
	DefaultQuality   string `json:"default_quality"`
	SubtitleLanguage string `json:"subtitle_language"`
	AutoPlay         bool   `json:"auto_play"`
	SkipIntro        bool   `json:"skip_intro"`
}

// WatchHistory represents a user's watch history for a media item.
type WatchHistory struct {
	ID          uuid.UUID  `json:"id"                   db:"id"`
	UserID      uuid.UUID  `json:"user_id"              db:"user_id"`
	MediaID     uuid.UUID  `json:"media_id"             db:"media_id"`
	EpisodeID   *uuid.UUID `json:"episode_id,omitempty" db:"episode_id"`
	Position    int        `json:"position"             db:"position"` // in seconds
	Duration    int        `json:"duration"             db:"duration"` // total duration
	Completed   bool       `json:"completed"            db:"completed"`
	LastWatched time.Time  `json:"last_watched"         db:"last_watched"`
}

// UserProfile represents a user's profile within an account.
type UserProfile struct {
	ID           uuid.UUID `json:"id"                     db:"id"`
	UserID       uuid.UUID `json:"user_id"                db:"user_id"`
	Name         string    `json:"name"                   db:"name"`
	Avatar       string    `json:"avatar,omitempty"       db:"avatar"`
	IsKid        bool      `json:"is_kid"                 db:"is_kid"`
	PIN          string    `json:"-"                      db:"pin"` // Optional PIN for profile
	Restrictions []string  `json:"restrictions,omitempty"`
	Created      time.Time `json:"created"                db:"created"`
}
