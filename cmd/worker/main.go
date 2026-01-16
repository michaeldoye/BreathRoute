package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Version and BuildTime are set at compile time via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("BreatheRoute Worker v%s (built %s)\n", Version, BuildTime)

	// Get port from environment or default to 8080
	// Worker also exposes health endpoint for Cloud Run
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create HTTP server for health checks
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","version":"%s"}`, Version)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Start health check server
	go func() {
		fmt.Printf("Health check server on :%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Health server error: %v\n", err)
		}
	}()

	// Start worker loop
	go func() {
		fmt.Println("Worker started, waiting for messages...")
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Worker context cancelled")
				return
			case <-ticker.C:
				fmt.Println("Worker heartbeat...")
				// TODO: Process Pub/Sub messages
				// TODO: Handle provider refresh jobs
				// TODO: Handle alert evaluation jobs
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down worker...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Health server forced to shutdown: %v\n", err)
	}

	fmt.Println("Worker stopped")
}
