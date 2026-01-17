package featureflags_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/featureflags"
)

func TestService_GetFlag(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()

	// Test getting a default flag
	flag := service.GetFlag(ctx, featureflags.FlagDisableTrainMode)
	if flag == nil {
		t.Fatal("expected flag to be returned")
	}
	if flag.Key != featureflags.FlagDisableTrainMode {
		t.Errorf("expected key %q, got %q", featureflags.FlagDisableTrainMode, flag.Key)
	}
	if flag.BoolValue(true) != false {
		t.Error("expected disable_train_mode to be false by default")
	}
}

func TestService_SetFlag(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()

	// Set a flag
	err := service.SetFlag(ctx, &featureflags.Flag{
		Key:   featureflags.FlagDisableTrainMode,
		Value: true,
	})
	if err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Verify it was updated
	flag := service.GetFlag(ctx, featureflags.FlagDisableTrainMode)
	if flag == nil {
		t.Fatal("expected flag to be returned")
	}
	if flag.BoolValue(false) != true {
		t.Error("expected disable_train_mode to be true after update")
	}
}

func TestService_SetFlags(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()

	// Set multiple flags
	err := service.SetFlags(ctx, []*featureflags.Flag{
		{Key: featureflags.FlagDisableTrainMode, Value: true},
		{Key: featureflags.FlagCachedOnlyAirQuality, Value: true},
	})
	if err != nil {
		t.Fatalf("failed to set flags: %v", err)
	}

	// Verify both were updated
	if !service.IsTrainModeDisabled(ctx) {
		t.Error("expected train mode to be disabled")
	}
	if !service.IsCachedOnlyAirQuality(ctx) {
		t.Error("expected cached only air quality to be true")
	}
}

func TestService_GetAllFlags(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()
	flags := service.GetAllFlags(ctx)

	// Should have all default flags
	expectedFlags := []string{
		featureflags.FlagDisableTrainMode,
		featureflags.FlagCachedOnlyAirQuality,
		featureflags.FlagDisableAlertsSending,
		featureflags.FlagDisablePollenFactor,
		featureflags.FlagRoutingBikeOnly,
		featureflags.FlagEnableTimeShift,
	}

	for _, key := range expectedFlags {
		if _, ok := flags[key]; !ok {
			t.Errorf("expected flag %q to be present", key)
		}
	}
}

func TestService_InvalidateCache(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Hour, // Long TTL to test cache
	})

	ctx := context.Background()

	// Get a flag to populate cache
	_ = service.GetFlag(ctx, featureflags.FlagDisableTrainMode)

	// Directly update the repository (bypassing service)
	_ = repo.SetFlag(ctx, &featureflags.Flag{
		Key:   featureflags.FlagDisableTrainMode,
		Value: true,
	})

	// Without invalidation, cache should still return old value
	// (Note: this depends on implementation details, but tests the concept)

	// Invalidate cache
	service.InvalidateCache()

	// Now should get fresh value from repository
	flag := service.GetFlag(ctx, featureflags.FlagDisableTrainMode)
	if flag.BoolValue(false) != true {
		t.Error("expected updated value after cache invalidation")
	}
}

func TestService_IsEnabled(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()

	// Default flags should be disabled (except time_shift)
	if service.IsEnabled(ctx, featureflags.FlagDisableTrainMode) {
		t.Error("expected disable_train_mode to be disabled by default")
	}

	if !service.IsEnabled(ctx, featureflags.FlagEnableTimeShift) {
		t.Error("expected enable_time_shift to be enabled by default")
	}

	// IsDisabled should be inverse
	if !service.IsDisabled(ctx, featureflags.FlagDisableTrainMode) {
		t.Error("expected IsDisabled to return true for disabled flag")
	}
}

func TestService_ConvenienceMethods(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository: repo,
		Logger:     zerolog.Nop(),
		CacheTTL:   1 * time.Minute,
	})

	ctx := context.Background()

	// Test all convenience methods with default values
	if service.IsTrainModeDisabled(ctx) {
		t.Error("expected train mode to not be disabled by default")
	}
	if service.IsCachedOnlyAirQuality(ctx) {
		t.Error("expected cached only air quality to be false by default")
	}
	if service.IsAlertsSendingDisabled(ctx) {
		t.Error("expected alerts sending to not be disabled by default")
	}
	if service.IsPollenFactorDisabled(ctx) {
		t.Error("expected pollen factor to not be disabled by default")
	}
	if service.IsBikeOnlyRouting(ctx) {
		t.Error("expected bike only routing to be false by default")
	}
	if !service.IsTimeShiftEnabled(ctx) {
		t.Error("expected time shift to be enabled by default")
	}
}

