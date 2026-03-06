package main

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/archplatform/collaboration-service/pkg/config"
	"github.com/archplatform/collaboration-service/pkg/server"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Initialize logger
	logger, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	
	// Load configuration
	cfg, err := config.LoadConfig(".")
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}
	
	logger.Info("Starting collaboration service",
		zap.String("name", cfg.Service.Name),
		zap.String("version", cfg.Service.Version),
		zap.String("environment", cfg.Service.Environment),
	)
	
	// Create server
	srv, err := server.NewCollaborationServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}
	
	// Start servers in goroutines
	var wg sync.WaitGroup
	
	// Start gRPC server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.StartGRPCServer(srv, cfg); err != nil {
			logger.Error("gRPC server error", zap.Error(err))
		}
	}()
	
	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.StartHTTPServer(srv, cfg); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	logger.Info("Shutting down collaboration service...")
	
	// TODO: Graceful shutdown
	
	wg.Wait()
	logger.Info("Collaboration service stopped")
}

func initLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.CallerKey = "caller"
	config.EncoderConfig.StacktraceKey = "stacktrace"
	
	// Use console encoder for development
	if os.Getenv("APP_ENV") == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	
	return config.Build()
}
