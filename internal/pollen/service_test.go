package pollen_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/featureflags"
	"github.com/breatheroute/breatheroute/internal/pollen"
)

// mockProvider is a mock pollen provider for testing.
type mockProvider struct {
	mu        sync.Mutex
	callCount int
	data      *pollen.RegionalPollen
	forecast  *pollen.Forecast
	err       error
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		data: &pollen.RegionalPollen{
			Region:     "NL",
			RegionName: "Netherlands",
			Lat:        52.370,
			Lon:        4.895,
			Readings: map[pollen.Type]*pollen.Reading{
				pollen.PollenGrass: {
					Type:  pollen.PollenGrass,
					Index: 2.0,
					Risk:  pollen.RiskModerate,
				},
				pollen.PollenTree: {
					Type:  pollen.PollenTree,
					Index: 1.0,
					Risk:  pollen.RiskLow,
				},
			},
			OverallRisk:  pollen.RiskModerate,
			OverallIndex: 1.5,
			ValidFor:     time.Now(),
			FetchedAt:    time.Now(),
			Provider:     "mock",
		},
		forecast: &pollen.Forecast{
			Region: "NL",
			Daily: []pollen.DailyForecast{
				{
					Date: time.Now().Add(24 * time.Hour),
					Readings: map[pollen.Type]*pollen.Reading{
						pollen.PollenGrass: {
							Type:  pollen.PollenGrass,
							Index: 2.5,
							Risk:  pollen.RiskModerate,
						},
					},
					OverallRisk:  pollen.RiskModerate,
					OverallIndex: 2.5,
				},
			},
			FetchedAt: time.Now(),
		},
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) GetRegionalPollen(_ context.Context, _, _ float64) (*pollen.RegionalPollen, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

func (m *mockProvider) GetForecast(_ context.Context, _, _ float64) (*pollen.Forecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}
	return m.forecast, nil
}

func (m *mockProvider) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func (m *mockProvider) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func TestService_GetRegionalPollen(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	data, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "NL", data.Region)
	assert.Equal(t, pollen.RiskModerate, data.OverallRisk)
	assert.NotNil(t, data.Readings[pollen.PollenGrass])
}

func TestService_GetRegionalPollen_Caching(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Second call should use cache
	_, err = service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Only one provider call
	assert.Equal(t, 1, provider.getCallCount())
}

func TestService_GetRegionalPollen_RegionalCaching(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// Two nearby points in same region (0.5 degree grid)
	_, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	_, err = service.GetRegionalPollen(context.Background(), 52.400, 4.850)
	require.NoError(t, err)

	// Should use same cache entry (same 0.5 degree grid cell)
	assert.Equal(t, 1, provider.getCallCount())

	// Point in different region
	_, err = service.GetRegionalPollen(context.Background(), 53.0, 5.0)
	require.NoError(t, err)

	// Should call provider again
	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_GetRegionalPollen_InvalidCoordinates(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	tests := []struct {
		name string
		lat  float64
		lon  float64
	}{
		{"lat too high", 91.0, 4.895},
		{"lat too low", -91.0, 4.895},
		{"lon too high", 52.370, 181.0},
		{"lon too low", 52.370, -181.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetRegionalPollen(context.Background(), tt.lat, tt.lon)
			require.Error(t, err)
			assert.ErrorIs(t, err, pollen.ErrInvalidCoordinates)
		})
	}
}

func TestService_GetRegionalPollen_ProviderError(t *testing.T) {
	provider := newMockProvider()
	provider.setError(errors.New("api error"))

	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	_, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.ErrorIs(t, err, pollen.ErrProviderUnavailable)
}

func TestService_GetRegionalPollen_StaleOnError(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider:        provider,
		Logger:          zerolog.Nop(),
		CacheTTL:        100 * time.Millisecond,
		StaleIfErrorTTL: 1 * time.Hour,
	})

	// First call succeeds
	data1, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, data1)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Set error on provider
	provider.setError(errors.New("api error"))

	// Second call should return stale data
	data2, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, data2)
}

func TestService_GetRegionalPollen_FeatureFlagDisabled(t *testing.T) {
	provider := newMockProvider()

	// Create feature flags service with pollen disabled
	ffRepo := featureflags.NewInMemoryRepositoryWithFlags(map[string]*featureflags.Flag{
		featureflags.FlagDisablePollenFactor: {
			Key:       featureflags.FlagDisablePollenFactor,
			Value:     true, // Disabled
			UpdatedAt: time.Now(),
		},
	})
	ffService := featureflags.NewService(featureflags.ServiceConfig{
		Repository: ffRepo,
		Logger:     zerolog.Nop(),
	})

	service := pollen.NewService(pollen.ServiceConfig{
		Provider:     provider,
		FeatureFlags: ffService,
		Logger:       zerolog.Nop(),
	})

	// Should return ErrPollenDisabled when disabled
	data, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.ErrorIs(t, err, pollen.ErrPollenDisabled)
	assert.Nil(t, data)

	// Provider should not be called
	assert.Equal(t, 0, provider.getCallCount())
}

func TestService_GetForecast(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	forecast, err := service.GetForecast(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, forecast)

	assert.Equal(t, "NL", forecast.Region)
	assert.Len(t, forecast.Daily, 1)
}

func TestService_GetExposureFactor(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	// Normal case - should return factor based on risk
	factor := service.GetExposureFactor(context.Background(), 52.370, 4.895)
	assert.Equal(t, 1.1, factor) // RiskModerate = 1.1

	// Error case - should return 1.0
	provider.setError(errors.New("api error"))
	factor = service.GetExposureFactor(context.Background(), 53.0, 5.0) // Different location
	assert.Equal(t, 1.0, factor)
}

func TestService_IsEnabled(t *testing.T) {
	provider := newMockProvider()

	t.Run("enabled by default", func(t *testing.T) {
		service := pollen.NewService(pollen.ServiceConfig{
			Provider: provider,
			Logger:   zerolog.Nop(),
		})
		assert.True(t, service.IsEnabled(context.Background()))
	})

	t.Run("disabled via feature flag", func(t *testing.T) {
		ffRepo := featureflags.NewInMemoryRepositoryWithFlags(map[string]*featureflags.Flag{
			featureflags.FlagDisablePollenFactor: {
				Key:       featureflags.FlagDisablePollenFactor,
				Value:     true,
				UpdatedAt: time.Now(),
			},
		})
		ffService := featureflags.NewService(featureflags.ServiceConfig{
			Repository: ffRepo,
			Logger:     zerolog.Nop(),
		})

		service := pollen.NewService(pollen.ServiceConfig{
			Provider:     provider,
			FeatureFlags: ffService,
			Logger:       zerolog.Nop(),
		})
		assert.False(t, service.IsEnabled(context.Background()))
	})
}

func TestService_InvalidateCache(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Invalidate cache
	service.InvalidateCache()

	// Second call should hit provider again
	_, err = service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_CacheStats(t *testing.T) {
	provider := newMockProvider()
	service := pollen.NewService(pollen.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// Empty cache
	stats := service.CacheStats()
	assert.Equal(t, 0, stats.PollenEntries)
	assert.Equal(t, "mock", stats.Provider)

	// Add entries
	_, _ = service.GetRegionalPollen(context.Background(), 52.370, 4.895)
	_, _ = service.GetForecast(context.Background(), 52.370, 4.895)

	stats = service.CacheStats()
	assert.Equal(t, 1, stats.PollenEntries)
	assert.Equal(t, 1, stats.ForecastEntries)
	assert.Equal(t, 1, stats.PollenFreshEntries)
	assert.Equal(t, 1, stats.ForecastFreshEntries)
}