func TestFlag_ValueHelpers(t *testing.T) {
	tests := []struct {
		name          string
		value         interface{}
		wantBool      bool
		wantString    string
		wantInt       int
		wantFloat     float64
		defaultBool   bool
		defaultString string
		defaultInt    int
		defaultFloat  float64
	}{
		{
			name:          "boolean true",
			value:         true,
			wantBool:      true,
			wantString:    "default",
			wantInt:       42,
			wantFloat:     3.14,
			defaultBool:   false,
			defaultString: "default",
			defaultInt:    42,
			defaultFloat:  3.14,
		},
		{
			name:          "boolean false",
			value:         false,
			wantBool:      false,
			defaultBool:   true,
			defaultString: "default",
			defaultInt:    42,
			defaultFloat:  3.14,
			wantString:    "default",
			wantInt:       42,
			wantFloat:     3.14,
		},
		{
			name:          "string value",
			value:         "hello",
			wantBool:      false,
			wantString:    "hello",
			wantInt:       42,
			wantFloat:     3.14,
			defaultBool:   false,
			defaultString: "default",
			defaultInt:    42,
			defaultFloat:  3.14,
		},
		{
			name:          "float64 value",
			value:         42.5,
			wantBool:      true, // non-zero
			wantString:    "default",
			wantInt:       42,
			wantFloat:     42.5,
			defaultBool:   false,
			defaultString: "default",
			defaultInt:    0,
			defaultFloat:  0.0,
		},
		{
			name:          "int value (as float64 from JSON)",
			value:         float64(100),
			wantBool:      true, // non-zero
			wantString:    "default",
			wantInt:       100,
			wantFloat:     100.0,
			defaultBool:   false,
			defaultString: "default",
			defaultInt:    0,
			defaultFloat:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := &featureflags.Flag{
				Key:       "test",
				Value:     tt.value,
				UpdatedAt: time.Now(),
			}

			if got := flag.BoolValue(tt.defaultBool); got != tt.wantBool {
				t.Errorf("BoolValue() = %v, want %v", got, tt.wantBool)
			}
			if got := flag.StringValue(tt.defaultString); got != tt.wantString {
				t.Errorf("StringValue() = %v, want %v", got, tt.wantString)
			}
			if got := flag.IntValue(tt.defaultInt); got != tt.wantInt {
				t.Errorf("IntValue() = %v, want %v", got, tt.wantInt)
			}
			if got := flag.Float64Value(tt.defaultFloat); got != tt.wantFloat {
				t.Errorf("Float64Value() = %v, want %v", got, tt.wantFloat)
			}
		})
	}
}

func TestFlag_NilFlag(t *testing.T) {
	var flag *featureflags.Flag

	if flag.BoolValue(true) != true {
		t.Error("expected default value for nil flag")
	}
	if flag.StringValue("default") != "default" {
		t.Error("expected default value for nil flag")
	}
	if flag.IntValue(42) != 42 {
		t.Error("expected default value for nil flag")
	}
	if flag.Float64Value(3.14) != 3.14 {
		t.Error("expected default value for nil flag")
	}
}

func TestInMemoryRepository_GetFlag_NotFound(t *testing.T) {
	repo := featureflags.NewInMemoryRepositoryWithFlags(make(map[string]*featureflags.Flag))
	ctx := context.Background()

	_, err := repo.GetFlag(ctx, "nonexistent")
	if !errors.Is(err, featureflags.ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestInMemoryRepository_DeleteFlag(t *testing.T) {
	repo := featureflags.NewInMemoryRepository()
	ctx := context.Background()

	// Delete existing flag
	err := repo.DeleteFlag(ctx, featureflags.FlagDisableTrainMode)
	if err != nil {
		t.Fatalf("failed to delete flag: %v", err)
	}

	// Should not be found now
	_, err = repo.GetFlag(ctx, featureflags.FlagDisableTrainMode)
	if !errors.Is(err, featureflags.ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound after delete, got %v", err)
	}

	// Delete non-existent flag should error
	err = repo.DeleteFlag(ctx, "nonexistent")
	if !errors.Is(err, featureflags.ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound for non-existent flag, got %v", err)
	}
}

func TestService_FallbackToDefaults(t *testing.T) {
	// Create service with empty repository but defaults
	repo := featureflags.NewInMemoryRepositoryWithFlags(make(map[string]*featureflags.Flag))
	service := featureflags.NewService(featureflags.ServiceConfig{
		Repository:   repo,
		Logger:       zerolog.Nop(),
		CacheTTL:     1 * time.Minute,
		DefaultFlags: featureflags.DefaultFlags(),
	})

	ctx := context.Background()

	// Should fallback to default value
	flag := service.GetFlag(ctx, featureflags.FlagEnableTimeShift)
	if flag == nil {
		t.Fatal("expected flag to be returned from defaults")
	}
	if flag.BoolValue(false) != true {
		t.Error("expected enable_time_shift to be true from defaults")
	}
}
