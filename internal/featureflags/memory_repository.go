package featureflags

import (
	"context"
	"sync"
	"time"
)

// InMemoryRepository is an in-memory implementation of Repository for testing.
type InMemoryRepository struct {
	mu    sync.RWMutex
	flags map[string]*Flag
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		flags: make(map[string]*Flag),
	}
}

// NewInMemoryRepositoryWithFlags creates a new in-memory repository with initial flags.
func NewInMemoryRepositoryWithFlags(flags map[string]*Flag) *InMemoryRepository {
	repo := &InMemoryRepository{
		flags: make(map[string]*Flag),
	}
	for k, v := range flags {
		repo.flags[k] = v
	}
	return repo
}

// GetFlag retrieves a single feature flag by key.
func (r *InMemoryRepository) GetFlag(ctx context.Context, key string) (*Flag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flag, ok := r.flags[key]
	if !ok {
		return nil, ErrFlagNotFound
	}
	return flag, nil
}

// GetAllFlags retrieves all feature flags.
func (r *InMemoryRepository) GetAllFlags(ctx context.Context) (map[string]*Flag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*Flag, len(r.flags))
	for k, v := range r.flags {
		result[k] = v
	}
	return result, nil
}

// SetFlag creates or updates a feature flag.
func (r *InMemoryRepository) SetFlag(ctx context.Context, flag *Flag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	flag.UpdatedAt = time.Now()
	r.flags[flag.Key] = flag
	return nil
}

// SetFlags creates or updates multiple feature flags.
func (r *InMemoryRepository) SetFlags(ctx context.Context, flags []*Flag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, flag := range flags {
		flag.UpdatedAt = now
		r.flags[flag.Key] = flag
	}
	return nil
}

// DeleteFlag removes a feature flag by key.
func (r *InMemoryRepository) DeleteFlag(ctx context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.flags, key)
	return nil
}

// Ensure InMemoryRepository implements Repository interface.
var _ Repository = (*InMemoryRepository)(nil)
