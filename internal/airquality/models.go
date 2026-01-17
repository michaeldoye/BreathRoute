// Package airquality provides air quality data access and caching.
package airquality

import (
	"errors"
	"time"
)

// Provider errors.
var (
	ErrStationNotFound     = errors.New("station not found")
	ErrNoMeasurements      = errors.New("no measurements available")
	ErrProviderUnavailable = errors.New("air quality provider unavailable")
)

// Pollutant represents an air quality pollutant type.
type Pollutant string

const (
	PollutantNO2  Pollutant = "NO2"
	PollutantPM25 Pollutant = "PM25"
	PollutantPM10 Pollutant = "PM10"
	PollutantO3   Pollutant = "O3"
)

// Station represents an air quality monitoring station.
type Station struct {
	ID         string
	Name       string
	Lat        float64
	Lon        float64
	Pollutants []Pollutant
	UpdatedAt  time.Time
}

// Measurement represents a single pollutant measurement at a station.
type Measurement struct {
	StationID  string
	Pollutant  Pollutant
	Value      float64
	Unit       string
	MeasuredAt time.Time
}

// AQSnapshot represents a point-in-time snapshot of air quality data.
// This is the internal normalized format for all air quality data.
type AQSnapshot struct {
	// Stations is a map of station ID to station metadata.
	Stations map[string]*Station

	// Measurements contains the latest measurements per station.
	// Key format: "stationID:pollutant" (e.g., "NL10938:NO2")
	Measurements map[string]*Measurement

	// FetchedAt is when this snapshot was retrieved from the provider.
	FetchedAt time.Time

	// Provider identifies the data source.
	Provider string
}

// NewAQSnapshot creates a new empty snapshot.
func NewAQSnapshot(provider string) *AQSnapshot {
	return &AQSnapshot{
		Stations:     make(map[string]*Station),
		Measurements: make(map[string]*Measurement),
		FetchedAt:    time.Now(),
		Provider:     provider,
	}
}

// GetMeasurement retrieves a measurement for a station and pollutant.
func (s *AQSnapshot) GetMeasurement(stationID string, pollutant Pollutant) *Measurement {
	key := stationID + ":" + string(pollutant)
	return s.Measurements[key]
}

// SetMeasurement adds or updates a measurement in the snapshot.
func (s *AQSnapshot) SetMeasurement(m *Measurement) {
	key := m.StationID + ":" + string(m.Pollutant)
	s.Measurements[key] = m
}

// StationList returns all stations as a slice.
func (s *AQSnapshot) StationList() []*Station {
	stations := make([]*Station, 0, len(s.Stations))
	for _, station := range s.Stations {
		stations = append(stations, station)
	}
	return stations
}

// GetStationMeasurements returns all measurements for a given station.
func (s *AQSnapshot) GetStationMeasurements(stationID string) []*Measurement {
	var measurements []*Measurement
	for _, pollutant := range []Pollutant{PollutantNO2, PollutantPM25, PollutantPM10, PollutantO3} {
		if m := s.GetMeasurement(stationID, pollutant); m != nil {
			measurements = append(measurements, m)
		}
	}
	return measurements
}
