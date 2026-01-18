package resilience

import (
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
)

// ProviderHealth represents the health status of a provider.
type ProviderHealth struct {
	// Name is the provider identifier.
	Name string

	// CircuitState is the current circuit breaker state.
	CircuitState gobreaker.State

	// Counts contains circuit breaker statistics.
	Counts gobreaker.Counts

	// LastSuccessAt is the timestamp of the last successful request.
	LastSuccessAt *time.Time

	// LastFailureAt is the timestamp of the last failed request.
	LastFailureAt *time.Time

	// LastError is the most recent error message, if any.
	LastError string
}

// IsHealthy returns true if the provider is considered healthy.
func (h *ProviderHealth) IsHealthy() bool {
	return h.CircuitState == gobreaker.StateClosed
}

// IsDegraded returns true if the provider is in a degraded state (half-open).
func (h *ProviderHealth) IsDegraded() bool {
	return h.CircuitState == gobreaker.StateHalfOpen
}

// IsUnhealthy returns true if the provider is unhealthy (circuit open).
func (h *ProviderHealth) IsUnhealthy() bool {
	return h.CircuitState == gobreaker.StateOpen
}

// Registry tracks registered providers and their health status.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]*registeredProvider
}

type registeredProvider struct {
	client        *Client
	lastSuccessAt *time.Time
	lastFailureAt *time.Time
	lastError     string
}

// GlobalRegistry is the default provider registry.
var GlobalRegistry = NewRegistry()

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]*registeredProvider),
	}
}

// Register adds a provider client to the registry.
func (r *Registry) Register(name string, client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = &registeredProvider{
		client: client,
	}
}

// Unregister removes a provider from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
}

// RecordSuccess records a successful request for a provider.
func (r *Registry) RecordSuccess(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[name]; ok {
		now := time.Now()
		p.lastSuccessAt = &now
	}
}

// RecordFailure records a failed request for a provider.
func (r *Registry) RecordFailure(name string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.providers[name]; ok {
		now := time.Now()
		p.lastFailureAt = &now
		if err != nil {
			p.lastError = err.Error()
		}
	}
}

// GetHealth returns the health status of a specific provider.
func (r *Registry) GetHealth(name string) *ProviderHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil
	}

	return &ProviderHealth{
		Name:          name,
		CircuitState:  p.client.CircuitBreakerState(),
		Counts:        p.client.CircuitBreakerCounts(),
		LastSuccessAt: p.lastSuccessAt,
		LastFailureAt: p.lastFailureAt,
		LastError:     p.lastError,
	}
}

// GetAllHealth returns the health status of all registered providers.
func (r *Registry) GetAllHealth() []*ProviderHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := make([]*ProviderHealth, 0, len(r.providers))
	for name, p := range r.providers {
		health = append(health, &ProviderHealth{
			Name:          name,
			CircuitState:  p.client.CircuitBreakerState(),
			Counts:        p.client.CircuitBreakerCounts(),
			LastSuccessAt: p.lastSuccessAt,
			LastFailureAt: p.lastFailureAt,
			LastError:     p.lastError,
		})
	}

	return health
}

// GetProviderNames returns the names of all registered providers.
func (r *Registry) GetProviderNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ProviderCount returns the number of registered providers.
func (r *Registry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.providers)
}
