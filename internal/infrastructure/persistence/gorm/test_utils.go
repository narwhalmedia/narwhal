package gorm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewTestDB creates a new in-memory SQLite database for testing
func NewTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&SeriesModel{},
		&EpisodeModel{},
		&MovieModel{},
		&EventModel{},
	)
	require.NoError(t, err)

	return db
}

// CleanupDB cleans up the test database
func CleanupDB(t *testing.T, db *gorm.DB) {
	err := db.Migrator().DropTable(
		&SeriesModel{},
		&EpisodeModel{},
		&MovieModel{},
		&EventModel{},
	)
	require.NoError(t, err)
} 