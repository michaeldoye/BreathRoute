// Package api provides the HTTP API for BreatheRoute.
package api

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/api/handler"
	"github.com/breatheroute/breatheroute/internal/api/middleware"
	"github.com/breatheroute/breatheroute/internal/auth"
	"github.com/breatheroute/breatheroute/internal/commute"
	"github.com/breatheroute/breatheroute/internal/featureflags"
	"github.com/breatheroute/breatheroute/internal/user"
)

// RouterConfig holds configuration for the router.
type RouterConfig struct {
	Version            string
	BuildTime          string
	Logger             zerolog.Logger
	ServiceName        string
	Metrics            *middleware.Metrics
	AuthService        *auth.Service
	UserService        *user.Service
	FeatureFlagService *featureflags.Service
	CommuteService     *commute.Service
}

// NewRouter creates a new chi router with all API routes configured.
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Set default service name if not provided
	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "breatheroute-api"
	}

	// Global middleware - order matters
	r.Use(middleware.RequestID)            // Generate/propagate request ID first
	r.Use(middleware.Tracing(serviceName)) // Distributed tracing
	if cfg.Metrics != nil {
		r.Use(cfg.Metrics.Middleware()) // HTTP metrics
	}
	r.Use(middleware.Logger(cfg.Logger))   // Structured logging
	r.Use(middleware.Recovery(cfg.Logger)) // Panic recovery
	r.Use(chimiddleware.RealIP)            // Real IP extraction
	r.Use(middleware.SecurityHeaders)      // Security headers (HSTS, CSP, etc.)
	r.Use(middleware.RequireTLS)           // TLS enforcement (enabled via REQUIRE_TLS=true)
	r.Use(middleware.ContentTypeJSON)      // JSON content type

	// Initialize handlers
	opsHandler := handler.NewOpsHandler(cfg.Version, cfg.BuildTime)
	authHandler := handler.NewAuthHandler(cfg.AuthService)
	meHandler := handler.NewMeHandler(cfg.UserService)
	profileHandler := handler.NewProfileHandler(cfg.UserService)
	commuteHandler := handler.NewCommuteHandler(cfg.CommuteService)
	routeHandler := handler.NewRouteHandler()
	alertHandler := handler.NewAlertHandler()
	deviceHandler := handler.NewDeviceHandler()
	gdprHandler := handler.NewGDPRHandler()
	metadataHandler := handler.NewMetadataHandler()
	featureFlagsHandler := handler.NewFeatureFlagsHandler(cfg.FeatureFlagService)

	// Create auth middleware
	authMiddleware := middleware.Auth(cfg.AuthService)

	// Create rate limit middleware for different endpoint categories
	authRateLimit := middleware.RateLimitByIP(middleware.AuthRateLimit)           // 10 req/min
	expensiveRateLimit := middleware.RateLimitByIP(middleware.ExpensiveRateLimit) // 30 req/min
	standardRateLimit := middleware.RateLimitByIP(middleware.StandardRateLimit)   // 100 req/min

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Auth endpoints (public) - strict rate limiting
		r.Route("/auth", func(r chi.Router) {
			r.Use(authRateLimit) // 10 requests per minute per IP
			r.Post("/siwa", authHandler.SignInWithApple)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/logout", authHandler.Logout)
			// logout-all requires authentication
			r.With(authMiddleware).Post("/logout-all", authHandler.LogoutAll)
		})

		// Ops endpoints (public)
		r.Route("/ops", func(r chi.Router) {
			r.Get("/health", opsHandler.HealthCheck)
			r.Get("/ready", opsHandler.ReadinessCheck)
			// Status endpoint requires authentication
			r.With(authMiddleware).Get("/status", opsHandler.SystemStatus)
		})

		// Metadata endpoints (public) - standard rate limiting
		r.Route("/metadata", func(r chi.Router) {
			r.Use(standardRateLimit)
			r.Get("/air-quality/stations", metadataHandler.ListAirQualityStations)
			r.Get("/enums", metadataHandler.GetEnums)
		})

		// Me endpoints (authenticated) - user-based rate limiting
		r.Route("/me", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RateLimitByUser(middleware.StandardRateLimit)) // 100 req/min per user
			r.Get("/", meHandler.GetMe)
			r.Put("/", meHandler.UpdateMe)

			// Consents
			r.Get("/consents", meHandler.GetConsents)
			r.Put("/consents", meHandler.UpdateConsents)

			// Profile
			r.Get("/profile", profileHandler.GetProfile)
			r.Put("/profile", profileHandler.UpsertProfile)

			// Commutes
			r.Route("/commutes", func(r chi.Router) {
				r.Get("/", commuteHandler.ListCommutes)
				r.Post("/", commuteHandler.CreateCommute)
				r.Route("/{commuteId}", func(r chi.Router) {
					r.Get("/", commuteHandler.GetCommute)
					r.Put("/", commuteHandler.UpdateCommute)
					r.Delete("/", commuteHandler.DeleteCommute)
				})
			})

			// Alert subscriptions
			r.Route("/alerts/subscriptions", func(r chi.Router) {
				r.Get("/", alertHandler.ListAlertSubscriptions)
				r.Post("/", alertHandler.CreateAlertSubscription)
				r.Route("/{subscriptionId}", func(r chi.Router) {
					r.Get("/", alertHandler.GetAlertSubscription)
					r.Put("/", alertHandler.UpdateAlertSubscription)
					r.Delete("/", alertHandler.DeleteAlertSubscription)
				})
			})

			// Devices
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", deviceHandler.ListDevices)
				r.Post("/", deviceHandler.RegisterDevice)
				r.Delete("/{deviceId}", deviceHandler.UnregisterDevice)
			})
		})

		// Routes endpoint - expensive compute, strict rate limiting
		r.With(expensiveRateLimit).Post("/routes:compute", routeHandler.ComputeRoutes)

		// Alerts preview endpoint - standard rate limiting
		r.With(standardRateLimit).Post("/alerts/preview", alertHandler.PreviewDepartureWindows)

		// GDPR endpoints (authenticated) - user-based rate limiting
		r.Route("/gdpr", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(middleware.RateLimitByUser(middleware.StandardRateLimit)) // 100 req/min per user
			r.Route("/export-requests", func(r chi.Router) {
				r.Get("/", gdprHandler.ListExportRequests)
				r.Post("/", gdprHandler.CreateExportRequest)
				r.Get("/{exportRequestId}", gdprHandler.GetExportRequest)
			})
			r.Route("/deletion-requests", func(r chi.Router) {
				r.Get("/", gdprHandler.ListDeletionRequests)
				r.Post("/", gdprHandler.CreateDeletionRequest)
				r.Get("/{deletionRequestId}", gdprHandler.GetDeletionRequest)
			})
		})

		// Admin endpoints (authenticated) - for internal operations
		r.Route("/admin", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(standardRateLimit)

			// Feature flags management
			r.Route("/feature-flags", func(r chi.Router) {
				r.Get("/", featureFlagsHandler.ListFeatureFlags)
				r.Put("/", featureFlagsHandler.UpsertFeatureFlags)
				r.Post("/invalidate", featureFlagsHandler.InvalidateCache)
			})
		})
	})

	return r
}
