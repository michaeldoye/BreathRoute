package airquality_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/airquality"
)

// mockProvider is a test provider that returns configurable data.
type mockProvider struct {
	snapshot     *airquality.AQSnapshot
	err          error
	fetchCount   atomic.Int32
	fetchDelay   time.Duration
}

func (m *mockProvider) FetchSnapshot(ctx context.Context) (*airquality.AQSnapshot, error) {
	m.fetchCount.Add(1)
	if m.fetchDelay > 0 {
		select {
		case <-time.After(m.fetchDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshot, nil
}

func (m *mockProvider) FetchStations(_ context.Context) ([]*airquality.Station, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.snapshot.StationList(), nil
}

func (m *mockProvider) FetchLatestMeasurements(_ context.Context) ([]*airquality.Measurement, error) {
	if m.err != nil {
		return nil, m.err
	}
	measurements := make([]*airquality.Measurement, 0, len(m.snapshot.Measurements))
	for _, measurement := range m.snapshot.Measurements {
		measurements = append(measurements, measurement)
	}
	return measurements, nil
}

func testSnapshot() *airquality.AQSnapshot {
	snapshot := airquality.NewAQSnapshot("test")
	snapshot.Stations["NL10001"] = &airquality.Station{
		ID:         "NL10001",
		Name:       "Amsterdam-Centrum",
		Lat:        52.370216,
		Lon:        4.895168,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM25},
		UpdatedAt:  time.Now(),
	}
	snapshot.Stations["NL10002"] = &airquality.Station{
		ID:         "NL10002",
		Name:       "Rotterdam-Noord",
		Lat:        51.9225,
		Lon:        4.47917,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM10},
		UpdatedAt:  time.Now(),
	}
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10001",
		Pollutant:  airquality.PollutantNO2,
		Value:      32.5,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10001",
		Pollutant:  airquality.PollutantPM25,
		Value:      12.3,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10002",
		Pollutant:  airquality.PollutantNO2,
		Value:      28.1,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	return snapshot
}

func TestService_GetSnapshot(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
		CacheTTL: 5 * time.Minute,
	})

	ctx := context.Background()

	// First call should fetch from provider
	snapshot, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(snapshot.Stations))
	assert.Equal(t, int32(1), provider.fetchCount.Load())

	// Second call should use cache
	snapshot2, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, snapshot, snapshot2)
	assert.Equal(t, int32(1), provider.fetchCount.Load()) // Still 1
}

func TestService_GetSnapshot_CacheExpiry(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
		CacheTTL: 50 * time.Millisecond, // Very short TTL for testing
	})

	ctx := context.Background()

	// First call
	_, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(1), provider.fetchCount.Load())

	// Wait for cache to expire
	time.Sleep(60 * time.Millisecond)

	// Should fetch again
	_, err = svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(2), provider.fetchCount.Load())
}

func TestService_GetSnapshot_ProviderError_StaleData(t *testing.T) {
	snapshot := testSnapshot()
	provider := &mockProvider{snapshot: snapshot}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider:        provider,
		Logger:          zerolog.New(io.Discard),
		CacheTTL:        50 * time.Millisecond,
		StaleIfErrorTTL: 1 * time.Hour, // Allow stale data for 1 hour
	})

	ctx := context.Background()

	// Populate cache
	_, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)

	// Wait for cache to expire
	time.Sleep(60 * time.Millisecond)

	// Simulate provider failure
	provider.err = errors.New("provider unavailable")

	// Should return stale data
	result, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(result.Stations))
}

func TestService_GetSnapshot_ProviderError_NoCache(t *testing.T) {
	provider := &mockProvider{err: errors.New("provider unavailable")}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
	})

	ctx := context.Background()

	_, err := svc.GetSnapshot(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrProviderUnavailable)
}

func TestService_GetStations(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
	})

	ctx := context.Background()

	stations, err := svc.GetStations(ctx)
	require.NoError(t, err)
	assert.Len(t, stations, 2)
}

func TestService_GetMeasurement(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
	})

	ctx := context.Background()

	// Existing measurement
	m, err := svc.GetMeasurement(ctx, "NL10001", airquality.PollutantNO2)
	require.NoError(t, err)
	assert.Equal(t, 32.5, m.Value)

	// Non-existent measurement
	_, err = svc.GetMeasurement(ctx, "NL10001", airquality.PollutantO3)
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrNoMeasurements)
}

func TestService_GetStationMeasurements(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
	})

	ctx := context.Background()

	// Station with multiple measurements
	measurements, err := svc.GetStationMeasurements(ctx, "NL10001")
	require.NoError(t, err)
	assert.Len(t, measurements, 2)

	// Non-existent station
	_, err = svc.GetStationMeasurements(ctx, "INVALID")
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrStationNotFound)
}

func TestService_InvalidateCache(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
		CacheTTL: 10 * time.Minute,
	})

	ctx := context.Background()

	// Populate cache
	_, err := svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(1), provider.fetchCount.Load())

	// Invalidate
	svc.InvalidateCache()

	// Should fetch again
	_, err = svc.GetSnapshot(ctx)
	require.NoError(t, err)
	assert.Equal(t, int32(2), provider.fetchCount.Load())
}

func TestService_CacheStatus(t *testing.T) {
	provider := &mockProvider{snapshot: testSnapshot()}
	svc := airquality.NewService(airquality.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.New(io.Discard),
		CacheTTL: 5 * time.Minute,
	})

	// Empty cache
	status := svc.CacheStatus()
	assert.False(t, status.HasData)

	// Populate cache
	ctx := context.Background()
	_, _ = svc.GetSnapshot(ctx)

	status = svc.CacheStatus()
	assert.True(t, status.HasData)
	assert.Equal(t, 2, status.StationCount)
	assert.Equal(t, "test", status.Provider)
	assert.False(t, status.IsExpired)
}
