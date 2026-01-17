package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/api"
	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/auth"
)

// testAuthService creates an auth service for testing.
func testAuthService() *auth.Service {
	siwaVerifier := auth.NewSIWAVerifier(auth.SIWAConfig{
		BundleID: "nl.breatheroute.app",
	})

	jwtService := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	userRepo := auth.NewInMemoryUserRepository()
	refreshRepo := auth.NewInMemoryRefreshTokenRepository()

	return auth.NewService(auth.ServiceConfig{
		SIWAVerifier:  siwaVerifier,
		JWTService:    jwtService,
		UserRepo:      userRepo,
		RefreshRepo:   refreshRepo,
		DefaultLocale: "nl-NL",
	})
}

// testJWTService creates a JWT service for generating test tokens.
func testJWTService() *auth.JWTService {
	return auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})
}

// generateTestToken generates a valid test token for a user.
func generateTestToken(t *testing.T) string {
	t.Helper()
	jwtService := testJWTService()
	user := &auth.User{
		ID:        "usr_testuser123",
		AppleSub:  "apple.123",
		Locale:    "nl-NL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	token, _, err := jwtService.GenerateAccessToken(user)
	require.NoError(t, err)
	return token
}

func newTestRouter() http.Handler {
	logger := zerolog.New(io.Discard)
	return api.NewRouter(api.RouterConfig{
		Version:     "test",
		BuildTime:   "2024-01-01T00:00:00Z",
		Logger:      logger,
		AuthService: testAuthService(),
	})
}

// addAuthHeader adds a valid Bearer token to the request.
func addAuthHeader(t *testing.T, req *http.Request) {
	t.Helper()
	token := generateTestToken(t)
	req.Header.Set("Authorization", "Bearer "+token)
}

func TestRouter_HealthCheck(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/ops/health", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.NotEmpty(t, w.Header().Get("X-Request-Id"))

	var health models.Health
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err)

	assert.Equal(t, models.HealthStatusOK, health.Status)
	assert.NotEmpty(t, health.Time)
}

func TestRouter_ReadinessCheck(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/ops/ready", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var health models.Health
	err := json.Unmarshal(w.Body.Bytes(), &health)
	require.NoError(t, err)

	assert.Equal(t, models.HealthStatusOK, health.Status)
}

func TestRouter_SystemStatus(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/ops/status", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status models.SystemStatus
	err := json.Unmarshal(w.Body.Bytes(), &status)
	require.NoError(t, err)

	assert.Equal(t, models.HealthStatusOK, status.Status)
	assert.NotEmpty(t, status.Subsystems)
	assert.NotEmpty(t, status.Providers)
}

func TestRouter_GetMe(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/me", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var me models.Me
	err := json.Unmarshal(w.Body.Bytes(), &me)
	require.NoError(t, err)

	assert.NotEmpty(t, me.UserID)
	assert.NotEmpty(t, me.Locale)
}

func TestRouter_GetProfile(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/me/profile", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var profile models.Profile
	err := json.Unmarshal(w.Body.Bytes(), &profile)
	require.NoError(t, err)

	assert.NotZero(t, profile.Weights.NO2)
}

