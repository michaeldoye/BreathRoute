package device

import "context"

// Repository defines the interface for device persistence.
type Repository interface {
	// Get retrieves a device by user ID and device ID.
	Get(ctx context.Context, userID, deviceID string) (*Device, error)

	// GetByToken retrieves a device by token.
	GetByToken(ctx context.Context, token string) (*Device, error)

	// ListByUser retrieves all devices for a user.
	ListByUser(ctx context.Context, userID string, opts ListOptions) (*ListResult, error)

	// Create creates a new device.
	Create(ctx context.Context, device *Device) error

	// Update updates an existing device.
	Update(ctx context.Context, device *Device) error

	// Upsert creates or updates a device based on the token.
	// Returns true if a new device was created, false if updated.
	Upsert(ctx context.Context, device *Device) (created bool, err error)

	// Delete deletes a device.
	Delete(ctx context.Context, userID, deviceID string) error

	// DeleteByUser deletes all devices for a user.
	DeleteByUser(ctx context.Context, userID string) error
}
