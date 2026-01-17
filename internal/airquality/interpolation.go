package airquality

import (
	"errors"
	"math"
	"sort"
)

// Interpolation errors.
var (
	ErrNoStationsInRange = errors.New("no stations within range")
	ErrInsufficientData  = errors.New("insufficient data for interpolation")
)

// Confidence represents the confidence level of an interpolated value.
type Confidence string

const (
	ConfidenceLow    Confidence = "LOW"
	ConfidenceMedium Confidence = "MEDIUM"
	ConfidenceHigh   Confidence = "HIGH"
)

// InterpolationConfig holds configuration for the interpolation algorithm.
type InterpolationConfig struct {
	// MaxDistance is the maximum distance (in meters) to consider stations.
	// Stations beyond this distance are ignored. Default: 50000 (50km).
	MaxDistance float64

	// MinStations is the minimum number of stations required for interpolation.
	// If fewer stations are in range, returns ErrInsufficientData. Default: 1.
	MinStations int

	// MaxStations is the maximum number of nearest stations to use.
	// Using fewer stations is faster but less accurate. Default: 5.
	MaxStations int

	// Power is the power parameter for inverse distance weighting.
	// Higher values give more weight to closer stations. Default: 2.0.
	Power float64

	// HighConfidenceMaxDistance is the max distance for HIGH confidence.
	// Default: 5000 (5km).
	HighConfidenceMaxDistance float64

	// MediumConfidenceMaxDistance is the max distance for MEDIUM confidence.
	// Default: 15000 (15km).
	MediumConfidenceMaxDistance float64
}

// DefaultInterpolationConfig returns the default configuration.
func DefaultInterpolationConfig() InterpolationConfig {
	return InterpolationConfig{
		MaxDistance:                 50000, // 50km
		MinStations:                 1,
		MaxStations:                 5,
		Power:                       2.0,
		HighConfidenceMaxDistance:   5000,  // 5km
		MediumConfidenceMaxDistance: 15000, // 15km
	}
}

// InterpolatedValue represents an interpolated air quality value at a point.
type InterpolatedValue struct {
	// Pollutant is the pollutant type.
	Pollutant Pollutant

	// Value is the interpolated value in µg/m³.
	Value float64

	// Confidence indicates the data quality.
	Confidence Confidence

	// StationsUsed is the number of stations used in interpolation.
	StationsUsed int

	// NearestStationDistance is the distance to the nearest station in meters.
	NearestStationDistance float64

	// ContributingStations lists the stations that contributed to this value.
	ContributingStations []StationContribution
}

// StationContribution describes a station's contribution to an interpolated value.
type StationContribution struct {
	StationID string
	Distance  float64 // meters
	Value     float64 // measured value
	Weight    float64 // normalized weight (0-1)
}

// InterpolatedPoint represents all interpolated values at a geographic point.
type InterpolatedPoint struct {
	Lat    float64
	Lon    float64
	Values map[Pollutant]*InterpolatedValue
}

// stationDistance pairs a station with its distance from the query point.
type stationDistance struct {
	station  *Station
	distance float64
}

// Interpolator performs spatial interpolation of air quality data.
type Interpolator struct {
	config InterpolationConfig
}

// NewInterpolator creates a new Interpolator with the given configuration.
func NewInterpolator(config InterpolationConfig) *Interpolator {
	if config.MaxDistance <= 0 {
		config.MaxDistance = DefaultInterpolationConfig().MaxDistance
	}
	if config.MinStations <= 0 {
		config.MinStations = DefaultInterpolationConfig().MinStations
	}
	if config.MaxStations <= 0 {
		config.MaxStations = DefaultInterpolationConfig().MaxStations
	}
	if config.Power <= 0 {
		config.Power = DefaultInterpolationConfig().Power
	}
	if config.HighConfidenceMaxDistance <= 0 {
		config.HighConfidenceMaxDistance = DefaultInterpolationConfig().HighConfidenceMaxDistance
	}
	if config.MediumConfidenceMaxDistance <= 0 {
		config.MediumConfidenceMaxDistance = DefaultInterpolationConfig().MediumConfidenceMaxDistance
	}
	return &Interpolator{config: config}
}

