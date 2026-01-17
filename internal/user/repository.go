package user

import (
	"context"
	"errors"
	"sync"
)

// Repository errors.
var (
	ErrUserNotFound = errors.New("user not found")
)

// Repository defines the interface for user data persistence.
type Repository interface {
	// Get retrieves a user by ID.
	Get(ctx context.Context, id string) (*User, error)

	// Create creates a new user.
	Create(ctx context.Context, user *User) error

	// Update updates an existing user.
	Update(ctx context.Context, user *User) error

	// Delete deletes a user and all associated data.
	Delete(ctx context.Context, id string) error
}

// InMemoryRepository is an in-memory implementation of Repository.
// This is intended for MVP/testing. Production should use a database-backed implementation.
type InMemoryRepository struct {
	mu    sync.RWMutex
	users map[string]*User
}

// NewInMemoryRepository creates a new in-memory user repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users: make(map[string]*User),
	}
}

// Get retrieves a user by ID.
func (r *InMemoryRepository) Get(_ context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	// Return a deep copy to prevent mutation
	return copyUser(user), nil
}

// Create creates a new user.
func (r *InMemoryRepository) Create(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = copyUser(user)
	return nil
}

// Update updates an existing user.
func (r *InMemoryRepository) Update(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[user.ID]; !ok {
		return ErrUserNotFound
	}

	r.users[user.ID] = copyUser(user)
	return nil
}

// Delete deletes a user.
func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.users, id)
	return nil
}

// copyUser creates a deep copy of a user.
func copyUser(u *User) *User {
	if u == nil {
		return nil
	}

	userCopy := &User{
		ID:        u.ID,
		Locale:    u.Locale,
		Units:     u.Units,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}

	if u.Profile != nil {
		userCopy.Profile = &Profile{
			Weights:     u.Profile.Weights,
			Constraints: u.Profile.Constraints,
			CreatedAt:   u.Profile.CreatedAt,
			UpdatedAt:   u.Profile.UpdatedAt,
		}
		// Copy pointer fields
		if u.Profile.Constraints.PreferParks != nil {
			val := *u.Profile.Constraints.PreferParks
			userCopy.Profile.Constraints.PreferParks = &val
		}
		if u.Profile.Constraints.MaxExtraMinutesVsFastest != nil {
			val := *u.Profile.Constraints.MaxExtraMinutesVsFastest
			userCopy.Profile.Constraints.MaxExtraMinutesVsFastest = &val
		}
		if u.Profile.Constraints.MaxTransfers != nil {
			val := *u.Profile.Constraints.MaxTransfers
			userCopy.Profile.Constraints.MaxTransfers = &val
		}
	}

	if u.Consents != nil {
		userCopy.Consents = &Consents{
			Analytics:         u.Consents.Analytics,
			Marketing:         u.Consents.Marketing,
			PushNotifications: u.Consents.PushNotifications,
			UpdatedAt:         u.Consents.UpdatedAt,
		}
	}

	return userCopy
}
