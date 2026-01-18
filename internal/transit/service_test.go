package transit_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/transit"
)

// mockProvider is a mock transit provider for testing.
type mockProvider struct {
	mu          sync.Mutex
	callCount   int
	disruptions []*transit.Disruption
	stations    []*transit.Station
	routeData   *transit.RouteDisruptions
	err         error
	disruptErr  error
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		disruptions: []*transit.Disruption{
			{
				ID:               "d1",
				Type:             transit.DisruptionMaintenance,
				Title:            "Track maintenance",
				Description:      "Planned maintenance work",
				Impact:           transit.ImpactModerate,
				AffectedStations: []string{"ASD", "UT"},
				AffectedRoutes:   []string{"Amsterdam - Utrecht"},
				Start:            time.Now().Add(-1 * time.Hour),
				End:              time.Now().Add(2 * time.Hour),
				IsPlanned:        true,
				Provider:         "mock",
			},
			{
				ID:               "d2",
				Type:             transit.DisruptionDisturbance,
				Title:            "Signal failure",
				Description:      "Unexpected signal failure",
				Impact:           transit.ImpactMajor,
				AffectedStations: []string{"RTD"},
				AffectedRoutes:   []string{"Rotterdam - Den Haag"},
				Start:            time.Now().Add(-30 * time.Minute),
				IsPlanned:        false,
				Provider:         "mock",
			},
		},
		stations: []*transit.Station{
			{Code: "ASD", Name: "Amsterdam Centraal", Lat: 52.378901, Lon: 4.900272, Country: "NL"},
			{Code: "UT", Name: "Utrecht Centraal", Lat: 52.089444, Lon: 5.110278, Country: "NL"},
			{Code: "RTD", Name: "Rotterdam Centraal", Lat: 51.924419, Lon: 4.469756, Country: "NL"},
		},
	}
}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) GetAllDisruptions(_ context.Context) ([]*transit.Disruption, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.disruptErr != nil {
		return nil, m.disruptErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.disruptions, nil
}

func (m *mockProvider) GetDisruptionsForRoute(_ context.Context, origin, destination string) (*transit.RouteDisruptions, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}

	if m.routeData != nil {
		return m.routeData, nil
	}

	// Filter disruptions for route
	relevant := make([]*transit.Disruption, 0)
	for _, d := range m.disruptions {
		if d.AffectsStation(origin) || d.AffectsStation(destination) {
			relevant = append(relevant, d)
		}
	}

	return &transit.RouteDisruptions{
		Origin:         origin,
		Destination:    destination,
		Disruptions:    relevant,
		HasDisruptions: len(relevant) > 0,
		OverallImpact:  transit.CalculateOverallImpact(relevant),
		FetchedAt:      time.Now(),
	}, nil
}

func (m *mockProvider) GetStations(_ context.Context) ([]*transit.Station, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount++

	if m.err != nil {
		return nil, m.err
	}
	return m.stations, nil
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

func (m *mockProvider) setDisruptionError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disruptErr = err
}

func TestService_GetAllDisruptions(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	disruptions, err := service.GetAllDisruptions(context.Background())
	require.NoError(t, err)
	require.Len(t, disruptions, 2)

	assert.Equal(t, "d1", disruptions[0].ID)
	assert.Equal(t, transit.DisruptionMaintenance, disruptions[0].Type)
}

func TestService_GetAllDisruptions_Caching(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetAllDisruptions(context.Background())
	require.NoError(t, err)

	// Second call should use cache
	_, err = service.GetAllDisruptions(context.Background())
	require.NoError(t, err)

	// Only one provider call
	assert.Equal(t, 1, provider.getCallCount())
}

func TestService_GetActiveDisruptions(t *testing.T) {
	provider := newMockProvider()
	// Add an inactive (future) disruption
	provider.disruptions = append(provider.disruptions, &transit.Disruption{
		ID:       "d3",
		Type:     transit.DisruptionConstruction,
		Title:    "Future construction",
		Start:    time.Now().Add(24 * time.Hour),
		End:      time.Now().Add(48 * time.Hour),
		Provider: "mock",
	})

	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	active, err := service.GetActiveDisruptions(context.Background())
	require.NoError(t, err)

	// Only the two active disruptions should be returned
	assert.Len(t, active, 2)
}

func TestService_GetDisruptionsForRoute(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	result, err := service.GetDisruptionsForRoute(context.Background(), "ASD", "UT")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "ASD", result.Origin)
	assert.Equal(t, "UT", result.Destination)
	assert.True(t, result.HasDisruptions)
	assert.Len(t, result.Disruptions, 1) // Only d1 affects ASD-UT
}

func TestService_GetDisruptionsForRoute_Caching(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetDisruptionsForRoute(context.Background(), "ASD", "UT")
	require.NoError(t, err)

	// Second call should use cache
	_, err = service.GetDisruptionsForRoute(context.Background(), "ASD", "UT")
	require.NoError(t, err)

	// Only one provider call
	assert.Equal(t, 1, provider.getCallCount())

	// Different route should call provider again
	_, err = service.GetDisruptionsForRoute(context.Background(), "RTD", "DH")
	require.NoError(t, err)
	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_GetDisruptionSummary(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	summary, err := service.GetDisruptionSummary(context.Background())
	require.NoError(t, err)
	require.NotNil(t, summary)

	assert.Equal(t, 2, summary.TotalDisruptions)
	assert.Equal(t, transit.ImpactMajor, summary.MostSevere)
	assert.Equal(t, 1, summary.ByImpact[transit.ImpactModerate])
	assert.Equal(t, 1, summary.ByImpact[transit.ImpactMajor])
	assert.Equal(t, 1, summary.ByType[transit.DisruptionMaintenance])
	assert.Equal(t, 1, summary.ByType[transit.DisruptionDisturbance])
}

func TestService_GetStation(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	station, err := service.GetStation(context.Background(), "ASD")
	require.NoError(t, err)
	require.NotNil(t, station)

	assert.Equal(t, "ASD", station.Code)
	assert.Equal(t, "Amsterdam Centraal", station.Name)
	assert.Equal(t, "NL", station.Country)
}

func TestService_GetStation_NotFound(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	_, err := service.GetStation(context.Background(), "XXX")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "station not found")
}

