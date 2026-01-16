// Package models provides request and response models for the BreatheRoute API.
// These models match the OpenAPI specification defined in prod-api.yaml.
package models

import "time"

// Point represents a geographic coordinate.
type Point struct {
	Lat float64 `json:"lat" validate:"required,gte=-90,lte=90"`
	Lon float64 `json:"lon" validate:"required,gte=-180,lte=180"`
}

// GeoBox represents a geographic bounding box.
type GeoBox struct {
	MinLat float64 `json:"minLat" validate:"required,gte=-90,lte=90"`
	MinLon float64 `json:"minLon" validate:"required,gte=-180,lte=180"`
	MaxLat float64 `json:"maxLat" validate:"required,gte=-90,lte=90"`
	MaxLon float64 `json:"maxLon" validate:"required,gte=-180,lte=180"`
}

// Mode represents a transportation mode.
type Mode string

const (
	ModeWalk  Mode = "WALK"
	ModeBike  Mode = "BIKE"
	ModeTrain Mode = "TRAIN"
)

// Objective represents a routing objective.
type Objective string

const (
	ObjectiveFastest       Objective = "FASTEST"
	ObjectiveLowestExposure Objective = "LOWEST_EXPOSURE"
	ObjectiveBalanced      Objective = "BALANCED"
)

// Confidence represents the confidence level of a calculation.
type Confidence string

const (
	ConfidenceLow    Confidence = "LOW"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceHigh   Confidence = "HIGH"
)

// PagedResponseMeta contains pagination metadata.
type PagedResponseMeta struct {
	Limit      int     `json:"limit"`
	NextCursor *string `json:"nextCursor,omitempty"`
}

// Pollutant represents a pollutant type.
type Pollutant string

const (
	PollutantNO2    Pollutant = "NO2"
	PollutantPM25   Pollutant = "PM25"
	PollutantPM10   Pollutant = "PM10"
	PollutantO3     Pollutant = "O3"
	PollutantPollen Pollutant = "POLLEN"
)

// PushPlatform represents a push notification platform.
type PushPlatform string

const (
	PushPlatformFCM  PushPlatform = "FCM"
	PushPlatformAPNS PushPlatform = "APNS"
)

// HealthStatus represents the health status of a service.
type HealthStatus string

const (
	HealthStatusOK       HealthStatus = "OK"
	HealthStatusDegraded HealthStatus = "DEGRADED"
	HealthStatusFail     HealthStatus = "FAIL"
)

// ExportRequestStatus represents the status of an export request.
type ExportRequestStatus string

const (
	ExportStatusPending ExportRequestStatus = "PENDING"
	ExportStatusRunning ExportRequestStatus = "RUNNING"
	ExportStatusReady   ExportRequestStatus = "READY"
	ExportStatusFailed  ExportRequestStatus = "FAILED"
	ExportStatusExpired ExportRequestStatus = "EXPIRED"
)

// DeletionRequestStatus represents the status of a deletion request.
type DeletionRequestStatus string

const (
	DeletionStatusPending   DeletionRequestStatus = "PENDING"
	DeletionStatusScheduled DeletionRequestStatus = "SCHEDULED"
	DeletionStatusRunning   DeletionRequestStatus = "RUNNING"
	DeletionStatusCompleted DeletionRequestStatus = "COMPLETED"
	DeletionStatusFailed    DeletionRequestStatus = "FAILED"
)

// AlertThresholdType represents the type of alert threshold.
type AlertThresholdType string

const (
	ThresholdAbsoluteScore       AlertThresholdType = "ABSOLUTE_SCORE"
	ThresholdPercentWorseThanBaseline AlertThresholdType = "PERCENT_WORSE_THAN_BASELINE"
)

// TransitAlertSeverity represents the severity of a transit alert.
type TransitAlertSeverity string

const (
	SeverityInfo    TransitAlertSeverity = "INFO"
	SeverityWarning TransitAlertSeverity = "WARNING"
	SeveritySevere  TransitAlertSeverity = "SEVERE"
)

// Timestamp is a helper type for time.Time with custom JSON formatting.
type Timestamp time.Time

// MarshalJSON implements json.Marshaler for Timestamp.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler for Timestamp.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	// Remove quotes
	s := string(data[1 : len(data)-1])
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*t = Timestamp(parsed)
	return nil
}

// Time returns the underlying time.Time.
func (t Timestamp) Time() time.Time {
	return time.Time(t)
}
