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
	"github.com/breatheroute/breatheroute/internal/auth"
	"github.com/breatheroute/breatheroute/internal/commute"
	"github.com/breatheroute/breatheroute/internal/database"
	"github.com/breatheroute/breatheroute/internal/featureflags"
	"github.com/breatheroute/breatheroute/internal/telemetry"
	"github.com/breatheroute/breatheroute/internal/user"
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

	// Connect to database
	dbConfig := database.ConfigFromEnv()
	pool, err := database.Connect(ctx, dbConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer pool.Close()
	log.Info().
		Str("host", dbConfig.Host).
		Int("port", dbConfig.Port).
		Str("database", dbConfig.Database).
		Msg("database connected")

	// Initialize auth repositories and service
	authUserRepo := auth.NewPostgresUserRepository(pool)
	authRefreshRepo := auth.NewPostgresRefreshTokenRepository(pool)

	// Initialize JWT service (get signing key from environment)
	jwtSigningKey := os.Getenv("JWT_SIGNING_KEY")
	if jwtSigningKey == "" {
		jwtSigningKey = "local-dev-signing-key-change-in-production"
		log.Warn().Msg("using default JWT signing key - not secure for production")
	}

	jwtService := auth.NewJWTService(auth.JWTConfig{
		SigningKey: jwtSigningKey,
	})

	// Initialize SIWA verifier (may be nil if not configured)
	var siwaVerifier *auth.SIWAVerifier
	appleBundleID := os.Getenv("APPLE_CLIENT_ID") // Bundle ID for iOS app
	if appleBundleID != "" {
		siwaVerifier = auth.NewSIWAVerifier(auth.SIWAConfig{
			BundleID: appleBundleID,
		})
		log.Info().Msg("Sign in with Apple verifier initialized")
	} else {
		log.Warn().Msg("Sign in with Apple not configured - auth endpoints will fail")
	}

	authService := auth.NewService(auth.ServiceConfig{
		SIWAVerifier:  siwaVerifier,
		JWTService:    jwtService,
		UserRepo:      authUserRepo,
		RefreshRepo:   authRefreshRepo,
		DefaultLocale: "nl-NL",
	})
	log.Info().Msg("auth service initialized")

	// Initialize user repository and service
	userRepo := user.NewPostgresRepository(pool)
	userService := user.NewService(userRepo)
	log.Info().Msg("user service initialized")

	// Initialize commute repository and service
	commuteRepo := commute.NewPostgresRepository(pool)
	commuteService := commute.NewService(commuteRepo)
	log.Info().Msg("commute service initialized")

	// Initialize feature flags repository and service
	ffRepo := featureflags.NewPostgresRepository(pool)
	ffService := featureflags.NewService(featureflags.ServiceConfig{
		Repository: ffRepo,
		Logger:     log,
		CacheTTL:   1 * time.Minute,
	})
	log.Info().Msg("feature flags service initialized")

	// Create router with configuration
	router := api.NewRouter(api.RouterConfig{
		Version:            Version,
		BuildTime:          BuildTime,
		Logger:             log,
		ServiceName:        serviceName,
		Metrics:            metrics,
		AuthService:        authService,
		UserService:        userService,
		FeatureFlagService: ffService,
		CommuteService:     commuteService,
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
