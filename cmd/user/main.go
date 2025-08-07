package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	"github.com/narwhalmedia/narwhal/internal/user/handler"
	"github.com/narwhalmedia/narwhal/internal/user/repository"
	"github.com/narwhalmedia/narwhal/pkg/models"
	"github.com/narwhalmedia/narwhal/internal/user/service"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	authpb "github.com/narwhalmedia/narwhal/pkg/auth/v1"
	"github.com/narwhalmedia/narwhal/pkg/config"
	"github.com/narwhalmedia/narwhal/pkg/database"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/middleware"
	"github.com/narwhalmedia/narwhal/pkg/utils"
)

func main() {
	// Load configuration
	cfg := config.MustLoadServiceConfig("user", config.GetDefaultUserConfig())

	// Initialize logger
	log := logger.New()

	log.Info("User service starting",
		interfaces.String("version", config.GetServiceVersion(&cfg.Service)),
		interfaces.String("environment", cfg.Service.Environment))

	// Connect to database
	log.Info("Connecting to database...")
	db, err := database.NewGormDB(cfg.Database.ToDatabaseConfig())
	if err != nil {
		log.Fatal("Failed to connect to database", interfaces.Error(err))
	}

	// Run migrations
	log.Info("Running database migrations...")
	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations", interfaces.Error(err))
	}

	// Seed initial data
	if err := seedInitialData(db); err != nil {
		log.Fatal("Failed to seed initial data", interfaces.Error(err))
	}

	// Initialize cache
	cacheClient := utils.NewInMemoryCache()

	// Initialize event bus
	eventBus := events.NewLocalEventBus(log)

	// Initialize JWT manager
	jwtSecret := cfg.Auth.JWTSecret
	if jwtSecret == "" || jwtSecret == "development-secret-change-in-production" {
		if config.IsProduction(&cfg.Service) {
			log.Fatal("JWT secret must be set in production")
		}
		// Generate a consistent secret for development
		jwtSecret = auth.GenerateSecret()
		log.Warn("Using generated JWT secret for development")
	}

	jwtManager := auth.NewJWTManager(
		jwtSecret,
		jwtSecret, // Use same secret for refresh tokens
		cfg.Service.Name,
		cfg.Auth.JWTAccessExpiry,
		cfg.Auth.JWTRefreshExpiry,
	)

	// Initialize repository
	repo := repository.NewGormRepository(db)

	// Initialize services
	authService := service.NewAuthService(repo, jwtManager, eventBus, log)
	userService := service.NewUserService(repo, eventBus, cacheClient, log)

	// Initialize gRPC handler
	grpcHandler := handler.NewGRPCHandler(authService, userService, log)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.AuthInterceptor(jwtManager, middleware.PublicMethods())),
		grpc.StreamInterceptor(middleware.StreamAuthInterceptor(jwtManager, middleware.PublicMethods())),
	)

	// Register services
	authpb.RegisterAuthServiceServer(grpcServer, grpcHandler)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("narwhal.auth.v1.AuthService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Enable reflection
	reflection.Register(grpcServer)

	// Start session cleanup routine
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
			if err := authService.CleanupExpiredSessions(ctx); err != nil {
				log.Error("Failed to cleanup expired sessions", interfaces.Error(err))
			}
			cancel()
		}
	}()

	// Start gRPC server
	grpcAddr := config.GetGRPCListenAddress(&cfg.Service)
	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("Failed to listen", interfaces.Error(err))
	}

	log.Info("gRPC server starting", interfaces.String("address", grpcAddr))

	// Start server in goroutine
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatal("Failed to serve gRPC", interfaces.Error(err))
		}
	}()

	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics, log)
	}

	// Start health check server
	go startHealthServer(cfg.Service.Port, db, log)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down user service...")

	// Graceful shutdown with timeout
	_, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Close database connection
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	log.Info("User service stopped")
}

func startMetricsServer(cfg config.MetricsConfig, log interfaces.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement Prometheus metrics
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# Metrics endpoint\n"))
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("Metrics server starting", interfaces.String("address", addr))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Metrics server failed", interfaces.Error(err))
	}
}

func startHealthServer(port int, db *gorm.DB, log interfaces.Logger) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// Check database connection
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not ready"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	addr := fmt.Sprintf(":%d", port)
	log.Info("Health server starting", interfaces.String("address", addr))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Health server failed", interfaces.Error(err))
	}
}

