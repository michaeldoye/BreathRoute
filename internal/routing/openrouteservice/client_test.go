package openrouteservice

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/routing"
)

func TestClient_GetDirections_Success(t *testing.T) {
	// Load test fixture
	respBody, err := os.ReadFile("testdata/directions_response.json")
	if err != nil {
		t.Fatalf("failed to load test fixture: %v", err)
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "mock123" {
			t.Errorf("expected Authorization header 'mock123', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Verify URL path contains profile
		expectedPath := "/v2/directions/cycling-regular"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBody)
	}))
	defer server.Close()

	// Create client
	client := NewClient(ClientConfig{
		APIKey:     "mock123",
		BaseURL:    server.URL,
		HTTPClient: &mockHTTPClient{client: server.Client()},
		Logger:     zerolog.Nop(),
	})

	// Make request
	resp, err := client.GetDirections(context.Background(), routing.DirectionsRequest{
		Origin:          routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination:     routing.Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:         routing.ProfileBike,
		MaxAlternatives: 2,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response
	if resp.Provider != ProviderName {
		t.Errorf("expected provider %s, got %s", ProviderName, resp.Provider)
	}
	if len(resp.Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(resp.Routes))
	}

	// Verify first route
	route := resp.Routes[0]
	if route.DistanceMeters != 12345 {
		t.Errorf("expected distance 12345, got %d", route.DistanceMeters)
	}
	if route.DurationSeconds != 2456 {
		t.Errorf("expected duration 2456, got %d", route.DurationSeconds)
	}
	if route.GeometryPolyline == "" {
		t.Error("expected non-empty geometry polyline")
	}
	if route.BoundingBox == nil {
		t.Error("expected bounding box to be set")
	}
	if len(route.Instructions) == 0 {
		t.Error("expected instructions to be present")
	}
}

func TestClient_GetDirections_NoRouteFound(t *testing.T) {
	// Load test fixture
	respBody, err := os.ReadFile("testdata/error_response.json")
	if err != nil {
		t.Fatalf("failed to load test fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(respBody)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:     "mock123",
		BaseURL:    server.URL,
		HTTPClient: &mockHTTPClient{client: server.Client()},
		Logger:     zerolog.Nop(),
	})

	_, err = client.GetDirections(context.Background(), routing.DirectionsRequest{
		Origin:      routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: routing.Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     routing.ProfileBike,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var routingErr *routing.Error
	if !errors.As(err, &routingErr) {
		t.Fatalf("expected routing.Error, got %T", err)
	}
	if !errors.Is(routingErr.Err, routing.ErrNoRouteFound) {
		t.Errorf("expected ErrNoRouteFound, got %v", routingErr.Err)
	}
}

func TestClient_GetDirections_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"code":403,"message":"Rate limit exceeded"}}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:     "mock123",
		BaseURL:    server.URL,
		HTTPClient: &mockHTTPClient{client: server.Client()},
		Logger:     zerolog.Nop(),
	})

	_, err := client.GetDirections(context.Background(), routing.DirectionsRequest{
		Origin:      routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: routing.Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     routing.ProfileBike,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var routingErr *routing.Error
	if !errors.As(err, &routingErr) {
		t.Fatalf("expected routing.Error, got %T", err)
	}
	if !errors.Is(routingErr.Err, routing.ErrRateLimitExceeded) {
		t.Errorf("expected ErrRateLimitExceeded, got %v", routingErr.Err)
	}
}

func TestClient_GetDirections_InvalidCoordinates(t *testing.T) {
	tests := []struct {
		name        string
		origin      routing.Coordinate
		destination routing.Coordinate
	}{
		{
			name:        "latitude out of range",
			origin:      routing.Coordinate{Lat: 91.0, Lon: 4.9},
			destination: routing.Coordinate{Lat: 52.0, Lon: 5.1},
		},
		{
			name:        "negative latitude out of range",
			origin:      routing.Coordinate{Lat: -91.0, Lon: 4.9},
			destination: routing.Coordinate{Lat: 52.0, Lon: 5.1},
		},
		{
			name:        "longitude out of range",
			origin:      routing.Coordinate{Lat: 52.0, Lon: 4.9},
			destination: routing.Coordinate{Lat: 52.0, Lon: 181.0},
		},
		{
			name:        "negative longitude out of range",
			origin:      routing.Coordinate{Lat: 52.0, Lon: 4.9},
			destination: routing.Coordinate{Lat: 52.0, Lon: -181.0},
		},
	}

	client := NewClient(ClientConfig{
		APIKey: "mock123",
		Logger: zerolog.Nop(),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetDirections(context.Background(), routing.DirectionsRequest{
				Origin:      tt.origin,
				Destination: tt.destination,
				Profile:     routing.ProfileBike,
			})

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var routingErr *routing.Error
			if !errors.As(err, &routingErr) {
				t.Fatalf("expected routing.Error, got %T", err)
			}
			if !errors.Is(routingErr.Err, routing.ErrInvalidCoordinates) {
				t.Errorf("expected ErrInvalidCoordinates, got %v", routingErr.Err)
			}
		})
	}
}

