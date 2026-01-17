package featureflags

import (
	"context"
	"errors"
)

// ErrFlagNotFound is returned when a feature flag is not found.
var ErrFlagNotFound = errors.New("feature flag not found")

// Repository defines the interface for feature flag storage.
type Repository interface {
	// GetFlag retrieves a single feature flag by key.
	GetFlag(ctx context.Context, key string) (*Flag, error)

	// GetAllFlags retrieves all feature flags.
	GetAllFlags(ctx context.Context) (map[string]*Flag, error)

	// SetFlag creates or updates a feature flag.
	SetFlag(ctx context.Context, flag *Flag) error

	// SetFlags creates or updates multiple feature flags atomically.
	SetFlags(ctx context.Context, flags []*Flag) error

	// DeleteFlag removes a feature flag by key.
	DeleteFlag(ctx context.Context, key string) error
}
