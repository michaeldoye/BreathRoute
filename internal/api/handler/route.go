package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// RouteHandler handles routing endpoints.
type RouteHandler struct{}

// NewRouteHandler creates a new RouteHandler.
func NewRouteHandler() *RouteHandler {
	return &RouteHandler{}
}

// ComputeRoutes handles POST /v1/routes:compute - compute route options.
func (h *RouteHandler) ComputeRoutes(w http.ResponseWriter, r *http.Request) {
	var input models.RouteComputeRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate: either commuteId or origin+destination required
	if input.CommuteID == nil && (input.Origin == nil || input.Destination == nil) {
		response.BadRequest(w, r, "either commuteId or origin and destination are required", []models.FieldError{
			{Field: "commuteId", Message: "required if origin/destination not provided"},
			{Field: "origin", Message: "required if commuteId not provided"},
			{Field: "destination", Message: "required if commuteId not provided"},
		})
		return
	}

	// TODO: Actually compute routes using routing engine
	now := models.Timestamp(time.Now())
	resp := models.RouteComputeResponse{
		GeneratedAt: now,
		Options: []models.RouteOption{
			{
				ID:              "opt_" + uuid.New().String()[:12],
				Objective:       input.Objective,
				DurationSeconds: 2760,
				DistanceMeters:  intPtr(12340),
				ExposureScore:   42.7,
				Confidence:      models.ConfidenceHigh,
				Legs: []models.RouteLeg{
					{
						Mode:            models.ModeBike,
						Provider:        "graphhopper",
						Start:           models.LegPoint{Name: "Home", Point: *input.Origin},
						End:             models.LegPoint{Name: "Amsterdam Centraal", Point: models.Point{Lat: 52.378, Lon: 4.900}},
						DurationSeconds: 900,
						DistanceMeters:  intPtr(2800),
					},
					{
						Mode:            models.ModeTrain,
						Provider:        "ns",
						Start:           models.LegPoint{Name: "Amsterdam Centraal", Point: models.Point{Lat: 52.378, Lon: 4.900}},
						End:             models.LegPoint{Name: "Schiphol", Point: models.Point{Lat: 52.308, Lon: 4.764}},
						DurationSeconds: 1800,
						Transit: &models.TransitLeg{
							ServiceName:   "Intercity",
							Line:          strPtr("IC 1432"),
							DepartureTime: now,
							ArrivalTime:   models.Timestamp(time.Time(now).Add(30 * time.Minute)),
							Platform:      strPtr("5b"),
						},
					},
				},
				Summary: models.RouteSummary{
					Title: "Fastest route via Amsterdam Centraal",
					Highlights: []string{
						"Direct intercity connection",
						"Bike-friendly route to station",
					},
				},
			},
		},
	}

	w.Header().Set("Cache-Control", "private, max-age=60")
	response.JSON(w, r, http.StatusOK, resp)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
