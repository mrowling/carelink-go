package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrowling/carelink-go/internal/carelink"
	"github.com/mrowling/carelink-go/internal/config"
	"github.com/mrowling/carelink-go/internal/database"
	"github.com/mrowling/carelink-go/internal/logger"
	"github.com/mrowling/carelink-go/internal/paths"
	"github.com/mrowling/carelink-go/internal/poller"
	"github.com/mrowling/carelink-go/internal/server"
)

func main() {
	// Initialize logger
	logger.Init()

	// Handle help command
	if len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	// Reject unknown commands
	if len(os.Args) > 1 {
		logger.Fatal("Main", "Unknown command: %s. Run 'carelink-go help' for usage.", os.Args[1])
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Main", "Failed to load config: %v", err)
	}

	// Check logindata.json exists
	if _, err := paths.FindFile("logindata.json"); err != nil {
		logger.Fatal("Main", "No logindata.json found — see README for authentication setup")
	}

	// Create CareLink client
	client, err := carelink.NewClient(cfg)
	if err != nil {
		logger.Fatal("Main", "Failed to create client: %v", err)
	}

	// Initialize database
	dbPath := os.Getenv("CARELINK_DB_PATH")
	if dbPath == "" {
		dbPath, err = paths.GetDefaultDBPath()
		if err != nil {
			logger.Fatal("Main", "Failed to get default database path: %v", err)
		}
	}

	db, err := database.New(dbPath)
	if err != nil {
		logger.Fatal("Main", "Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Main", "Failed to close database: %v", err)
		}
	}()
	logger.Info("Main", "Database initialized at %s", dbPath)

	// Create HTTP server
	srv := server.New(db)

	// Create poller with callback to update server's last fetch time
	poll := poller.New(client, db, cfg, srv.UpdateLastFetch)

	// Start polling in background
	go poll.Start()

	// Setup graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in separate goroutine
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Main", "HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-done
	logger.Info("Main", "Received shutdown signal")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Main", "Server shutdown error: %v", err)
	}

	logger.Info("Main", "Shutdown complete")
}

func printHelp() {
	fmt.Println(`CareLink Go - Continuous glucose monitoring bridge with HTTP API

Usage:
  carelink-go       Start the bridge (polls CareLink API + serves HTTP API)
  carelink-go help  Show this help message

HTTP Endpoints:
  GET /latest       Returns the most recent glucose reading
                    Response: {"sgv_mmol": 12.3, "direction": "Flat", "trend": 4, "age_minutes": 5}
  
  GET /health       Health check endpoint (for Kubernetes liveness/readiness probes)
                    Returns 200 OK if system is healthy, 503 if data is stale

Environment Variables:
  CARELINK_CONFIG_DIR              Config directory (default: ~/.carelink/config/)
  CARELINK_DATA_DIR                Data directory (default: ~/.carelink/data/)
  CARELINK_DB_PATH                 SQLite database path (default: <data_dir>/carelink.db)
  CARELINK_PORT                    HTTP server port (default: 8080)
  CARELINK_LOG_LEVEL               Logging level: DEBUG, INFO, WARN, ERROR (default: INFO)
  CARELINK_HEALTH_CHECK_STALE_MINS Minutes before health check fails (default: 15)

Configuration Files:
  Config directory (CARELINK_CONFIG_DIR or ~/.carelink/config/):
    .env, my.env      Environment configuration
    logindata.json    OAuth tokens
    https.txt         Proxy list (optional)
  
  Data directory (CARELINK_DATA_DIR or ~/.carelink/data/):
    carelink.db       SQLite database

Examples:
  # Start the server
  ./carelink-go

  # Query latest glucose reading
  curl http://localhost:8080/latest

  # Check health
  curl http://localhost:8080/health

  # Run on custom port
  CARELINK_PORT=3000 ./carelink-go

  # Enable debug logging
  CARELINK_LOG_LEVEL=DEBUG ./carelink-go`)
}
