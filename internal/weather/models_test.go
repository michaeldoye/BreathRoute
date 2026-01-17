package weather_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/breatheroute/breatheroute/internal/weather"
)

func TestObservation_GetWindCategory(t *testing.T) {
	tests := []struct {
		name      string
		windSpeed float64
		expected  weather.WindCategory
	}{
		{"calm - zero", 0, weather.WindCalm},
		{"calm - low", 0.5, weather.WindCalm},
		{"calm - boundary", 0.9, weather.WindCalm},
		{"light - boundary", 1.0, weather.WindLight},
		{"light - mid", 2.0, weather.WindLight},
		{"light - high", 2.9, weather.WindLight},
		{"moderate - boundary", 3.0, weather.WindModerate},
		{"moderate - mid", 5.0, weather.WindModerate},
		{"moderate - high", 7.9, weather.WindModerate},
		{"strong - boundary", 8.0, weather.WindStrong},
		{"strong - high", 15.0, weather.WindStrong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &weather.Observation{WindSpeed: tt.windSpeed}
			assert.Equal(t, tt.expected, obs.GetWindCategory())
		})
	}
}

func TestObservation_DispersionFactor(t *testing.T) {
	tests := []struct {
		name     string
		wind     float64
		expected float64
	}{
		{"calm - accumulation", 0.5, 1.3},
		{"light - slight accumulation", 2.0, 1.1},
		{"moderate - good dispersion", 5.0, 0.9},
		{"strong - excellent dispersion", 10.0, 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &weather.Observation{WindSpeed: tt.wind}
			assert.Equal(t, tt.expected, obs.DispersionFactor())
		})
	}
}

func TestHourlyForecast_GetWindCategory(t *testing.T) {
	tests := []struct {
		name      string
		windSpeed float64
		expected  weather.WindCategory
	}{
		{"calm", 0.5, weather.WindCalm},
		{"light", 2.0, weather.WindLight},
		{"moderate", 5.0, weather.WindModerate},
		{"strong", 10.0, weather.WindStrong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &weather.HourlyForecast{WindSpeed: tt.windSpeed}
			assert.Equal(t, tt.expected, h.GetWindCategory())
		})
	}
}

func TestBoundingBox_Contains(t *testing.T) {
	box := weather.BoundingBox{
		MinLat: 52.0,
		MaxLat: 53.0,
		MinLon: 4.0,
		MaxLon: 5.0,
	}

	tests := []struct {
		name     string
		lat, lon float64
		expected bool
	}{
		{"center", 52.5, 4.5, true},
		{"min corner", 52.0, 4.0, true},
		{"max corner", 53.0, 5.0, true},
		{"edge", 52.0, 4.5, true},
		{"outside north", 53.1, 4.5, false},
		{"outside south", 51.9, 4.5, false},
		{"outside east", 52.5, 5.1, false},
		{"outside west", 52.5, 3.9, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, box.Contains(tt.lat, tt.lon))
		})
	}
}

func TestBoundingBox_Center(t *testing.T) {
	box := weather.BoundingBox{
		MinLat: 52.0,
		MaxLat: 53.0,
		MinLon: 4.0,
		MaxLon: 5.0,
	}

	lat, lon := box.Center()
	assert.Equal(t, 52.5, lat)
	assert.Equal(t, 4.5, lon)
}

func TestConditionConstants(t *testing.T) {
	// Verify all conditions are distinct
	conditions := []weather.Condition{
		weather.ConditionClear,
		weather.ConditionClouds,
		weather.ConditionRain,
		weather.ConditionDrizzle,
		weather.ConditionThunderstorm,
		weather.ConditionSnow,
		weather.ConditionMist,
		weather.ConditionFog,
		weather.ConditionHaze,
		weather.ConditionUnknown,
	}

	seen := make(map[weather.Condition]bool)
	for _, c := range conditions {
		assert.False(t, seen[c], "duplicate condition: %s", c)
		seen[c] = true
	}
}

func TestWindCategoryConstants(t *testing.T) {
	// Verify all categories are distinct
	categories := []weather.WindCategory{
		weather.WindCalm,
		weather.WindLight,
		weather.WindModerate,
		weather.WindStrong,
	}

	seen := make(map[weather.WindCategory]bool)
	for _, c := range categories {
		assert.False(t, seen[c], "duplicate category: %s", c)
		seen[c] = true
	}
}
