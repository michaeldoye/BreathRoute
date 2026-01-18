package routing

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ServiceConfig holds configuration for the routing service.
type ServiceConfig struct {
	// Provider is the routing data provider.
	Provider Provider

	// Logger for service operations.
	Logger zerolog.Logger

	// CacheTTL is how long to cache routing data (default: 5 minutes).
	CacheTTL time.Duration

	// CacheGridSize is the size of cache grid cells in degrees (default: 0.01 ~ 1.1km).
	// Points within the same grid cell share cached data.
	CacheGridSize float64

	// StaleIfErrorTTL allows serving stale data on provider errors (default: 15 minutes).
	StaleIfErrorTTL time.Duration

	// CleanupInterval is how often to clean up expired entries (default: 5 minutes).
	CleanupInterval time.Duration
}

// Service provides routing data with caching.
type Service struct {
	provider        Provider
	logger          zerolog.Logger
	cacheTTL        time.Duration
	cacheGridSize   float64
	staleIfErrorTTL time.Duration
	cleanupInterval time.Duration

	mu          sync.RWMutex
	cache       map[string]*cachedDirections
	lastCleanup time.Time
}

type cachedDirections struct {
	response  *DirectionsResponse
	fetchedAt time.Time
	expiresAt time.Time
}

// NewService creates a new routing service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}

	cacheGridSize := cfg.CacheGridSize
	if cacheGridSize == 0 {
		cacheGridSize = 0.01 // ~1.1km at equator
	}

	staleIfErrorTTL := cfg.StaleIfErrorTTL
	if staleIfErrorTTL == 0 {
		staleIfErrorTTL = 15 * time.Minute
	}

	cleanupInterval := cfg.CleanupInterval
	if cleanupInterval == 0 {
		cleanupInterval = 5 * time.Minute
	}

	return &Service{
		provider:        cfg.Provider,
		logger:          cfg.Logger,
		cacheTTL:        cacheTTL,
		cacheGridSize:   cacheGridSize,
		staleIfErrorTTL: staleIfErrorTTL,
		cleanupInterval: cleanupInterval,
		cache:           make(map[string]*cachedDirections),
	}
}

// GetDirections returns route directions between two points.
// Uses cached data if available and not expired.
func (s *Service) GetDirections(ctx context.Context, req DirectionsRequest) (*DirectionsResponse, error) {
	// Validate coordinates
	if err := validateCoordinates(req.Origin); err != nil {
		return nil, &Error{
			Provider: s.provider.Name(),
			Code:     "INVALID_ORIGIN",
			Message:  "invalid origin coordinates",
			Err:      ErrInvalidCoordinates,
		}
	}
	if err := validateCoordinates(req.Destination); err != nil {
		return nil, &Error{
			Provider: s.provider.Name(),
			Code:     "INVALID_DESTINATION",
			Message:  "invalid destination coordinates",
			Err:      ErrInvalidCoordinates,
		}
	}

	cacheKey := s.cacheKey(req)

	// Check cache (read lock)
	s.mu.RLock()
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		s.logger.Debug().
			Str("cache_key", cacheKey).
			Msg("cache hit for directions")
		return cached.response, nil
	}
	s.mu.RUnlock()

	// Fetch from provider
	return s.fetchDirections(ctx, req, cacheKey)
}

// fetchDirections fetches directions from provider and updates cache.
func (s *Service) fetchDirections(ctx context.Context, req DirectionsRequest, cacheKey string) (*DirectionsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache (prevents thundering herd)
	if cached, ok := s.cache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.logger.Debug().
			Str("cache_key", cacheKey).
			Msg("cache hit after double-check")
		return cached.response, nil
	}

	s.logger.Debug().
		Float64("origin_lat", req.Origin.Lat).
		Float64("origin_lon", req.Origin.Lon).
		Float64("dest_lat", req.Destination.Lat).
		Float64("dest_lon", req.Destination.Lon).
		Str("profile", string(req.Profile)).
		Str("provider", s.provider.Name()).
		Msg("fetching directions from provider")

	resp, err := s.provider.GetDirections(ctx, req)
	if err != nil {
		s.logger.Error().Err(err).
			Float64("origin_lat", req.Origin.Lat).
			Float64("origin_lon", req.Origin.Lon).
			Float64("dest_lat", req.Destination.Lat).
			Float64("dest_lon", req.Destination.Lon).
			Str("profile", string(req.Profile)).
			Msg("failed to fetch directions")

		// Check for stale data (stale-if-error pattern)
		if cached, ok := s.cache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Str("cache_key", cacheKey).
					Msg("serving stale directions data due to provider error")
				return cached.response, nil
			}
		}

		return nil, err
	}

	// Update cache
	now := time.Now()
	s.cache[cacheKey] = &cachedDirections{
		response:  resp,
		fetchedAt: now,
		expiresAt: now.Add(s.cacheTTL),
	}

	s.logger.Debug().
		Str("cache_key", cacheKey).
		Int("route_count", len(resp.Routes)).
		Msg("cached directions response")

	// Periodic cleanup
	s.cleanupIfNeeded()

	return resp, nil
}

// cacheKey generates a cache key for a routing request.
// Uses grid-based quantization for both origin and destination.
// Format: {profile}:{gridOriginLat},{gridOriginLon}:{gridDestLat},{gridDestLon}.
func (s *Service) cacheKey(req DirectionsRequest) string {
	gridOriginLat := math.Floor(req.Origin.Lat/s.cacheGridSize) * s.cacheGridSize
	gridOriginLon := math.Floor(req.Origin.Lon/s.cacheGridSize) * s.cacheGridSize
	gridDestLat := math.Floor(req.Destination.Lat/s.cacheGridSize) * s.cacheGridSize
	gridDestLon := math.Floor(req.Destination.Lon/s.cacheGridSize) * s.cacheGridSize

	return fmt.Sprintf("%s:%.2f,%.2f:%.2f,%.2f",
		req.Profile,
		gridOriginLat, gridOriginLon,
		gridDestLat, gridDestLon,
	)
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
		// Remove entries that are past the stale-if-error window
		if now.After(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
			delete(s.cache, key)
			expired++
		}
	}

	if expired > 0 {
		s.logger.Debug().
			Int("expired_entries", expired).
			Msg("cleaned up expired routing cache entries")
	}
}

// InvalidateCache clears all cached data.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*cachedDirections)
}

// CacheStats returns cache statistics.
func (s *Service) CacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	fresh := 0
	stale := 0

	for _, c := range s.cache {
		if now.Before(c.expiresAt) {
			fresh++
		} else if now.Before(c.fetchedAt.Add(s.staleIfErrorTTL)) {
			stale++
		}
	}

	return CacheStats{
		TotalEntries: len(s.cache),
		FreshEntries: fresh,
		StaleEntries: stale,
		Provider:     s.provider.Name(),
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	TotalEntries int
	FreshEntries int
	StaleEntries int
	Provider     string
}

// ProviderName returns the name of the underlying provider.
func (s *Service) ProviderName() string {
	return s.provider.Name()
}

// validateCoordinates checks if coordinates are within valid ranges.
func validateCoordinates(c Coordinate) error {
	if c.Lat < -90 || c.Lat > 90 {
		return fmt.Errorf("latitude %f out of range [-90, 90]", c.Lat)
	}
	if c.Lon < -180 || c.Lon > 180 {
		return fmt.Errorf("longitude %f out of range [-180, 180]", c.Lon)
	}
	return nil
}
