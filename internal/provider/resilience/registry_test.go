package resilience_test

import (
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

func TestRegistry_RegisterAndGetHealth(t *testing.T) {
	registry := resilience.NewRegistry()
	cfg := resilience.DefaultClientConfig("test-provider")
	cfg.Registry = registry

	client := resilience.NewClient(cfg)

	// Client should be registered
	assert.Equal(t, 1, registry.ProviderCount())

	// Get health
	health := registry.GetHealth("test-provider")
	require.NotNil(t, health)
	assert.Equal(t, "test-provider", health.Name)
	assert.Equal(t, gobreaker.StateClosed, health.CircuitState)
	assert.True(t, health.IsHealthy())
	assert.False(t, health.IsDegraded())
	assert.False(t, health.IsUnhealthy())

	// Verify client name
	assert.Equal(t, "test-provider", client.Name())
}

func TestRegistry_Unregister(t *testing.T) {
	registry := resilience.NewRegistry()
	cfg := resilience.DefaultClientConfig("test-provider")
	cfg.Registry = registry

	_ = resilience.NewClient(cfg)

	assert.Equal(t, 1, registry.ProviderCount())

	registry.Unregister("test-provider")

	assert.Equal(t, 0, registry.ProviderCount())
	assert.Nil(t, registry.GetHealth("test-provider"))
}

func TestRegistry_RecordSuccess(t *testing.T) {
	registry := resilience.NewRegistry()
	cfg := resilience.DefaultClientConfig("test-provider")
	cfg.Registry = registry

	_ = resilience.NewClient(cfg)

	// Before recording success
	health := registry.GetHealth("test-provider")
	require.NotNil(t, health)
	assert.Nil(t, health.LastSuccessAt)

	// Record success
	registry.RecordSuccess("test-provider")

	// After recording success
	health = registry.GetHealth("test-provider")
	require.NotNil(t, health)
	require.NotNil(t, health.LastSuccessAt)
	assert.WithinDuration(t, time.Now(), *health.LastSuccessAt, time.Second)
}

func TestRegistry_RecordFailure(t *testing.T) {
	registry := resilience.NewRegistry()
	cfg := resilience.DefaultClientConfig("test-provider")
	cfg.Registry = registry

	_ = resilience.NewClient(cfg)

	// Before recording failure
	health := registry.GetHealth("test-provider")
	require.NotNil(t, health)
	assert.Nil(t, health.LastFailureAt)
	assert.Empty(t, health.LastError)

	// Record failure
	registry.RecordFailure("test-provider", assert.AnError)

	// After recording failure
	health = registry.GetHealth("test-provider")
	require.NotNil(t, health)
	require.NotNil(t, health.LastFailureAt)
	assert.WithinDuration(t, time.Now(), *health.LastFailureAt, time.Second)
	assert.Equal(t, assert.AnError.Error(), health.LastError)
}

func TestRegistry_GetAllHealth(t *testing.T) {
	registry := resilience.NewRegistry()

	// Register multiple providers
	for _, name := range []string{"provider-a", "provider-b", "provider-c"} {
		cfg := resilience.DefaultClientConfig(name)
		cfg.Registry = registry
		_ = resilience.NewClient(cfg)
	}

	healthList := registry.GetAllHealth()
	assert.Len(t, healthList, 3)

	names := make(map[string]bool)
	for _, h := range healthList {
		names[h.Name] = true
		assert.Equal(t, gobreaker.StateClosed, h.CircuitState)
	}

	assert.True(t, names["provider-a"])
	assert.True(t, names["provider-b"])
	assert.True(t, names["provider-c"])
}

func TestRegistry_GetProviderNames(t *testing.T) {
	registry := resilience.NewRegistry()

	// Empty registry
	names := registry.GetProviderNames()
	assert.Empty(t, names)

	// Add providers
	for _, name := range []string{"provider-a", "provider-b"} {
		cfg := resilience.DefaultClientConfig(name)
		cfg.Registry = registry
		_ = resilience.NewClient(cfg)
	}

	names = registry.GetProviderNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "provider-a")
	assert.Contains(t, names, "provider-b")
}

func TestRegistry_GetHealthNotFound(t *testing.T) {
	registry := resilience.NewRegistry()

	health := registry.GetHealth("nonexistent")
	assert.Nil(t, health)
}

func TestRegistry_RecordSuccessNotFound(t *testing.T) {
	registry := resilience.NewRegistry()

	// Should not panic
	registry.RecordSuccess("nonexistent")
}

func TestRegistry_RecordFailureNotFound(t *testing.T) {
	registry := resilience.NewRegistry()

	// Should not panic
	registry.RecordFailure("nonexistent", assert.AnError)
}

func TestGlobalRegistry(t *testing.T) {
	// Global registry should exist
	assert.NotNil(t, resilience.GlobalRegistry)
}

func TestProviderHealth_States(t *testing.T) {
	tests := []struct {
		state      gobreaker.State
		isHealthy  bool
		isDegraded bool
		isUnhealth bool
	}{
		{gobreaker.StateClosed, true, false, false},
		{gobreaker.StateHalfOpen, false, true, false},
		{gobreaker.StateOpen, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			h := &resilience.ProviderHealth{CircuitState: tt.state}
			assert.Equal(t, tt.isHealthy, h.IsHealthy())
			assert.Equal(t, tt.isDegraded, h.IsDegraded())
			assert.Equal(t, tt.isUnhealth, h.IsUnhealthy())
		})
	}
}
