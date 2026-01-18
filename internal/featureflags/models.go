// Package featureflags provides feature flag management.
package featureflags

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

// ErrFlagNotFound is returned when a feature flag is not found.
var ErrFlagNotFound = errors.New("feature flag not found")

// Feature flag keys.
const (
	// FlagDisablePollenFactor disables pollen factor in route calculations.
	FlagDisablePollenFactor = "pollen_factor_disabled"
)

// Flag represents a feature flag.
type Flag struct {
	Key       string
	Value     interface{}
	UpdatedAt time.Time
}

// Repository defines the interface for feature flag storage.
type Repository interface {
	GetFlag(ctx context.Context, key string) (*Flag, error)
	GetAllFlags(ctx context.Context) (map[string]*Flag, error)
	SetFlag(ctx context.Context, flag *Flag) error
	SetFlags(ctx context.Context, flags []*Flag) error
	DeleteFlag(ctx context.Context, key string) error
}

// ServiceConfig holds configuration for the feature flags service.
type ServiceConfig struct {
	Repository Repository
	Logger     zerolog.Logger
	CacheTTL   time.Duration
}

// Service provides feature flag functionality.
type Service struct {
	repo     Repository
	logger   zerolog.Logger
	cacheTTL time.Duration
}

// NewService creates a new feature flags service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = time.Minute
	}
	return &Service{
		repo:     cfg.Repository,
		logger:   cfg.Logger,
		cacheTTL: cacheTTL,
	}
}

// IsPollenFactorDisabled checks if the pollen factor is disabled.
func (s *Service) IsPollenFactorDisabled(ctx context.Context) bool {
	if s == nil || s.repo == nil {
		return false
	}
	flag, err := s.repo.GetFlag(ctx, "pollen_factor_disabled")
	if err != nil {
		return false
	}
	if disabled, ok := flag.Value.(bool); ok {
		return disabled
	}
	return false
}
