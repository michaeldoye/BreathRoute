package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/airquality"
	"github.com/breatheroute/breatheroute/internal/pollen"
	"github.com/breatheroute/breatheroute/internal/transit"
	"github.com/breatheroute/breatheroute/internal/weather"
)

// RefreshJob handles provider cache refresh operations.
type RefreshJob struct {
	config RefreshConfig
	logger zerolog.Logger

	// Services (optional, nil if not configured)
	airQualityService *airquality.Service
	weatherService    *weather.Service
	pollenService     *pollen.Service
	transitService    *transit.Service

	// Metrics
	metrics *RefreshMetrics
}

// RefreshMetrics tracks refresh job statistics.
type RefreshMetrics struct {
	mu sync.RWMutex

	// Counters
	TotalRefreshes    int64
	SuccessfulRefresh int64
	FailedRefreshes   int64
	AirQualityRefresh int64
	WeatherRefresh    int64
	PollenRefresh     int64
	TransitRefresh    int64

	// Timings
	LastRefreshAt       time.Time
	LastRefreshDuration time.Duration
	TotalDuration       time.Duration

	// Cache stats
	CacheHits   int64
	CacheMisses int64
}

// RefreshJobConfig holds configuration for creating a RefreshJob.
type RefreshJobConfig struct {
	Config            RefreshConfig
	Logger            zerolog.Logger
	AirQualityService *airquality.Service
	WeatherService    *weather.Service
	PollenService     *pollen.Service
	TransitService    *transit.Service
}

// NewRefreshJob creates a new refresh job processor.
func NewRefreshJob(cfg RefreshJobConfig) *RefreshJob {
	config := cfg.Config
	if len(config.Targets) == 0 {
		config = DefaultRefreshConfig()
	}

	return &RefreshJob{
		config:            config,
		logger:            cfg.Logger,
		airQualityService: cfg.AirQualityService,
		weatherService:    cfg.WeatherService,
		pollenService:     cfg.PollenService,
		transitService:    cfg.TransitService,
		metrics:           &RefreshMetrics{},
	}
}

// RefreshResult contains the result of a refresh operation.
type RefreshResult struct {
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	TotalPoints int
	Successful  int
	Failed      int
	Errors      []RefreshError
	CacheHits   int
	CacheMisses int
}

// RefreshError represents an error during refresh.
type RefreshError struct {
	Provider string
	Point    Point
	Error    string
}

// Run executes the refresh job for all configured targets.
func (j *RefreshJob) Run(ctx context.Context) *RefreshResult {
	startTime := time.Now()
	result := &RefreshResult{
		StartTime:   startTime,
		TotalPoints: j.config.TotalPoints(),
	}

	j.logger.Info().
		Int("total_points", result.TotalPoints).
		Int("concurrency", j.config.Concurrency).
		Msg("starting provider refresh job")

	// Get all points to refresh
	points := j.config.AllPoints()

	// Create work channels
	pointsChan := make(chan Point, len(points))
	resultsChan := make(chan pointResult, len(points))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < j.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			j.refreshWorker(ctx, workerID, pointsChan, resultsChan)
		}(i)
	}

	// Send points to workers
	for _, p := range points {
		pointsChan <- p
	}
	close(pointsChan)

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for pr := range resultsChan {
		if pr.success {
			result.Successful++
		} else {
			result.Failed++
		}
		result.CacheHits += pr.cacheHits
		result.CacheMisses += pr.cacheMisses
		result.Errors = append(result.Errors, pr.errors...)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(startTime)

	// Update metrics
	j.updateMetrics(result)

	j.logger.Info().
		Dur("duration", result.Duration).
		Int("successful", result.Successful).
		Int("failed", result.Failed).
		Int("cache_hits", result.CacheHits).
		Int("cache_misses", result.CacheMisses).
		Msg("provider refresh job completed")

	return result
}

type pointResult struct {
	point       Point
	success     bool
	cacheHits   int
	cacheMisses int
	errors      []RefreshError
}

func (j *RefreshJob) refreshWorker(ctx context.Context, _ int, points <-chan Point, results chan<- pointResult) {
	for point := range points {
		select {
		case <-ctx.Done():
			return
		default:
			result := j.refreshPoint(ctx, point)
			results <- result
		}
	}
}

