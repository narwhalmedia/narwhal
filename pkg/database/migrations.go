package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	userRepo "github.com/narwhalmedia/narwhal/internal/user/repository"
)

// Migration represents a database migration
type Migration struct {
	ID        uint      `gorm:"primaryKey"`
	Version   string    `gorm:"uniqueIndex;not null"`
	Name      string    `gorm:"not null"`
	AppliedAt time.Time `gorm:"not null"`
}

// MigrationFunc is a function that performs a migration
type MigrationFunc func(*gorm.DB) error

// MigrationEntry represents a single migration
type MigrationEntry struct {
	Version string
	Name    string
	Up      MigrationFunc
}

// Migrator handles database migrations
type Migrator struct {
	db         *gorm.DB
	migrations []MigrationEntry
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: getAllMigrations(),
	}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate() error {
	// Ensure migrations table exists
	if err := m.db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	var appliedMigrations []Migration
	if err := m.db.Find(&appliedMigrations).Error; err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map of applied versions
	applied := make(map[string]bool)
	for _, migration := range appliedMigrations {
		applied[migration.Version] = true
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if applied[migration.Version] {
			continue
		}

		fmt.Printf("Running migration %s: %s\n", migration.Version, migration.Name)
		
		// Run migration in a transaction
		err := m.db.Transaction(func(tx *gorm.DB) error {
			if err := migration.Up(tx); err != nil {
				return err
			}

			// Record migration
			return tx.Create(&Migration{
				Version:   migration.Version,
				Name:      migration.Name,
				AppliedAt: time.Now(),
			}).Error
		})

		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.Version, err)
		}

		fmt.Printf("Completed migration %s\n", migration.Version)
	}

	return nil
}

// GetPendingMigrations returns a list of pending migrations
func (m *Migrator) GetPendingMigrations() ([]MigrationEntry, error) {
	// Get applied migrations
	var appliedMigrations []Migration
	if err := m.db.Find(&appliedMigrations).Error; err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map of applied versions
	applied := make(map[string]bool)
	for _, migration := range appliedMigrations {
		applied[migration.Version] = true
	}

	// Find pending migrations
	var pending []MigrationEntry
	for _, migration := range m.migrations {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}

	return pending, nil
}

// getAllMigrations returns all migrations in order
func getAllMigrations() []MigrationEntry {
	return []MigrationEntry{
		{
			Version: "20240101_001",
			Name:    "Create initial schema",
			Up:      migration001CreateInitialSchema,
		},
		{
			Version: "20240101_002",
			Name:    "Add indexes for performance",
			Up:      migration002AddIndexes,
		},
		{
			Version: "20240101_003",
			Name:    "Add composite constraints",
			Up:      migration003AddConstraints,
		},
	}
}

// migration001CreateInitialSchema creates the initial database schema
func migration001CreateInitialSchema(tx *gorm.DB) error {
	// Enable UUID extension
	if err := tx.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Create library tables
	if err := tx.AutoMigrate(
		&repository.Library{},
		&repository.MediaItem{},
		&repository.Episode{},
		&repository.MetadataProvider{},
		&repository.ScanHistory{},
	); err != nil {
		return fmt.Errorf("failed to migrate library models: %w", err)
	}

	// Create user tables
	if err := tx.AutoMigrate(
		&userRepo.User{},
		&userRepo.Role{},
		&userRepo.Permission{},
		&userRepo.Session{},
		&userRepo.UserRole{},
		&userRepo.RolePermission{},
	); err != nil {
		return fmt.Errorf("failed to migrate user models: %w", err)
	}

	return nil
}

// migration002AddIndexes adds performance indexes
func migration002AddIndexes(tx *gorm.DB) error {
	// Add composite indexes for better query performance
	indexes := []string{
		// Library indexes
		"CREATE INDEX IF NOT EXISTS idx_media_items_library_type ON media_items(library_id, media_type)",
		"CREATE INDEX IF NOT EXISTS idx_media_items_library_status ON media_items(library_id, status)",
		"CREATE INDEX IF NOT EXISTS idx_episodes_media_season ON episodes(media_id, season_number)",
		"CREATE INDEX IF NOT EXISTS idx_episodes_media_season_episode ON episodes(media_id, season_number, episode_number)",
		
		// User indexes
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_expires ON sessions(user_id, expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_user_roles_composite ON user_roles(user_id, role_id)",
		"CREATE INDEX IF NOT EXISTS idx_role_permissions_composite ON role_permissions(role_id, permission_id)",
		
		// Search indexes
		"CREATE INDEX IF NOT EXISTS idx_media_items_title_trgm ON media_items USING gin(title gin_trgm_ops)",
		"CREATE INDEX IF NOT EXISTS idx_media_items_original_title_trgm ON media_items USING gin(original_title gin_trgm_ops)",
	}

	// Enable pg_trgm extension for fuzzy search
	if err := tx.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return fmt.Errorf("failed to create pg_trgm extension: %w", err)
	}

	// Create indexes
	for _, index := range indexes {
		if err := tx.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// migration003AddConstraints adds database constraints
func migration003AddConstraints(tx *gorm.DB) error {
	// Add unique constraints
	constraints := []string{
		// Ensure unique episodes per series
		"ALTER TABLE episodes ADD CONSTRAINT unique_episode_per_series UNIQUE (media_id, season_number, episode_number) WHERE deleted_at IS NULL",
		
		// Ensure unique user roles
		"ALTER TABLE user_roles ADD CONSTRAINT unique_user_role UNIQUE (user_id, role_id)",
		
		// Ensure unique role permissions
		"ALTER TABLE role_permissions ADD CONSTRAINT unique_role_permission UNIQUE (role_id, permission_id)",
		
		// Ensure unique permissions
		"ALTER TABLE permissions ADD CONSTRAINT unique_permission UNIQUE (resource, action)",
	}

	for _, constraint := range constraints {
		if err := tx.Exec(constraint).Error; err != nil {
			// Check if constraint already exists
			if !isConstraintExistsError(err) {
				return fmt.Errorf("failed to add constraint: %w", err)
			}
		}
	}

	return nil
}

// isConstraintExistsError checks if the error is due to constraint already existing
func isConstraintExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "duplicate key")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

// findSubstring checks if substring exists in string
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}