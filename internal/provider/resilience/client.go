package resilience

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sony/gobreaker/v2"
)

// Predefined errors for resilient operations.
var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrMaxRetriesExceeded is returned when all retry attempts have been exhausted.
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// ClientConfig holds configuration for the resilient HTTP client.
type ClientConfig struct {
	// Name identifies this client for circuit breaker naming.
	Name string

	// Timeout is the request timeout for individual HTTP calls.
	// Default: 10 seconds
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts.
	// Default: 3
	MaxRetries uint64

	// InitialInterval is the initial retry backoff interval.
	// Default: 100ms
	InitialInterval time.Duration

	// MaxInterval is the maximum retry backoff interval.
	// Default: 5 seconds
	MaxInterval time.Duration

	// CircuitBreaker is the circuit breaker configuration.
	// If nil, uses DefaultCircuitBreakerConfig.
	CircuitBreaker *CircuitBreakerConfig
}

// DefaultClientConfig returns sensible defaults for the resilient client.
func DefaultClientConfig(name string) ClientConfig {
	cbConfig := DefaultCircuitBreakerConfig(name)
	return ClientConfig{
		Name:            name,
		Timeout:         10 * time.Second,
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		CircuitBreaker:  &cbConfig,
	}
}

// Client is a resilient HTTP client with circuit breaker and retry logic.
type Client struct {
	httpClient     *http.Client
	circuitBreaker *gobreaker.CircuitBreaker[*http.Response]
	config         ClientConfig
}

// NewClient creates a new resilient HTTP client.
func NewClient(cfg ClientConfig) *Client {
	// Set defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialInterval == 0 {
		cfg.InitialInterval = 100 * time.Millisecond
	}
	if cfg.MaxInterval == 0 {
		cfg.MaxInterval = 5 * time.Second
	}

	// Create circuit breaker
	var cb *gobreaker.CircuitBreaker[*http.Response]
	if cfg.CircuitBreaker != nil {
		cb = NewCircuitBreaker[*http.Response](*cfg.CircuitBreaker) //nolint:bodyclose // type param, not response
	} else {
		defaultCB := DefaultCircuitBreakerConfig(cfg.Name)
		cb = NewCircuitBreaker[*http.Response](defaultCB) //nolint:bodyclose // type param, not response
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		circuitBreaker: cb,
		config:         cfg,
	}
}

// Do executes an HTTP request with circuit breaker protection and retry logic.
// The request is retried on transient failures (5xx, network errors) with exponential backoff.
// Returns immediately with ErrCircuitOpen if the circuit breaker is open.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.DoWithContext(req.Context(), req)
}

// DoWithContext executes an HTTP request with the given context.
func (c *Client) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Create exponential backoff
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = c.config.InitialInterval
	bo.MaxInterval = c.config.MaxInterval
	bo.MaxElapsedTime = 0 // Unlimited, we control retries via WithMaxRetries

	// Wrap with max retries and context
	backoffWithRetries := backoff.WithMaxRetries(bo, c.config.MaxRetries)
	backoffWithContext := backoff.WithContext(backoffWithRetries, ctx)

	var lastResp *http.Response

	operation := func() error {
		// Execute through circuit breaker
		// Note: 5xx errors are returned as errors to trip the circuit breaker
		resp, err := c.circuitBreaker.Execute(func() (*http.Response, error) { //nolint:bodyclose // caller is responsible for closing
			// Clone the request for retry safety (body needs special handling)
			reqClone := req.Clone(ctx)
			r, err := c.httpClient.Do(reqClone)
			if err != nil {
				return nil, err
			}

			// Treat 5xx as errors for circuit breaker
			if r.StatusCode >= 500 {
				return r, &ServerError{StatusCode: r.StatusCode}
			}

			return r, nil
		})

		if err != nil {
			// Check if circuit breaker is open
			if errors.Is(err, gobreaker.ErrOpenState) {
				return backoff.Permanent(ErrCircuitOpen)
			}
			if errors.Is(err, gobreaker.ErrTooManyRequests) {
				return backoff.Permanent(ErrCircuitOpen)
			}

			// Store response if available (5xx case)
			if resp != nil {
				lastResp = resp
			}
			// Network and server errors are retryable
			return err
		}

		lastResp = resp

		// Success or client error (not retryable)
		return nil
	}

	err := backoff.Retry(operation, backoffWithContext)
	if err != nil {
		// If we have a last response (e.g., 5xx that exhausted retries), return it
		if lastResp != nil {
			return lastResp, nil
		}
		return nil, err
	}

	return lastResp, nil
}

// ServerError represents an HTTP 5xx server error.
type ServerError struct {
	StatusCode int
}

func (e *ServerError) Error() string {
	return "server error: " + http.StatusText(e.StatusCode)
}

// CircuitBreakerState returns the current state of the circuit breaker.
func (c *Client) CircuitBreakerState() gobreaker.State {
	return c.circuitBreaker.State()
}

// CircuitBreakerCounts returns the current counts of the circuit breaker.
func (c *Client) CircuitBreakerCounts() gobreaker.Counts {
	return c.circuitBreaker.Counts()
}
