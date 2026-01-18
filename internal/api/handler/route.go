package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/routing"
)

// RouteHandler handles routing endpoints.
type RouteHandler struct {
	routingService *routing.Service
	logger         zerolog.Logger
}

// NewRouteHandler creates a new RouteHandler.
func NewRouteHandler(routingService *routing.Service, logger zerolog.Logger) *RouteHandler {
	return &RouteHandler{
		routingService: routingService,
		logger:         logger,
	}
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

	ctx := r.Context()
	now := models.Timestamp(time.Now())

	// Determine which modes to compute routes for
	modes := input.Modes
	if len(modes) == 0 {
		// Default modes: BIKE and WALK
		modes = []models.Mode{models.ModeBike, models.ModeWalk}
	}

	var options []models.RouteOption
	var warnings []models.Warning

	// Compute routes for each mode
	for _, mode := range modes {
		profile := modeToProfile(mode)
		if profile == "" {
			// Skip unsupported modes like TRAIN (handled separately via NS API)
			continue
		}

		routeOptions, modeWarnings := h.computeRoutesForMode(ctx, input, mode, profile)
		options = append(options, routeOptions...)
		warnings = append(warnings, modeWarnings...)
	}

	// Sort options by objective
	h.sortOptionsByObjective(options, input.Objective)

	// Apply maxOptions limit
	maxOptions := 5
	if input.MaxOptions != nil && *input.MaxOptions > 0 {
		maxOptions = *input.MaxOptions
	}
	if len(options) > maxOptions {
		options = options[:maxOptions]
	}

	resp := models.RouteComputeResponse{
		GeneratedAt: now,
		Options:     options,
		Warnings:    warnings,
	}

	w.Header().Set("Cache-Control", "private, max-age=60")
	response.JSON(w, r, http.StatusOK, resp)
}

// computeRoutesForMode computes routes for a specific mode.
func (h *RouteHandler) computeRoutesForMode(
	ctx context.Context,
	input models.RouteComputeRequest,
	mode models.Mode,
	profile routing.RouteProfile,
) ([]models.RouteOption, []models.Warning) {
	options := make([]models.RouteOption, 0, 3) // Pre-allocate for typical route count
	warnings := make([]models.Warning, 0, 1)

	req := routing.DirectionsRequest{
		Origin: routing.Coordinate{
			Lat: input.Origin.Lat,
			Lon: input.Origin.Lon,
		},
		Destination: routing.Coordinate{
			Lat: input.Destination.Lat,
			Lon: input.Destination.Lon,
		},
		Profile:         profile,
		MaxAlternatives: 3, // Request up to 3 alternatives per mode
	}

	resp, err := h.routingService.GetDirections(ctx, req)
	if err != nil {
		h.logger.Warn().
			Err(err).
			Str("mode", string(mode)).
			Str("profile", string(profile)).
			Msg("failed to get directions for mode")

		// Add warning for failed mode
		provider := h.routingService.ProviderName()
		var routingErr *routing.Error
		warningCode := "PROVIDER_ERROR"
		warningMsg := "routing provider temporarily unavailable for " + string(mode)

		if errors.As(err, &routingErr) {
			warningCode = routingErr.Code
			warningMsg = routingErr.Message
		}

		warnings = append(warnings, models.Warning{
			Code:     warningCode,
			Message:  warningMsg,
			Provider: &provider,
		})

		return options, warnings
	}

	// Convert routes to RouteOptions
	for i, route := range resp.Routes {
		option := h.routeToOption(route, mode, input.Objective, i, *input.Origin, *input.Destination)
		options = append(options, option)
	}

	return options, warnings
}

