package device

import (
	"context"
	"errors"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
)

// Service provides device operations.
type Service struct {
	repo Repository
}

// NewService creates a new device service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List retrieves all devices for a user.
func (s *Service) List(ctx context.Context, userID string, limit int) (*models.PagedDevices, error) {
	result, err := s.repo.ListByUser(ctx, userID, ListOptions{Limit: limit})
	if err != nil {
		return nil, err
	}

	items := make([]models.Device, 0, len(result.Items))
	for _, d := range result.Items {
		items = append(items, s.toAPIDevice(d))
	}

	var nextCursor *string
	if result.NextCursor != "" {
		nextCursor = &result.NextCursor
	}

	return &models.PagedDevices{
		Items: items,
		Meta: models.PagedResponseMeta{
			Limit:      limit,
			NextCursor: nextCursor,
		},
	}, nil
}

// Get retrieves a device by ID for a user.
func (s *Service) Get(ctx context.Context, userID, deviceID string) (*models.Device, error) {
	device, err := s.repo.Get(ctx, userID, deviceID)
	if err != nil {
		return nil, err
	}

	result := s.toAPIDevice(device)
	return &result, nil
}

// Register registers or updates a device.
// Returns the device and whether it was newly created.
func (s *Service) Register(ctx context.Context, userID string, input *models.DeviceRegisterRequest) (*models.Device, bool, error) {
	now := time.Now()

	device := &Device{
		ID:          input.DeviceID,
		UserID:      userID,
		Platform:    Platform(input.Platform),
		Token:       input.Token,
		DeviceModel: input.DeviceModel,
		OSVersion:   input.OSVersion,
		AppVersion:  input.AppVersion,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.repo.Upsert(ctx, device)
	if err != nil {
		return nil, false, err
	}

	result := s.toAPIDevice(device)
	return &result, created, nil
}

// Unregister removes a device registration.
func (s *Service) Unregister(ctx context.Context, userID, deviceID string) error {
	err := s.repo.Delete(ctx, userID, deviceID)
	if err != nil {
		if errors.Is(err, ErrDeviceNotFound) {
			return ErrDeviceNotFound
		}
		return err
	}
	return nil
}

// toAPIDevice converts a domain Device to an API Device.
func (s *Service) toAPIDevice(d *Device) models.Device {
	tokenLast4 := d.TokenLast4()
	return models.Device{
		ID:          d.ID,
		Platform:    models.PushPlatform(d.Platform),
		TokenLast4:  &tokenLast4,
		DeviceModel: d.DeviceModel,
		OSVersion:   d.OSVersion,
		AppVersion:  d.AppVersion,
		CreatedAt:   models.Timestamp(d.CreatedAt),
		UpdatedAt:   models.Timestamp(d.UpdatedAt),
	}
}
