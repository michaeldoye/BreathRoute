package pollen

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/featureflags"
)

// Provider defines the interface for pollen data providers.
type Provider interface {
	// GetRegionalPollen fetches pollen data for a region containing the given coordinates.
	GetRegionalPollen(ctx context.Context, lat, lon float64) (*RegionalPollen, error)

	// GetForecast fetches pollen forecast for a region.
	GetForecast(ctx context.Context, lat, lon float64) (*Forecast, error)

	// Name returns the provider name for logging.
	Name() string
}

// ServiceConfig holds configuration for the pollen service.
type ServiceConfig struct {
	// Provider is the pollen data provider.
	Provider Provider

	// FeatureFlags is the feature flag service (optional).
	// If provided, pollen data can be disabled via feature flag.
	FeatureFlags *featureflags.Service

	// Logger for service operations.
	Logger zerolog.Logger

	// CacheTTL is how long to cache pollen data (default: 1 hour).
	// Pollen data changes slowly, so longer cache is appropriate.
	CacheTTL time.Duration

	// StaleIfErrorTTL allows serving stale data on provider errors (default: 6 hours).
	StaleIfErrorTTL time.Duration
}

// Service provides pollen data with caching and feature flag control.
type Service struct {
	provider        Provider
	featureFlags    *featureflags.Service
	logger          zerolog.Logger
	cacheTTL        time.Duration
	staleIfErrorTTL time.Duration

	mu              sync.RWMutex
	cache           map[string]*cachedPollen
	forecastCache   map[string]*cachedForecast
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

type cachedPollen struct {
	data      *RegionalPollen
	fetchedAt time.Time
	expiresAt time.Time
}

type cachedForecast struct {
	data      *Forecast
	fetchedAt time.Time
	expiresAt time.Time
}

// NewService creates a new pollen service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 1 * time.Hour
	}

	staleIfErrorTTL := cfg.StaleIfErrorTTL
	if staleIfErrorTTL == 0 {
		staleIfErrorTTL = 6 * time.Hour
	}

	return &Service{
		provider:        cfg.Provider,
		featureFlags:    cfg.FeatureFlags,
		logger:          cfg.Logger,
		cacheTTL:        cacheTTL,
		staleIfErrorTTL: staleIfErrorTTL,
		cache:           make(map[string]*cachedPollen),
		forecastCache:   make(map[string]*cachedForecast),
		cleanupInterval: 30 * time.Minute,
	}
}

// GetRegionalPollen returns pollen data for a location.
// Returns ErrPollenDisabled if pollen factor is disabled via feature flag.
func (s *Service) GetRegionalPollen(ctx context.Context, lat, lon float64) (*RegionalPollen, error) {
	// Check feature flag
	if s.isPollenDisabled(ctx) {
		s.logger.Debug().Msg("pollen factor disabled by feature flag")
		return nil, ErrPollenDisabled
	}

	if err := validateCoordinates(lat, lon); err != nil {
		return nil, err
	}

	cacheKey := s.cacheKey(lat, lon)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		return cached.data, nil
	}
	s.mu.RUnlock()

	// Fetch from provider
	return s.fetchPollen(ctx, lat, lon, cacheKey)
}

// GetForecast returns pollen forecast for a location.
// Returns ErrPollenDisabled if pollen factor is disabled via feature flag.
func (s *Service) GetForecast(ctx context.Context, lat, lon float64) (*Forecast, error) {
	// Check feature flag
	if s.isPollenDisabled(ctx) {
		s.logger.Debug().Msg("pollen factor disabled by feature flag")
		return nil, ErrPollenDisabled
	}

	if err := validateCoordinates(lat, lon); err != nil {
		return nil, err
	}

	cacheKey := s.cacheKey(lat, lon)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.forecastCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		return cached.data, nil
	}
	s.mu.RUnlock()

	// Fetch from provider
	return s.fetchForecast(ctx, lat, lon, cacheKey)
}

// GetExposureFactor returns the pollen exposure factor for a location.
// Returns 1.0 (neutral) if pollen is disabled or data is unavailable.
func (s *Service) GetExposureFactor(ctx context.Context, lat, lon float64) float64 {
	data, err := s.GetRegionalPollen(ctx, lat, lon)
	if err != nil || data == nil {
		return 1.0
	}
	return data.ExposureFactor()
}

// IsEnabled returns true if pollen factor is enabled.
func (s *Service) IsEnabled(ctx context.Context) bool {
	return !s.isPollenDisabled(ctx)
}

