package weather

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Provider defines the interface for weather data providers.
type Provider interface {
	// GetCurrentWeather fetches current weather for a location.
	GetCurrentWeather(ctx context.Context, lat, lon float64) (*Observation, error)

	// GetForecast fetches hourly forecast for a location.
	GetForecast(ctx context.Context, lat, lon float64) (*Forecast, error)

	// Name returns the provider name for logging.
	Name() string
}

// ServiceConfig holds configuration for the weather service.
type ServiceConfig struct {
	// Provider is the weather data provider.
	Provider Provider

	// Logger for service operations.
	Logger zerolog.Logger

	// CacheTTL is how long to cache weather data (default: 10 minutes).
	// Weather changes slower than AQ data, so longer cache is acceptable.
	CacheTTL time.Duration

	// CacheGridSize is the size of cache grid cells in degrees (default: 0.1).
	// Points within the same grid cell share cached data.
	CacheGridSize float64

	// StaleIfErrorTTL allows serving stale data on provider errors (default: 1 hour).
	StaleIfErrorTTL time.Duration
}

// Service provides weather data with caching.
type Service struct {
	provider        Provider
	logger          zerolog.Logger
	cacheTTL        time.Duration
	cacheGridSize   float64
	staleIfErrorTTL time.Duration

	mu              sync.RWMutex
	weatherCache    map[string]*cachedObservation
	forecastCache   map[string]*cachedForecast
	lastCleanup     time.Time
	cleanupInterval time.Duration
}

type cachedObservation struct {
	observation *Observation
	fetchedAt   time.Time
	expiresAt   time.Time
}

type cachedForecast struct {
	forecast  *Forecast
	fetchedAt time.Time
	expiresAt time.Time
}

// NewService creates a new weather service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 10 * time.Minute
	}

	cacheGridSize := cfg.CacheGridSize
	if cacheGridSize == 0 {
		cacheGridSize = 0.1 // ~11km at equator
	}

	staleIfErrorTTL := cfg.StaleIfErrorTTL
	if staleIfErrorTTL == 0 {
		staleIfErrorTTL = 1 * time.Hour
	}

	return &Service{
		provider:        cfg.Provider,
		logger:          cfg.Logger,
		cacheTTL:        cacheTTL,
		cacheGridSize:   cacheGridSize,
		staleIfErrorTTL: staleIfErrorTTL,
		weatherCache:    make(map[string]*cachedObservation),
		forecastCache:   make(map[string]*cachedForecast),
		cleanupInterval: 5 * time.Minute,
	}
}

// GetCurrentWeather returns current weather for a location.
// Uses cached data if available and not expired.
func (s *Service) GetCurrentWeather(ctx context.Context, lat, lon float64) (*Observation, error) {
	if err := validateCoordinates(lat, lon); err != nil {
		return nil, err
	}

	cacheKey := s.cacheKey(lat, lon)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.weatherCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		return cached.observation, nil
	}
	s.mu.RUnlock()

	// Fetch from provider
	return s.fetchWeather(ctx, lat, lon, cacheKey)
}

// GetForecast returns hourly forecast for a location.
func (s *Service) GetForecast(ctx context.Context, lat, lon float64) (*Forecast, error) {
	if err := validateCoordinates(lat, lon); err != nil {
		return nil, err
	}

	cacheKey := s.cacheKey(lat, lon)

	// Check cache
	s.mu.RLock()
	if cached, ok := s.forecastCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		s.mu.RUnlock()
		return cached.forecast, nil
	}
	s.mu.RUnlock()

	// Fetch from provider
	return s.fetchForecast(ctx, lat, lon, cacheKey)
}

// GetWeatherForPoints returns current weather for multiple points.
// Uses caching to minimize provider calls.
func (s *Service) GetWeatherForPoints(ctx context.Context, points []struct{ Lat, Lon float64 }) ([]*Observation, error) {
	results := make([]*Observation, len(points))

	for i, p := range points {
		obs, err := s.GetCurrentWeather(ctx, p.Lat, p.Lon)
		if err != nil {
			s.logger.Warn().
				Float64("lat", p.Lat).
				Float64("lon", p.Lon).
				Err(err).
				Msg("failed to get weather for point")
			// Continue with nil for failed points
			continue
		}
		results[i] = obs
	}

	return results, nil
}

// GetWeatherForBoundingBox returns weather for a bounding box.
// Samples the center point of the box for simplicity.
func (s *Service) GetWeatherForBoundingBox(ctx context.Context, box BoundingBox) (*Observation, error) {
	centerLat, centerLon := box.Center()
	return s.GetCurrentWeather(ctx, centerLat, centerLon)
}

