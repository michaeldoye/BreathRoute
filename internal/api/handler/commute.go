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

// CommuteHandler handles commute endpoints.
type CommuteHandler struct{}

// NewCommuteHandler creates a new CommuteHandler.
func NewCommuteHandler() *CommuteHandler {
	return &CommuteHandler{}
}

// ListCommutes handles GET /v1/me/commutes - list saved commutes.
func (h *CommuteHandler) ListCommutes(w http.ResponseWriter, _ *http.Request) {
	// TODO: Get actual commutes from database with pagination
	now := models.Timestamp(time.Now())
	commutes := models.PagedCommutes{
		Items: []models.Commute{
			{
				ID:    "cmt_01HY2ABCDEF0123456789",
				Label: "Home → Office",
				Origin: models.CommuteLocation{
					Point: models.Point{Lat: 52.370216, Lon: 4.895168},
				},
				Destination: models.CommuteLocation{
					Point: models.Point{Lat: 52.308056, Lon: 4.763889},
				},
				DaysOfWeek:                []int{1, 2, 3, 4, 5},
				PreferredArrivalTimeLocal: "09:00",
				CreatedAt:                 now,
				UpdatedAt:                 now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, http.StatusOK, commutes)
}

// CreateCommute handles POST /v1/me/commutes - create a saved commute.
func (h *CommuteHandler) CreateCommute(w http.ResponseWriter, r *http.Request) {
	var input models.CommuteCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Validate input using validator
	// TODO: Save commute to database

	now := models.Timestamp(time.Now())
	commuteID := "cmt_" + uuid.New().String()[:22]
	commute := models.Commute{
		ID:                        commuteID,
		Label:                     input.Label,
		Origin:                    input.Origin,
		Destination:               input.Destination,
		DaysOfWeek:                input.DaysOfWeek,
		PreferredArrivalTimeLocal: input.PreferredArrivalTimeLocal,
		Notes:                     input.Notes,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	location := fmt.Sprintf("/v1/me/commutes/%s", commuteID)
	response.Created(w, location, commute)
}

// GetCommute handles GET /v1/me/commutes/{commuteId} - get a saved commute.
func (h *CommuteHandler) GetCommute(w http.ResponseWriter, r *http.Request) {
	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	// TODO: Get actual commute from database
	now := models.Timestamp(time.Now())
	commute := models.Commute{
		ID:    commuteID,
		Label: "Home → Office",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.370216, Lon: 4.895168},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.308056, Lon: 4.763889},
		},
		DaysOfWeek:                []int{1, 2, 3, 4, 5},
		PreferredArrivalTimeLocal: "09:00",
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}
	response.JSON(w, http.StatusOK, commute)
}

// UpdateCommute handles PUT /v1/me/commutes/{commuteId} - update a saved commute.
func (h *CommuteHandler) UpdateCommute(w http.ResponseWriter, r *http.Request) {
	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	var input models.CommuteUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// TODO: Validate input using validator
	// TODO: Update commute in database

	now := models.Timestamp(time.Now())
	commute := models.Commute{
		ID:    commuteID,
		Label: "Home → Office",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.370216, Lon: 4.895168},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.308056, Lon: 4.763889},
		},
		DaysOfWeek:                []int{1, 2, 3, 4, 5},
		PreferredArrivalTimeLocal: "09:00",
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}

	if input.Label != nil {
		commute.Label = *input.Label
	}
	if input.Origin != nil {
		commute.Origin = *input.Origin
	}
	if input.Destination != nil {
		commute.Destination = *input.Destination
	}
	if input.DaysOfWeek != nil {
		commute.DaysOfWeek = input.DaysOfWeek
	}
	if input.PreferredArrivalTimeLocal != nil {
		commute.PreferredArrivalTimeLocal = *input.PreferredArrivalTimeLocal
	}
	if input.Notes != nil {
		commute.Notes = input.Notes
	}

	response.JSON(w, http.StatusOK, commute)
}

// DeleteCommute handles DELETE /v1/me/commutes/{commuteId} - delete a saved commute.
func (h *CommuteHandler) DeleteCommute(w http.ResponseWriter, r *http.Request) {
	commuteID := chi.URLParam(r, "commuteId")
	if commuteID == "" {
		response.BadRequest(w, r, "commuteId is required", nil)
		return
	}

	// TODO: Delete commute from database

	response.NoContent(w)
}
