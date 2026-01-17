// Package commute provides commute management services.
package commute

import (
	"errors"
	"time"
)

// Repository errors.
var (
	ErrCommuteNotFound = errors.New("commute not found")
)

// Commute represents a saved commute.
type Commute struct {
	ID                        string
	UserID                    string
	Label                     string
	Origin                    Location
	Destination               Location
	DaysOfWeek                []int
	PreferredArrivalTimeLocal string
	Notes                     *string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

// Location represents a geographic location.
type Location struct {
	Point   Point
	Geohash *string
}

// Point represents a geographic point.
type Point struct {
	Lat float64
	Lon float64
}
