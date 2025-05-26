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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	mediav1 "github.com/narwhalmedia/narwhal/api/proto/media/v1"
	"github.com/narwhalmedia/narwhal/internal/config"
	"github.com/narwhalmedia/narwhal/internal/container"
	"github.com/narwhalmedia/narwhal/internal/logger"
	grpcmiddleware "github.com/narwhalmedia/narwhal/internal/middleware/grpc"
)

const serviceName = "media-service"

func main() {
	// Load configuration
	cfg, err := config.Load(serviceName)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	// Initialize logger
	log, err := logger.New(cfg.Server.ServiceName, cfg.Server.Environment, cfg.Server.LogLevel, cfg.Observability.LogFormat)
	if err != nil {
		panic(fmt.Sprintf("failed to create logger: %v", err))
	}
	defer log.Sync()

	// Log startup
	log.Info("starting service",
		zap.String("version", "1.0.0"), // TODO: Get from build info
		zap.String("environment", cfg.Server.Environment),
	)

	// Initialize service container with all dependencies
	serviceContainer, cleanup, err := container.InitializeMediaService(cfg, log)
	if err != nil {
		log.Fatal("failed to initialize service", zap.Error(err))
	}
	defer cleanup()

	// Setup event consumers (if enabled)
	if cfg.Server.Environment != "test" {
		// Note: download and transcode services are nil until implemented
		if err := container.SetupEventConsumers(ctx, serviceContainer, nil, nil); err != nil {
			log.Error("failed to setup event consumers", zap.Error(err))
			// Continue without consumers - they're not critical for basic operation
		}
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpcmiddleware.LoggingInterceptor(log),
			// TODO: Add tracing interceptor
			// TODO: Add metrics interceptor
			// TODO: Add auth interceptor
		),
		grpc.ChainStreamInterceptor(
			grpcmiddleware.StreamLoggingInterceptor(log),
			// TODO: Add stream interceptors
		),
	)

	// Register services
	mediav1.RegisterMediaServiceServer(grpcServer, serviceContainer.GRPCService)
	
	// Register health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)
	
	// Register reflection for grpcurl
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcLis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GRPCPort))
	if err != nil {
		log.Fatal("failed to listen on gRPC port", zap.Error(err))
	}
	
	go func() {
		log.Info("starting gRPC server", zap.Int("port", cfg.Server.GRPCPort))
		if err := grpcServer.Serve(grpcLis); err != nil {
			log.Fatal("failed to serve gRPC", zap.Error(err))
		}
	}()

	// Create context for the service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
	)
	
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort)
	
	if err := mediav1.RegisterMediaServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
		log.Fatal("failed to register gateway", zap.Error(err))
	}

	// Add health check endpoint
	httpMux := http.NewServeMux()
	httpMux.Handle("/", mux)
	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: httpMux,
	}

	// Start HTTP server
	go func() {
		log.Info("starting HTTP server", zap.Int("port", cfg.Server.HTTPPort))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to serve HTTP", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Info("shutting down service")

	// Set health status to not serving
	healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTime)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to shutdown HTTP server", zap.Error(err))
	}

	// Shutdown gRPC server
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		log.Warn("shutdown timeout exceeded, forcing stop")
		grpcServer.Stop()
	case <-stopped:
		log.Info("gRPC server stopped gracefully")
	}

	log.Info("service shutdown complete")
} 