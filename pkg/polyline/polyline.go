// Package polyline provides encoding and decoding utilities for Google's polyline algorithm.
// The polyline algorithm is documented at: https://developers.google.com/maps/documentation/utilities/polylinealgorithm
package polyline

import (
	"math"
)

// Coordinate represents a geographic point with latitude and longitude.
type Coordinate struct {
	Lat float64
	Lon float64
}

// Decode decodes a polyline-encoded string into a slice of coordinates.
// The polyline format uses precision of 5 decimal places (standard Google/ORS format).
func Decode(encoded string) []Coordinate {
	if encoded == "" {
		return nil
	}

	var coords []Coordinate
	index := 0
	lat := 0
	lon := 0

	for index < len(encoded) {
		// Decode latitude
		latDelta, newIndex := decodeValue(encoded, index)
		index = newIndex
		lat += latDelta

		// Decode longitude
		lonDelta, newIndex := decodeValue(encoded, index)
		index = newIndex
		lon += lonDelta

		coords = append(coords, Coordinate{
			Lat: float64(lat) / 1e5,
			Lon: float64(lon) / 1e5,
		})
	}

	return coords
}

// decodeValue decodes a single value from the polyline at the given index.
// Returns the decoded delta value and the new index position.
func decodeValue(encoded string, index int) (int, int) {
	shift := 0
	result := 0

	for index < len(encoded) {
		b := int(encoded[index]) - 63
		index++
		result |= (b & 0x1f) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}

	// Apply two's complement for negative values
	if result&1 != 0 {
		return ^(result >> 1), index
	}
	return result >> 1, index
}

// Encode encodes a slice of coordinates into a polyline-encoded string.
// The polyline format uses precision of 5 decimal places (standard Google/ORS format).
func Encode(coords []Coordinate) string {
	if len(coords) == 0 {
		return ""
	}

	encoded := make([]byte, 0, len(coords)*4)
	prevLat := 0
	prevLon := 0

	for _, coord := range coords {
		lat := int(math.Round(coord.Lat * 1e5))
		lon := int(math.Round(coord.Lon * 1e5))

		encoded = encodeValue(encoded, lat-prevLat)
		encoded = encodeValue(encoded, lon-prevLon)

		prevLat = lat
		prevLon = lon
	}

	return string(encoded)
}

// encodeValue encodes a single integer value using the polyline algorithm.
func encodeValue(buf []byte, value int) []byte {
	// Invert if negative
	if value < 0 {
		value = ^(value << 1)
	} else {
		value <<= 1
	}

	// Encode in 5-bit chunks
	for value >= 0x20 {
		buf = append(buf, byte((value&0x1f)|0x20)+63)
		value >>= 5
	}
	buf = append(buf, byte(value)+63)

	return buf
}

// Length calculates the total length of a polyline in meters using the haversine formula.
func Length(coords []Coordinate) float64 {
	if len(coords) < 2 {
		return 0
	}

	var total float64
	for i := 1; i < len(coords); i++ {
		total += haversineDistance(coords[i-1], coords[i])
	}
	return total
}

// Sample returns coordinates sampled at approximately the specified interval along the polyline.
// This is useful for sampling points for air quality exposure scoring.
func Sample(coords []Coordinate, intervalMeters float64) []Coordinate {
	if len(coords) == 0 {
		return nil
	}
	if intervalMeters <= 0 {
		return coords
	}

	sampled := []Coordinate{coords[0]}
	accumulated := 0.0

	for i := 1; i < len(coords); i++ {
		segmentDist := haversineDistance(coords[i-1], coords[i])

		// Check if we need to add sample points within this segment
		for accumulated+segmentDist >= intervalMeters {
			// Calculate how far along this segment we need to go
			remaining := intervalMeters - accumulated
			fraction := remaining / segmentDist

			// Interpolate the point
			newLat := coords[i-1].Lat + fraction*(coords[i].Lat-coords[i-1].Lat)
			newLon := coords[i-1].Lon + fraction*(coords[i].Lon-coords[i-1].Lon)
			sampled = append(sampled, Coordinate{Lat: newLat, Lon: newLon})

			// Update for next iteration
			segmentDist -= remaining
			accumulated = 0
		}

		accumulated += segmentDist
	}

	// Always include the last point if it's not already included
	last := coords[len(coords)-1]
	if len(sampled) == 0 || sampled[len(sampled)-1] != last {
		sampled = append(sampled, last)
	}

	return sampled
}

// haversineDistance calculates the distance between two coordinates in meters.
const earthRadiusMeters = 6371000

func haversineDistance(a, b Coordinate) float64 {
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := (b.Lat - a.Lat) * math.Pi / 180
	dLon := (b.Lon - a.Lon) * math.Pi / 180

	sinDLat := math.Sin(dLat / 2)
	sinDLon := math.Sin(dLon / 2)

	h := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLon*sinDLon
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}
