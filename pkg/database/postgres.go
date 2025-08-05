package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MinConnections  int
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	LogLevel        logger.LogLevel
}

// DefaultPostgresConfig returns a default PostgreSQL configuration
func DefaultPostgresConfig() *PostgresConfig {
	return &PostgresConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "narwhal",
		Password:        "narwhal_dev",
		Database:        "narwhal_dev",
		SSLMode:         "disable",
		MaxConnections:  25,
		MinConnections:  5,
		MaxConnLifetime: time.Hour,
		MaxConnIdleTime: 30 * time.Minute,
		LogLevel:        logger.Info,
	}
}

// NewGormDB creates a new GORM database connection
func NewGormDB(cfg *PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode)

	// Configure GORM logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  cfg.LogLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		PrepareStmt: true, // Prepare statements for better performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL database: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MinConnections)
	sqlDB.SetConnMaxLifetime(cfg.MaxConnLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.MaxConnIdleTime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// MigrateDatabase runs GORM auto-migrations for the given models (deprecated)
// Use RunMigrations instead for versioned migrations
func MigrateDatabase(db *gorm.DB, models ...interface{}) error {
	// Enable UUID extension for PostgreSQL
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create UUID extension: %w", err)
	}

	// Run auto-migration
	if err := db.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run auto-migration: %w", err)
	}

	return nil
}

// RunMigrations runs all pending database migrations
func RunMigrations(db *gorm.DB) error {
	migrator := NewMigrator(db)
	return migrator.Migrate()
}

// GetPendingMigrations returns a list of migrations that haven't been applied yet
func GetPendingMigrations(db *gorm.DB) ([]MigrationEntry, error) {
	migrator := NewMigrator(db)
	return migrator.GetPendingMigrations()
}
