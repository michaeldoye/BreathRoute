package auth

import (
	"context"
	"sync"
	"time"
)

// InMemoryUserRepository is an in-memory implementation of UserRepository.
// This is intended for MVP/testing. Production should use a database-backed implementation.
type InMemoryUserRepository struct {
	mu      sync.RWMutex
	users   map[string]*User  // keyed by user ID
	byApple map[string]string // appleSub -> userID
}

// NewInMemoryUserRepository creates a new in-memory user repository.
func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{
		users:   make(map[string]*User),
		byApple: make(map[string]string),
	}
}

// FindByAppleSub finds a user by their Apple subject identifier.
func (r *InMemoryUserRepository) FindByAppleSub(_ context.Context, appleSub string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.byApple[appleSub]
	if !ok {
		return nil, ErrUserNotFound
	}

	user, ok := r.users[userID]
	if !ok {
		return nil, ErrUserNotFound
	}

	// Return a copy to avoid mutation
	userCopy := *user
	return &userCopy, nil
}

// Create creates a new user.
func (r *InMemoryUserRepository) Create(_ context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Store the user
	userCopy := *user
	r.users[user.ID] = &userCopy
	r.byApple[user.AppleSub] = user.ID

	return nil
}

// FindByID finds a user by their internal ID.
func (r *InMemoryUserRepository) FindByID(_ context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	// Return a copy to avoid mutation
	userCopy := *user
	return &userCopy, nil
}

// InMemoryRefreshTokenRepository is an in-memory implementation of RefreshTokenRepository.
// This is intended for MVP/testing. Production should use a database-backed implementation.
type InMemoryRefreshTokenRepository struct {
	mu     sync.RWMutex
	tokens map[string]*RefreshToken // keyed by token value
	byUser map[string][]string      // userID -> list of token values
}

// NewInMemoryRefreshTokenRepository creates a new in-memory refresh token repository.
func NewInMemoryRefreshTokenRepository() *InMemoryRefreshTokenRepository {
	return &InMemoryRefreshTokenRepository{
		tokens: make(map[string]*RefreshToken),
		byUser: make(map[string][]string),
	}
}

// Create stores a new refresh token.
func (r *InMemoryRefreshTokenRepository) Create(_ context.Context, token *RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tokenCopy := *token
	r.tokens[token.Token] = &tokenCopy
	r.byUser[token.UserID] = append(r.byUser[token.UserID], token.Token)

	return nil
}

// FindByToken finds a refresh token by its value.
func (r *InMemoryRefreshTokenRepository) FindByToken(_ context.Context, tokenValue string) (*RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[tokenValue]
	if !ok {
		return nil, ErrInvalidRefreshToken
	}

	tokenCopy := *token
	return &tokenCopy, nil
}

// Revoke marks a refresh token as revoked.
func (r *InMemoryRefreshTokenRepository) Revoke(_ context.Context, tokenValue string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	token, ok := r.tokens[tokenValue]
	if !ok {
		return nil // Token not found, consider already revoked
	}

	now := time.Now()
	token.RevokedAt = &now

	return nil
}

// RevokeAllForUser revokes all refresh tokens for a user.
func (r *InMemoryRefreshTokenRepository) RevokeAllForUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tokenValues, ok := r.byUser[userID]
	if !ok {
		return nil
	}

	now := time.Now()
	for _, tokenValue := range tokenValues {
		if token, ok := r.tokens[tokenValue]; ok {
			token.RevokedAt = &now
		}
	}

	return nil
}