// routeToOption converts a routing.Route to a models.RouteOption.
func (h *RouteHandler) routeToOption(
	route routing.Route,
	mode models.Mode,
	objective models.Objective,
	index int,
	origin, destination models.Point,
) models.RouteOption {
	// Generate unique ID
	optionID := "opt_" + uuid.New().String()[:12]

	// Create the route leg
	leg := models.RouteLeg{
		Mode:     mode,
		Provider: h.routingService.ProviderName(),
		Start: models.LegPoint{
			Name:  "Origin",
			Point: origin,
		},
		End: models.LegPoint{
			Name:  "Destination",
			Point: destination,
		},
		DurationSeconds:  route.DurationSeconds,
		DistanceMeters:   intPtr(route.DistanceMeters),
		GeometryPolyline: strPtr(route.GeometryPolyline),
	}

	// Add instructions if available
	for _, inst := range route.Instructions {
		leg.Instructions = append(leg.Instructions, models.Instruction{
			Text:           inst.Text,
			DistanceMeters: inst.DistanceMeters,
		})
	}

	// Build summary and highlights
	summary := buildRouteSummary(mode, route, index)

	// TODO: Calculate actual exposure score based on air quality data along route
	// For now, use a placeholder score based on route index
	exposureScore := 30.0 + float64(index)*5.0

	return models.RouteOption{
		ID:              optionID,
		Objective:       objective,
		DurationSeconds: route.DurationSeconds,
		DistanceMeters:  intPtr(route.DistanceMeters),
		ExposureScore:   exposureScore,
		Confidence:      models.ConfidenceMedium, // Medium until we have AQ data
		Legs:            []models.RouteLeg{leg},
		Summary:         summary,
	}
}

// sortOptionsByObjective sorts route options based on the requested objective.
func (h *RouteHandler) sortOptionsByObjective(options []models.RouteOption, objective models.Objective) {
	sort.Slice(options, func(i, j int) bool {
		switch objective {
		case models.ObjectiveFastest:
			return options[i].DurationSeconds < options[j].DurationSeconds
		case models.ObjectiveLowestExposure:
			return options[i].ExposureScore < options[j].ExposureScore
		case models.ObjectiveBalanced:
			// Balanced: weighted combination of duration and exposure
			scoreI := float64(options[i].DurationSeconds)/60.0 + options[i].ExposureScore
			scoreJ := float64(options[j].DurationSeconds)/60.0 + options[j].ExposureScore
			return scoreI < scoreJ
		default:
			return options[i].DurationSeconds < options[j].DurationSeconds
		}
	})
}

// modeToProfile maps API modes to ORS routing profiles.
func modeToProfile(mode models.Mode) routing.RouteProfile {
	switch mode {
	case models.ModeBike:
		return routing.ProfileBike
	case models.ModeWalk:
		return routing.ProfileWalk
	case models.ModeTrain:
		// TRAIN mode is handled separately via NS API
		return ""
	}
	return ""
}

// buildRouteSummary creates a human-readable summary for a route.
func buildRouteSummary(mode models.Mode, route routing.Route, index int) models.RouteSummary {
	var title string
	var highlights []string

	durationMins := route.DurationSeconds / 60
	distanceKm := float64(route.DistanceMeters) / 1000.0

	switch mode {
	case models.ModeBike:
		if index == 0 {
			title = "Fastest cycling route"
		} else {
			title = "Alternative cycling route"
		}
		highlights = append(highlights, formatDuration(durationMins)+" cycling")
	case models.ModeWalk:
		if index == 0 {
			title = "Fastest walking route"
		} else {
			title = "Alternative walking route"
		}
		highlights = append(highlights, formatDuration(durationMins)+" walking")
	case models.ModeTrain:
		// TRAIN mode is not handled here
		title = "Train route"
	}

	highlights = append(highlights, formatDistance(distanceKm))

	// Add route summary text if available
	if route.Summary != "" {
		highlights = append(highlights, "Via "+route.Summary)
	}

	return models.RouteSummary{
		Title:      title,
		Highlights: highlights,
	}
}

// formatDuration formats a duration in minutes to a human-readable string.
func formatDuration(mins int) string {
	if mins < 60 {
		return fmt.Sprintf("%d min", mins)
	}
	hours := mins / 60
	remainingMins := mins % 60
	if remainingMins == 0 {
		return fmt.Sprintf("%d hr", hours)
	}
	return fmt.Sprintf("%d hr %d min", hours, remainingMins)
}

// formatDistance formats a distance in km to a human-readable string.
func formatDistance(km float64) string {
	if km < 1 {
		return fmt.Sprintf("%d m", int(km*1000))
	}
	return fmt.Sprintf("%.1f km", km)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