// isPollenDisabled checks the feature flag.
func (s *Service) isPollenDisabled(ctx context.Context) bool {
	if s.featureFlags == nil {
		return false
	}
	return s.featureFlags.IsPollenFactorDisabled(ctx)
}

// fetchPollen fetches pollen data from provider and updates cache.
func (s *Service) fetchPollen(ctx context.Context, lat, lon float64, cacheKey string) (*RegionalPollen, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		return cached.data, nil
	}

	s.logger.Debug().
		Float64("lat", lat).
		Float64("lon", lon).
		Str("provider", s.provider.Name()).
		Msg("fetching pollen data from provider")

	data, err := s.provider.GetRegionalPollen(ctx, lat, lon)
	if err != nil {
		s.logger.Error().Err(err).
			Float64("lat", lat).
			Float64("lon", lon).
			Msg("failed to fetch pollen data")

		// Check for stale data
		if cached, ok := s.cache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Msg("serving stale pollen data due to provider error")
				return cached.data, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.cache[cacheKey] = &cachedPollen{
		data:      data,
		fetchedAt: now,
		expiresAt: now.Add(s.cacheTTL),
	}

	// Periodic cleanup
	s.cleanupIfNeeded()

	return data, nil
}

// fetchForecast fetches forecast from provider and updates cache.
func (s *Service) fetchForecast(ctx context.Context, lat, lon float64, cacheKey string) (*Forecast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if cached, ok := s.forecastCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		return cached.data, nil
	}

	s.logger.Debug().
		Float64("lat", lat).
		Float64("lon", lon).
		Str("provider", s.provider.Name()).
		Msg("fetching pollen forecast from provider")

	data, err := s.provider.GetForecast(ctx, lat, lon)
	if err != nil {
		s.logger.Error().Err(err).
			Float64("lat", lat).
			Float64("lon", lon).
			Msg("failed to fetch pollen forecast")

		// Check for stale data
		if cached, ok := s.forecastCache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Msg("serving stale pollen forecast due to provider error")
				return cached.data, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.forecastCache[cacheKey] = &cachedForecast{
		data:      data,
		fetchedAt: now,
		expiresAt: now.Add(s.cacheTTL),
	}

	// Periodic cleanup
	s.cleanupIfNeeded()

	return data, nil
}

// cacheKey generates a cache key for a location.
// Uses region-level granularity (0.5 degrees, ~55km).
func (s *Service) cacheKey(lat, lon float64) string {
	// Round to 0.5 degree grid for regional pollen data
	gridLat := float64(int(lat*2)) / 2
	gridLon := float64(int(lon*2)) / 2
	return fmt.Sprintf("%.1f:%.1f", gridLat, gridLon)
}

// cleanupIfNeeded removes expired entries if cleanup interval has passed.
func (s *Service) cleanupIfNeeded() {
	now := time.Now()
	if now.Sub(s.lastCleanup) < s.cleanupInterval {
		return
	}

	s.lastCleanup = now
	expired := 0

	for key, cached := range s.cache {
		if now.After(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
			delete(s.cache, key)
			expired++
		}
	}

	for key, cached := range s.forecastCache {
		if now.After(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
			delete(s.forecastCache, key)
			expired++
		}
	}

	if expired > 0 {
		s.logger.Debug().
			Int("expired_entries", expired).
			Msg("cleaned up expired pollen cache entries")
	}
}

// InvalidateCache clears all cached data.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*cachedPollen)
	s.forecastCache = make(map[string]*cachedForecast)
}

// CacheStats returns cache statistics.
func (s *Service) CacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	pollenFresh := 0
	forecastFresh := 0

	for _, c := range s.cache {
		if now.Before(c.expiresAt) {
			pollenFresh++
		}
	}
	for _, c := range s.forecastCache {
		if now.Before(c.expiresAt) {
			forecastFresh++
		}
	}

	return CacheStats{
		PollenEntries:        len(s.cache),
		PollenFreshEntries:   pollenFresh,
		ForecastEntries:      len(s.forecastCache),
		ForecastFreshEntries: forecastFresh,
		Provider:             s.provider.Name(),
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	PollenEntries        int
	PollenFreshEntries   int
	ForecastEntries      int
	ForecastFreshEntries int
	Provider             string
}

// validateCoordinates checks if coordinates are valid.
func validateCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}
