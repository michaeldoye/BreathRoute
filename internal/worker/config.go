// Package worker provides background job processing for BreatheRoute.
package worker

import (
	"time"
)

// RefreshTarget represents a geographic region to refresh.
type RefreshTarget struct {
	// Name is the human-readable name of the target.
	Name string

	// Points are the lat/lon coordinates to refresh.
	// Typically the centers of major cities or commuter hubs.
	Points []Point

	// Priority determines refresh order (lower = higher priority).
	Priority int
}

// Point represents a geographic coordinate.
type Point struct {
	Lat float64
	Lon float64
}

// RefreshConfig holds configuration for the provider refresh job.
type RefreshConfig struct {
	// Targets are the geographic regions to refresh.
	// If empty, uses DefaultRefreshTargets.
	Targets []RefreshTarget

	// Concurrency is the number of concurrent refresh operations.
	// Default: 3
	Concurrency int

	// Timeout is the timeout for each refresh operation.
	// Default: 30 seconds
	Timeout time.Duration

	// RefreshAirQuality enables air quality refresh.
	// Default: true
	RefreshAirQuality bool

	// RefreshWeather enables weather refresh.
	// Default: true
	RefreshWeather bool

	// RefreshPollen enables pollen refresh.
	// Default: true
	RefreshPollen bool

	// RefreshTransit enables transit disruption refresh.
	// Default: true
	RefreshTransit bool
}

// DefaultRefreshConfig returns the default refresh configuration.
func DefaultRefreshConfig() RefreshConfig {
	return RefreshConfig{
		Targets:           DefaultRefreshTargets(),
		Concurrency:       3,
		Timeout:           30 * time.Second,
		RefreshAirQuality: true,
		RefreshWeather:    true,
		RefreshPollen:     true,
		RefreshTransit:    true,
	}
}

// DefaultRefreshTargets returns the default refresh targets for the Netherlands.
// Focuses on the Randstad metropolitan area and major commuter corridors.
func DefaultRefreshTargets() []RefreshTarget {
	return []RefreshTarget{
		{
			Name:     "Amsterdam",
			Priority: 1,
			Points: []Point{
				{Lat: 52.3676, Lon: 4.9041}, // Amsterdam Centraal
				{Lat: 52.3386, Lon: 4.8919}, // Amsterdam Zuid
				{Lat: 52.3114, Lon: 4.9469}, // Amsterdam Zuidoost
				{Lat: 52.3894, Lon: 4.9006}, // Amsterdam Noord
			},
		},
		{
			Name:     "Rotterdam",
			Priority: 1,
			Points: []Point{
				{Lat: 51.9244, Lon: 4.4777}, // Rotterdam Centraal
				{Lat: 51.9062, Lon: 4.4874}, // Rotterdam Zuid
				{Lat: 51.9161, Lon: 4.3895}, // Rotterdam West
			},
		},
		{
			Name:     "Den Haag",
			Priority: 1,
			Points: []Point{
				{Lat: 52.0705, Lon: 4.3007}, // Den Haag Centraal
				{Lat: 52.0887, Lon: 4.3234}, // Den Haag HS
				{Lat: 52.1024, Lon: 4.2828}, // Scheveningen
			},
		},
		{
			Name:     "Utrecht",
			Priority: 1,
			Points: []Point{
				{Lat: 52.0894, Lon: 5.1102}, // Utrecht Centraal
				{Lat: 52.0627, Lon: 5.1179}, // Utrecht Science Park
			},
		},
		{
			Name:     "Eindhoven",
			Priority: 2,
			Points: []Point{
				{Lat: 51.4416, Lon: 5.4697}, // Eindhoven Centraal
				{Lat: 51.4548, Lon: 5.4553}, // High Tech Campus
			},
		},
		{
			Name:     "Schiphol",
			Priority: 2,
			Points: []Point{
				{Lat: 52.3105, Lon: 4.7683}, // Schiphol Airport
			},
		},
		{
			Name:     "Leiden",
			Priority: 3,
			Points: []Point{
				{Lat: 52.1664, Lon: 4.4819}, // Leiden Centraal
			},
		},
		{
			Name:     "Haarlem",
			Priority: 3,
			Points: []Point{
				{Lat: 52.3874, Lon: 4.6462}, // Haarlem
			},
		},
		{
			Name:     "Delft",
			Priority: 3,
			Points: []Point{
				{Lat: 52.0116, Lon: 4.3571}, // Delft
			},
		},
		{
			Name:     "Amersfoort",
			Priority: 3,
			Points: []Point{
				{Lat: 52.1530, Lon: 5.3711}, // Amersfoort Centraal
			},
		},
	}
}

// AllPoints returns all points from all targets, ordered by priority.
func (c RefreshConfig) AllPoints() []Point {
	var points []Point
	for _, target := range c.Targets {
		points = append(points, target.Points...)
	}
	return points
}

// TotalPoints returns the total number of points to refresh.
func (c RefreshConfig) TotalPoints() int {
	total := 0
	for _, target := range c.Targets {
		total += len(target.Points)
	}
	return total
}
