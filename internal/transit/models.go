package transit

import (
	"errors"
	"time"
)

// Transit errors.
var (
	ErrProviderUnavailable = errors.New("transit provider unavailable")
	ErrNoDisruptions       = errors.New("no disruptions found")
)

// DisruptionType represents the type of transit disruption.
type DisruptionType string

const (
	DisruptionMaintenance  DisruptionType = "MAINTENANCE"
	DisruptionDisturbance  DisruptionType = "DISTURBANCE"
	DisruptionConstruction DisruptionType = "CONSTRUCTION"
	DisruptionStrike       DisruptionType = "STRIKE"
	DisruptionWeather      DisruptionType = "WEATHER"
	DisruptionUnknown      DisruptionType = "UNKNOWN"
)

// Impact represents the severity of a disruption.
type Impact string

const (
	ImpactMinor    Impact = "MINOR"    // Delays < 15 min
	ImpactModerate Impact = "MODERATE" // Delays 15-60 min or partial service
	ImpactMajor    Impact = "MAJOR"    // Delays > 60 min or significant cancellations
	ImpactSevere   Impact = "SEVERE"   // No service on affected routes
)

// Disruption represents a transit service disruption.
type Disruption struct {
	// ID is the unique identifier for this disruption.
	ID string

	// Type categorizes the disruption.
	Type DisruptionType

	// Title is a brief description.
	Title string

	// Description provides detailed information.
	Description string

	// Impact indicates severity.
	Impact Impact

	// AffectedRoutes lists the train lines/routes affected (e.g., "IC Amsterdam-Rotterdam").
	AffectedRoutes []string

	// AffectedStations lists station codes affected (e.g., "ASD", "RTD").
	AffectedStations []string

	// ExpectedDuration is the estimated delay in minutes (0 if unknown).
	ExpectedDuration int

	// Start is when the disruption began.
	Start time.Time

	// End is when the disruption is expected to end (zero if unknown).
	End time.Time

	// IsPlanned indicates if this was scheduled maintenance.
	IsPlanned bool

	// AlternativeTransport describes alternatives (e.g., "Bus replacement service").
	AlternativeTransport string

	// Cause provides the reason for the disruption.
	Cause string

	// LastUpdated is when this disruption info was last updated.
	LastUpdated time.Time

	// Provider identifies the data source.
	Provider string
}

// IsActive returns true if the disruption is currently active.
func (d *Disruption) IsActive() bool {
	now := time.Now()
	if now.Before(d.Start) {
		return false
	}
	if !d.End.IsZero() && now.After(d.End) {
		return false
	}
	return true
}

// AffectsStation returns true if the disruption affects the given station code.
func (d *Disruption) AffectsStation(stationCode string) bool {
	for _, s := range d.AffectedStations {
		if s == stationCode {
			return true
		}
	}
	return false
}

// AffectsRoute returns true if the disruption affects the given route.
func (d *Disruption) AffectsRoute(route string) bool {
	for _, r := range d.AffectedRoutes {
		if r == route {
			return true
		}
	}
	return false
}

// DisruptionSummary provides a snapshot of current transit status.
type DisruptionSummary struct {
	// TotalDisruptions is the count of all active disruptions.
	TotalDisruptions int

	// ByImpact groups disruption counts by impact level.
	ByImpact map[Impact]int

	// ByType groups disruption counts by type.
	ByType map[DisruptionType]int

	// MostSevere is the highest impact level among active disruptions.
	MostSevere Impact

	// FetchedAt is when this summary was generated.
	FetchedAt time.Time

	// Provider identifies the data source.
	Provider string
}

// Station represents a train station.
type Station struct {
	// Code is the station code (e.g., "ASD" for Amsterdam Centraal).
	Code string

	// Name is the station name.
	Name string

	// Lat/Lon for geolocation.
	Lat float64
	Lon float64

	// Country code (e.g., "NL").
	Country string
}

// RouteDisruptions contains disruptions relevant to a specific route.
type RouteDisruptions struct {
	// Origin station code.
	Origin string

	// Destination station code.
	Destination string

	// Disruptions affecting this route.
	Disruptions []*Disruption

	// OverallImpact is the highest impact among relevant disruptions.
	OverallImpact Impact

	// HasDisruptions indicates if any disruptions affect this route.
	HasDisruptions bool

	// AdvisoryMessage is a user-friendly summary.
	AdvisoryMessage string

	// FetchedAt is when this was retrieved.
	FetchedAt time.Time
}

// CalculateOverallImpact determines the highest impact from disruptions.
func CalculateOverallImpact(disruptions []*Disruption) Impact {
	if len(disruptions) == 0 {
		return ""
	}

	impactOrder := map[Impact]int{
		ImpactMinor:    1,
		ImpactModerate: 2,
		ImpactMajor:    3,
		ImpactSevere:   4,
	}

	highest := ImpactMinor
	for _, d := range disruptions {
		if impactOrder[d.Impact] > impactOrder[highest] {
			highest = d.Impact
		}
	}

	return highest
}
