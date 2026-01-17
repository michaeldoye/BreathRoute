// Package device provides device registration and management for push notifications.
package device

import (
	"errors"
	"time"
)

// Repository errors.
var (
	ErrDeviceNotFound = errors.New("device not found")
)

// Platform represents a push notification platform.
type Platform string

const (
	PlatformFCM  Platform = "FCM"
	PlatformAPNS Platform = "APNS"
)

// Device represents a registered push notification device.
type Device struct {
	ID          string
	UserID      string
	Platform    Platform
	Token       string
	DeviceModel *string
	OSVersion   *string
	AppVersion  *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TokenLast4 returns the last 4 characters of the token for display purposes.
func (d *Device) TokenLast4() string {
	if len(d.Token) < 4 {
		return d.Token
	}
	return d.Token[len(d.Token)-4:]
}

// ListOptions contains options for listing devices.
type ListOptions struct {
	Limit int
}

// ListResult contains the result of listing devices.
type ListResult struct {
	Items      []*Device
	NextCursor string
}
