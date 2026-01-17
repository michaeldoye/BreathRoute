package featureflags

import (
	"context"
	"sync"
	"time"
)

// InMemoryRepository is an in-memory implementation of the Repository interface.
// This is intended for MVP/testing. Production should use a database-backed implementation.
type InMemoryRepository struct {
	mu    sync.RWMutex
	flags map[string]*Flag
}

// NewInMemoryRepository creates a new in-memory repository with default flags.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		flags: DefaultFlags(),
	}
}

// NewInMemoryRepositoryWithFlags creates a new in-memory repository with custom flags.
func NewInMemoryRepositoryWithFlags(flags map[string]*Flag) *InMemoryRepository {
	if flags == nil {
		flags = make(map[string]*Flag)
	}
	return &InMemoryRepository{
		flags: flags,
	}
}

// GetFlag retrieves a single feature flag by key.
func (r *InMemoryRepository) GetFlag(_ context.Context, key string) (*Flag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flag, ok := r.flags[key]
	if !ok {
		return nil, ErrFlagNotFound
	}

	// Return a copy to prevent mutation
	return &Flag{
		Key:       flag.Key,
		Value:     flag.Value,
		UpdatedAt: flag.UpdatedAt,
	}, nil
}

// GetAllFlags retrieves all feature flags.
func (r *InMemoryRepository) GetAllFlags(_ context.Context) (map[string]*Flag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return copies to prevent mutation
	result := make(map[string]*Flag, len(r.flags))
	for k, v := range r.flags {
		result[k] = &Flag{
			Key:       v.Key,
			Value:     v.Value,
			UpdatedAt: v.UpdatedAt,
		}
	}
	return result, nil
}

// SetFlag creates or updates a feature flag.
func (r *InMemoryRepository) SetFlag(_ context.Context, flag *Flag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.flags[flag.Key] = &Flag{
		Key:       flag.Key,
		Value:     flag.Value,
		UpdatedAt: time.Now(),
	}
	return nil
}

// SetFlags creates or updates multiple feature flags atomically.
func (r *InMemoryRepository) SetFlags(_ context.Context, flags []*Flag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, flag := range flags {
		r.flags[flag.Key] = &Flag{
			Key:       flag.Key,
			Value:     flag.Value,
			UpdatedAt: now,
		}
	}
	return nil
}

// DeleteFlag removes a feature flag by key.
func (r *InMemoryRepository) DeleteFlag(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.flags[key]; !ok {
		return ErrFlagNotFound
	}
	delete(r.flags, key)
	return nil
}
