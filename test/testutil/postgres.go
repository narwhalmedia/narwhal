package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresContainer wraps a postgres test container
type PostgresContainer struct {
	*tcpostgres.PostgresContainer
	ConnectionString string
	DB               *gorm.DB
}

// SetupPostgresContainer creates a new postgres container for testing
func SetupPostgresContainer(t *testing.T) *PostgresContainer {
	ctx := context.Background()

	// Create postgres container
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start postgres container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Connect with GORM
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Cleanup function
	t.Cleanup(func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate postgres container: %v", err)
		}
	})

	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
		DB:                db,
	}
}

// MigrateModels runs GORM auto-migration for the given models
func (pc *PostgresContainer) MigrateModels(models ...interface{}) error {
	return pc.DB.AutoMigrate(models...)
}

// TruncateTables truncates all tables to clean data between tests
func (pc *PostgresContainer) TruncateTables(tableNames ...string) error {
	for _, table := range tableNames {
		if err := pc.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error; err != nil {
			return err
		}
	}
	return nil
}

// BeginTx starts a new transaction for testing
func (pc *PostgresContainer) BeginTx() *gorm.DB {
	return pc.DB.Begin()
}