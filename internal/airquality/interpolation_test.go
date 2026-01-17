package airquality_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/airquality"
)

func createTestSnapshot() *airquality.AQSnapshot {
	snapshot := airquality.NewAQSnapshot("test")

	// Amsterdam area stations
	snapshot.Stations["NL10001"] = &airquality.Station{
		ID:         "NL10001",
		Name:       "Amsterdam-Centrum",
		Lat:        52.370216,
		Lon:        4.895168,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM25},
		UpdatedAt:  time.Now(),
	}
	snapshot.Stations["NL10002"] = &airquality.Station{
		ID:         "NL10002",
		Name:       "Amsterdam-West",
		Lat:        52.375,
		Lon:        4.85,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM25, airquality.PollutantPM10},
		UpdatedAt:  time.Now(),
	}
	snapshot.Stations["NL10003"] = &airquality.Station{
		ID:         "NL10003",
		Name:       "Amsterdam-Oost",
		Lat:        52.365,
		Lon:        4.94,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantO3},
		UpdatedAt:  time.Now(),
	}

	// Rotterdam station (far from Amsterdam)
	snapshot.Stations["NL10004"] = &airquality.Station{
		ID:         "NL10004",
		Name:       "Rotterdam-Centrum",
		Lat:        51.9225,
		Lon:        4.47917,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM25},
		UpdatedAt:  time.Now(),
	}

	// Measurements
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10001",
		Pollutant:  airquality.PollutantNO2,
		Value:      30.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10001",
		Pollutant:  airquality.PollutantPM25,
		Value:      15.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10002",
		Pollutant:  airquality.PollutantNO2,
		Value:      25.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10002",
		Pollutant:  airquality.PollutantPM25,
		Value:      12.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10002",
		Pollutant:  airquality.PollutantPM10,
		Value:      20.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10003",
		Pollutant:  airquality.PollutantNO2,
		Value:      35.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10003",
		Pollutant:  airquality.PollutantO3,
		Value:      50.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10004",
		Pollutant:  airquality.PollutantNO2,
		Value:      40.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10004",
		Pollutant:  airquality.PollutantPM25,
		Value:      18.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})

	return snapshot
}

func TestInterpolator_Interpolate_BasicIDW(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	// Point between Amsterdam stations
	result, err := interpolator.Interpolate(52.370, 4.89, snapshot)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have NO2 (all 3 Amsterdam stations have it)
	no2 := result.Values[airquality.PollutantNO2]
	require.NotNil(t, no2, "should have NO2 value")
	assert.True(t, no2.Value > 25 && no2.Value < 35, "NO2 should be interpolated between station values")
	assert.True(t, no2.StationsUsed >= 2, "should use multiple stations")

	// Should have PM25 (2 Amsterdam stations have it)
	pm25 := result.Values[airquality.PollutantPM25]
	require.NotNil(t, pm25, "should have PM25 value")
	assert.True(t, pm25.Value > 12 && pm25.Value < 15, "PM25 should be interpolated")
}

func TestInterpolator_Interpolate_ExactStationLocation(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	// Point at exact station location
	result, err := interpolator.Interpolate(52.370216, 4.895168, snapshot)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should be very close to station value
	no2 := result.Values[airquality.PollutantNO2]
	require.NotNil(t, no2)
	assert.InDelta(t, 30.0, no2.Value, 0.5, "should be very close to station value")
}

func TestInterpolator_Interpolate_NoStationsInRange(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.InterpolationConfig{
		MaxDistance: 1000, // Only 1km range
		MinStations: 1,
		MaxStations: 5,
		Power:       2.0,
	})

	// Point far from any station (somewhere in the North Sea)
	_, err := interpolator.Interpolate(55.0, 3.0, snapshot)
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrNoStationsInRange)
}

func TestInterpolator_Interpolate_EmptySnapshot(t *testing.T) {
	snapshot := airquality.NewAQSnapshot("test")
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	_, err := interpolator.Interpolate(52.370, 4.89, snapshot)
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrNoStationsInRange)
}

func TestInterpolator_Interpolate_NilSnapshot(t *testing.T) {
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	_, err := interpolator.Interpolate(52.370, 4.89, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, airquality.ErrNoStationsInRange)
}

func TestInterpolator_Interpolate_MissingPollutantData(t *testing.T) {
	snapshot := airquality.NewAQSnapshot("test")
	// Add station with pollutant but no measurement
	snapshot.Stations["NL10001"] = &airquality.Station{
		ID:         "NL10001",
		Name:       "Test Station",
		Lat:        52.370,
		Lon:        4.89,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2, airquality.PollutantPM25},
		UpdatedAt:  time.Now(),
	}
	// Only add NO2 measurement, not PM25
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID:  "NL10001",
		Pollutant:  airquality.PollutantNO2,
		Value:      30.0,
		Unit:       "µg/m³",
		MeasuredAt: time.Now(),
	})

	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())
	result, err := interpolator.Interpolate(52.370, 4.89, snapshot)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have NO2
	assert.NotNil(t, result.Values[airquality.PollutantNO2])

	// Should NOT have PM25 (no measurement)
	assert.Nil(t, result.Values[airquality.PollutantPM25])
}

func TestInterpolator_Interpolate_Confidence(t *testing.T) {
	snapshot := createTestSnapshot()

	tests := []struct {
		name               string
		lat, lon           float64
		expectedConfidence airquality.Confidence
		description        string
	}{
		{
			name:               "high confidence - near station",
			lat:                52.370216,
			lon:                4.895168,
			expectedConfidence: airquality.ConfidenceHigh,
			description:        "at station location",
		},
		{
			name:               "medium confidence - moderate distance",
			lat:                52.45, // ~10km from Amsterdam stations
			lon:                4.90,
			expectedConfidence: airquality.ConfidenceMedium,
			description:        "~10km from stations",
		},
	}

	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpolator.Interpolate(tt.lat, tt.lon, snapshot)
			require.NoError(t, err, tt.description)

			no2 := result.Values[airquality.PollutantNO2]
			require.NotNil(t, no2)
			assert.Equal(t, tt.expectedConfidence, no2.Confidence, tt.description)
		})
	}
}

