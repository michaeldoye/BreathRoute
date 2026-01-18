package routing

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockProvider is a mock routing provider for testing.
type mockProvider struct {
	name      string
	profiles  []RouteProfile
	response  *DirectionsResponse
	err       error
	callCount atomic.Int32
	delay     time.Duration
}

func (m *mockProvider) GetDirections(ctx context.Context, req DirectionsRequest) (*DirectionsResponse, error) {
	m.callCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SupportedProfiles() []RouteProfile {
	return m.profiles
}

func TestService_GetDirections_CacheMiss(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike, ProfileWalk},
		response: &DirectionsResponse{
			Routes: []Route{
				{
					GeometryPolyline: "_p~iF~ps|U_ulLnnqC",
					DistanceMeters:   12345,
					DurationSeconds:  2456,
				},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	resp, err := service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call, got %d", provider.callCount.Load())
	}

	if len(resp.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(resp.Routes))
	}

	if resp.Routes[0].DistanceMeters != 12345 {
		t.Errorf("expected distance 12345, got %d", resp.Routes[0].DistanceMeters)
	}
}

func TestService_GetDirections_CacheHit(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike, ProfileWalk},
		response: &DirectionsResponse{
			Routes: []Route{
				{
					GeometryPolyline: "_p~iF~ps|U_ulLnnqC",
					DistanceMeters:   12345,
					DurationSeconds:  2456,
				},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	req := DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	}

	// First call
	_, err := service.GetDirections(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	// Second call (should hit cache)
	_, err = service.GetDirections(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call (cache hit), got %d", provider.callCount.Load())
	}
}

func TestService_GetDirections_GridCaching(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike, ProfileWalk},
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider:      provider,
		CacheTTL:      5 * time.Minute,
		CacheGridSize: 0.01, // ~1.1km grid
	})

	// Request 1
	_, _ = service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	})

	// Request 2 - slightly different coordinates but same grid cell
	_, _ = service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3678, Lon: 4.9045}, // Small offset
		Destination: Coordinate{Lat: 52.0909, Lon: 5.1210}, // Small offset
		Profile:     ProfileBike,
	})

	// Should only have called provider once due to grid caching
	if provider.callCount.Load() != 1 {
		t.Errorf("expected 1 provider call (grid cache hit), got %d", provider.callCount.Load())
	}
}

func TestService_GetDirections_DifferentProfilesNotCached(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike, ProfileWalk},
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	// Bike request
	_, _ = service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	})

	// Walk request - same coordinates, different profile
	_, _ = service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileWalk,
	})

	// Should call provider twice - different profiles
	if provider.callCount.Load() != 2 {
		t.Errorf("expected 2 provider calls (different profiles), got %d", provider.callCount.Load())
	}
}

func TestService_GetDirections_StaleIfError(t *testing.T) {
	callCount := atomic.Int32{}
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike},
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider:        provider,
		CacheTTL:        50 * time.Millisecond,
		StaleIfErrorTTL: 500 * time.Millisecond,
	})

	req := DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	}

	// First call - populates cache
	_, err := service.GetDirections(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	callCount.Store(provider.callCount.Load())

	// Wait for cache to expire (but still within stale window)
	time.Sleep(100 * time.Millisecond)

	// Make provider fail
	provider.err = errors.New("provider error")

	// This call should serve stale data
	resp, err := service.GetDirections(context.Background(), req)
	if err != nil {
		t.Fatalf("expected stale data to be served, got error: %v", err)
	}

	if resp.Routes[0].DistanceMeters != 12345 {
		t.Errorf("expected stale distance 12345, got %d", resp.Routes[0].DistanceMeters)
	}
}

