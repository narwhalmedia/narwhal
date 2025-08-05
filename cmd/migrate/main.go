package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/narwhalmedia/narwhal/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	var (
		host     = flag.String("host", getEnv("DB_HOST", "localhost"), "Database host")
		port     = flag.Int("port", getEnvAsInt("DB_PORT", 5432), "Database port")
		user     = flag.String("user", getEnv("DB_USER", "narwhal"), "Database user")
		password = flag.String("password", getEnv("DB_PASSWORD", "narwhal_dev"), "Database password")
		dbname   = flag.String("dbname", getEnv("DB_NAME", "narwhal_dev"), "Database name")
		sslmode  = flag.String("sslmode", getEnv("DB_SSLMODE", "disable"), "SSL mode")
		status   = flag.Bool("status", false, "Show migration status")
		dryRun   = flag.Bool("dry-run", false, "Show pending migrations without applying them")
	)
	flag.Parse()

	// Create database configuration
	cfg := &database.PostgresConfig{
		Host:     *host,
		Port:     *port,
		User:     *user,
		Password: *password,
		Database: *dbname,
		SSLMode:  *sslmode,
		LogLevel: logger.Info,
	}

	// Connect to database
	db, err := database.NewGormDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Handle different commands
	switch {
	case *status:
		showMigrationStatus(db)
	case *dryRun:
		showPendingMigrations(db)
	default:
		runMigrations(db)
	}
}

// runMigrations applies all pending migrations
func runMigrations(db *gorm.DB) {
	fmt.Println("Running database migrations...")
	
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	
	fmt.Println("Migrations completed successfully!")
}

// showMigrationStatus displays the current migration status
func showMigrationStatus(db *gorm.DB) {
	// Get all migrations
	var migrations []database.Migration
	if err := db.Order("applied_at DESC").Find(&migrations).Error; err != nil {
		log.Fatalf("Failed to get migrations: %v", err)
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations have been applied yet.")
		return
	}

	fmt.Println("Applied migrations:")
	fmt.Println("==================")
	for _, m := range migrations {
		fmt.Printf("%s | %s | Applied at: %s\n", m.Version, m.Name, m.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	// Show pending migrations
	pending, err := database.GetPendingMigrations(db)
	if err != nil {
		log.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) > 0 {
		fmt.Println("\nPending migrations:")
		fmt.Println("==================")
		for _, m := range pending {
			fmt.Printf("%s | %s\n", m.Version, m.Name)
		}
	} else {
		fmt.Println("\nAll migrations are up to date!")
	}
}

// showPendingMigrations displays migrations that would be applied
func showPendingMigrations(db *gorm.DB) {
	pending, err := database.GetPendingMigrations(db)
	if err != nil {
		log.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations.")
		return
	}

	fmt.Println("Pending migrations that would be applied:")
	fmt.Println("========================================")
	for _, m := range pending {
		fmt.Printf("%s | %s\n", m.Version, m.Name)
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intVal int
		if _, err := fmt.Sscanf(value, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultValue
}