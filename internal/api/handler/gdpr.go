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

// GDPRHandler handles GDPR endpoints.
type GDPRHandler struct{}

// NewGDPRHandler creates a new GDPRHandler.
func NewGDPRHandler() *GDPRHandler {
	return &GDPRHandler{}
}

// CreateExportRequest handles POST /v1/gdpr/export-requests - create export request.
func (h *GDPRHandler) CreateExportRequest(w http.ResponseWriter, r *http.Request) {
	var input models.ExportRequestCreate
	// Body is optional, ignore decode errors
	_ = json.NewDecoder(r.Body).Decode(&input)
	// TODO: Create export job
	now := models.Timestamp(time.Now())
	requestID := "exp_" + uuid.New().String()[:22]

	exportRequest := models.ExportRequest{
		ID:        requestID,
		Status:    models.ExportStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	location := fmt.Sprintf("/v1/gdpr/export-requests/%s", requestID)
	response.Accepted(w, r, location, exportRequest)
}

// ListExportRequests handles GET /v1/gdpr/export-requests - list export requests.
func (h *GDPRHandler) ListExportRequests(w http.ResponseWriter, r *http.Request) {
	// TODO: Get actual export requests from database
	now := models.Timestamp(time.Now())
	requests := models.PagedExportRequests{
		Items: []models.ExportRequest{
			{
				ID:        "exp_01HY5ABCDEF0123456789",
				Status:    models.ExportStatusReady,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, r, http.StatusOK, requests)
}

// GetExportRequest handles GET /v1/gdpr/export-requests/{exportRequestId}.
func (h *GDPRHandler) GetExportRequest(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "exportRequestId")
	if requestID == "" {
		response.BadRequest(w, r, "exportRequestId is required", nil)
		return
	}

	// TODO: Get actual export request from database
	now := models.Timestamp(time.Now())
	exportRequest := models.ExportRequest{
		ID:        requestID,
		Status:    models.ExportStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	response.JSON(w, r, http.StatusOK, exportRequest)
}

// CreateDeletionRequest handles POST /v1/gdpr/deletion-requests - create deletion request.
func (h *GDPRHandler) CreateDeletionRequest(w http.ResponseWriter, r *http.Request) {
	var input models.DeletionRequestCreate
	// Body is optional, ignore decode errors
	_ = json.NewDecoder(r.Body).Decode(&input)
	// TODO: Create deletion job
	now := models.Timestamp(time.Now())
	requestID := "del_" + uuid.New().String()[:22]

	deletionRequest := models.DeletionRequest{
		ID:        requestID,
		Status:    models.DeletionStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	location := fmt.Sprintf("/v1/gdpr/deletion-requests/%s", requestID)
	response.Accepted(w, r, location, deletionRequest)
}

// ListDeletionRequests handles GET /v1/gdpr/deletion-requests - list deletion requests.
func (h *GDPRHandler) ListDeletionRequests(w http.ResponseWriter, r *http.Request) {
	// TODO: Get actual deletion requests from database
	now := models.Timestamp(time.Now())
	requests := models.PagedDeletionRequests{
		Items: []models.DeletionRequest{
			{
				ID:        "del_01HY6ABCDEF0123456789",
				Status:    models.DeletionStatusPending,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		Meta: models.PagedResponseMeta{
			Limit: 50,
		},
	}
	response.JSON(w, r, http.StatusOK, requests)
}

// GetDeletionRequest handles GET /v1/gdpr/deletion-requests/{deletionRequestId}.
func (h *GDPRHandler) GetDeletionRequest(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "deletionRequestId")
	if requestID == "" {
		response.BadRequest(w, r, "deletionRequestId is required", nil)
		return
	}

	// TODO: Get actual deletion request from database
	now := models.Timestamp(time.Now())
	deletionRequest := models.DeletionRequest{
		ID:        requestID,
		Status:    models.DeletionStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	response.JSON(w, r, http.StatusOK, deletionRequest)
}
