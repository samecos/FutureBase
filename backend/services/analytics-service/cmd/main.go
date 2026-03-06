package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/archplatform/analytics-service/internal/config"
	"github.com/archplatform/analytics-service/internal/handler"
	"github.com/archplatform/analytics-service/internal/storage"
	"github.com/gorilla/mux"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize storage
	postgresStorage, err := storage.NewPostgresStorage(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer postgresStorage.Close()

	// Create handler
	h := handler.NewHandler(postgresStorage)

	// Setup HTTP router
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	// HTTP server
	httpServer := &http.Server{
		Addr:    ":" + getEnv("HTTP_PORT", "8090"),
		Handler: router,
	}

	// Start HTTP server in background
	go func() {
		log.Printf("HTTP server starting on port %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
