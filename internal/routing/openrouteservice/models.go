package openrouteservice

// orsRequest represents the ORS directions API request body.
type orsRequest struct {
	Coordinates       [][]float64            `json:"coordinates"`
	AlternativeRoutes *alternativeRoutesOpts `json:"alternative_routes,omitempty"`
	Instructions      bool                   `json:"instructions"`
	Geometry          bool                   `json:"geometry"`
	Units             string                 `json:"units"`
	Language          string                 `json:"language"`
}

// alternativeRoutesOpts configures alternative route generation.
type alternativeRoutesOpts struct {
	TargetCount int `json:"target_count"`
}

// orsResponse represents the ORS directions API response.
type orsResponse struct {
	Routes  []orsRoute `json:"routes"`
	BBox    []float64  `json:"bbox,omitempty"`
	Metdata *metadata  `json:"metadata,omitempty"`
}

// metadata contains response metadata.
type metadata struct {
	Attribution string `json:"attribution,omitempty"`
	Service     string `json:"service,omitempty"`
	Timestamp   int64  `json:"timestamp,omitempty"`
}

// orsRoute represents a single route in the ORS response.
type orsRoute struct {
	Summary   routeSummary     `json:"summary"`
	Segments  []routeSegment   `json:"segments,omitempty"`
	BBox      []float64        `json:"bbox,omitempty"`
	Geometry  string           `json:"geometry"`
	WayPoints []int            `json:"way_points,omitempty"`
	Warnings  []routeWarning   `json:"warnings,omitempty"`
	Extras    map[string]extra `json:"extras,omitempty"`
}

// routeSummary contains summary information for a route.
type routeSummary struct {
	Distance float64 `json:"distance"` // Distance in meters
	Duration float64 `json:"duration"` // Duration in seconds
}

// routeSegment represents a segment of the route.
type routeSegment struct {
	Distance float64          `json:"distance"`
	Duration float64          `json:"duration"`
	Steps    []routeStep      `json:"steps,omitempty"`
	Warnings []segmentWarning `json:"warnings,omitempty"`
}

// routeStep represents a single step (instruction) in a segment.
type routeStep struct {
	Distance    float64 `json:"distance"`
	Duration    float64 `json:"duration"`
	Type        int     `json:"type"`
	Instruction string  `json:"instruction"`
	Name        string  `json:"name"`
	WayPoints   []int   `json:"way_points,omitempty"`
}

// routeWarning represents a warning for the entire route.
type routeWarning struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// segmentWarning represents a warning for a specific segment.
type segmentWarning struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// extra contains additional route information like elevation, surface type, etc.
type extra struct {
	Values  [][]int   `json:"values,omitempty"`
	Summary []summary `json:"summary,omitempty"`
}

// summary provides summary statistics for extras.
type summary struct {
	Value    float64 `json:"value"`
	Distance float64 `json:"distance"`
	Amount   float64 `json:"amount"`
}

// orsErrorResponse represents an error response from ORS.
type orsErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Info string `json:"info,omitempty"`
}

// ORS error codes for error mapping.
const (
	orsErrorCodeNotFound     = 2009 // Route not found
	orsErrorCodeInvalidParam = 2003 // Invalid parameter
	orsErrorCodeRateLimit    = 403  // Rate limit exceeded (HTTP status)
)