// fetchWeather fetches weather from provider and updates cache.
func (s *Service) fetchWeather(ctx context.Context, lat, lon float64, cacheKey string) (*Observation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if cached, ok := s.weatherCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		return cached.observation, nil
	}

	s.logger.Debug().
		Float64("lat", lat).
		Float64("lon", lon).
		Str("provider", s.provider.Name()).
		Msg("fetching weather from provider")

	obs, err := s.provider.GetCurrentWeather(ctx, lat, lon)
	if err != nil {
		s.logger.Error().Err(err).
			Float64("lat", lat).
			Float64("lon", lon).
			Msg("failed to fetch weather")

		// Check for stale data
		if cached, ok := s.weatherCache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Msg("serving stale weather data due to provider error")
				return cached.observation, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.weatherCache[cacheKey] = &cachedObservation{
		observation: obs,
		fetchedAt:   now,
		expiresAt:   now.Add(s.cacheTTL),
	}

	// Periodic cleanup
	s.cleanupIfNeeded()

	return obs, nil
}

// fetchForecast fetches forecast from provider and updates cache.
func (s *Service) fetchForecast(ctx context.Context, lat, lon float64, cacheKey string) (*Forecast, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check cache
	if cached, ok := s.forecastCache[cacheKey]; ok && time.Now().Before(cached.expiresAt) {
		return cached.forecast, nil
	}

	s.logger.Debug().
		Float64("lat", lat).
		Float64("lon", lon).
		Str("provider", s.provider.Name()).
		Msg("fetching forecast from provider")

	forecast, err := s.provider.GetForecast(ctx, lat, lon)
	if err != nil {
		s.logger.Error().Err(err).
			Float64("lat", lat).
			Float64("lon", lon).
			Msg("failed to fetch forecast")

		// Check for stale data
		if cached, ok := s.forecastCache[cacheKey]; ok {
			if time.Now().Before(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
				s.logger.Warn().
					Time("fetched_at", cached.fetchedAt).
					Msg("serving stale forecast data due to provider error")
				return cached.forecast, nil
			}
		}

		return nil, ErrProviderUnavailable
	}

	// Update cache
	now := time.Now()
	s.forecastCache[cacheKey] = &cachedForecast{
		forecast:  forecast,
		fetchedAt: now,
		expiresAt: now.Add(s.cacheTTL),
	}

	// Periodic cleanup
	s.cleanupIfNeeded()

	return forecast, nil
}

// cacheKey generates a cache key for a location.
// Groups nearby points into grid cells to reduce API calls.
func (s *Service) cacheKey(lat, lon float64) string {
	// Round to grid cell
	gridLat := math.Floor(lat/s.cacheGridSize) * s.cacheGridSize
	gridLon := math.Floor(lon/s.cacheGridSize) * s.cacheGridSize
	return fmt.Sprintf("%.2f:%.2f", gridLat, gridLon)
}

// cleanupIfNeeded removes expired entries if cleanup interval has passed.
func (s *Service) cleanupIfNeeded() {
	now := time.Now()
	if now.Sub(s.lastCleanup) < s.cleanupInterval {
		return
	}

	s.lastCleanup = now
	expired := 0

	for key, cached := range s.weatherCache {
		if now.After(cached.fetchedAt.Add(s.staleIfErrorTTL)) {
			delete(s.weatherCache, key)
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
			Msg("cleaned up expired weather cache entries")
	}
}

// InvalidateCache clears all cached data.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weatherCache = make(map[string]*cachedObservation)
	s.forecastCache = make(map[string]*cachedForecast)
}

// CacheStats returns cache statistics.
func (s *Service) CacheStats() CacheStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	weatherFresh := 0
	forecastFresh := 0

	for _, c := range s.weatherCache {
		if now.Before(c.expiresAt) {
			weatherFresh++
		}
	}
	for _, c := range s.forecastCache {
		if now.Before(c.expiresAt) {
			forecastFresh++
		}
	}

	return CacheStats{
		WeatherEntries:       len(s.weatherCache),
		WeatherFreshEntries:  weatherFresh,
		ForecastEntries:      len(s.forecastCache),
		ForecastFreshEntries: forecastFresh,
		Provider:             s.provider.Name(),
	}
}

// CacheStats contains cache statistics.
type CacheStats struct {
	WeatherEntries       int
	WeatherFreshEntries  int
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
