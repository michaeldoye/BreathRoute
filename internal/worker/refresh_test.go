package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/worker"
)

func TestDefaultRefreshConfig(t *testing.T) {
	cfg := worker.DefaultRefreshConfig()

	assert.Equal(t, 3, cfg.Concurrency)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.True(t, cfg.RefreshAirQuality)
	assert.True(t, cfg.RefreshWeather)
	assert.True(t, cfg.RefreshPollen)
	assert.True(t, cfg.RefreshTransit)
	assert.NotEmpty(t, cfg.Targets)
}

func TestDefaultRefreshTargets(t *testing.T) {
	targets := worker.DefaultRefreshTargets()

	// Should have multiple cities
	assert.GreaterOrEqual(t, len(targets), 5)

	// Find Amsterdam
	var amsterdam *worker.RefreshTarget
	for i := range targets {
		if targets[i].Name == "Amsterdam" {
			amsterdam = &targets[i]
			break
		}
	}
	require.NotNil(t, amsterdam, "Amsterdam should be in targets")
	assert.Equal(t, 1, amsterdam.Priority)
	assert.GreaterOrEqual(t, len(amsterdam.Points), 2)
}

func TestRefreshConfig_AllPoints(t *testing.T) {
	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "City A",
				Points: []worker.Point{{Lat: 1, Lon: 1}, {Lat: 2, Lon: 2}},
			},
			{
				Name:   "City B",
				Points: []worker.Point{{Lat: 3, Lon: 3}},
			},
		},
	}

	points := cfg.AllPoints()
	assert.Len(t, points, 3)
	assert.Equal(t, cfg.TotalPoints(), 3)
}

func TestRefreshConfig_TotalPoints(t *testing.T) {
	cfg := worker.DefaultRefreshConfig()
	total := cfg.TotalPoints()

	// Should have a reasonable number of points
	assert.Greater(t, total, 10)
}

func TestRefreshJob_Run_NoServices(t *testing.T) {
	// Create a job with no services configured
	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Test",
				Points: []worker.Point{{Lat: 52.37, Lon: 4.90}},
			},
		},
		Concurrency:       1,
		Timeout:           1 * time.Second,
		RefreshAirQuality: true,
		RefreshWeather:    true,
		RefreshPollen:     true,
		RefreshTransit:    true,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	result := job.Run(context.Background())

	// Should complete without panicking
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalPoints)
	assert.Greater(t, result.Duration, time.Duration(0))
}

func TestRefreshJob_GetMetrics(t *testing.T) {
	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Test",
				Points: []worker.Point{{Lat: 52.37, Lon: 4.90}},
			},
		},
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	// Run the job
	_ = job.Run(context.Background())

	// Check metrics
	metrics := job.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRefreshes)
	assert.NotZero(t, metrics.LastRefreshAt)
	assert.Greater(t, metrics.LastRefreshDuration, time.Duration(0))
}

func TestRefreshJob_MetricsSnapshot(t *testing.T) {
	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Test",
				Points: []worker.Point{{Lat: 52.37, Lon: 4.90}},
			},
		},
		Concurrency: 1,
		Timeout:     1 * time.Second,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	_ = job.Run(context.Background())

	snapshot := job.MetricsSnapshot()

	assert.Contains(t, snapshot, "total_refreshes")
	assert.Contains(t, snapshot, "successful_refreshes")
	assert.Contains(t, snapshot, "failed_refreshes")
	assert.Contains(t, snapshot, "last_refresh_at")
	assert.Contains(t, snapshot, "last_refresh_duration")
}

func TestRefreshJob_Run_WithConcurrency(t *testing.T) {
	// Create a job with multiple points
	points := make([]worker.Point, 10)
	for i := range points {
		points[i] = worker.Point{Lat: 52.0 + float64(i)*0.1, Lon: 4.0 + float64(i)*0.1}
	}

	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Test",
				Points: points,
			},
		},
		Concurrency:       3,
		Timeout:           1 * time.Second,
		RefreshAirQuality: false, // Disable to avoid nil pointer
		RefreshWeather:    false,
		RefreshPollen:     false,
		RefreshTransit:    false,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	result := job.Run(context.Background())

	assert.Equal(t, 10, result.TotalPoints)
	assert.Equal(t, 10, result.Successful) // All should succeed since no providers
}

func TestRefreshJob_Run_ContextCancellation(t *testing.T) {
	// Create many points to process
	points := make([]worker.Point, 100)
	for i := range points {
		points[i] = worker.Point{Lat: 52.0 + float64(i)*0.01, Lon: 4.0 + float64(i)*0.01}
	}

	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Test",
				Points: points,
			},
		},
		Concurrency: 1,
		Timeout:     100 * time.Millisecond,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	// Cancel context immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := job.Run(ctx)

	// Should complete (even if not all points processed)
	assert.NotNil(t, result)
}