// seedInitialData creates initial roles and permissions.
func seedInitialData(db *gorm.DB) error {
	ctx := context.Background()
	repo := repository.NewGormRepository(db)

	// Check if roles already exist
	if _, err := repo.GetRoleByName(ctx, models.RoleAdmin); err == nil {
		return nil // Already seeded
	}

	// Create permissions
	permissions := []models.Permission{
		// System permissions
		{Resource: models.ResourceSystem, Action: models.ActionAdmin, Description: "Full system administration"},

		// User permissions
		{Resource: models.ResourceUser, Action: models.ActionRead, Description: "View users"},
		{Resource: models.ResourceUser, Action: models.ActionWrite, Description: "Create/update users"},
		{Resource: models.ResourceUser, Action: models.ActionDelete, Description: "Delete users"},
		{Resource: models.ResourceUser, Action: models.ActionAdmin, Description: "Manage user roles and permissions"},

		// Library permissions
		{Resource: models.ResourceLibrary, Action: models.ActionRead, Description: "View libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionWrite, Description: "Create/update libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionDelete, Description: "Delete libraries"},
		{Resource: models.ResourceLibrary, Action: models.ActionAdmin, Description: "Manage library settings"},

		// Media permissions
		{Resource: models.ResourceMedia, Action: models.ActionRead, Description: "View media"},
		{Resource: models.ResourceMedia, Action: models.ActionWrite, Description: "Create/update media"},
		{Resource: models.ResourceMedia, Action: models.ActionDelete, Description: "Delete media"},

		// Streaming permissions
		{Resource: models.ResourceStreaming, Action: models.ActionRead, Description: "Stream media"},
		{Resource: models.ResourceStreaming, Action: models.ActionAdmin, Description: "Manage streaming settings"},

		// Transcoding permissions
		{Resource: models.ResourceTranscoding, Action: models.ActionRead, Description: "View transcoding jobs"},
		{Resource: models.ResourceTranscoding, Action: models.ActionWrite, Description: "Create transcoding jobs"},
		{Resource: models.ResourceTranscoding, Action: models.ActionAdmin, Description: "Manage transcoding settings"},

		// Acquisition permissions
		{Resource: models.ResourceAcquisition, Action: models.ActionRead, Description: "View acquisition settings"},
		{Resource: models.ResourceAcquisition, Action: models.ActionWrite, Description: "Manage acquisition settings"},
		{Resource: models.ResourceAcquisition, Action: models.ActionAdmin, Description: "Full acquisition control"},

		// Analytics permissions
		{Resource: models.ResourceAnalytics, Action: models.ActionRead, Description: "View analytics"},
		{Resource: models.ResourceAnalytics, Action: models.ActionAdmin, Description: "Manage analytics settings"},
	}

	for i := range permissions {
		permissions[i].ID = uuid.New()
		if err := repo.CreatePermission(ctx, &permissions[i]); err != nil {
			return fmt.Errorf("failed to create permission: %w", err)
		}
	}

	// Create roles
	roles := []struct {
		name        string
		description string
		permissions []string
	}{
		{
			name:        models.RoleAdmin,
			description: "Administrator with full access",
			permissions: []string{"*:*"}, // All permissions
		},
		{
			name:        models.RoleUser,
			description: "Regular user with media access",
			permissions: []string{
				models.ResourceLibrary + ":" + models.ActionRead,
				models.ResourceMedia + ":" + models.ActionRead,
				models.ResourceStreaming + ":" + models.ActionRead,
				models.ResourceAnalytics + ":" + models.ActionRead,
			},
		},
		{
			name:        models.RoleGuest,
			description: "Guest with limited access",
			permissions: []string{
				models.ResourceMedia + ":" + models.ActionRead,
			},
		},
	}

	for _, r := range roles {
		role := &models.Role{
			ID:          uuid.New(),
			Name:        r.name,
			Description: r.description,
		}

		// Add permissions
		if r.name == models.RoleAdmin {
			// Admin gets all permissions
			role.Permissions = permissions
		} else {
			// Find specific permissions
			for _, permStr := range r.permissions {
				parts := strings.Split(permStr, ":")
				if len(parts) == 2 {
					for _, p := range permissions {
						if p.Resource == parts[0] && p.Action == parts[1] {
							role.Permissions = append(role.Permissions, p)
							break
						}
					}
				}
			}
		}

		if err := repo.CreateRole(ctx, role); err != nil {
			return fmt.Errorf("failed to create role %s: %w", r.name, err)
		}
	}

	return nil
}
