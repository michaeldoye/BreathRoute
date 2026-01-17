package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/breatheroute/breatheroute/internal/api/models"
	"github.com/breatheroute/breatheroute/internal/api/response"
	"github.com/breatheroute/breatheroute/internal/auth"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *auth.Service
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// SignInWithApple handles POST /v1/auth/siwa - Sign in with Apple authentication.
func (h *AuthHandler) SignInWithApple(w http.ResponseWriter, r *http.Request) {
	var req auth.SIWATokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		fieldErrors := make([]models.FieldError, len(errs))
		for i, e := range errs {
			fieldErrors[i] = models.FieldError{
				Field:   e.Field,
				Message: e.Message,
				Code:    e.Code,
			}
		}
		response.BadRequest(w, r, "validation error", fieldErrors)
		return
	}

	// Authenticate with Apple
	tokenResp, err := h.authService.AuthenticateWithApple(r.Context(), &req)
	if err != nil {
		// Map specific errors to appropriate responses
		if errors.Is(err, auth.ErrInvalidToken) ||
			errors.Is(err, auth.ErrInvalidIssuer) ||
			errors.Is(err, auth.ErrInvalidAudience) ||
			errors.Is(err, auth.ErrNonceMismatch) {
			response.Unauthorized(w, r, "invalid Apple identity token")
			return
		}
		if errors.Is(err, auth.ErrTokenExpired) {
			response.Unauthorized(w, r, "Apple identity token has expired")
			return
		}
		if errors.Is(err, auth.ErrKeyNotFound) ||
			errors.Is(err, auth.ErrFetchingAppleKeys) {
			response.ServiceUnavailable(w, r, "unable to verify Apple token at this time")
			return
		}

		// Generic error
		response.InternalError(w, r, "authentication failed")
		return
	}

	// Return the token response
	response.JSON(w, r, http.StatusOK, tokenResp)
}

// RefreshToken handles POST /v1/auth/refresh - refresh access token.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req auth.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		fieldErrors := make([]models.FieldError, len(errs))
		for i, e := range errs {
			fieldErrors[i] = models.FieldError{
				Field:   e.Field,
				Message: e.Message,
				Code:    e.Code,
			}
		}
		response.BadRequest(w, r, "validation error", fieldErrors)
		return
	}

	// Refresh the token
	tokenResp, err := h.authService.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			response.Unauthorized(w, r, "invalid refresh token")
			return
		}
		if errors.Is(err, auth.ErrRefreshTokenExpired) {
			response.Unauthorized(w, r, "refresh token has expired")
			return
		}
		if errors.Is(err, auth.ErrUserNotFound) {
			response.Unauthorized(w, r, "user not found")
			return
		}

		response.InternalError(w, r, "token refresh failed")
		return
	}

	response.JSON(w, r, http.StatusOK, tokenResp)
}

// Logout handles POST /v1/auth/logout - revoke current session.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, r, "invalid JSON body", nil)
		return
	}

	if req.RefreshToken == "" {
		response.BadRequest(w, r, "refreshToken is required", nil)
		return
	}

	// Revoke the refresh token
	if err := h.authService.RevokeRefreshToken(r.Context(), req.RefreshToken); err != nil {
		// Log error but don't expose details
		response.InternalError(w, r, "logout failed")
		return
	}

	response.NoContent(w, r)
}

// LogoutAll handles POST /v1/auth/logout-all - revoke all sessions for the user.
// This endpoint requires authentication.
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID := GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, r, "authentication required")
		return
	}

	// Revoke all refresh tokens for the user
	if err := h.authService.RevokeAllTokens(r.Context(), userID); err != nil {
		response.InternalError(w, r, "logout failed")
		return
	}

	response.NoContent(w, r)
}

// DevLogin handles POST /v1/auth/dev - development-only authentication.
// This endpoint is only available when AUTH_DEV_MODE=true.
// It creates a test user and returns valid tokens for local testing.
func (h *AuthHandler) DevLogin(w http.ResponseWriter, r *http.Request) {
	var req auth.DevAuthenticateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body - will create a new user with defaults
		req = auth.DevAuthenticateRequest{}
	}

	tokenResp, err := h.authService.DevAuthenticate(r.Context(), &req)
	if err != nil {
		response.InternalError(w, r, "dev authentication failed")
		return
	}

	response.JSON(w, r, http.StatusOK, tokenResp)
}