func TestService_GetDirections_InvalidCoordinates(t *testing.T) {
	provider := &mockProvider{
		name: "test-provider",
	}

	service := NewService(ServiceConfig{
		Provider: provider,
	})

	tests := []struct {
		name string
		req  DirectionsRequest
	}{
		{
			name: "invalid origin latitude",
			req: DirectionsRequest{
				Origin:      Coordinate{Lat: 91, Lon: 0},
				Destination: Coordinate{Lat: 0, Lon: 0},
				Profile:     ProfileBike,
			},
		},
		{
			name: "invalid destination longitude",
			req: DirectionsRequest{
				Origin:      Coordinate{Lat: 0, Lon: 0},
				Destination: Coordinate{Lat: 0, Lon: 181},
				Profile:     ProfileBike,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.GetDirections(context.Background(), tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var routingErr *Error
			if !errors.As(err, &routingErr) {
				t.Fatalf("expected Error, got %T", err)
			}
			if !errors.Is(routingErr.Err, ErrInvalidCoordinates) {
				t.Errorf("expected ErrInvalidCoordinates, got %v", routingErr.Err)
			}
		})
	}
}

func TestService_GetDirections_ConcurrentRequests(t *testing.T) {
	provider := &mockProvider{
		name:  "test-provider",
		delay: 50 * time.Millisecond, // Simulate slow provider
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	req := DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	}

	// Start 10 concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.GetDirections(context.Background(), req)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	// With double-check locking, only a few calls should reach the provider
	// (not all 10)
	calls := provider.callCount.Load()
	if calls > 3 {
		t.Errorf("expected <= 3 provider calls with double-check locking, got %d", calls)
	}
}

func TestService_CacheStats(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike},
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	// Initial stats
	stats := service.CacheStats()
	if stats.TotalEntries != 0 {
		t.Errorf("expected 0 entries, got %d", stats.TotalEntries)
	}
	if stats.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got '%s'", stats.Provider)
	}

	// Add an entry
	_, _ = service.GetDirections(context.Background(), DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	})

	stats = service.CacheStats()
	if stats.TotalEntries != 1 {
		t.Errorf("expected 1 entry, got %d", stats.TotalEntries)
	}
	if stats.FreshEntries != 1 {
		t.Errorf("expected 1 fresh entry, got %d", stats.FreshEntries)
	}
}

func TestService_InvalidateCache(t *testing.T) {
	provider := &mockProvider{
		name:     "test-provider",
		profiles: []RouteProfile{ProfileBike},
		response: &DirectionsResponse{
			Routes: []Route{
				{DistanceMeters: 12345},
			},
			Provider:  "test-provider",
			FetchedAt: time.Now(),
		},
	}

	service := NewService(ServiceConfig{
		Provider: provider,
		CacheTTL: 5 * time.Minute,
	})

	req := DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	}

	// Populate cache
	_, _ = service.GetDirections(context.Background(), req)
	if service.CacheStats().TotalEntries != 1 {
		t.Fatal("expected cache to have 1 entry")
	}

	// Invalidate
	service.InvalidateCache()

	if service.CacheStats().TotalEntries != 0 {
		t.Errorf("expected empty cache after invalidation, got %d entries", service.CacheStats().TotalEntries)
	}

	// New request should call provider again
	_, _ = service.GetDirections(context.Background(), req)
	if provider.callCount.Load() != 2 {
		t.Errorf("expected 2 provider calls after cache invalidation, got %d", provider.callCount.Load())
	}
}

func TestService_CacheKeyFormat(t *testing.T) {
	service := &Service{
		cacheGridSize: 0.01,
	}

	req := DirectionsRequest{
		Origin:      Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     ProfileBike,
	}

	key := service.cacheKey(req)

	// Should contain profile and 4 coordinate values
	expectedPrefix := "cycling-regular:"
	if len(key) < len(expectedPrefix) || key[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("cache key should start with '%s', got '%s'", expectedPrefix, key)
	}
}

func TestService_ProviderName(t *testing.T) {
	provider := &mockProvider{
		name: "my-routing-provider",
	}

	service := NewService(ServiceConfig{
		Provider: provider,
	})

	if service.ProviderName() != "my-routing-provider" {
		t.Errorf("expected 'my-routing-provider', got '%s'", service.ProviderName())
	}
}
