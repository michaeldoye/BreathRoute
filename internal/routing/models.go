// Package routing provides route computation for bike and walk modes.
package routing

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for routing operations.
var (
	// ErrProviderUnavailable indicates the routing provider is down or the circuit breaker is open.
	ErrProviderUnavailable = errors.New("routing provider unavailable")
	// ErrNoRouteFound indicates no valid route exists between the given points.
	ErrNoRouteFound = errors.New("no route found between the given points")
	// ErrRateLimitExceeded indicates the API quota has been exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	// ErrInvalidCoordinates indicates the provided coordinates are invalid or out of range.
	ErrInvalidCoordinates = errors.New("invalid coordinates")
)

// Provider defines the interface for routing providers.
type Provider interface {
	// GetDirections retrieves route directions between two points.
	// Returns multiple route alternatives when available.
	GetDirections(ctx context.Context, req DirectionsRequest) (*DirectionsResponse, error)
	// Name returns the provider identifier for logging and metrics.
	Name() string
	// SupportedProfiles returns the list of route profiles this provider supports.
	SupportedProfiles() []RouteProfile
}

// RouteProfile represents a routing profile (mode of transport).
type RouteProfile string

const (
	// ProfileWalk is the foot-walking profile for pedestrian routing.
	ProfileWalk RouteProfile = "foot-walking"
	// ProfileBike is the cycling-regular profile for bike routing.
	ProfileBike RouteProfile = "cycling-regular"
)

// Coordinate represents a geographic point.
type Coordinate struct {
	Lat float64
	Lon float64
}

// DirectionsRequest is the request for computing routes.
type DirectionsRequest struct {
	Origin          Coordinate
	Destination     Coordinate
	Profile         RouteProfile
	MaxAlternatives int // Maximum number of alternative routes to return (default: 2)
}

// DirectionsResponse is the response containing route alternatives.
type DirectionsResponse struct {
	Routes    []Route
	Provider  string
	FetchedAt time.Time
}

// Route represents a single route option.
type Route struct {
	GeometryPolyline string        // Encoded polyline (precision 5)
	DistanceMeters   int           // Total distance in meters
	DurationSeconds  int           // Total duration in seconds
	Summary          string        // Human-readable route summary
	BoundingBox      *BoundingBox  // Geographic bounding box
	Instructions     []Instruction // Turn-by-turn instructions
}

// BoundingBox represents a geographic bounding box.
type BoundingBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// Instruction represents a turn-by-turn instruction.
type Instruction struct {
	Text           string // Human-readable instruction text
	DistanceMeters int    // Distance for this segment
	DurationSecs   int    // Duration for this segment
	Type           int    // ORS instruction type code
}

// Error provides detailed error information from the routing provider.
type Error struct {
	Provider string // Provider that generated the error
	Code     string // Error code from the provider
	Message  string // Human-readable error message
	Err      error  // Underlying error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is transient and the request can be retried.
func (e *Error) IsRetryable() bool {
	return errors.Is(e.Err, ErrProviderUnavailable) || errors.Is(e.Err, ErrRateLimitExceeded)
}
