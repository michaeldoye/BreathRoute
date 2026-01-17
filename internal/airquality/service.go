package airquality

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Provider defines the interface for air quality data providers.
type Provider interface {
	// FetchSnapshot fetches a complete snapshot of stations and measurements.
	FetchSnapshot(ctx context.Context) (*AQSnapshot, error)

	// FetchStations fetches just the station metadata.
	FetchStations(ctx context.Context) ([]*Station, error)

	// FetchLatestMeasurements fetches the latest measurements.
	FetchLatestMeasurements(ctx context.Context) ([]*Measurement, error)
}

// ServiceConfig holds configuration for the air quality service.
type ServiceConfig struct {
	// Provider is the air quality data provider.
	Provider Provider

	// Logger for service operations.
	Logger zerolog.Logger

	// CacheTTL is how long to cache the snapshot (default: 5 minutes).
	CacheTTL time.Duration

	// StaleIfErrorTTL allows serving stale data on provider errors (default: 30 minutes).
	StaleIfErrorTTL time.Duration
}

// Service provides air quality data with caching.
type Service struct {
	provider        Provider
	logger          zerolog.Logger
	cacheTTL        time.Duration
	staleIfErrorTTL time.Duration

	mu          sync.RWMutex
	snapshot    *AQSnapshot
	cacheExpiry time.Time
}

// NewService creates a new air quality service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 5 * time.Minute
	}

	staleIfErrorTTL := cfg.StaleIfErrorTTL
	if staleIfErrorTTL == 0 {
		staleIfErrorTTL = 30 * time.Minute
	}

	return &Service{
		provider:        cfg.Provider,
		logger:          cfg.Logger,
		cacheTTL:        cacheTTL,
		staleIfErrorTTL: staleIfErrorTTL,
	}
}

// GetSnapshot returns the current air quality snapshot.
// It uses a cached version if available and not expired.
func (s *Service) GetSnapshot(ctx context.Context) (*AQSnapshot, error) {
	// Check for fresh cache
	s.mu.RLock()
	if s.snapshot != nil && time.Now().Before(s.cacheExpiry) {
		snapshot := s.snapshot
		s.mu.RUnlock()
		return snapshot, nil
	}
	s.mu.RUnlock()

	// Need to refresh
	return s.refreshSnapshot(ctx)
}

// GetStations returns all monitoring stations.
func (s *Service) GetStations(ctx context.Context) ([]*Station, error) {
	snapshot, err := s.GetSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return snapshot.StationList(), nil
}

// GetMeasurement retrieves a specific measurement.
func (s *Service) GetMeasurement(ctx context.Context, stationID string, pollutant Pollutant) (*Measurement, error) {
	snapshot, err := s.GetSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	m := snapshot.GetMeasurement(stationID, pollutant)
	if m == nil {
		return nil, ErrNoMeasurements
	}
	return m, nil
}

// GetStationMeasurements retrieves all measurements for a station.
func (s *Service) GetStationMeasurements(ctx context.Context, stationID string) ([]*Measurement, error) {
	snapshot, err := s.GetSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	if _, ok := snapshot.Stations[stationID]; !ok {
		return nil, ErrStationNotFound
	}

	return snapshot.GetStationMeasurements(stationID), nil
}

// RefreshSnapshot forces a cache refresh.
func (s *Service) RefreshSnapshot(ctx context.Context) error {
	_, err := s.refreshSnapshot(ctx)
	return err
}

// InvalidateCache clears the cached snapshot.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshot = nil
	s.cacheExpiry = time.Time{}
}

// CacheStatus returns information about the current cache state.
func (s *Service) CacheStatus() CacheStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.snapshot == nil {
		return CacheStatus{
			HasData: false,
		}
	}

	now := time.Now()
	return CacheStatus{
		HasData:      true,
		FetchedAt:    s.snapshot.FetchedAt,
		ExpiresAt:    s.cacheExpiry,
		IsExpired:    now.After(s.cacheExpiry),
		IsStale:      now.After(s.snapshot.FetchedAt.Add(s.staleIfErrorTTL)),
		StationCount: len(s.snapshot.Stations),
		Provider:     s.snapshot.Provider,
	}
}

// CacheStatus represents the current state of the cache.
type CacheStatus struct {
	HasData      bool
	FetchedAt    time.Time
	ExpiresAt    time.Time
	IsExpired    bool
	IsStale      bool
	StationCount int
	Provider     string
}

// refreshSnapshot fetches fresh data from the provider.
func (s *Service) refreshSnapshot(ctx context.Context) (*AQSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check: another goroutine might have refreshed while we waited
	if s.snapshot != nil && time.Now().Before(s.cacheExpiry) {
		return s.snapshot, nil
	}

	s.logger.Debug().Msg("refreshing air quality snapshot")

	snapshot, err := s.provider.FetchSnapshot(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch air quality snapshot")

		// If we have stale data that's not too old, return it
		if s.snapshot != nil && time.Now().Before(s.snapshot.FetchedAt.Add(s.staleIfErrorTTL)) {
			s.logger.Warn().
				Time("fetched_at", s.snapshot.FetchedAt).
				Msg("serving stale air quality data due to provider error")
			return s.snapshot, nil
		}

		return nil, ErrProviderUnavailable
	}

	s.snapshot = snapshot
	s.cacheExpiry = time.Now().Add(s.cacheTTL)

	s.logger.Info().
		Int("stations", len(snapshot.Stations)).
		Int("measurements", len(snapshot.Measurements)).
		Time("expires_at", s.cacheExpiry).
		Msg("air quality snapshot refreshed")

	return snapshot, nil
}