func (j *RefreshJob) refreshPoint(ctx context.Context, point Point) pointResult {
	result := pointResult{
		point:   point,
		success: true,
	}

	// Create timeout context for this point
	pointCtx, cancel := context.WithTimeout(ctx, j.config.Timeout)
	defer cancel()

	// Refresh air quality
	if j.config.RefreshAirQuality && j.airQualityService != nil {
		if err := j.refreshAirQuality(pointCtx, point); err != nil {
			result.errors = append(result.errors, RefreshError{
				Provider: "airquality",
				Point:    point,
				Error:    err.Error(),
			})
			result.success = false
		} else {
			result.cacheMisses++ // Successful refresh means cache was updated
			atomic.AddInt64(&j.metrics.AirQualityRefresh, 1)
		}
	}

	// Refresh weather
	if j.config.RefreshWeather && j.weatherService != nil {
		if err := j.refreshWeather(pointCtx, point); err != nil {
			result.errors = append(result.errors, RefreshError{
				Provider: "weather",
				Point:    point,
				Error:    err.Error(),
			})
			result.success = false
		} else {
			result.cacheMisses++
			atomic.AddInt64(&j.metrics.WeatherRefresh, 1)
		}
	}

	// Refresh pollen
	if j.config.RefreshPollen && j.pollenService != nil {
		if err := j.refreshPollen(pointCtx, point); err != nil {
			result.errors = append(result.errors, RefreshError{
				Provider: "pollen",
				Point:    point,
				Error:    err.Error(),
			})
			// Pollen errors are non-fatal (feature flag may disable it)
		} else {
			result.cacheMisses++
			atomic.AddInt64(&j.metrics.PollenRefresh, 1)
		}
	}

	return result
}

func (j *RefreshJob) refreshAirQuality(ctx context.Context, _ Point) error {
	// Air quality data is station-based, so we just refresh the snapshot
	// which triggers a fetch from the provider if cache is stale
	_, err := j.airQualityService.GetSnapshot(ctx)
	return err
}

func (j *RefreshJob) refreshWeather(ctx context.Context, point Point) error {
	_, err := j.weatherService.GetCurrentWeather(ctx, point.Lat, point.Lon)
	return err
}

func (j *RefreshJob) refreshPollen(ctx context.Context, point Point) error {
	_, err := j.pollenService.GetRegionalPollen(ctx, point.Lat, point.Lon)
	if errors.Is(err, pollen.ErrPollenDisabled) {
		// Not an error if pollen is disabled by feature flag
		return nil
	}
	return err
}

// RefreshTransit refreshes transit disruption data.
// Transit is not location-based, so we refresh all disruptions.
func (j *RefreshJob) RefreshTransit(ctx context.Context) error {
	if !j.config.RefreshTransit || j.transitService == nil {
		return nil
	}

	j.logger.Debug().Msg("refreshing transit disruptions")

	_, err := j.transitService.GetAllDisruptions(ctx)
	if err != nil {
		j.logger.Error().Err(err).Msg("failed to refresh transit disruptions")
		return err
	}

	atomic.AddInt64(&j.metrics.TransitRefresh, 1)
	return nil
}

func (j *RefreshJob) updateMetrics(result *RefreshResult) {
	j.metrics.mu.Lock()
	defer j.metrics.mu.Unlock()

	j.metrics.TotalRefreshes++
	j.metrics.SuccessfulRefresh += int64(result.Successful)
	j.metrics.FailedRefreshes += int64(result.Failed)
	j.metrics.LastRefreshAt = result.EndTime
	j.metrics.LastRefreshDuration = result.Duration
	j.metrics.TotalDuration += result.Duration
	j.metrics.CacheHits += int64(result.CacheHits)
	j.metrics.CacheMisses += int64(result.CacheMisses)
}

// GetMetrics returns a copy of the current metrics.
func (j *RefreshJob) GetMetrics() RefreshMetrics {
	j.metrics.mu.RLock()
	defer j.metrics.mu.RUnlock()

	return RefreshMetrics{
		TotalRefreshes:      j.metrics.TotalRefreshes,
		SuccessfulRefresh:   j.metrics.SuccessfulRefresh,
		FailedRefreshes:     j.metrics.FailedRefreshes,
		AirQualityRefresh:   j.metrics.AirQualityRefresh,
		WeatherRefresh:      j.metrics.WeatherRefresh,
		PollenRefresh:       j.metrics.PollenRefresh,
		TransitRefresh:      j.metrics.TransitRefresh,
		LastRefreshAt:       j.metrics.LastRefreshAt,
		LastRefreshDuration: j.metrics.LastRefreshDuration,
		TotalDuration:       j.metrics.TotalDuration,
		CacheHits:           j.metrics.CacheHits,
		CacheMisses:         j.metrics.CacheMisses,
	}
}

// MetricsSnapshot returns a snapshot of the current metrics as a map.
func (j *RefreshJob) MetricsSnapshot() map[string]interface{} {
	m := j.GetMetrics()
	return map[string]interface{}{
		"total_refreshes":       m.TotalRefreshes,
		"successful_refreshes":  m.SuccessfulRefresh,
		"failed_refreshes":      m.FailedRefreshes,
		"airquality_refreshes":  m.AirQualityRefresh,
		"weather_refreshes":     m.WeatherRefresh,
		"pollen_refreshes":      m.PollenRefresh,
		"transit_refreshes":     m.TransitRefresh,
		"last_refresh_at":       m.LastRefreshAt,
		"last_refresh_duration": m.LastRefreshDuration.String(),
		"total_duration":        m.TotalDuration.String(),
		"cache_hits":            m.CacheHits,
		"cache_misses":          m.CacheMisses,
	}
}