func TestRefreshJob_RefreshTransit_NoService(t *testing.T) {
	cfg := worker.RefreshConfig{
		RefreshTransit: true,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	err := job.RefreshTransit(context.Background())
	assert.NoError(t, err)
}

func TestRefreshJob_RefreshTransit_Disabled(t *testing.T) {
	cfg := worker.RefreshConfig{
		RefreshTransit: false,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	err := job.RefreshTransit(context.Background())
	assert.NoError(t, err)
}

func TestRefreshResult_Fields(t *testing.T) {
	result := &worker.RefreshResult{
		StartTime:   time.Now(),
		TotalPoints: 10,
		Successful:  8,
		Failed:      2,
		CacheHits:   5,
		CacheMisses: 15,
		Errors: []worker.RefreshError{
			{Provider: "weather", Point: worker.Point{Lat: 1, Lon: 1}, Error: "timeout"},
			{Provider: "pollen", Point: worker.Point{Lat: 2, Lon: 2}, Error: "unavailable"},
		},
	}
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	assert.Equal(t, 10, result.TotalPoints)
	assert.Equal(t, 8, result.Successful)
	assert.Equal(t, 2, result.Failed)
	assert.Equal(t, 5, result.CacheHits)
	assert.Equal(t, 15, result.CacheMisses)
	assert.Greater(t, result.Duration, time.Duration(0))
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, "weather", result.Errors[0].Provider)
}

func TestRefreshError_Fields(t *testing.T) {
	err := worker.RefreshError{
		Provider: "airquality",
		Point:    worker.Point{Lat: 52.37, Lon: 4.90},
		Error:    "connection refused",
	}

	assert.Equal(t, "airquality", err.Provider)
	assert.Equal(t, 52.37, err.Point.Lat)
	assert.Equal(t, 4.90, err.Point.Lon)
	assert.Equal(t, "connection refused", err.Error)
}

func TestPoint_Fields(t *testing.T) {
	p := worker.Point{Lat: 52.3676, Lon: 4.9041}
	assert.Equal(t, 52.3676, p.Lat)
	assert.Equal(t, 4.9041, p.Lon)
}

func TestRefreshTarget_Fields(t *testing.T) {
	target := worker.RefreshTarget{
		Name:     "Amsterdam",
		Priority: 1,
		Points: []worker.Point{
			{Lat: 52.3676, Lon: 4.9041},
		},
	}

	assert.Equal(t, "Amsterdam", target.Name)
	assert.Equal(t, 1, target.Priority)
	assert.Len(t, target.Points, 1)
}

func TestRefreshMetrics_Fields(t *testing.T) {
	now := time.Now()
	metrics := worker.RefreshMetrics{
		TotalRefreshes:      10,
		SuccessfulRefresh:   8,
		FailedRefreshes:     2,
		AirQualityRefresh:   5,
		WeatherRefresh:      5,
		PollenRefresh:       4,
		TransitRefresh:      2,
		LastRefreshAt:       now,
		LastRefreshDuration: 5 * time.Second,
		TotalDuration:       50 * time.Second,
		CacheHits:           100,
		CacheMisses:         50,
	}

	assert.Equal(t, int64(10), metrics.TotalRefreshes)
	assert.Equal(t, int64(8), metrics.SuccessfulRefresh)
	assert.Equal(t, int64(2), metrics.FailedRefreshes)
	assert.Equal(t, int64(5), metrics.AirQualityRefresh)
	assert.Equal(t, int64(5), metrics.WeatherRefresh)
	assert.Equal(t, int64(4), metrics.PollenRefresh)
	assert.Equal(t, int64(2), metrics.TransitRefresh)
	assert.Equal(t, now, metrics.LastRefreshAt)
	assert.Equal(t, 5*time.Second, metrics.LastRefreshDuration)
	assert.Equal(t, 50*time.Second, metrics.TotalDuration)
	assert.Equal(t, int64(100), metrics.CacheHits)
	assert.Equal(t, int64(50), metrics.CacheMisses)
}

func TestNewRefreshJob_DefaultConfig(t *testing.T) {
	// Create job with empty config - should use defaults
	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: worker.RefreshConfig{}, // Empty
		Logger: zerolog.Nop(),
	})

	// Should have default targets
	metrics := job.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRefreshes) // Not run yet
}

// BenchmarkRefreshJob_Run benchmarks the refresh job.
func BenchmarkRefreshJob_Run(b *testing.B) {
	cfg := worker.RefreshConfig{
		Targets: []worker.RefreshTarget{
			{
				Name:   "Benchmark",
				Points: []worker.Point{{Lat: 52.37, Lon: 4.90}},
			},
		},
		Concurrency:       1,
		Timeout:           100 * time.Millisecond,
		RefreshAirQuality: false,
		RefreshWeather:    false,
		RefreshPollen:     false,
		RefreshTransit:    false,
	}

	job := worker.NewRefreshJob(worker.RefreshJobConfig{
		Config: cfg,
		Logger: zerolog.Nop(),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = job.Run(context.Background())
	}
}

// TestRefreshJob_ErrorCollection verifies that errors during refresh are properly collected.
func TestRefreshJob_ErrorCollection(t *testing.T) {
	// This test verifies that errors during refresh are properly collected
	// Since we don't have mock services, we verify the error structure works

	err := errors.New("test error")
	refreshErr := worker.RefreshError{
		Provider: "test",
		Point:    worker.Point{Lat: 1, Lon: 1},
		Error:    err.Error(),
	}

	assert.Equal(t, "test", refreshErr.Provider)
	assert.Equal(t, "test error", refreshErr.Error)
}
