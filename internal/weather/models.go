package weather

import (
	"errors"
	"time"
)

// Weather errors.
var (
	ErrProviderUnavailable = errors.New("weather provider unavailable")
	ErrNoDataForLocation   = errors.New("no weather data for location")
	ErrInvalidCoordinates  = errors.New("invalid coordinates")
)

// Observation represents weather data at a specific point and time.
type Observation struct {
	// Location coordinates
	Lat float64
	Lon float64

	// Temperature in Celsius
	Temperature float64

	// Humidity percentage (0-100)
	Humidity float64

	// Wind data
	WindSpeed     float64 // m/s
	WindDirection float64 // degrees (0-360, 0=N, 90=E, 180=S, 270=W)
	WindGust      float64 // m/s (optional, 0 if not available)

	// Atmospheric pressure in hPa
	Pressure float64

	// Weather condition
	Condition   Condition
	Description string

	// Cloud cover percentage (0-100)
	CloudCover float64

	// Visibility in meters
	Visibility float64

	// Timestamps
	ObservedAt time.Time
	FetchedAt  time.Time
}

// Condition represents the general weather condition.
type Condition string

const (
	ConditionClear        Condition = "CLEAR"
	ConditionClouds       Condition = "CLOUDS"
	ConditionRain         Condition = "RAIN"
	ConditionDrizzle      Condition = "DRIZZLE"
	ConditionThunderstorm Condition = "THUNDERSTORM"
	ConditionSnow         Condition = "SNOW"
	ConditionMist         Condition = "MIST"
	ConditionFog          Condition = "FOG"
	ConditionHaze         Condition = "HAZE"
	ConditionUnknown      Condition = "UNKNOWN"
)

// WindCategory categorizes wind speed for air quality impact assessment.
type WindCategory string

const (
	WindCalm     WindCategory = "CALM"     // < 1 m/s - pollutants accumulate
	WindLight    WindCategory = "LIGHT"    // 1-3 m/s - minimal dispersion
	WindModerate WindCategory = "MODERATE" // 3-8 m/s - good dispersion
	WindStrong   WindCategory = "STRONG"   // > 8 m/s - excellent dispersion
)

// GetWindCategory returns the wind category for the observation.
func (o *Observation) GetWindCategory() WindCategory {
	switch {
	case o.WindSpeed < 1:
		return WindCalm
	case o.WindSpeed < 3:
		return WindLight
	case o.WindSpeed < 8:
		return WindModerate
	default:
		return WindStrong
	}
}

// DispersionFactor returns a multiplier (0.5-1.5) indicating how wind affects
// air quality dispersion. Lower values mean pollutants disperse faster.
func (o *Observation) DispersionFactor() float64 {
	switch o.GetWindCategory() {
	case WindCalm:
		return 1.3 // Pollutants accumulate - worse AQ
	case WindLight:
		return 1.1
	case WindModerate:
		return 0.9
	case WindStrong:
		return 0.7 // Good dispersion - better AQ
	default:
		return 1.0
	}
}

// BoundingBox represents a geographic bounding box.
type BoundingBox struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// Contains checks if a point is within the bounding box.
func (b BoundingBox) Contains(lat, lon float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat &&
		lon >= b.MinLon && lon <= b.MaxLon
}

// Center returns the center point of the bounding box.
func (b BoundingBox) Center() (lat, lon float64) {
	return (b.MinLat + b.MaxLat) / 2, (b.MinLon + b.MaxLon) / 2
}

// Forecast represents weather forecast data.
type Forecast struct {
	// Location
	Lat float64
	Lon float64

	// Hourly forecasts
	Hourly []HourlyForecast

	// When the forecast was fetched
	FetchedAt time.Time
}

// HourlyForecast represents weather for a specific hour.
type HourlyForecast struct {
	Time          time.Time
	Temperature   float64
	Humidity      float64
	WindSpeed     float64
	WindDirection float64
	WindGust      float64
	Condition     Condition
	Description   string
	CloudCover    float64
	Visibility    float64
	PrecipProb    float64 // Probability of precipitation (0-1)
}

// GetWindCategory returns the wind category for the hourly forecast.
func (h *HourlyForecast) GetWindCategory() WindCategory {
	switch {
	case h.WindSpeed < 1:
		return WindCalm
	case h.WindSpeed < 3:
		return WindLight
	case h.WindSpeed < 8:
		return WindModerate
	default:
		return WindStrong
	}
}
