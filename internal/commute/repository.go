package commute

import (
	"context"
	"errors"
	"sync"
)

// Repository errors.
var (
	ErrCommuteNotFound = errors.New("commute not found")
)

// ListOptions contains options for listing commutes.
type ListOptions struct {
	Limit  int
	Cursor string
}

// ListResult contains the result of listing commutes.
type ListResult struct {
	Items      []*Commute
	NextCursor string
}

// Repository defines the interface for commute data persistence.
type Repository interface {
	// Get retrieves a commute by ID.
	Get(ctx context.Context, id string) (*Commute, error)

	// GetByUserAndID retrieves a commute by user ID and commute ID.
	GetByUserAndID(ctx context.Context, userID, commuteID string) (*Commute, error)

	// List retrieves all commutes for a user.
	List(ctx context.Context, userID string, opts ListOptions) (*ListResult, error)

	// Create creates a new commute.
	Create(ctx context.Context, commute *Commute) error

	// Update updates an existing commute.
	Update(ctx context.Context, commute *Commute) error

	// Delete deletes a commute by ID.
	Delete(ctx context.Context, id string) error
}

// InMemoryRepository is an in-memory implementation of Repository.
// This is intended for MVP/testing. Production should use a database-backed implementation.
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

	commute, ok := r.commutes[id]
	if !ok {
		return nil, ErrCommuteNotFound
	}

	return commute.Copy(), nil
}

// GetByUserAndID retrieves a commute by user ID and commute ID.
func (r *InMemoryRepository) GetByUserAndID(_ context.Context, userID, commuteID string) (*Commute, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commute, ok := r.commutes[commuteID]
	if !ok {
		return nil, ErrCommuteNotFound
	}

	// Verify ownership
	if commute.UserID != userID {
		return nil, ErrCommuteNotFound
	}

	return commute.Copy(), nil
}

// List retrieves all commutes for a user.
func (r *InMemoryRepository) List(_ context.Context, userID string, opts ListOptions) (*ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Set default limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	// Collect commutes for user
	var items []*Commute
	for _, c := range r.commutes {
		if c.UserID == userID {
			items = append(items, c.Copy())
		}
	}

	// Simple pagination (for MVP, just truncate)
	if len(items) > limit {
		items = items[:limit]
	}

	return &ListResult{
		Items:      items,
		NextCursor: "", // Simplified for MVP
	}, nil
}

// Create creates a new commute.
func (r *InMemoryRepository) Create(_ context.Context, commute *Commute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.commutes[commute.ID] = commute.Copy()
	return nil
}

// Update updates an existing commute.
func (r *InMemoryRepository) Update(_ context.Context, commute *Commute) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.commutes[commute.ID]; !ok {
		return ErrCommuteNotFound
	}

	r.commutes[commute.ID] = commute.Copy()
	return nil
}

// Delete deletes a commute by ID.
func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.commutes[id]; !ok {
		return ErrCommuteNotFound
	}

	delete(r.commutes, id)
	return nil
}
