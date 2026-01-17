package commute

import "context"

// ListOptions contains options for listing commutes.
type ListOptions struct {
	Limit  int
	Cursor string
}

// ListResult contains the results of listing commutes.
type ListResult struct {
	Items      []*Commute
	NextCursor string
}

// Repository defines the interface for commute data persistence.
type Repository interface {
	// Get retrieves a commute by ID.
	Get(ctx context.Context, id string) (*Commute, error)

	// GetByUserAndID retrieves a commute by user ID and commute ID.
	// Returns ErrCommuteNotFound if the commute doesn't exist or doesn't belong to the user.
	GetByUserAndID(ctx context.Context, userID, commuteID string) (*Commute, error)

	// List retrieves all commutes for a user with pagination.
	List(ctx context.Context, userID string, opts ListOptions) (*ListResult, error)

	// Create creates a new commute.
	Create(ctx context.Context, commute *Commute) error

	// Update updates an existing commute.
	Update(ctx context.Context, commute *Commute) error

	// Delete deletes a commute by ID.
	Delete(ctx context.Context, id string) error
}