// Interpolate estimates air quality values at the given location.
func (i *Interpolator) Interpolate(lat, lon float64, snapshot *AQSnapshot) (*InterpolatedPoint, error) {
	if snapshot == nil || len(snapshot.Stations) == 0 {
		return nil, ErrNoStationsInRange
	}

	// Calculate distances to all stations
	var stationDistances []stationDistance

	for _, station := range snapshot.Stations {
		dist := haversineDistance(lat, lon, station.Lat, station.Lon)
		if dist <= i.config.MaxDistance {
			stationDistances = append(stationDistances, stationDistance{
				station:  station,
				distance: dist,
			})
		}
	}

	if len(stationDistances) < i.config.MinStations {
		return nil, ErrNoStationsInRange
	}

	// Sort by distance
	sort.Slice(stationDistances, func(a, b int) bool {
		return stationDistances[a].distance < stationDistances[b].distance
	})

	// Limit to MaxStations
	if len(stationDistances) > i.config.MaxStations {
		stationDistances = stationDistances[:i.config.MaxStations]
	}

	// Interpolate each pollutant
	result := &InterpolatedPoint{
		Lat:    lat,
		Lon:    lon,
		Values: make(map[Pollutant]*InterpolatedValue),
	}

	for _, pollutant := range []Pollutant{PollutantNO2, PollutantPM25, PollutantPM10, PollutantO3} {
		value, err := i.interpolatePollutant(pollutant, stationDistances, snapshot)
		if err != nil {
			// Skip pollutants with no data
			continue
		}
		result.Values[pollutant] = value
	}

	if len(result.Values) == 0 {
		return nil, ErrInsufficientData
	}

	return result, nil
}

// InterpolateMultiple estimates air quality values at multiple points.
func (i *Interpolator) InterpolateMultiple(points []struct{ Lat, Lon float64 }, snapshot *AQSnapshot) ([]*InterpolatedPoint, error) {
	results := make([]*InterpolatedPoint, 0, len(points))

	for _, p := range points {
		result, err := i.Interpolate(p.Lat, p.Lon, snapshot)
		if err != nil {
			// Include nil for failed interpolations
			results = append(results, nil)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// interpolatePollutant performs IDW interpolation for a single pollutant.
func (i *Interpolator) interpolatePollutant(
	pollutant Pollutant,
	stationDistances []stationDistance,
	snapshot *AQSnapshot,
) (*InterpolatedValue, error) {
	contributions := make([]StationContribution, 0, len(stationDistances))
	var totalWeight float64

	for _, sd := range stationDistances {
		// Check if station has this pollutant
		hasPollutant := false
		for _, p := range sd.station.Pollutants {
			if p == pollutant {
				hasPollutant = true
				break
			}
		}
		if !hasPollutant {
			continue
		}

		// Get measurement
		m := snapshot.GetMeasurement(sd.station.ID, pollutant)
		if m == nil {
			continue
		}

		// Calculate weight using inverse distance weighting
		var weight float64
		if sd.distance < 1 {
			// Very close to station - use station value directly
			weight = 1e10 // Very high weight
		} else {
			weight = 1.0 / math.Pow(sd.distance, i.config.Power)
		}

		contributions = append(contributions, StationContribution{
			StationID: sd.station.ID,
			Distance:  sd.distance,
			Value:     m.Value,
			Weight:    weight,
		})
		totalWeight += weight
	}

	if len(contributions) == 0 {
		return nil, ErrInsufficientData
	}

	// Normalize weights and calculate weighted average
	var interpolatedValue float64
	for idx := range contributions {
		contributions[idx].Weight /= totalWeight
		interpolatedValue += contributions[idx].Value * contributions[idx].Weight
	}

	// Determine confidence based on nearest station distance
	nearestDistance := contributions[0].Distance
	confidence := i.calculateConfidence(nearestDistance, len(contributions))

	return &InterpolatedValue{
		Pollutant:              pollutant,
		Value:                  interpolatedValue,
		Confidence:             confidence,
		StationsUsed:           len(contributions),
		NearestStationDistance: nearestDistance,
		ContributingStations:   contributions,
	}, nil
}

// calculateConfidence determines confidence level based on distance and station count.
func (i *Interpolator) calculateConfidence(nearestDistance float64, stationCount int) Confidence {
	// High confidence: close to station and multiple stations
	if nearestDistance <= i.config.HighConfidenceMaxDistance && stationCount >= 2 {
		return ConfidenceHigh
	}

	// Medium confidence: moderate distance or fewer stations
	if nearestDistance <= i.config.MediumConfidenceMaxDistance && stationCount >= 1 {
		return ConfidenceMedium
	}

	// Low confidence: far from stations
	return ConfidenceLow
}

// haversineDistance calculates the distance between two points in meters
// using the Haversine formula.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
