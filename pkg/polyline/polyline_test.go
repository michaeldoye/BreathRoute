package polyline

import (
	"math"
	"testing"
)

func TestDecode_ValidPolyline(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected []Coordinate
	}{
		{
			name:    "single point",
			encoded: "_p~iF~ps|U",
			expected: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
			},
		},
		{
			name:    "two points",
			encoded: "_p~iF~ps|U_ulLnnqC",
			expected: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
				{Lat: 40.7, Lon: -120.95},
			},
		},
		{
			name:    "three points - Google example",
			encoded: "_p~iF~ps|U_ulLnnqC_mqNvxq`@",
			expected: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
				{Lat: 40.7, Lon: -120.95},
				{Lat: 43.252, Lon: -126.453},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Decode(tt.encoded)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d coordinates, got %d", len(tt.expected), len(result))
			}

			for i, coord := range result {
				if !coordsEqual(coord, tt.expected[i], 0.001) {
					t.Errorf("coordinate %d: expected %+v, got %+v", i, tt.expected[i], coord)
				}
			}
		})
	}
}

func TestDecode_EmptyString(t *testing.T) {
	result := Decode("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

func TestEncode_ValidCoordinates(t *testing.T) {
	tests := []struct {
		name   string
		coords []Coordinate
	}{
		{
			name: "single point",
			coords: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
			},
		},
		{
			name: "two points",
			coords: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
				{Lat: 40.7, Lon: -120.95},
			},
		},
		{
			name: "three points",
			coords: []Coordinate{
				{Lat: 38.5, Lon: -120.2},
				{Lat: 40.7, Lon: -120.95},
				{Lat: 43.252, Lon: -126.453},
			},
		},
		{
			name: "Amsterdam to Utrecht",
			coords: []Coordinate{
				{Lat: 52.3676, Lon: 4.9041},
				{Lat: 52.0907, Lon: 5.1214},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := Encode(tt.coords)
			if encoded == "" {
				t.Fatal("expected non-empty encoded string")
			}

			// Verify round-trip
			decoded := Decode(encoded)
			if len(decoded) != len(tt.coords) {
				t.Fatalf("round-trip: expected %d coordinates, got %d", len(tt.coords), len(decoded))
			}

			for i, coord := range decoded {
				if !coordsEqual(coord, tt.coords[i], 0.00001) {
					t.Errorf("round-trip coordinate %d: expected %+v, got %+v", i, tt.coords[i], coord)
				}
			}
		})
	}
}

func TestEncode_EmptyCoordinates(t *testing.T) {
	result := Encode(nil)
	if result != "" {
		t.Errorf("expected empty string for nil coordinates, got %q", result)
	}

	result = Encode([]Coordinate{})
	if result != "" {
		t.Errorf("expected empty string for empty coordinates, got %q", result)
	}
}

func TestLength_ValidRoute(t *testing.T) {
	tests := []struct {
		name           string
		coords         []Coordinate
		expectedMeters float64
		tolerance      float64
	}{
		{
			name:           "empty",
			coords:         nil,
			expectedMeters: 0,
			tolerance:      0,
		},
		{
			name:           "single point",
			coords:         []Coordinate{{Lat: 52.0, Lon: 4.0}},
			expectedMeters: 0,
			tolerance:      0,
		},
		{
			name: "Amsterdam to Utrecht - roughly 35km",
			coords: []Coordinate{
				{Lat: 52.3676, Lon: 4.9041},
				{Lat: 52.0907, Lon: 5.1214},
			},
			expectedMeters: 35000,
			tolerance:      2000, // Allow 2km tolerance
		},
		{
			name: "1 degree latitude at equator - roughly 111km",
			coords: []Coordinate{
				{Lat: 0.0, Lon: 0.0},
				{Lat: 1.0, Lon: 0.0},
			},
			expectedMeters: 111000,
			tolerance:      1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Length(tt.coords)
			diff := math.Abs(result - tt.expectedMeters)
			if diff > tt.tolerance {
				t.Errorf("expected ~%.0fm (±%.0f), got %.0fm", tt.expectedMeters, tt.tolerance, result)
			}
		})
	}
}

func TestSample_ValidRoute(t *testing.T) {
	// Create a simple route
	coords := []Coordinate{
		{Lat: 52.0, Lon: 4.0},
		{Lat: 52.01, Lon: 4.0}, // ~1.1km north
		{Lat: 52.02, Lon: 4.0}, // ~1.1km more north
		{Lat: 52.03, Lon: 4.0}, // ~1.1km more north
	}

	t.Run("sample every 500m", func(t *testing.T) {
		sampled := Sample(coords, 500)
		// Total distance is ~3.3km, so we expect ~7 samples (every 500m) plus endpoints
		if len(sampled) < 5 {
			t.Errorf("expected at least 5 samples, got %d", len(sampled))
		}
		// First and last should match
		if !coordsEqual(sampled[0], coords[0], 0.0001) {
			t.Errorf("first sample should be first coordinate")
		}
		if !coordsEqual(sampled[len(sampled)-1], coords[len(coords)-1], 0.0001) {
			t.Errorf("last sample should be last coordinate")
		}
	})

	t.Run("sample every 10km exceeds route length", func(t *testing.T) {
		sampled := Sample(coords, 10000)
		// Route is ~3.3km, so we should just get first and last
		if len(sampled) != 2 {
			t.Errorf("expected 2 samples (start and end), got %d", len(sampled))
		}
	})

	t.Run("empty coordinates", func(t *testing.T) {
		sampled := Sample(nil, 500)
		if sampled != nil {
			t.Errorf("expected nil for empty coordinates")
		}
	})

	t.Run("zero interval returns all", func(t *testing.T) {
		sampled := Sample(coords, 0)
		if len(sampled) != len(coords) {
			t.Errorf("expected all coordinates for zero interval")
		}
	})
}

func TestRoundTrip_HighPrecision(t *testing.T) {
	// Test that encode->decode preserves coordinates to 5 decimal places
	coords := []Coordinate{
		{Lat: 52.37403, Lon: 4.88969},
		{Lat: 52.37234, Lon: 4.89231},
		{Lat: 52.37001, Lon: 4.89534},
	}

	encoded := Encode(coords)
	decoded := Decode(encoded)

	for i, coord := range decoded {
		// Precision of 5 decimal places = 0.00001
		if !coordsEqual(coord, coords[i], 0.00001) {
			t.Errorf("coordinate %d lost precision: expected %+v, got %+v", i, coords[i], coord)
		}
	}
}

// coordsEqual checks if two coordinates are equal within a tolerance.
func coordsEqual(a, b Coordinate, tolerance float64) bool {
	return math.Abs(a.Lat-b.Lat) <= tolerance && math.Abs(a.Lon-b.Lon) <= tolerance
}

// BenchmarkDecode benchmarks encoding/decoding for performance testing.
func BenchmarkDecode(b *testing.B) {
	// A moderately complex polyline (Amsterdam area route)
	encoded := "_p~iF~ps|U_ulLnnqC_mqNvxq`@"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Decode(encoded)
	}
}

func BenchmarkEncode(b *testing.B) {
	coords := []Coordinate{
		{Lat: 38.5, Lon: -120.2},
		{Lat: 40.7, Lon: -120.95},
		{Lat: 43.252, Lon: -126.453},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Encode(coords)
	}
}
