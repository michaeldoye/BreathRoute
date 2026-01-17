package commute

import (
	"context"
	"sync"
)

// InMemoryRepository is an in-memory implementation of Repository.
// This is intended for testing. Production should use PostgresRepository.
type InMemoryRepository struct {
	mu       sync.RWMutex
	commutes map[string]*Commute
}

// NewInMemoryRepository creates a new in-memory commute repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		commutes: make(map[string]*Commute),
	}
}

// Get retrieves a commute by ID.
func (r *InMemoryRepository) Get(_ context.Context, id string) (*Commute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.commutes[id]
	if !ok {
		return nil, ErrCommuteNotFound
	}

	// Return a copy
	cpy := *c
	return &cpy, nil
}

// GetByUserAndID retrieves a commute by user ID and commute ID.
func (r *InMemoryRepository) GetByUserAndID(_ context.Context, userID, commuteID string) (*Commute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.commutes[commuteID]
	if !ok {
		return nil, ErrCommuteNotFound
	}

	if c.UserID != userID {
		return nil, ErrCommuteNotFound
	}

	// Return a copy
	cpy := *c
	return &cpy, nil
}

// List retrieves all commutes for a user with pagination.
func (r *InMemoryRepository) List(_ context.Context, userID string, opts ListOptions) (*ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var commutes []*Commute
	for _, c := range r.commutes {
		if c.UserID == userID {
			cpy := *c
			commutes = append(commutes, &cpy)
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	result := &ListResult{
		Items: commutes,
	}

	if len(commutes) > limit {
		result.Items = commutes[:limit]
		result.NextCursor = commutes[limit-1].ID
	}

	return result, nil
}

// Create creates a new commute.
func (r *InMemoryRepository) Create(_ context.Context, c *Commute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cpy := *c
	r.commutes[c.ID] = &cpy
	return nil
}

// Update updates an existing commute.
func (r *InMemoryRepository) Update(_ context.Context, c *Commute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.commutes[c.ID]; !ok {
		return ErrCommuteNotFound
	}

	cpy := *c
	r.commutes[c.ID] = &cpy
	return nil
}

// Delete deletes a commute by ID.
func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.commutes, id)
	return nil
}

// Ensure InMemoryRepository implements Repository interface.
var _ Repository = (*InMemoryRepository)(nil)