func TestInterpolator_Interpolate_MaxStationsLimit(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.InterpolationConfig{
		MaxDistance: 100000, // 100km
		MinStations: 1,
		MaxStations: 2, // Only use 2 nearest
		Power:       2.0,
	})

	result, err := interpolator.Interpolate(52.370, 4.89, snapshot)
	require.NoError(t, err)

	no2 := result.Values[airquality.PollutantNO2]
	require.NotNil(t, no2)
	assert.LessOrEqual(t, no2.StationsUsed, 2, "should use at most 2 stations")
}

func TestInterpolator_InterpolateMultiple(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	points := []struct{ Lat, Lon float64 }{
		{52.370, 4.89}, // Amsterdam
		{52.375, 4.85}, // Near Amsterdam-West
		{55.0, 3.0},    // North Sea (no stations in range)
		{52.365, 4.94}, // Near Amsterdam-Oost
	}

	results, err := interpolator.InterpolateMultiple(points, snapshot)
	require.NoError(t, err)
	assert.Len(t, results, 4)

	// First point should have data
	assert.NotNil(t, results[0])
	assert.NotNil(t, results[0].Values[airquality.PollutantNO2])

	// Second point should have data
	assert.NotNil(t, results[1])

	// Third point (North Sea) should be nil (no stations)
	assert.Nil(t, results[2])

	// Fourth point should have data
	assert.NotNil(t, results[3])
}

func TestInterpolator_StationContributions(t *testing.T) {
	snapshot := createTestSnapshot()
	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	result, err := interpolator.Interpolate(52.370, 4.89, snapshot)
	require.NoError(t, err)

	no2 := result.Values[airquality.PollutantNO2]
	require.NotNil(t, no2)

	// Check contributions sum to 1
	var totalWeight float64
	for _, c := range no2.ContributingStations {
		totalWeight += c.Weight
		assert.True(t, c.Distance >= 0, "distance should be non-negative")
		assert.True(t, c.Value > 0, "value should be positive")
	}
	assert.InDelta(t, 1.0, totalWeight, 0.001, "weights should sum to 1")
}

func TestInterpolator_IDW_CloserStationsDominate(t *testing.T) {
	snapshot := airquality.NewAQSnapshot("test")

	// Two stations with very different values
	snapshot.Stations["close"] = &airquality.Station{
		ID:         "close",
		Name:       "Close Station",
		Lat:        52.370,
		Lon:        4.89,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2},
	}
	snapshot.Stations["far"] = &airquality.Station{
		ID:         "far",
		Name:       "Far Station",
		Lat:        52.5, // ~15km away
		Lon:        4.89,
		Pollutants: []airquality.Pollutant{airquality.PollutantNO2},
	}

	snapshot.SetMeasurement(&airquality.Measurement{
		StationID: "close",
		Pollutant: airquality.PollutantNO2,
		Value:     10.0, // Low value
	})
	snapshot.SetMeasurement(&airquality.Measurement{
		StationID: "far",
		Pollutant: airquality.PollutantNO2,
		Value:     100.0, // High value
	})

	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())

	// Point very close to "close" station
	result, err := interpolator.Interpolate(52.3701, 4.89, snapshot)
	require.NoError(t, err)

	no2 := result.Values[airquality.PollutantNO2]
	require.NotNil(t, no2)

	// Value should be much closer to 10 than to 100
	assert.True(t, no2.Value < 20, "closer station should dominate: got %f", no2.Value)
}

func TestHaversineDistance(t *testing.T) {
	// Test known distances
	tests := []struct {
		name             string
		lat1, lon1       float64
		lat2, lon2       float64
		expectedDistance float64 // in meters
		tolerance        float64
	}{
		{
			name:             "same point",
			lat1:             52.370,
			lon1:             4.89,
			lat2:             52.370,
			lon2:             4.89,
			expectedDistance: 0,
			tolerance:        1,
		},
		{
			name:             "Amsterdam to Rotterdam",
			lat1:             52.370216,
			lon1:             4.895168,
			lat2:             51.9225,
			lon2:             4.47917,
			expectedDistance: 57000, // ~57km
			tolerance:        2000,  // 2km tolerance
		},
	}

	interpolator := airquality.NewInterpolator(airquality.DefaultInterpolationConfig())
	// We can't access haversineDistance directly, but we can test through interpolation
	_ = interpolator

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a snapshot with a single station at lat2, lon2
			snapshot := airquality.NewAQSnapshot("test")
			snapshot.Stations["test"] = &airquality.Station{
				ID:         "test",
				Lat:        tt.lat2,
				Lon:        tt.lon2,
				Pollutants: []airquality.Pollutant{airquality.PollutantNO2},
			}
			snapshot.SetMeasurement(&airquality.Measurement{
				StationID: "test",
				Pollutant: airquality.PollutantNO2,
				Value:     30.0,
			})

			result, err := interpolator.Interpolate(tt.lat1, tt.lon1, snapshot)
			if tt.expectedDistance > 50000 { // Beyond default max distance
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			no2 := result.Values[airquality.PollutantNO2]
			require.NotNil(t, no2)
			assert.InDelta(t, tt.expectedDistance, no2.NearestStationDistance, tt.tolerance)
		})
	}
}
