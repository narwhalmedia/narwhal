package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/narwhalmedia/narwhal/internal/library/handler"
	"github.com/narwhalmedia/narwhal/internal/library/repository"
	"github.com/narwhalmedia/narwhal/internal/library/service"
	"github.com/narwhalmedia/narwhal/pkg/auth"
	"github.com/narwhalmedia/narwhal/pkg/config"
	"github.com/narwhalmedia/narwhal/pkg/database"
	"github.com/narwhalmedia/narwhal/pkg/events"
	"github.com/narwhalmedia/narwhal/pkg/interfaces"
	librarypb "github.com/narwhalmedia/narwhal/pkg/library/v1"
	"github.com/narwhalmedia/narwhal/pkg/logger"
	"github.com/narwhalmedia/narwhal/pkg/pagination"
	"github.com/narwhalmedia/narwhal/pkg/utils"
)

func main() {
	// Load configuration
	cfg := config.MustLoadServiceConfig("library", config.GetDefaultLibraryConfig())

	// Initialize logger
	// TODO: Update logger package to support configuration
	logger := logger.New()

	logger.Info("Library service starting",
		interfaces.String("version", config.GetServiceVersion(&cfg.Service)),
		interfaces.String("environment", cfg.Service.Environment))

	// Connect to database
	logger.Info("Connecting to database...")
	db, err := database.NewGormDB(cfg.Database.ToDatabaseConfig())
	if err != nil {
		logger.Fatal("Failed to connect to database", interfaces.Error(err))
	}

	// Run migrations
	logger.Info("Running database migrations...")
	if err := database.RunMigrations(db); err != nil {
		logger.Fatal("Failed to run migrations", interfaces.Error(err))
	}

	// Initialize repository
	repo, err := repository.NewGormRepository(db)
	if err != nil {
		logger.Fatal("Failed to create repository", interfaces.Error(err))
	}

	// Initialize components
	cache := utils.NewInMemoryCache()
	eventBus := events.NewInMemoryEventBus(logger)

	// Start event bus
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := eventBus.Start(ctx); err != nil {
		logger.Fatal("Failed to start event bus", interfaces.Error(err))
	}

	// Initialize library service
	libraryService := service.NewLibraryService(
		repo,
		eventBus,
		cache,
		logger,
	)

	logger.Info("Media Library Service starting...")

	// Initialize JWT manager for auth middleware
	jwtManager := auth.NewJWTManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTSecret, // Use same secret for refresh tokens
		"narwhal-library-service",
		cfg.Auth.AccessTokenDuration,
		cfg.Auth.RefreshTokenDuration,
	)

	// Initialize RBAC
	rbacConfig := auth.RBACConfig{
		Type:             auth.RBACType(cfg.Auth.RBACType),
		CasbinModelPath:  cfg.Auth.RBACModelPath,
		CasbinPolicyPath: cfg.Auth.RBACPolicyPath,
		Logger:           logger,
	}

	rbac, err := auth.NewRBACFromConfig(rbacConfig)
	if err != nil {
		logger.Fatal("Failed to initialize RBAC", interfaces.Error(err))
	}

	// Create auth interceptor
	authInterceptor := auth.NewAuthInterceptor(jwtManager, rbac)

	// Create gRPC server with auth interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor.UnaryServerInterceptor()),
		grpc.StreamInterceptor(authInterceptor.StreamServerInterceptor()),
	)

	// Initialize pagination encoder
	var paginationEncoder *pagination.CursorEncoder
	if cfg.Pagination.CursorEncryptionKey != "" {
		// Ensure key is 32 bytes
		key := []byte(cfg.Pagination.CursorEncryptionKey)
		if len(key) < 32 {
			// Pad with zeros if too short
			padded := make([]byte, 32)
			copy(padded, key)
			key = padded
		} else if len(key) > 32 {
			// Truncate if too long
			key = key[:32]
		}

		encoder, err := pagination.NewCursorEncoder(key)
		if err != nil {
			logger.Error("Failed to create pagination encoder", interfaces.Error(err))
			// Continue without pagination encryption
		} else {
			paginationEncoder = encoder
		}
	}

	// Create and register gRPC handler
	grpcHandler := handler.NewGRPCHandler(libraryService, logger, paginationEncoder)
	librarypb.RegisterLibraryServiceServer(grpcServer, grpcHandler)

	// Register reflection service for grpcurl
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcAddr := config.GetGRPCListenAddress(&cfg.Service)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatal("Failed to listen", interfaces.Error(err))
	}

	go func() {
		logger.Info("gRPC server starting", interfaces.String("address", grpcAddr))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve gRPC", interfaces.Error(err))
		}
	}()

	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg.Metrics, logger)
	}

	// Start health check server
	go startHealthServer(cfg.Service.Port, logger)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")

	// Graceful shutdown with timeout
	_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop gRPC server
	grpcServer.GracefulStop()

	// Stop event bus
	eventBus.Stop()

	// Close database connection
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	logger.Info("Library service stopped")
}

func startMetricsServer(cfg config.MetricsConfig, log interfaces.Logger) {
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.Path, func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement Prometheus metrics
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# Metrics endpoint\n"))
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Info("Metrics server starting", interfaces.String("address", addr))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Metrics server failed", interfaces.Error(err))
	}
}

func startHealthServer(port int, log interfaces.Logger) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Readiness check endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Check database connection, etc.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	addr := fmt.Sprintf(":%d", port)
	log.Info("Health server starting", interfaces.String("address", addr))

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error("Health server failed", interfaces.Error(err))
	}
}
