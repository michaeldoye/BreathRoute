// Package featureflags provides feature flag management for runtime configuration.
package featureflags

import (
	"encoding/json"
	"time"
)

// Well-known feature flag keys.
const (
	// FlagDisableTrainMode disables train/transit routing options.
	FlagDisableTrainMode = "disable_train_mode"

	// FlagCachedOnlyAirQuality forces air quality lookups to use cache only.
	FlagCachedOnlyAirQuality = "cached_only_air_quality"

	// FlagDisableAlertsSending prevents sending push notifications.
	FlagDisableAlertsSending = "disable_alerts_sending"

	// FlagDisablePollenFactor excludes pollen from exposure calculations.
	FlagDisablePollenFactor = "disable_pollen_factor"

	// FlagRoutingBikeOnly restricts routing to bike mode only.
	FlagRoutingBikeOnly = "routing_bike_only"

	// FlagEnableTimeShift enables time-shift route recommendations.
	FlagEnableTimeShift = "enable_time_shift"
)

// Flag represents a feature flag with its current value.
type Flag struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

// FlagList represents a list of feature flags.
type FlagList struct {
	Items []Flag `json:"items"`
}

// FlagUpdate represents a single flag update request.
type FlagUpdate struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// FlagUpdateRequest represents a request to update feature flags.
type FlagUpdateRequest struct {
	Updates []FlagUpdate `json:"updates"`
	Reason  string       `json:"reason"`
}

// BoolValue returns the flag value as a boolean.
// Returns the default value if the flag is nil, not found, or not a boolean.
func (f *Flag) BoolValue(defaultValue bool) bool {
	if f == nil {
		return defaultValue
	}
	switch v := f.Value.(type) {
	case bool:
		return v
	case float64:
		// JSON unmarshals numbers as float64
		return v != 0
	default:
		return defaultValue
	}
}

// StringValue returns the flag value as a string.
// Returns the default value if the flag is nil, not found, or not a string.
func (f *Flag) StringValue(defaultValue string) string {
	if f == nil {
		return defaultValue
	}
	switch v := f.Value.(type) {
	case string:
		return v
	default:
		return defaultValue
	}
}

// IntValue returns the flag value as an integer.
// Returns the default value if the flag is nil, not found, or not a number.
func (f *Flag) IntValue(defaultValue int) int {
	if f == nil {
		return defaultValue
	}
	switch v := f.Value.(type) {
	case float64:
		// JSON unmarshals numbers as float64
		return int(v)
	case int:
		return v
	default:
		return defaultValue
	}
}

// Float64Value returns the flag value as a float64.
// Returns the default value if the flag is nil, not found, or not a number.
func (f *Flag) Float64Value(defaultValue float64) float64 {
	if f == nil {
		return defaultValue
	}
	switch v := f.Value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return defaultValue
	}
}

// JSONValue unmarshals the flag value into the target struct.
// Returns an error if unmarshaling fails.
func (f *Flag) JSONValue(target interface{}) error {
	if f == nil {
		return nil
	}
	data, err := json.Marshal(f.Value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// DefaultFlags returns the default feature flags for the application.
func DefaultFlags() map[string]*Flag {
	now := time.Now()
	return map[string]*Flag{
		FlagDisableTrainMode: {
			Key:       FlagDisableTrainMode,
			Value:     false,
			UpdatedAt: now,
		},
		FlagCachedOnlyAirQuality: {
			Key:       FlagCachedOnlyAirQuality,
			Value:     false,
			UpdatedAt: now,
		},
		FlagDisableAlertsSending: {
			Key:       FlagDisableAlertsSending,
			Value:     false,
			UpdatedAt: now,
		},
		FlagDisablePollenFactor: {
			Key:       FlagDisablePollenFactor,
			Value:     false,
			UpdatedAt: now,
		},
		FlagRoutingBikeOnly: {
			Key:       FlagRoutingBikeOnly,
			Value:     false,
			UpdatedAt: now,
		},
		FlagEnableTimeShift: {
			Key:       FlagEnableTimeShift,
			Value:     true,
			UpdatedAt: now,
		},
	}
}
