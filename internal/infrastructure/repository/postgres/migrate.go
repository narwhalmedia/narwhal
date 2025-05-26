package postgres

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	Version   int       `gorm:"primary_key"`
	AppliedAt time.Time `gorm:"not null"`
}

// TableName specifies the table name for Migration
func (Migration) TableName() string {
	return "migrations"
}

// MigrationFile represents a migration file
type MigrationFile struct {
	Version int
	Up      string
	Down    string
}

// RunMigrations runs all migrations in order using GORM
func RunMigrations(db *gorm.DB, migrationsDir string) error {
	// Create migrations table if it doesn't exist
	if err := db.AutoMigrate(&Migration{}); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	var appliedMigrations []Migration
	if err := db.Order("version").Find(&appliedMigrations).Error; err != nil {
		return fmt.Errorf("failed to query applied migrations: %w", err)
	}

	applied := make(map[int]bool)
	for _, m := range appliedMigrations {
		applied[m.Version] = true
	}

	// Read migration files
	files, err := ioutil.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []MigrationFile
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}

		version := parseVersion(file.Name())
		if version == 0 {
			continue
		}

		// Read up migration
		upPath := filepath.Join(migrationsDir, file.Name())
		upSQL, err := ioutil.ReadFile(upPath)
		if err != nil {
			return fmt.Errorf("failed to read up migration %s: %w", file.Name(), err)
		}

		// Read down migration
		downPath := filepath.Join(migrationsDir, fmt.Sprintf("%06d.down.sql", version))
		downSQL, err := ioutil.ReadFile(downPath)
		if err != nil {
			return fmt.Errorf("failed to read down migration %s: %w", file.Name(), err)
		}

		migrations = append(migrations, MigrationFile{
			Version: version,
			Up:      string(upSQL),
			Down:    string(downSQL),
		})
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply pending migrations
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}

		// Run migration in transaction
		err := db.Transaction(func(tx *gorm.DB) error {
			// Apply migration
			if err := tx.Exec(migration.Up).Error; err != nil {
				return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
			}

			// Record migration
			m := Migration{
				Version:   migration.Version,
				AppliedAt: time.Now(),
			}
			if err := tx.Create(&m).Error; err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// parseVersion extracts the version number from a migration filename
func parseVersion(filename string) int {
	var version int
	fmt.Sscanf(filename, "%d.up.sql", &version)
	return version
} 