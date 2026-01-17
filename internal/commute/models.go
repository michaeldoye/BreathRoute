// Package commute provides commute management functionality.
package commute

import (
	"time"
)

// Point represents a geographic coordinate.
type Point struct {
	Lat float64
	Lon float64
}

// Location represents a commute endpoint with coordinates and optional geohash.
type Location struct {
	Point   Point
	Geohash *string
}

// Commute represents a saved commute route.
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

// copyStringPtr creates a copy of a string pointer.
func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

// copyIntSlice creates a copy of an int slice.
func copyIntSlice(s []int) []int {
	if s == nil {
		return nil
	}
	result := make([]int, len(s))
	copy(result, s)
	return result
}

// Copy creates a deep copy of a Commute.
func (c *Commute) Copy() *Commute {
	if c == nil {
		return nil
	}
	return &Commute{
		ID:     c.ID,
		UserID: c.UserID,
		Label:  c.Label,
		Origin: Location{
			Point:   c.Origin.Point,
			Geohash: copyStringPtr(c.Origin.Geohash),
		},
		Destination: Location{
			Point:   c.Destination.Point,
			Geohash: copyStringPtr(c.Destination.Geohash),
		},
		DaysOfWeek:                copyIntSlice(c.DaysOfWeek),
		PreferredArrivalTimeLocal: c.PreferredArrivalTimeLocal,
		Notes:                     copyStringPtr(c.Notes),
		CreatedAt:                 c.CreatedAt,
		UpdatedAt:                 c.UpdatedAt,
	}
}
