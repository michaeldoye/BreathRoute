package pollen

import (
	"errors"
	"time"
)

// Pollen errors.
var (
	ErrProviderUnavailable = errors.New("pollen provider unavailable")
	ErrNoDataForRegion     = errors.New("no pollen data for region")
	ErrInvalidCoordinates  = errors.New("invalid coordinates")
	ErrPollenDisabled      = errors.New("pollen factor disabled by feature flag")
)

// Type represents a category of pollen.
type Type string

const (
	PollenGrass Type = "GRASS"
	PollenTree  Type = "TREE"
	PollenWeed  Type = "WEED"
)

// AllTypes returns all supported pollen types.
func AllTypes() []Type {
	return []Type{PollenGrass, PollenTree, PollenWeed}
}

// RiskLevel represents the pollen risk level.
type RiskLevel string

const (
	RiskNone     RiskLevel = "NONE"
	RiskLow      RiskLevel = "LOW"
	RiskModerate RiskLevel = "MODERATE"
	RiskHigh     RiskLevel = "HIGH"
	RiskVeryHigh RiskLevel = "VERY_HIGH"
)

// RiskLevelFromIndex converts a numeric index (0-5 scale) to RiskLevel.
func RiskLevelFromIndex(index float64) RiskLevel {
	switch {
	case index <= 0:
		return RiskNone
	case index <= 1:
		return RiskLow
	case index <= 2:
		return RiskModerate
	case index <= 3:
		return RiskHigh
	default:
		return RiskVeryHigh
	}
}

// Reading represents a pollen measurement for a specific type.
type Reading struct {
	// Type is the pollen category.
	Type Type

	// Index is the raw pollen index (0-5 scale typically).
	Index float64

	// Risk is the categorical risk level.
	Risk RiskLevel

	// Species contains specific pollen species if available (e.g., "Birch", "Oak").
	Species []string
}

// RegionalPollen represents pollen data for a geographic region.
type RegionalPollen struct {
	// Region identifier (e.g., "NL", "NL-NH" for Noord-Holland).
	Region string

	// RegionName is the human-readable region name.
	RegionName string

	// Center coordinates of the region.
	Lat float64
	Lon float64

	// Readings contains pollen data by type.
	Readings map[Type]*Reading

	// OverallRisk is the highest risk level across all pollen types.
	OverallRisk RiskLevel

	// OverallIndex is the combined/average pollen index.
	OverallIndex float64

	// ValidFor indicates the date/period this data is valid for.
	ValidFor time.Time

	// FetchedAt is when the data was retrieved.
	FetchedAt time.Time

	// Provider identifies the data source.
	Provider string
}

// GetReading returns the reading for a specific pollen type.
func (r *RegionalPollen) GetReading(pollenType Type) *Reading {
	if r.Readings == nil {
		return nil
	}
	return r.Readings[pollenType]
}

// ExposureFactor returns a multiplier (1.0-1.5) for exposure scoring.
// Higher pollen means slightly worse conditions for sensitive users.
func (r *RegionalPollen) ExposureFactor() float64 {
	switch r.OverallRisk {
	case RiskNone:
		return 1.0
	case RiskLow:
		return 1.05
	case RiskModerate:
		return 1.1
	case RiskHigh:
		return 1.2
	case RiskVeryHigh:
		return 1.3
	default:
		return 1.0
	}
}

// Forecast represents pollen forecast data.
type Forecast struct {
	// Region identifier.
	Region string

	// Daily forecasts.
	Daily []DailyForecast

	// FetchedAt is when the forecast was retrieved.
	FetchedAt time.Time
}

// DailyForecast represents pollen forecast for a single day.
type DailyForecast struct {
	// Date of the forecast.
	Date time.Time

	// Readings contains predicted pollen data by type.
	Readings map[Type]*Reading

	// OverallRisk is the highest predicted risk level.
	OverallRisk RiskLevel

	// OverallIndex is the predicted combined pollen index.
	OverallIndex float64
}
