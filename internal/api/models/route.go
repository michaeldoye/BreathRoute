package models

// ClientContext provides optional context from the client.
type ClientContext struct {
	Locale         *string `json:"locale,omitempty"`
	DeviceTimeZone *string `json:"deviceTimeZone,omitempty"`
	AppVersion     *string `json:"appVersion,omitempty"`
}

// RouteComputeRequest is the request body for computing routes.
type RouteComputeRequest struct {
	CommuteID             *string        `json:"commuteId,omitempty"`
	Origin                *Point         `json:"origin,omitempty"`
	Destination           *Point         `json:"destination,omitempty"`
	DepartureTime         string         `json:"departureTime" validate:"required"`
	Modes                 []Mode         `json:"modes,omitempty"`
	Objective             Objective      `json:"objective" validate:"required,oneof=FASTEST LOWEST_EXPOSURE BALANCED"`
	MaxOptions            *int           `json:"maxOptions,omitempty" validate:"omitempty,gte=1,lte=10"`
	ProfileOverride       *ProfileInput  `json:"profileOverride,omitempty"`
	IncludeExplainability *bool          `json:"includeExplainability,omitempty"`
	ClientContext         *ClientContext `json:"clientContext,omitempty"`
}

// RouteComputeResponse is the response for route computation.
type RouteComputeResponse struct {
	GeneratedAt Timestamp     `json:"generatedAt"`
	Options     []RouteOption `json:"options"`
	Warnings    []Warning     `json:"warnings,omitempty"`
}

// Warning represents a non-fatal issue in the response.
type Warning struct {
	Code     string  `json:"code"`
	Message  string  `json:"message"`
	Provider *string `json:"provider,omitempty"`
}

// RouteOption represents a single route alternative.
type RouteOption struct {
	ID              string             `json:"id"`
	Objective       Objective          `json:"objective"`
	DurationSeconds int                `json:"durationSeconds"`
	Transfers       *int               `json:"transfers,omitempty"`
	DistanceMeters  *int               `json:"distanceMeters,omitempty"`
	ExposureScore   float64            `json:"exposureScore"`
	Confidence      Confidence         `json:"confidence"`
	DeltaVsFastest  *Delta             `json:"deltaVsFastest,omitempty"`
	Breakdown       *ExposureBreakdown `json:"breakdown,omitempty"`
	Explainability  *Explainability    `json:"explainability,omitempty"`
	Legs            []RouteLeg         `json:"legs"`
	Summary         RouteSummary       `json:"summary"`
}

// Delta represents the difference versus the fastest option.
type Delta struct {
	ExtraSeconds int     `json:"extraSeconds"`
	ExposurePct  float64 `json:"exposurePct"`
}

// RouteSummary provides a human-readable summary of a route.
type RouteSummary struct {
	Title      string   `json:"title"`
	Highlights []string `json:"highlights"`
}

// RouteLeg represents a segment of a route.
type RouteLeg struct {
	Mode             Mode          `json:"mode"`
	Provider         string        `json:"provider"`
	Start            LegPoint      `json:"start"`
	End              LegPoint      `json:"end"`
	DurationSeconds  int           `json:"durationSeconds"`
	DistanceMeters   *int          `json:"distanceMeters,omitempty"`
	GeometryPolyline *string       `json:"geometryPolyline,omitempty"`
	Transit          *TransitLeg   `json:"transit,omitempty"`
	Instructions     []Instruction `json:"instructions,omitempty"`
}

// LegPoint represents a point in a route leg.
type LegPoint struct {
	Name  string `json:"name"`
	Point Point  `json:"point"`
}

// TransitLeg contains transit-specific information for a leg.
type TransitLeg struct {
	ServiceName   string         `json:"serviceName"`
	Line          *string        `json:"line,omitempty"`
	DepartureTime Timestamp      `json:"departureTime"`
	ArrivalTime   Timestamp      `json:"arrivalTime"`
	Platform      *string        `json:"platform,omitempty"`
	Alerts        []TransitAlert `json:"alerts,omitempty"`
}

// TransitAlert represents a service alert for transit.
type TransitAlert struct {
	Severity TransitAlertSeverity `json:"severity"`
	Message  string               `json:"message"`
}

// Instruction represents a turn-by-turn instruction.
type Instruction struct {
	Text           string `json:"text"`
	DistanceMeters int    `json:"distanceMeters"`
}

// ExposureBreakdown provides per-factor exposure contributions.
type ExposureBreakdown struct {
	Normalized *NormalizedExposure  `json:"normalized,omitempty"`
	Raw        *ExposureRawAverages `json:"raw,omitempty"`
}

// NormalizedExposure contains normalized exposure values.
type NormalizedExposure struct {
	NO2    *float64 `json:"no2,omitempty"`
	PM25   *float64 `json:"pm25,omitempty"`
	O3     *float64 `json:"o3,omitempty"`
	Pollen *float64 `json:"pollen,omitempty"`
}

// ExposureRawAverages contains raw route-average values.
type ExposureRawAverages struct {
	NO2Ugm3     *float64 `json:"no2_ugm3,omitempty"`
	PM25Ugm3    *float64 `json:"pm25_ugm3,omitempty"`
	O3Ugm3      *float64 `json:"o3_ugm3,omitempty"`
	PollenIndex *float64 `json:"pollen_index,omitempty"`
}

// Explainability provides provenance and scoring context.
type Explainability struct {
	DataFreshness  *DataFreshness     `json:"dataFreshness,omitempty"`
	StationSamples []StationReference `json:"stationSamples,omitempty"`
	ScoringNotes   []string           `json:"scoringNotes,omitempty"`
}

// DataFreshness indicates how recent the data is.
type DataFreshness struct {
	AirQuality *ProviderFreshness `json:"airQuality,omitempty"`
	Pollen     *ProviderFreshness `json:"pollen,omitempty"`
	Weather    *ProviderFreshness `json:"weather,omitempty"`
	Transit    *ProviderFreshness `json:"transit,omitempty"`
}

// ProviderFreshness indicates the freshness of data from a provider.
type ProviderFreshness struct {
	Provider   string    `json:"provider"`
	AsOf       Timestamp `json:"asOf"`
	TTLSeconds *int      `json:"ttlSeconds,omitempty"`
}

// StationReference identifies a monitoring station used in scoring.
type StationReference struct {
	StationID           string      `json:"stationId"`
	Name                string      `json:"name"`
	Point               Point       `json:"point"`
	DistanceMeters      int         `json:"distanceMeters"`
	PollutantsAvailable []Pollutant `json:"pollutantsAvailable,omitempty"`
}
