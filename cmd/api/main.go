// Package main provides the entrypoint for the BreatheRoute API server.
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/api"
	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/telemetry"
)

// Version and BuildTime are set at compile time via ldflags.
var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	const serviceName = "breatheroute-api"

	// Setup structured logging
	log := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", serviceName).
		Str("version", Version).
		Logger()

	log.Info().
		Str("build_time", BuildTime).
		Msg("starting BreatheRoute API")

	// Get configuration from environment
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = "localhost:4317"
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Initialize OpenTelemetry
	ctx := context.Background()
	telemetryEnabled := os.Getenv("OTEL_ENABLED") == "true"

	tp, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    serviceName,
		ServiceVersion: Version,
		Environment:    env,
		OTLPEndpoint:   otlpEndpoint,
		Enabled:        telemetryEnabled,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize telemetry")
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := tp.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Error().Err(shutdownErr).Msg("failed to shutdown telemetry")
		}
	}()

	if telemetryEnabled {
		log.Info().
			Str("otlp_endpoint", otlpEndpoint).
			Msg("OpenTelemetry initialized")
	}

	// Initialize metrics
	metrics, err := middleware.NewMetrics()
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize metrics")
		os.Exit(1) //nolint:gocritic // intentional exit, telemetry cleanup is best-effort
	}

	// Create router with configuration
	router := api.NewRouter(api.RouterConfig{
		Version:     Version,
		BuildTime:   BuildTime,
		Logger:      log,
		ServiceName: serviceName,
		Metrics:     metrics,
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().
			Str("addr", server.Addr).
			Msg("server listening")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
		os.Exit(1)
	}

	log.Info().Msg("server stopped")
}
