package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"github.com/rs/zerolog"
)

// PubSubHandler handles Pub/Sub messages for the worker.
type PubSubHandler struct {
	client           *pubsub.Client
	subscriber       *pubsub.Subscriber
	subscriptionName string
	refreshJob       *RefreshJob
	logger           zerolog.Logger
}

// PubSubConfig holds configuration for the Pub/Sub handler.
type PubSubConfig struct {
	ProjectID        string
	SubscriptionName string
	RefreshJob       *RefreshJob
	Logger           zerolog.Logger
}

// RefreshMessage represents a provider refresh job message.
type RefreshMessage struct {
	JobType    string `json:"job_type"`
	RefreshAll bool   `json:"refresh_all,omitempty"`
	CheckOnly  bool   `json:"check_only,omitempty"`
}

// NewPubSubHandler creates a new Pub/Sub handler.
func NewPubSubHandler(ctx context.Context, cfg PubSubConfig) (*PubSubHandler, error) {
	client, err := pubsub.NewClient(ctx, cfg.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("creating pubsub client: %w", err)
	}

	subscriber := client.Subscriber(cfg.SubscriptionName)

	// Configure receive settings.
	subscriber.ReceiveSettings.MaxOutstandingMessages = 10
	subscriber.ReceiveSettings.MaxExtension = 10 * time.Minute

	return &PubSubHandler{
		client:           client,
		subscriber:       subscriber,
		subscriptionName: cfg.SubscriptionName,
		refreshJob:       cfg.RefreshJob,
		logger:           cfg.Logger,
	}, nil
}

// Start begins processing Pub/Sub messages.
func (h *PubSubHandler) Start(ctx context.Context) error {
	h.logger.Info().
		Str("subscription", h.subscriptionName).
		Msg("starting pubsub handler")

	return h.subscriber.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		h.handleMessage(ctx, msg)
	})
}

// Close closes the Pub/Sub client.
func (h *PubSubHandler) Close() error {
	return h.client.Close()
}

func (h *PubSubHandler) handleMessage(ctx context.Context, msg *pubsub.Message) {
	startTime := time.Now()

	logger := h.logger.With().
		Str("message_id", msg.ID).
		Str("publish_time", msg.PublishTime.Format(time.RFC3339)).
		Logger()

	logger.Debug().Msg("received pubsub message")

	// Parse message.
	var refreshMsg RefreshMessage
	if err := json.Unmarshal(msg.Data, &refreshMsg); err != nil {
		logger.Error().Err(err).Msg("failed to parse message")
		msg.Nack()
		return
	}

	// Handle based on job type.
	var err error
	switch refreshMsg.JobType {
	case "provider_refresh":
		err = h.handleProviderRefresh(ctx, refreshMsg)
	case "health_check":
		err = h.handleHealthCheck(ctx)
	default:
		logger.Warn().Str("job_type", refreshMsg.JobType).Msg("unknown job type")
		msg.Ack() // Ack unknown messages to prevent redelivery
		return
	}

	if err != nil {
		logger.Error().Err(err).Msg("job failed")
		msg.Nack()
		return
	}

	duration := time.Since(startTime)
	logger.Info().
		Str("job_type", refreshMsg.JobType).
		Dur("duration", duration).
		Msg("job completed successfully")

	msg.Ack()
}

func (h *PubSubHandler) handleProviderRefresh(ctx context.Context, msg RefreshMessage) error {
	h.logger.Info().
		Bool("refresh_all", msg.RefreshAll).
		Msg("starting provider refresh")

	// Run the refresh job.
	result := h.refreshJob.Run(ctx)

	// Also refresh transit data.
	if err := h.refreshJob.RefreshTransit(ctx); err != nil {
		h.logger.Warn().Err(err).Msg("transit refresh failed")
	}

	// Log summary.
	h.logger.Info().
		Dur("duration", result.Duration).
		Int("successful", result.Successful).
		Int("failed", result.Failed).
		Int("total_points", result.TotalPoints).
		Msg("provider refresh completed")

	// Consider it successful if more than half succeeded.
	if result.Failed > result.Successful {
		return fmt.Errorf("too many refresh failures: %d/%d", result.Failed, result.TotalPoints)
	}

	return nil
}

func (h *PubSubHandler) handleHealthCheck(ctx context.Context) error {
	h.logger.Debug().Msg("running health check")

	// Just refresh a single point to verify provider connectivity.
	testPoint := Point{Lat: 52.3676, Lon: 4.9041} // Amsterdam

	// Create a single-point config.
	singlePointConfig := RefreshConfig{
		Targets: []RefreshTarget{
			{
				Name:     "health-check",
				Priority: 1,
				Points:   []Point{testPoint},
			},
		},
		Concurrency:       1,
		Timeout:           10 * time.Second,
		RefreshAirQuality: true,
		RefreshWeather:    true,
		RefreshPollen:     false, // Skip pollen for health check
		RefreshTransit:    false, // Skip transit for health check
	}

	// Create a temporary refresh job for health check.
	healthCheckJob := NewRefreshJob(RefreshJobConfig{
		Config:            singlePointConfig,
		Logger:            h.logger,
		AirQualityService: h.refreshJob.airQualityService,
		WeatherService:    h.refreshJob.weatherService,
	})

	result := healthCheckJob.Run(ctx)

	if result.Failed > 0 {
		return fmt.Errorf("health check failed: %d errors", result.Failed)
	}

	h.logger.Debug().Msg("health check passed")
	return nil
}