func TestRouter_UpsertProfile(t *testing.T) {
	router := newTestRouter()

	input := models.ProfileInput{
		Weights: models.ExposureWeights{
			NO2:    0.5,
			PM25:   0.3,
			O3:     0.1,
			Pollen: 0.1,
		},
		Constraints: models.RouteConstraints{
			AvoidMajorRoads: true,
		},
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPut, "/v1/me/profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var profile models.Profile
	err := json.Unmarshal(w.Body.Bytes(), &profile)
	require.NoError(t, err)

	assert.Equal(t, 0.5, profile.Weights.NO2)
	assert.True(t, profile.Constraints.AvoidMajorRoads)
}

func TestRouter_ListCommutes(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/me/commutes", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var commutes models.PagedCommutes
	err := json.Unmarshal(w.Body.Bytes(), &commutes)
	require.NoError(t, err)

	assert.NotNil(t, commutes.Items)
	assert.NotZero(t, commutes.Meta.Limit)
}

func TestRouter_CreateCommute(t *testing.T) {
	router := newTestRouter()

	input := models.CommuteCreateRequest{
		Label: "Home → Work",
		Origin: models.CommuteLocation{
			Point: models.Point{Lat: 52.37, Lon: 4.89},
		},
		Destination: models.CommuteLocation{
			Point: models.Point{Lat: 52.31, Lon: 4.76},
		},
		DaysOfWeek:                []int{1, 2, 3, 4, 5},
		PreferredArrivalTimeLocal: "09:00",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/me/commutes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotEmpty(t, w.Header().Get("Location"))

	var commute models.Commute
	err := json.Unmarshal(w.Body.Bytes(), &commute)
	require.NoError(t, err)

	assert.Equal(t, "Home → Work", commute.Label)
	assert.NotEmpty(t, commute.ID)
}

func TestRouter_GetCommute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/me/commutes/cmt_test123", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var commute models.Commute
	err := json.Unmarshal(w.Body.Bytes(), &commute)
	require.NoError(t, err)

	assert.Equal(t, "cmt_test123", commute.ID)
}

func TestRouter_DeleteCommute(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodDelete, "/v1/me/commutes/cmt_test123", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestRouter_ComputeRoutes(t *testing.T) {
	router := newTestRouter()

	input := models.RouteComputeRequest{
		Origin:        &models.Point{Lat: 52.37, Lon: 4.89},
		Destination:   &models.Point{Lat: 52.31, Lon: 4.76},
		DepartureTime: "2026-01-15T08:00:00+01:00",
		Objective:     models.ObjectiveFastest,
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/routes:compute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.RouteComputeResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Options)
	assert.NotEmpty(t, resp.GeneratedAt)
}

func TestRouter_ComputeRoutes_ValidationError(t *testing.T) {
	router := newTestRouter()

	// Missing origin, destination, and commuteId
	input := models.RouteComputeRequest{
		DepartureTime: "2026-01-15T08:00:00+01:00",
		Objective:     models.ObjectiveFastest,
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/routes:compute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var problem models.Problem
	err := json.Unmarshal(w.Body.Bytes(), &problem)
	require.NoError(t, err)

	assert.Equal(t, models.ProblemTypeValidation, problem.Type)
	assert.NotEmpty(t, problem.TraceID)
}

func TestRouter_PreviewDepartureWindows(t *testing.T) {
	router := newTestRouter()

	input := models.AlertPreviewRequest{
		Origin:              &models.Point{Lat: 52.37, Lon: 4.89},
		Destination:         &models.Point{Lat: 52.31, Lon: 4.76},
		TargetDepartureTime: strPtr("2026-01-15T08:00:00+01:00"),
		Objective:           models.ObjectiveLowestExposure,
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/alerts/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp models.AlertPreviewResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.NotEmpty(t, resp.Recommended)
}

func TestRouter_ListDevices(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/me/devices", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var devices models.PagedDevices
	err := json.Unmarshal(w.Body.Bytes(), &devices)
	require.NoError(t, err)

	assert.NotNil(t, devices.Items)
}

func TestRouter_RegisterDevice(t *testing.T) {
	router := newTestRouter()

	input := models.DeviceRegisterRequest{
		DeviceID: "dev_test123",
		Platform: models.PushPlatformAPNS,
		Token:    "abc123token456xyz789",
	}
	body, _ := json.Marshal(input)

	req := httptest.NewRequest(http.MethodPost, "/v1/me/devices", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.NotEmpty(t, w.Header().Get("Location"))

	var device models.Device
	err := json.Unmarshal(w.Body.Bytes(), &device)
	require.NoError(t, err)

	assert.Equal(t, "dev_test123", device.ID)
	assert.Equal(t, models.PushPlatformAPNS, device.Platform)
}

func TestRouter_GetEnums(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/metadata/enums", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var enums models.Enums
	err := json.Unmarshal(w.Body.Bytes(), &enums)
	require.NoError(t, err)

	assert.Contains(t, enums.Modes, models.ModeWalk)
	assert.Contains(t, enums.Modes, models.ModeBike)
	assert.Contains(t, enums.Modes, models.ModeTrain)
	assert.Contains(t, enums.Objectives, models.ObjectiveFastest)
	assert.Contains(t, enums.Confidence, models.ConfidenceHigh)
}

func TestRouter_ListAirQualityStations(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/metadata/air-quality/stations", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stations models.PagedStations
	err := json.Unmarshal(w.Body.Bytes(), &stations)
	require.NoError(t, err)

	assert.NotEmpty(t, stations.Items)
}

func TestRouter_GDPR_ExportRequest(t *testing.T) {
	router := newTestRouter()

	// Create export request
	req := httptest.NewRequest(http.MethodPost, "/v1/gdpr/export-requests", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.NotEmpty(t, w.Header().Get("Location"))

	var exportReq models.ExportRequest
	err := json.Unmarshal(w.Body.Bytes(), &exportReq)
	require.NoError(t, err)

	assert.NotEmpty(t, exportReq.ID)
	assert.Equal(t, models.ExportStatusPending, exportReq.Status)
}

func TestRouter_GDPR_DeletionRequest(t *testing.T) {
	router := newTestRouter()

	// Create deletion request
	req := httptest.NewRequest(http.MethodPost, "/v1/gdpr/deletion-requests", http.NoBody)
	addAuthHeader(t, req)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.NotEmpty(t, w.Header().Get("Location"))

	var deleteReq models.DeletionRequest
	err := json.Unmarshal(w.Body.Bytes(), &deleteReq)
	require.NoError(t, err)

	assert.NotEmpty(t, deleteReq.ID)
	assert.Equal(t, models.DeletionStatusPending, deleteReq.Status)
}

func TestRouter_RequestID_Generated(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/ops/health", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-Id")
	assert.NotEmpty(t, requestID)
	assert.Contains(t, requestID, "req_")
}

func TestRouter_RequestID_Preserved(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/ops/health", http.NoBody)
	req.Header.Set("X-Request-Id", "custom_request_id")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, "custom_request_id", w.Header().Get("X-Request-Id"))
}

func TestRouter_NotFound(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/nonexistent", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func strPtr(s string) *string {
	return &s
}
