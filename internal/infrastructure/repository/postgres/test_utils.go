package postgres

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Get database connection details from environment variables
	host := getEnvOrDefault("TEST_DB_HOST", "localhost")
	port := getEnvOrDefault("TEST_DB_PORT", "5432")
	user := getEnvOrDefault("TEST_DB_USER", "postgres")
	password := getEnvOrDefault("TEST_DB_PASSWORD", "postgres")
	dbname := getEnvOrDefault("TEST_DB_NAME", "narwhal_test")

	// First connect to postgres db to create test database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		host, port, user, password)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Drop test database if it exists
	err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbname)).Error
	require.NoError(t, err)

	// Create test database
	err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname)).Error
	require.NoError(t, err)

	// Connect to test database
	dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	testDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Run migrations using GORM AutoMigrate
	err = AutoMigrate(testDB)
	require.NoError(t, err)

	return testDB
}

// AutoMigrate runs GORM auto-migrations for all models
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Movie{},
		&Series{},
		&Episode{},
		&Event{},
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
} 