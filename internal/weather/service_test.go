package weather_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/weather"
)

// mockProvider is a mock weather provider for testing.
type mockProvider struct {
	mu           sync.Mutex
	callCount    int
	observations map[string]*weather.Observation
	forecasts    map[string]*weather.Forecast
	err          error
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		observations: make(map[string]*weather.Observation),
		forecasts:    make(map[string]*weather.Forecast),
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) GetCurrentWeather(_ context.Context, lat, lon float64) (*weather.Observation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}

	key := cacheKey(lat, lon)
	if obs, ok := m.observations[key]; ok {
		return obs, nil
	}

	// Return default observation
	return &weather.Observation{
		Lat:           lat,
		Lon:           lon,
		Temperature:   20.0,
		Humidity:      65.0,
		WindSpeed:     5.0,
		WindDirection: 180.0,
		Condition:     weather.ConditionClear,
		ObservedAt:    time.Now(),
		FetchedAt:     time.Now(),
	}, nil
}

func (m *mockProvider) GetForecast(_ context.Context, lat, lon float64) (*weather.Forecast, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}

	key := cacheKey(lat, lon)
	if f, ok := m.forecasts[key]; ok {
		return f, nil
	}

	// Return default forecast
	return &weather.Forecast{
		Lat: lat,
		Lon: lon,
		Hourly: []weather.HourlyForecast{
			{
				Time:          time.Now().Add(1 * time.Hour),
				Temperature:   21.0,
				WindSpeed:     4.0,
				WindDirection: 180.0,
				Condition:     weather.ConditionClear,
			},
		},
		FetchedAt: time.Now(),
	}, nil
}

func cacheKey(lat, lon float64) string {
	return string(rune(int(lat*10))) + ":" + string(rune(int(lon*10)))
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

func TestService_GetCurrentWeather(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 5 * time.Minute,
	})

	obs, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, obs)

	assert.Equal(t, 52.370, obs.Lat)
	assert.Equal(t, 4.895, obs.Lon)
	assert.Equal(t, 20.0, obs.Temperature)
	assert.Equal(t, weather.ConditionClear, obs.Condition)
}

func TestService_GetCurrentWeather_Caching(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 5 * time.Minute,
	})

	// First call
	_, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Second call should use cache
	_, err = service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Only one provider call (cached)
	assert.Equal(t, 1, provider.getCallCount())
}

func TestService_GetCurrentWeather_CacheGriding(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider:      provider,
		Logger:        zerolog.Nop(),
		CacheTTL:      5 * time.Minute,
		CacheGridSize: 0.1, // ~11km grid
	})

	// Two nearby points in same grid cell
	_, err := service.GetCurrentWeather(context.Background(), 52.371, 4.891)
	require.NoError(t, err)

	_, err = service.GetCurrentWeather(context.Background(), 52.375, 4.895)
	require.NoError(t, err)

	// Should only call provider once (same grid cell)
	assert.Equal(t, 1, provider.getCallCount())

	// Point in different grid cell
	_, err = service.GetCurrentWeather(context.Background(), 52.5, 4.9)
	require.NoError(t, err)

	// Should call provider again
	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_GetCurrentWeather_InvalidCoordinates(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
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
			_, err := service.GetCurrentWeather(context.Background(), tt.lat, tt.lon)
			require.Error(t, err)
			assert.ErrorIs(t, err, weather.ErrInvalidCoordinates)
		})
	}
}

func TestService_GetCurrentWeather_ProviderError(t *testing.T) {
	provider := newMockProvider()
	provider.setError(errors.New("api error"))

	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	_, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.ErrorIs(t, err, weather.ErrProviderUnavailable)
}

func TestService_GetCurrentWeather_StaleOnError(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider:        provider,
		Logger:          zerolog.Nop(),
		CacheTTL:        100 * time.Millisecond,
		StaleIfErrorTTL: 1 * time.Hour,
	})

	// First call succeeds
	obs1, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, obs1)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Set error on provider
	provider.setError(errors.New("api error"))

	// Second call should return stale data
	obs2, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, obs2)
}

func TestService_GetForecast(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	forecast, err := service.GetForecast(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, forecast)

	assert.Equal(t, 52.370, forecast.Lat)
	assert.Len(t, forecast.Hourly, 1)
	assert.Equal(t, 21.0, forecast.Hourly[0].Temperature)
}

func TestService_GetWeatherForPoints(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	points := []struct{ Lat, Lon float64 }{
		{52.370, 4.895},
		{52.375, 4.850},
		{52.365, 4.940},
	}

	results, err := service.GetWeatherForPoints(context.Background(), points)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	for i, obs := range results {
		require.NotNil(t, obs, "observation %d should not be nil", i)
	}
}

func TestService_GetWeatherForBoundingBox(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	box := weather.BoundingBox{
		MinLat: 52.3,
		MaxLat: 52.4,
		MinLon: 4.8,
		MaxLon: 5.0,
	}

	obs, err := service.GetWeatherForBoundingBox(context.Background(), box)
	require.NoError(t, err)
	require.NotNil(t, obs)

	// Should fetch for center point
	centerLat, centerLon := box.Center()
	assert.InDelta(t, centerLat, obs.Lat, 0.1)
	assert.InDelta(t, centerLon, obs.Lon, 0.1)
}

func TestService_InvalidateCache(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 5 * time.Minute,
	})

	// First call
	_, err := service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	// Invalidate cache
	service.InvalidateCache()

	// Second call should hit provider again
	_, err = service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)

	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_CacheStats(t *testing.T) {
	provider := newMockProvider()
	service := weather.NewService(weather.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 5 * time.Minute,
	})

	// Empty cache
	stats := service.CacheStats()
	assert.Equal(t, 0, stats.WeatherEntries)
	assert.Equal(t, "mock", stats.Provider)

	// Add some entries
	_, _ = service.GetCurrentWeather(context.Background(), 52.370, 4.895)
	_, _ = service.GetForecast(context.Background(), 52.370, 4.895)

	stats = service.CacheStats()
	assert.Equal(t, 1, stats.WeatherEntries)
	assert.Equal(t, 1, stats.ForecastEntries)
	assert.Equal(t, 1, stats.WeatherFreshEntries)
	assert.Equal(t, 1, stats.ForecastFreshEntries)
}
