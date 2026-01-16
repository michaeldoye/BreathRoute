package models

// Device represents a registered push notification device.
type Device struct {
	ID          string       `json:"id"`
	Platform    PushPlatform `json:"platform"`
	TokenLast4  *string      `json:"tokenLast4,omitempty"`
	DeviceModel *string      `json:"deviceModel,omitempty"`
	OSVersion   *string      `json:"osVersion,omitempty"`
	AppVersion  *string      `json:"appVersion,omitempty"`
	CreatedAt   Timestamp    `json:"createdAt"`
	UpdatedAt   Timestamp    `json:"updatedAt"`
}

// DeviceRegisterRequest is the request body for registering a device.
type DeviceRegisterRequest struct {
	DeviceID    string       `json:"deviceId" validate:"required"`
	Platform    PushPlatform `json:"platform" validate:"required,oneof=FCM APNS"`
	Token       string       `json:"token" validate:"required,min=16"`
	DeviceModel *string      `json:"deviceModel,omitempty"`
	OSVersion   *string      `json:"osVersion,omitempty"`
	AppVersion  *string      `json:"appVersion,omitempty"`
}

// PagedDevices represents a paginated list of devices.
type PagedDevices struct {
	Items []Device          `json:"items"`
	Meta  PagedResponseMeta `json:"meta"`
}
