package transit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Provider defines the interface for transit disruption data providers.
type Provider interface {
	// GetAllDisruptions fetches all current disruptions.
	GetAllDisruptions(ctx context.Context) ([]*Disruption, error)

	// GetDisruptionsForRoute fetches disruptions affecting a specific route.
	GetDisruptionsForRoute(ctx context.Context, origin, destination string) (*RouteDisruptions, error)

	// GetStations fetches the list of stations (for station code lookup).
	GetStations(ctx context.Context) ([]*Station, error)

	// Name returns the provider name for logging.
	Name() string
}

// ServiceConfig holds configuration for the transit service.
type ServiceConfig struct {
	// Provider is the transit data provider.
	Provider Provider

	// Logger for service operations.
	Logger zerolog.Logger

	// CacheTTL is how long to cache disruption data (default: 5 minutes).
	// Disruptions can change quickly, so shorter cache is appropriate.
	CacheTTL time.Duration

	// StationCacheTTL is how long to cache station data (default: 24 hours).
	// Station data rarely changes.
	StationCacheTTL time.Duration

	// StaleIfErrorTTL allows serving stale data on provider errors (default: 30 minutes).
	StaleIfErrorTTL time.Duration
}

// Service provides transit disruption data with caching.
type Service struct {
	provider        Provider
	logger          zerolog.Logger
	cacheTTL        time.Duration
	stationCacheTTL time.Duration
	staleIfErrorTTL time.Duration

	mu              sync.RWMutex
	disruptionCache *cachedDisruptions
	stationCache    *cachedStations
	routeCache      map[string]*cachedRouteDisruptions
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

type cachedDisruptions struct {
	disruptions []*Disruption
	fetchedAt   time.Time
	expiresAt   time.Time
}

type cachedStations struct {
	stations   []*Station
	stationMap map[string]*Station // code -> station
	fetchedAt  time.Time
	expiresAt  time.Time
}

type cachedRouteDisruptions struct {
	data      *RouteDisruptions
	fetchedAt time.Time
	expiresAt time.Time
}

// NewService creates a new transit service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}

	stationCacheTTL := cfg.StationCacheTTL
	if stationCacheTTL == 0 {
		stationCacheTTL = 24 * time.Hour
	}

	staleIfErrorTTL := cfg.StaleIfErrorTTL
	if staleIfErrorTTL == 0 {
		staleIfErrorTTL = 30 * time.Minute
	}

	return &Service{
		provider:        cfg.Provider,
		logger:          cfg.Logger,
		cacheTTL:        cacheTTL,
		stationCacheTTL: stationCacheTTL,
		staleIfErrorTTL: staleIfErrorTTL,
		routeCache:      make(map[string]*cachedRouteDisruptions),
		cleanupInterval: 10 * time.Minute,
	}
}

// GetAllDisruptions returns all current disruptions.
func (s *Service) GetAllDisruptions(ctx context.Context) ([]*Disruption, error) {
	s.mu.RLock()
	if s.disruptionCache != nil && time.Now().Before(s.disruptionCache.expiresAt) {
		disruptions := s.disruptionCache.disruptions
		s.mu.RUnlock()
		return disruptions, nil
	}
	s.mu.RUnlock()

	return s.fetchDisruptions(ctx)
}

// GetActiveDisruptions returns only currently active disruptions.
func (s *Service) GetActiveDisruptions(ctx context.Context) ([]*Disruption, error) {
	all, err := s.GetAllDisruptions(ctx)
	if err != nil {
		return nil, err
	}

	active := make([]*Disruption, 0, len(all))
	for _, d := range all {
		if d.IsActive() {
			active = append(active, d)
		}
	}

	return active, nil
}

// GetDisruptionsForRoute returns disruptions affecting a specific route.
func (s *Service) GetDisruptionsForRoute(ctx context.Context, origin, destination string) (*RouteDisruptions, error) {
	cacheKey := s.routeCacheKey(origin, destination)

	s.mu.RLock()
	if cached, ok := s.routeCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		return cached.data, nil
	}
	s.mu.RUnlock()

	return s.fetchRouteDisruptions(ctx, origin, destination, cacheKey)
}

// GetDisruptionSummary returns a summary of current disruptions.
func (s *Service) GetDisruptionSummary(ctx context.Context) (*DisruptionSummary, error) {
	disruptions, err := s.GetActiveDisruptions(ctx)
	if err != nil {
		return nil, err
	}

	summary := &DisruptionSummary{
		TotalDisruptions: len(disruptions),
		ByImpact:         make(map[Impact]int),
		ByType:           make(map[DisruptionType]int),
		FetchedAt:        time.Now(),
		Provider:         s.provider.Name(),
	}

	for _, d := range disruptions {
		summary.ByImpact[d.Impact]++
		summary.ByType[d.Type]++
	}

	summary.MostSevere = CalculateOverallImpact(disruptions)

	return summary, nil
}

// GetStation returns station info by code.
func (s *Service) GetStation(ctx context.Context, code string) (*Station, error) {
	s.mu.RLock()
	if s.stationCache != nil && time.Now().Before(s.stationCache.expiresAt) {
		if station, ok := s.stationCache.stationMap[code]; ok {
			s.mu.RUnlock()
			return station, nil
		}
		s.mu.RUnlock()
		return nil, fmt.Errorf("station not found: %s", code)
	}
	s.mu.RUnlock()

	// Refresh stations cache
	if _, err := s.fetchStations(ctx); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if station, ok := s.stationCache.stationMap[code]; ok {
		return station, nil
	}
	return nil, fmt.Errorf("station not found: %s", code)
}

