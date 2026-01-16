package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
)

// AlertHandler handles alert endpoints.
type AlertHandler struct{}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler() *AlertHandler {
	return &AlertHandler{}
}

// PreviewDepartureWindows handles POST /v1/alerts/preview - preview best departure windows.
func (h *AlertHandler) PreviewDepartureWindows(w http.ResponseWriter, r *http.Request) {
	var input models.AlertPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Actually compute departure windows
	now := time.Now()
	resp := models.AlertPreviewResponse{
		Recommended: []models.DepartureRecommendation{
			{
				DepartureTime:   models.Timestamp(now.Add(5 * time.Minute)),
				DurationSeconds: 2820,
				ExposureScore:   39.4,
				Confidence:      models.ConfidenceHigh,
				Rationale:       "Lower NO₂ expected before peak traffic; similar travel time.",
			},
			{
				DepartureTime:   models.Timestamp(now.Add(-15 * time.Minute)),
				DurationSeconds: 2760,
				ExposureScore:   35.2,
				Confidence:      models.ConfidenceMedium,
				Rationale:       "Best exposure score with 20 minute earlier departure.",
			},
		},
		EvaluatedCount: intPtr(13),
		Objective:      &input.Objective,
	}
	response.JSON(w, http.StatusOK, resp)
}

// ListAlertSubscriptions handles GET /v1/me/alerts/subscriptions - list alert subscriptions.
func (h *AlertHandler) ListAlertSubscriptions(w http.ResponseWriter, r *http.Request) {
	// TODO: Get actual subscriptions from database
	now := models.Timestamp(time.Now())
	subscriptions := models.PagedAlertSubscriptions{
		Items: []models.AlertSubscription{
			{
				ID:        "sub_01HY3ABCDEF0123456789",
				CommuteID: "cmt_01HY2ABCDEF0123456789",
				Enabled:   true,
				Objective: models.ObjectiveLowestExposure,
				Threshold: models.AlertThreshold{
					Type:          models.ThresholdAbsoluteScore,
					AbsoluteScore: float64Ptr(55.0),
				},
				QuietHours: models.QuietHours{
					StartLocal: "22:00",
					EndLocal:   "07:00",
				},
				Schedule: &models.AlertSchedule{
					DaysOfWeek:          []int{1, 2, 3, 4, 5},
					EvaluationTimeLocal: "07:15",
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, http.StatusOK, subscriptions)
}

// CreateAlertSubscription handles POST /v1/me/alerts/subscriptions - create alert subscription.
func (h *AlertHandler) CreateAlertSubscription(w http.ResponseWriter, r *http.Request) {
	var input models.AlertSubscriptionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Validate input and save to database
	now := models.Timestamp(time.Now())
	subscriptionID := "sub_" + uuid.New().String()[:22]

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	subscription := models.AlertSubscription{
		ID:        subscriptionID,
		CommuteID: input.CommuteID,
		Enabled:   enabled,
		Objective: input.Objective,
		Threshold: input.Threshold,
		QuietHours: models.QuietHours{
			StartLocal: "22:00",
			EndLocal:   "07:00",
		},
		Schedule:  &input.Schedule,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if input.QuietHours != nil {
		subscription.QuietHours = *input.QuietHours
	}

	location := fmt.Sprintf("/v1/me/alerts/subscriptions/%s", subscriptionID)
	response.Created(w, location, subscription)
}

// GetAlertSubscription handles GET /v1/me/alerts/subscriptions/{subscriptionId}.
func (h *AlertHandler) GetAlertSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		response.BadRequest(w, r, "subscriptionId is required", nil)
		return
	}

	// TODO: Get actual subscription from database
	now := models.Timestamp(time.Now())
	subscription := models.AlertSubscription{
		ID:        subscriptionID,
		CommuteID: "cmt_01HY2ABCDEF0123456789",
		Enabled:   true,
		Objective: models.ObjectiveLowestExposure,
		Threshold: models.AlertThreshold{
			Type:          models.ThresholdAbsoluteScore,
			AbsoluteScore: float64Ptr(55.0),
		},
		QuietHours: models.QuietHours{
			StartLocal: "22:00",
			EndLocal:   "07:00",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	response.JSON(w, http.StatusOK, subscription)
}

// UpdateAlertSubscription handles PUT /v1/me/alerts/subscriptions/{subscriptionId}.
func (h *AlertHandler) UpdateAlertSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		response.BadRequest(w, r, "subscriptionId is required", nil)
		return
	}

	var input models.AlertSubscriptionUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Update subscription in database
	now := models.Timestamp(time.Now())
	subscription := models.AlertSubscription{
		ID:        subscriptionID,
		CommuteID: "cmt_01HY2ABCDEF0123456789",
		Enabled:   true,
		Objective: models.ObjectiveLowestExposure,
		Threshold: models.AlertThreshold{
			Type:          models.ThresholdAbsoluteScore,
			AbsoluteScore: float64Ptr(55.0),
		},
		QuietHours: models.QuietHours{
			StartLocal: "22:00",
			EndLocal:   "07:00",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if input.Enabled != nil {
		subscription.Enabled = *input.Enabled
	}
	if input.Objective != nil {
		subscription.Objective = *input.Objective
	}

	response.JSON(w, http.StatusOK, subscription)
}

// DeleteAlertSubscription handles DELETE /v1/me/alerts/subscriptions/{subscriptionId}.
func (h *AlertHandler) DeleteAlertSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := chi.URLParam(r, "subscriptionId")
	if subscriptionID == "" {
		response.BadRequest(w, r, "subscriptionId is required", nil)
		return
	}

	// TODO: Delete subscription from database
	response.NoContent(w)
}

func float64Ptr(f float64) *float64 {
	return &f
}
