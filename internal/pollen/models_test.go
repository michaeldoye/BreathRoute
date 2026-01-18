package pollen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/breatheroute/breatheroute/internal/pollen"
)

func TestRiskLevelFromIndex(t *testing.T) {
	tests := []struct {
		name     string
		index    float64
		expected pollen.RiskLevel
	}{
		{"zero", 0, pollen.RiskNone},
		{"negative", -1, pollen.RiskNone},
		{"low boundary", 0.5, pollen.RiskLow},
		{"low max", 1.0, pollen.RiskLow},
		{"moderate boundary", 1.5, pollen.RiskModerate},
		{"moderate max", 2.0, pollen.RiskModerate},
		{"high boundary", 2.5, pollen.RiskHigh},
		{"high max", 3.0, pollen.RiskHigh},
		{"very high boundary", 3.5, pollen.RiskVeryHigh},
		{"very high", 5.0, pollen.RiskVeryHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, pollen.RiskLevelFromIndex(tt.index))
		})
	}
}

func TestRegionalPollen_ExposureFactor(t *testing.T) {
	tests := []struct {
		name     string
		risk     pollen.RiskLevel
		expected float64
	}{
		{"none", pollen.RiskNone, 1.0},
		{"low", pollen.RiskLow, 1.05},
		{"moderate", pollen.RiskModerate, 1.1},
		{"high", pollen.RiskHigh, 1.2},
		{"very high", pollen.RiskVeryHigh, 1.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := &pollen.RegionalPollen{OverallRisk: tt.risk}
			assert.Equal(t, tt.expected, rp.ExposureFactor())
		})
	}
}

func TestRegionalPollen_GetReading(t *testing.T) {
	rp := &pollen.RegionalPollen{
		Readings: map[pollen.Type]*pollen.Reading{
			pollen.PollenGrass: {
				Type:  pollen.PollenGrass,
				Index: 2.5,
				Risk:  pollen.RiskModerate,
			},
			pollen.PollenTree: {
				Type:  pollen.PollenTree,
				Index: 1.0,
				Risk:  pollen.RiskLow,
			},
		},
	}

	t.Run("existing reading", func(t *testing.T) {
		reading := rp.GetReading(pollen.PollenGrass)
		assert.NotNil(t, reading)
		assert.Equal(t, pollen.PollenGrass, reading.Type)
		assert.Equal(t, 2.5, reading.Index)
	})

	t.Run("missing reading", func(t *testing.T) {
		reading := rp.GetReading(pollen.PollenWeed)
		assert.Nil(t, reading)
	})

	t.Run("nil readings map", func(t *testing.T) {
		rp := &pollen.RegionalPollen{}
		reading := rp.GetReading(pollen.PollenGrass)
		assert.Nil(t, reading)
	})
}

func TestAllTypes(t *testing.T) {
	types := pollen.AllTypes()
	assert.Len(t, types, 3)
	assert.Contains(t, types, pollen.PollenGrass)
	assert.Contains(t, types, pollen.PollenTree)
	assert.Contains(t, types, pollen.PollenWeed)
}

func TestPollenTypeConstants(t *testing.T) {
	// Verify all types are distinct
	types := []pollen.Type{
		pollen.PollenGrass,
		pollen.PollenTree,
		pollen.PollenWeed,
	}

	seen := make(map[pollen.Type]bool)
	for _, pt := range types {
		assert.False(t, seen[pt], "duplicate type: %s", pt)
		seen[pt] = true
	}
}

func TestRiskLevelConstants(t *testing.T) {
	// Verify all risk levels are distinct
	levels := []pollen.RiskLevel{
		pollen.RiskNone,
		pollen.RiskLow,
		pollen.RiskModerate,
		pollen.RiskHigh,
		pollen.RiskVeryHigh,
	}

	seen := make(map[pollen.RiskLevel]bool)
	for _, rl := range levels {
		assert.False(t, seen[rl], "duplicate level: %s", rl)
		seen[rl] = true
	}
}