// fetchDisruptions fetches from provider and updates cache.
func (s *Service) fetchDisruptions(ctx context.Context) ([]*Disruption, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if s.disruptionCache != nil && time.Now().Before(s.disruptionCache.expiresAt) {
		return s.disruptionCache.disruptions, nil
	}

	s.logger.Debug().
		Str("provider", s.provider.Name()).
		Msg("fetching disruptions from provider")

	disruptions, err := s.provider.GetAllDisruptions(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch disruptions")

		// Check for stale data
		if s.disruptionCache != nil {
			if time.Now().Before(s.disruptionCache.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", s.disruptionCache.fetchedAt).
					Msg("serving stale disruption data due to provider error")
				return s.disruptionCache.disruptions, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.disruptionCache = &cachedDisruptions{
		disruptions: disruptions,
		fetchedAt:   now,
		expiresAt:   now.Add(s.cacheTTL),
	}

	s.logger.Info().
		Int("disruptions", len(disruptions)).
		Msg("disruptions cache refreshed")

	return disruptions, nil
}

// fetchRouteDisruptions fetches route-specific disruptions.
func (s *Service) fetchRouteDisruptions(ctx context.Context, origin, destination, cacheKey string) (*RouteDisruptions, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if cached, ok := s.routeCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		return cached.data, nil
	}

	s.logger.Debug().
		Str("origin", origin).
		Str("destination", destination).
		Str("provider", s.provider.Name()).
		Msg("fetching route disruptions from provider")

	data, err := s.provider.GetDisruptionsForRoute(ctx, origin, destination)
	if err != nil {
		s.logger.Error().Err(err).
			Str("origin", origin).
			Str("destination", destination).
			Msg("failed to fetch route disruptions")

		// Check for stale data
		if cached, ok := s.routeCache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Msg("serving stale route disruption data due to provider error")
				return cached.data, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.routeCache[cacheKey] = &cachedRouteDisruptions{
		data:      data,
		fetchedAt: now,
		expiresAt: now.Add(s.cacheTTL),
	}

	// Periodic cleanup
	s.cleanupIfNeeded()

	return data, nil
}

// fetchStations fetches station list from provider.
func (s *Service) fetchStations(ctx context.Context) ([]*Station, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if s.stationCache != nil && time.Now().Before(s.stationCache.expiresAt) {
		return s.stationCache.stations, nil
	}

	s.logger.Debug().
		Str("provider", s.provider.Name()).
		Msg("fetching stations from provider")

	stations, err := s.provider.GetStations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch stations")

		// Check for stale data
		if s.stationCache != nil {
			if time.Now().Before(s.stationCache.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", s.stationCache.fetchedAt).
					Msg("serving stale station data due to provider error")
				return s.stationCache.stations, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Build station map
	stationMap := make(map[string]*Station, len(stations))
	for _, s := range stations {
		stationMap[s.Code] = s
	}

	// Update cache
	now := time.Now()
	s.stationCache = &cachedStations{
		stations:   stations,
		stationMap: stationMap,
		fetchedAt:  now,
		expiresAt:  now.Add(s.stationCacheTTL),
	}

	s.logger.Info().
		Int("stations", len(stations)).
		Msg("stations cache refreshed")

	return stations, nil
}

// routeCacheKey generates a cache key for a route.
func (s *Service) routeCacheKey(origin, destination string) string {
	return origin + ":" + destination
}

// cleanupIfNeeded removes expired route cache entries.
func (s *Service) cleanupIfNeeded() {
	now := time.Now()
	if now.Sub(s.lastCleanup) < s.cleanupInterval {
		return
	}

	s.lastCleanup = now
	expired := 0

	for key, cached := range s.routeCache {
		if now.After(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
			delete(s.routeCache, key)
			expired++
		}
	}

	if expired > 0 {
		s.logger.Debug().
			Int("expired_entries", expired).
			Msg("cleaned up expired route cache entries")
	}
}

// InvalidateCache clears all cached data.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.disruptionCache = nil
	s.stationCache = nil
	s.routeCache = make(map[string]*cachedRouteDisruptions)
}

// CacheStats returns cache statistics.
func (s *Service) CacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	stats := CacheStats{
		Provider:          s.provider.Name(),
		RouteCacheEntries: len(s.routeCache),
	}

	if s.disruptionCache != nil {
		stats.HasDisruptionCache = true
		stats.DisruptionCacheFresh = now.Before(s.disruptionCache.expiresAt)
		stats.DisruptionCount = len(s.disruptionCache.disruptions)
	}

	if s.stationCache != nil {
		stats.HasStationCache = true
		stats.StationCacheFresh = now.Before(s.stationCache.expiresAt)
		stats.StationCount = len(s.stationCache.stations)
	}

	return stats
}

// CacheStats contains cache statistics.
type CacheStats struct {
	Provider             string
	HasDisruptionCache   bool
	DisruptionCacheFresh bool
	DisruptionCount      int
	HasStationCache      bool
	StationCacheFresh    bool
	StationCount         int
	RouteCacheEntries    int
}
