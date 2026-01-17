// Package auth provides authentication services for BreatheRoute.
package auth

import "time"

// User represents an authenticated user in the system.
type User struct {
	ID        string    `json:"userId"`
	AppleSub  string    `json:"-"` // Apple's user identifier (never exposed in API)
	Email     string    `json:"email,omitempty"`
	Locale    string    `json:"locale"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SIWATokenRequest represents the request body for Sign in with Apple authentication.
type SIWATokenRequest struct {
	// IdentityToken is the JWT identity token received from Apple on the iOS device.
	IdentityToken string `json:"identityToken"`

	// AuthorizationCode is the short-lived code for server-to-server validation (optional).
	AuthorizationCode string `json:"authorizationCode,omitempty"`

	// Nonce is the nonce used when requesting the token from Apple (for replay protection).
	Nonce string `json:"nonce,omitempty"`
}

// Validate validates the SIWA token request.
func (r *SIWATokenRequest) Validate() []FieldError {
	var errors []FieldError

	if r.IdentityToken == "" {
		errors = append(errors, FieldError{
			Field:   "identityToken",
			Message: "identity token is required",
			Code:    "REQUIRED",
		})
	}

	return errors
}

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// TokenResponse represents the response after successful authentication.
type TokenResponse struct {
	// AccessToken is the JWT access token for API authentication.
	AccessToken string `json:"accessToken"`

	// TokenType is always "Bearer".
	TokenType string `json:"tokenType"`

	// ExpiresIn is the number of seconds until the access token expires.
	ExpiresIn int64 `json:"expiresIn"`

	// RefreshToken is the opaque token used to obtain new access tokens.
	RefreshToken string `json:"refreshToken,omitempty"`

	// User contains the authenticated user's information.
	User *User `json:"user"`
}

// RefreshTokenRequest represents the request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// Validate validates the refresh token request.
func (r *RefreshTokenRequest) Validate() []FieldError {
	var errors []FieldError

	if r.RefreshToken == "" {
		errors = append(errors, FieldError{
			Field:   "refreshToken",
			Message: "refresh token is required",
			Code:    "REQUIRED",
		})
	}

	return errors
}

// AppleClaims represents the claims in an Apple identity token.
type AppleClaims struct {
	// Issuer is always "https://appleid.apple.com".
	Issuer string `json:"iss"`

	// Subject is the unique identifier for the user.
	Subject string `json:"sub"`

	// Audience is the client_id (your app's bundle ID).
	Audience string `json:"aud"`

	// IssuedAt is when the token was issued.
	IssuedAt int64 `json:"iat"`

	// ExpiresAt is when the token expires.
	ExpiresAt int64 `json:"exp"`

	// Nonce is the nonce value passed to Apple when requesting the token.
	Nonce string `json:"nonce,omitempty"`

	// NonceSupported indicates if the nonce is supported.
	NonceSupported bool `json:"nonce_supported,omitempty"`

	// Email is the user's email (may not always be present).
	Email string `json:"email,omitempty"`

	// EmailVerified indicates if the email is verified.
	EmailVerified string `json:"email_verified,omitempty"`

	// IsPrivateEmail indicates if the email is a private relay email.
	IsPrivateEmail string `json:"is_private_email,omitempty"`

	// RealUserStatus indicates the likelihood that the user is real.
	// 0 = Unsupported, 1 = Unknown, 2 = LikelyReal
	RealUserStatus int `json:"real_user_status,omitempty"`

	// AuthTime is when the user authenticated with Apple.
	AuthTime int64 `json:"auth_time,omitempty"`
}
