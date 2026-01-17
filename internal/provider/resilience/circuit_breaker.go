// Package resilience provides resilient HTTP client wrappers with circuit breakers,
// timeouts, and retry logic for external provider calls.
package resilience

import (
	"time"

	"github.com/sony/gobreaker/v2"
)

// CircuitBreakerConfig holds configuration for the circuit breaker.
type CircuitBreakerConfig struct {
	// Name identifies the circuit breaker for logging/metrics.
	Name string

	// MaxRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	MaxRequests uint32

	// Interval is the cyclic period for clearing internal counts when closed.
	// Default: 0 (disabled)
	Interval time.Duration

	// Timeout is the period of open state before switching to half-open.
	// Default: 60 seconds
	Timeout time.Duration

	// ReadyToTrip determines when to trip the circuit breaker.
	// If nil, uses DefaultReadyToTrip (50% failure rate with 5+ requests).
	ReadyToTrip func(counts gobreaker.Counts) bool

	// OnStateChange is called when the circuit breaker state changes.
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
}

// DefaultCircuitBreakerConfig returns a sensible default configuration.
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 1,
		Interval:    0,
		Timeout:     60 * time.Second,
		ReadyToTrip: DefaultReadyToTrip,
	}
}

// DefaultReadyToTrip trips the circuit breaker when at least 5 requests have been made
// and the failure rate is 50% or higher.
func DefaultReadyToTrip(counts gobreaker.Counts) bool {
	failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
	return counts.Requests >= 5 && failureRatio >= 0.5
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker[T any](cfg CircuitBreakerConfig) *gobreaker.CircuitBreaker[T] {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: cfg.ReadyToTrip,
	}

	if cfg.OnStateChange != nil {
		settings.OnStateChange = cfg.OnStateChange
	}

	return gobreaker.NewCircuitBreaker[T](settings)
}
