package featureflags

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ServiceConfig holds configuration for the feature flag service.
type ServiceConfig struct {
	Repository   Repository
	Logger       zerolog.Logger
	CacheTTL     time.Duration // How long to cache flags in memory
	DefaultFlags map[string]*Flag
}

// Service provides feature flag evaluation with caching and fallback.
type Service struct {
	repo         Repository
	logger       zerolog.Logger
	cacheTTL     time.Duration
	defaultFlags map[string]*Flag

	mu          sync.RWMutex
	cache       map[string]*Flag
	cacheExpiry time.Time
}

// NewService creates a new feature flag service.
func NewService(cfg ServiceConfig) *Service {
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 1 * time.Minute // Default cache TTL
	}

	defaultFlags := cfg.DefaultFlags
	if defaultFlags == nil {
		defaultFlags = DefaultFlags()
	}

	return &Service{
		repo:         cfg.Repository,
		logger:       cfg.Logger,
		cacheTTL:     cacheTTL,
		defaultFlags: defaultFlags,
		cache:        make(map[string]*Flag),
	}
}

// GetFlag retrieves a feature flag by key.
// Uses cached value if available and not expired, with fallback to defaults.
func (s *Service) GetFlag(ctx context.Context, key string) *Flag {
	// Try cache first
	if flag := s.getCached(key); flag != nil {
		return flag
	}

	// Try repository
	flag, err := s.repo.GetFlag(ctx, key)
	if err == nil {
		s.setCached(key, flag)
		return flag
	}

	// Log error if not just "not found"
	if !errors.Is(err, ErrFlagNotFound) {
		s.logger.Warn().Err(err).Str("flag", key).Msg("failed to get feature flag from repository")
	}

	// Fallback to default
	if defaultFlag, ok := s.defaultFlags[key]; ok {
		return defaultFlag
	}

	return nil
}

// GetAllFlags retrieves all feature flags.
// Returns cached values merged with defaults.
func (s *Service) GetAllFlags(ctx context.Context) map[string]*Flag {
	// Start with defaults
	result := make(map[string]*Flag, len(s.defaultFlags))
	for k, v := range s.defaultFlags {
		result[k] = v
	}

	// Try to get from repository
	flags, err := s.repo.GetAllFlags(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get feature flags from repository, using defaults")
		return result
	}

	// Merge repository flags over defaults
	for k, v := range flags {
		result[k] = v
	}

	// Update cache
	s.mu.Lock()
	s.cache = flags
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.mu.Unlock()

	return result
}

// SetFlag updates a feature flag.
func (s *Service) SetFlag(ctx context.Context, flag *Flag) error {
	flag.UpdatedAt = time.Now()
	if err := s.repo.SetFlag(ctx, flag); err != nil {
		return err
	}

	// Update cache
	s.setCached(flag.Key, flag)
	return nil
}

// SetFlags updates multiple feature flags atomically.
func (s *Service) SetFlags(ctx context.Context, flags []*Flag) error {
	now := time.Now()
	for _, flag := range flags {
		flag.UpdatedAt = now
	}

	if err := s.repo.SetFlags(ctx, flags); err != nil {
		return err
	}

	// Update cache
	s.mu.Lock()
	for _, flag := range flags {
		s.cache[flag.Key] = flag
	}
	s.mu.Unlock()

	return nil
}

// InvalidateCache clears the cached flags, forcing a refresh on next access.
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]*Flag)
	s.cacheExpiry = time.Time{}
}

// IsEnabled returns true if the flag with the given key is enabled (truthy).
// This is a convenience method for boolean flags.
func (s *Service) IsEnabled(ctx context.Context, key string) bool {
	flag := s.GetFlag(ctx, key)
	return flag.BoolValue(false)
}

// IsDisabled returns true if the flag with the given key is disabled.
// This is the inverse of IsEnabled.
func (s *Service) IsDisabled(ctx context.Context, key string) bool {
	return !s.IsEnabled(ctx, key)
}

// getCached retrieves a flag from cache if valid.
func (s *Service) getCached(key string) *Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if time.Now().After(s.cacheExpiry) {
		return nil
	}

	flag, ok := s.cache[key]
	if !ok {
		return nil
	}
	return flag
}

// setCached stores a flag in the cache.
func (s *Service) setCached(key string, flag *Flag) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = flag
	// Extend cache expiry if setting individual flags
	if s.cacheExpiry.Before(time.Now()) {
		s.cacheExpiry = time.Now().Add(s.cacheTTL)
	}
}

// Convenience methods for well-known flags.

// IsTrainModeDisabled returns true if train/transit mode is disabled.
func (s *Service) IsTrainModeDisabled(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagDisableTrainMode)
}

// IsCachedOnlyAirQuality returns true if air quality should only use cached data.
func (s *Service) IsCachedOnlyAirQuality(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagCachedOnlyAirQuality)
}

// IsAlertsSendingDisabled returns true if sending alerts is disabled.
func (s *Service) IsAlertsSendingDisabled(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagDisableAlertsSending)
}

// IsPollenFactorDisabled returns true if pollen factor is disabled.
func (s *Service) IsPollenFactorDisabled(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagDisablePollenFactor)
}

// IsBikeOnlyRouting returns true if routing is restricted to bike mode only.
func (s *Service) IsBikeOnlyRouting(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagRoutingBikeOnly)
}

// IsTimeShiftEnabled returns true if time-shift recommendations are enabled.
func (s *Service) IsTimeShiftEnabled(ctx context.Context) bool {
	return s.IsEnabled(ctx, FlagEnableTimeShift)
}
