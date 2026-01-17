package device

import (
	"context"
	"sync"
)

// InMemoryRepository is an in-memory implementation of Repository.
// This is intended for testing. Production should use the PostgreSQL implementation.
type InMemoryRepository struct {
	mu      sync.RWMutex
	devices map[string]*Device // keyed by device ID
	tokens  map[string]string  // token -> device ID mapping
}

// NewInMemoryRepository creates a new in-memory device repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		devices: make(map[string]*Device),
		tokens:  make(map[string]string),
	}
}

// Get retrieves a device by user ID and device ID.
func (r *InMemoryRepository) Get(_ context.Context, userID, deviceID string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	device, ok := r.devices[deviceID]
	if !ok || device.UserID != userID {
		return nil, ErrDeviceNotFound
	}

	return copyDevice(device), nil
}

// GetByToken retrieves a device by token.
func (r *InMemoryRepository) GetByToken(_ context.Context, token string) (*Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deviceID, ok := r.tokens[token]
	if !ok {
		return nil, ErrDeviceNotFound
	}

	device, ok := r.devices[deviceID]
	if !ok {
		return nil, ErrDeviceNotFound
	}

	return copyDevice(device), nil
}

// ListByUser retrieves all devices for a user.
func (r *InMemoryRepository) ListByUser(_ context.Context, userID string, opts ListOptions) (*ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []*Device
	for _, device := range r.devices {
		if device.UserID == userID {
			items = append(items, copyDevice(device))
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(items) > limit {
		items = items[:limit]
	}

	return &ListResult{
		Items:      items,
		NextCursor: "",
	}, nil
}

// Create creates a new device.
func (r *InMemoryRepository) Create(_ context.Context, device *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.devices[device.ID] = copyDevice(device)
	r.tokens[device.Token] = device.ID
	return nil
}

// Update updates an existing device.
func (r *InMemoryRepository) Update(_ context.Context, device *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.devices[device.ID]
	if !ok {
		return ErrDeviceNotFound
	}

	// Remove old token mapping if token changed
	if existing.Token != device.Token {
		delete(r.tokens, existing.Token)
		r.tokens[device.Token] = device.ID
	}

	r.devices[device.ID] = copyDevice(device)
	return nil
}

// Upsert creates or updates a device based on the token.
// Returns true if a new device was created, false if updated.
func (r *InMemoryRepository) Upsert(_ context.Context, device *Device) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if device with this token already exists
	if existingID, ok := r.tokens[device.Token]; ok {
		existing := r.devices[existingID]
		// Update existing device with new info
		existing.Platform = device.Platform
		existing.DeviceModel = device.DeviceModel
		existing.OSVersion = device.OSVersion
		existing.AppVersion = device.AppVersion
		existing.UpdatedAt = device.UpdatedAt
		// If device ID changed, update the mapping
		if existingID != device.ID {
			delete(r.devices, existingID)
			r.devices[device.ID] = existing
			r.tokens[device.Token] = device.ID
		}
		return false, nil
	}

	// Create new device
	r.devices[device.ID] = copyDevice(device)
	r.tokens[device.Token] = device.ID
	return true, nil
}

// Delete deletes a device.
func (r *InMemoryRepository) Delete(_ context.Context, userID, deviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	device, ok := r.devices[deviceID]
	if !ok || device.UserID != userID {
		return ErrDeviceNotFound
	}

	delete(r.tokens, device.Token)
	delete(r.devices, deviceID)
	return nil
}

// DeleteByUser deletes all devices for a user.
func (r *InMemoryRepository) DeleteByUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, device := range r.devices {
		if device.UserID == userID {
			delete(r.tokens, device.Token)
			delete(r.devices, id)
		}
	}
	return nil
}

// copyDevice creates a deep copy of a device.
func copyDevice(d *Device) *Device {
	if d == nil {
		return nil
	}

	deviceCopy := &Device{
		ID:        d.ID,
		UserID:    d.UserID,
		Platform:  d.Platform,
		Token:     d.Token,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}

	if d.DeviceModel != nil {
		val := *d.DeviceModel
		deviceCopy.DeviceModel = &val
	}
	if d.OSVersion != nil {
		val := *d.OSVersion
		deviceCopy.OSVersion = &val
	}
	if d.AppVersion != nil {
		val := *d.AppVersion
		deviceCopy.AppVersion = &val
	}

	return deviceCopy
}
