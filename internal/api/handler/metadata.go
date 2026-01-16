package handler

import (
	"net/http"
	"time"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// MetadataHandler handles metadata endpoints.
type MetadataHandler struct{}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler() *MetadataHandler {
	return &MetadataHandler{}
}

// ListAirQualityStations handles GET /v1/metadata/air-quality/stations.
func (h *MetadataHandler) ListAirQualityStations(w http.ResponseWriter, _ *http.Request) {
	// TODO: Get actual stations from database/cache
	now := models.Timestamp(time.Now())
	stations := models.PagedStations{
		Items: []models.Station{
			{
				StationID:  "NL10938",
				Name:       "Amsterdam-Einsteinweg",
				Point:      models.Point{Lat: 52.366, Lon: 4.859},
				Pollutants: []models.Pollutant{models.PollutantNO2, models.PollutantPM10},
				UpdatedAt:  now,
			},
			{
				StationID:  "NL10937",
				Name:       "Amsterdam-Vondelpark",
				Point:      models.Point{Lat: 52.360, Lon: 4.871},
				Pollutants: []models.Pollutant{models.PollutantNO2, models.PollutantO3},
				UpdatedAt:  now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, http.StatusOK, stations)
}

// GetEnums handles GET /v1/metadata/enums - get enum values used by the API.
func (h *MetadataHandler) GetEnums(w http.ResponseWriter, _ *http.Request) {
	enums := models.Enums{
		Modes: []models.Mode{
			models.ModeWalk,
			models.ModeBike,
			models.ModeTrain,
		},
		Objectives: []models.Objective{
			models.ObjectiveFastest,
			models.ObjectiveLowestExposure,
			models.ObjectiveBalanced,
		},
		Confidence: []models.Confidence{
			models.ConfidenceLow,
			models.ConfidenceMedium,
			models.ConfidenceHigh,
		},
		Pollutants: []models.Pollutant{
			models.PollutantNO2,
			models.PollutantPM25,
			models.PollutantPM10,
			models.PollutantO3,
			models.PollutantPollen,
		},
	}
	response.JSON(w, http.StatusOK, enums)
}