func TestClient_GetDirections_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"code":500,"message":"Internal server error"}}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:     "mock123",
		BaseURL:    server.URL,
		HTTPClient: &mockHTTPClient{client: server.Client()},
		Logger:     zerolog.Nop(),
	})

	_, err := client.GetDirections(context.Background(), routing.DirectionsRequest{
		Origin:      routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: routing.Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     routing.ProfileBike,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var routingErr *routing.Error
	if !errors.As(err, &routingErr) {
		t.Fatalf("expected routing.Error, got %T", err)
	}
	if !errors.Is(routingErr.Err, routing.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", routingErr.Err)
	}
}

func TestClient_Name(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey: "test",
		Logger: zerolog.Nop(),
	})

	if client.Name() != ProviderName {
		t.Errorf("expected %s, got %s", ProviderName, client.Name())
	}
}

func TestClient_SupportedProfiles(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey: "test",
		Logger: zerolog.Nop(),
	})

	profiles := client.SupportedProfiles()
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}

	hasWalk := false
	hasBike := false
	for _, p := range profiles {
		if p == routing.ProfileWalk {
			hasWalk = true
		}
		if p == routing.ProfileBike {
			hasBike = true
		}
	}

	if !hasWalk {
		t.Error("expected ProfileWalk in supported profiles")
	}
	if !hasBike {
		t.Error("expected ProfileBike in supported profiles")
	}
}

// mockHTTPClient wraps http.Client to implement HTTPDoer interface.
type mockHTTPClient struct {
	client *http.Client
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.client.Do(req)
}

func TestValidateCoordinates(t *testing.T) {
	tests := []struct {
		name    string
		coord   routing.Coordinate
		wantErr bool
	}{
		{
			name:    "valid Amsterdam",
			coord:   routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
			wantErr: false,
		},
		{
			name:    "valid equator",
			coord:   routing.Coordinate{Lat: 0, Lon: 0},
			wantErr: false,
		},
		{
			name:    "valid extreme lat",
			coord:   routing.Coordinate{Lat: 90, Lon: 0},
			wantErr: false,
		},
		{
			name:    "valid extreme lon",
			coord:   routing.Coordinate{Lat: 0, Lon: 180},
			wantErr: false,
		},
		{
			name:    "invalid lat too high",
			coord:   routing.Coordinate{Lat: 90.1, Lon: 0},
			wantErr: true,
		},
		{
			name:    "invalid lat too low",
			coord:   routing.Coordinate{Lat: -90.1, Lon: 0},
			wantErr: true,
		},
		{
			name:    "invalid lon too high",
			coord:   routing.Coordinate{Lat: 0, Lon: 180.1},
			wantErr: true,
		},
		{
			name:    "invalid lon too low",
			coord:   routing.Coordinate{Lat: 0, Lon: -180.1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCoordinates(tt.coord)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCoordinates() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// mockFailingClient simulates network errors.
type mockFailingClient struct{}

func (m *mockFailingClient) Do(req *http.Request) (*http.Response, error) {
	return nil, errors.New("network error")
}

func TestClient_GetDirections_NetworkError(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:     "mock123",
		HTTPClient: &mockFailingClient{},
		Logger:     zerolog.Nop(),
	})

	_, err := client.GetDirections(context.Background(), routing.DirectionsRequest{
		Origin:      routing.Coordinate{Lat: 52.3676, Lon: 4.9041},
		Destination: routing.Coordinate{Lat: 52.0907, Lon: 5.1214},
		Profile:     routing.ProfileBike,
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var routingErr *routing.Error
	if !errors.As(err, &routingErr) {
		t.Fatalf("expected routing.Error, got %T", err)
	}
	if !errors.Is(routingErr.Err, routing.ErrProviderUnavailable) {
		t.Errorf("expected ErrProviderUnavailable, got %v", routingErr.Err)
	}
}

func TestError_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      *routing.Error
		expected bool
	}{
		{
			name: "provider unavailable is retryable",
			err: &routing.Error{
				Err: routing.ErrProviderUnavailable,
			},
			expected: true,
		},
		{
			name: "rate limit is retryable",
			err: &routing.Error{
				Err: routing.ErrRateLimitExceeded,
			},
			expected: true,
		},
		{
			name: "no route found is not retryable",
			err: &routing.Error{
				Err: routing.ErrNoRouteFound,
			},
			expected: false,
		},
		{
			name: "invalid coordinates is not retryable",
			err: &routing.Error{
				Err: routing.ErrInvalidCoordinates,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.IsRetryable() != tt.expected {
				t.Errorf("IsRetryable() = %v, expected %v", tt.err.IsRetryable(), tt.expected)
			}
		})
	}
}
