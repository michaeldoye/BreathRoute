package telemetry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/telemetry"
)

func TestInit_Disabled(t *testing.T) {
	ctx := context.Background()

	provider, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		OTLPEndpoint:   "localhost:4317",
		Enabled:        false,
	})

	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.Tracer)
	assert.NotNil(t, provider.Meter)

	// Noop provider should have nil TracerProvider and MeterProvider
	assert.Nil(t, provider.TracerProvider)
	assert.Nil(t, provider.MeterProvider)

	// Shutdown should not error
	err = provider.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestProvider_Shutdown_NilProviders(t *testing.T) {
	provider := &telemetry.Provider{}
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestTracer_ReturnsGlobalTracer(t *testing.T) {
	tracer := telemetry.Tracer("test-tracer")
	assert.NotNil(t, tracer)
}

func TestMeter_ReturnsGlobalMeter(t *testing.T) {
	meter := telemetry.Meter("test-meter")
	assert.NotNil(t, meter)
}
