package resilience_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

func TestClient_SuccessfulRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := resilience.DefaultClientConfig("test")
	client := resilience.NewClient(cfg)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_RetryOn5xx(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cbConfig := resilience.DefaultCircuitBreakerConfig("test-retry")
	// Increase threshold so circuit doesn't trip during test
	cbConfig.ReadyToTrip = func(counts gobreaker.Counts) bool {
		return counts.Requests >= 100 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
	}

	cfg := resilience.ClientConfig{
		Name:            "test-retry",
		Timeout:         5 * time.Second,
		MaxRetries:      5,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		CircuitBreaker:  &cbConfig,
	}
	client := resilience.NewClient(cfg)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), attempts.Load(), "should have retried until success")
}

func TestClient_CircuitBreakerTrips(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Configure circuit breaker to trip after 5 requests with 50% failure
	cbConfig := resilience.CircuitBreakerConfig{
		Name:        "test-trip",
		MaxRequests: 1,
		Timeout:     1 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.Requests >= 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
		},
	}

	cfg := resilience.ClientConfig{
		Name:            "test-trip",
		Timeout:         1 * time.Second,
		MaxRetries:      0, // No retries for this test
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		CircuitBreaker:  &cbConfig,
	}
	client := resilience.NewClient(cfg)

	// Make 5 failing requests to trip the circuit
	for i := 0; i < 5; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
		require.NoError(t, err)
		resp, _ := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Circuit should be open now
	assert.Equal(t, gobreaker.StateOpen, client.CircuitBreakerState())

	// Next request should fail immediately with ErrCircuitOpen
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.ErrorIs(t, err, resilience.ErrCircuitOpen)
}

func TestClient_TimeoutHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cbConfig := resilience.DefaultCircuitBreakerConfig("test-timeout")
	// Increase threshold so circuit doesn't trip
	cbConfig.ReadyToTrip = func(counts gobreaker.Counts) bool {
		return counts.Requests >= 100
	}

	cfg := resilience.ClientConfig{
		Name:            "test-timeout",
		Timeout:         100 * time.Millisecond, // Very short timeout
		MaxRetries:      0,                      // No retries
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		CircuitBreaker:  &cbConfig,
	}
	client := resilience.NewClient(cfg)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.Error(t, err, "should timeout")
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := resilience.DefaultClientConfig("test-cancel")
	client := resilience.NewClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	// Cancel the context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.Error(t, err, "should be canceled")
}

func TestClient_4xxNotRetried(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := resilience.ClientConfig{
		Name:            "test-4xx",
		Timeout:         5 * time.Second,
		MaxRetries:      3,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
	}
	client := resilience.NewClient(cfg)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, int32(1), attempts.Load(), "should not retry 4xx errors")
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := resilience.DefaultCircuitBreakerConfig("test")

	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, uint32(1), cfg.MaxRequests)
	assert.Equal(t, 60*time.Second, cfg.Timeout)
	assert.NotNil(t, cfg.ReadyToTrip)
}

func TestDefaultReadyToTrip(t *testing.T) {
	tests := []struct {
		name     string
		counts   gobreaker.Counts
		expected bool
	}{
		{
			name:     "not enough requests",
			counts:   gobreaker.Counts{Requests: 4, TotalFailures: 4},
			expected: false,
		},
		{
			name:     "enough requests but low failure rate",
			counts:   gobreaker.Counts{Requests: 10, TotalFailures: 4},
			expected: false,
		},
		{
			name:     "enough requests and high failure rate",
			counts:   gobreaker.Counts{Requests: 10, TotalFailures: 5},
			expected: true,
		},
		{
			name:     "exactly 5 requests all failing",
			counts:   gobreaker.Counts{Requests: 5, TotalFailures: 5},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resilience.DefaultReadyToTrip(tt.counts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := resilience.DefaultClientConfig("test-client")

	assert.Equal(t, "test-client", cfg.Name)
	assert.Equal(t, 10*time.Second, cfg.Timeout)
	assert.Equal(t, uint64(3), cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.InitialInterval)
	assert.Equal(t, 5*time.Second, cfg.MaxInterval)
	assert.NotNil(t, cfg.CircuitBreaker)
}

func TestServerError(t *testing.T) {
	err := &resilience.ServerError{StatusCode: http.StatusInternalServerError}
	assert.Contains(t, err.Error(), "Internal Server Error")
}