func TestService_GetStation_Caching(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider:        provider,
		Logger:          zerolog.Nop(),
		StationCacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetStation(context.Background(), "ASD")
	require.NoError(t, err)

	// Second call should use cache
	_, err = service.GetStation(context.Background(), "UT")
	require.NoError(t, err)

	// Only one provider call (stations are fetched once)
	assert.Equal(t, 1, provider.getCallCount())
}

func TestService_ProviderError(t *testing.T) {
	provider := newMockProvider()
	provider.setError(errors.New("api error"))

	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
	})

	_, err := service.GetAllDisruptions(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, transit.ErrProviderUnavailable)
}

func TestService_StaleOnError(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider:        provider,
		Logger:          zerolog.Nop(),
		CacheTTL:        100 * time.Millisecond,
		StaleIfErrorTTL: 1 * time.Hour,
	})

	// First call succeeds
	data1, err := service.GetAllDisruptions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, data1)

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Set error on provider
	provider.setDisruptionError(errors.New("api error"))

	// Second call should return stale data
	data2, err := service.GetAllDisruptions(context.Background())
	require.NoError(t, err)
	require.NotNil(t, data2)
	assert.Len(t, data2, 2)
}

func TestService_InvalidateCache(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// First call
	_, err := service.GetAllDisruptions(context.Background())
	require.NoError(t, err)

	// Invalidate cache
	service.InvalidateCache()

	// Second call should hit provider again
	_, err = service.GetAllDisruptions(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, provider.getCallCount())
}

func TestService_CacheStats(t *testing.T) {
	provider := newMockProvider()
	service := transit.NewService(transit.ServiceConfig{
		Provider: provider,
		Logger:   zerolog.Nop(),
		CacheTTL: 1 * time.Hour,
	})

	// Empty cache
	stats := service.CacheStats()
	assert.Equal(t, 0, stats.RouteCacheEntries)
	assert.False(t, stats.HasDisruptionCache)
	assert.Equal(t, "mock", stats.Provider)

	// Add entries
	_, _ = service.GetAllDisruptions(context.Background())
	_, _ = service.GetStation(context.Background(), "ASD")
	_, _ = service.GetDisruptionsForRoute(context.Background(), "ASD", "UT")

	stats = service.CacheStats()
	assert.True(t, stats.HasDisruptionCache)
	assert.True(t, stats.DisruptionCacheFresh)
	assert.Equal(t, 2, stats.DisruptionCount)
	assert.True(t, stats.HasStationCache)
	assert.Equal(t, 3, stats.StationCount)
	assert.Equal(t, 1, stats.RouteCacheEntries)
}

func TestDisruption_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected bool
	}{
		{
			name:     "active - started, no end",
			start:    time.Now().Add(-1 * time.Hour),
			end:      time.Time{},
			expected: true,
		},
		{
			name:     "active - started, end in future",
			start:    time.Now().Add(-1 * time.Hour),
			end:      time.Now().Add(1 * time.Hour),
			expected: true,
		},
		{
			name:     "inactive - not started",
			start:    time.Now().Add(1 * time.Hour),
			end:      time.Now().Add(2 * time.Hour),
			expected: false,
		},
		{
			name:     "inactive - ended",
			start:    time.Now().Add(-2 * time.Hour),
			end:      time.Now().Add(-1 * time.Hour),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &transit.Disruption{
				Start: tt.start,
				End:   tt.end,
			}
			assert.Equal(t, tt.expected, d.IsActive())
		})
	}
}

func TestDisruption_AffectsStation(t *testing.T) {
	d := &transit.Disruption{
		AffectedStations: []string{"ASD", "UT", "RTD"},
	}

	assert.True(t, d.AffectsStation("ASD"))
	assert.True(t, d.AffectsStation("UT"))
	assert.False(t, d.AffectsStation("DH"))
}

func TestDisruption_AffectsRoute(t *testing.T) {
	d := &transit.Disruption{
		AffectedRoutes: []string{"Amsterdam - Utrecht", "Rotterdam - Den Haag"},
	}

	assert.True(t, d.AffectsRoute("Amsterdam - Utrecht"))
	assert.False(t, d.AffectsRoute("Eindhoven - Maastricht"))
}

func TestCalculateOverallImpact(t *testing.T) {
	tests := []struct {
		name        string
		disruptions []*transit.Disruption
		expected    transit.Impact
	}{
		{
			name:        "empty",
			disruptions: nil,
			expected:    "",
		},
		{
			name: "single minor",
			disruptions: []*transit.Disruption{
				{Impact: transit.ImpactMinor},
			},
			expected: transit.ImpactMinor,
		},
		{
			name: "mixed - moderate highest",
			disruptions: []*transit.Disruption{
				{Impact: transit.ImpactMinor},
				{Impact: transit.ImpactModerate},
			},
			expected: transit.ImpactModerate,
		},
		{
			name: "mixed - severe highest",
			disruptions: []*transit.Disruption{
				{Impact: transit.ImpactMinor},
				{Impact: transit.ImpactMajor},
				{Impact: transit.ImpactSevere},
			},
			expected: transit.ImpactSevere,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, transit.CalculateOverallImpact(tt.disruptions))
		})
	}
}
