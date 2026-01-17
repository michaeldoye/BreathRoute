package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Predefined service errors.
var (
	ErrUserNotFound = errors.New("user not found")
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	// FindByAppleSub finds a user by their Apple subject identifier.
	FindByAppleSub(ctx context.Context, appleSub string) (*User, error)

	// Create creates a new user.
	Create(ctx context.Context, user *User) error

	// FindByID finds a user by their internal ID.
	FindByID(ctx context.Context, id string) (*User, error)
}

// RefreshTokenRepository defines the interface for refresh token operations.
type RefreshTokenRepository interface {
	// Create stores a new refresh token.
	Create(ctx context.Context, token *RefreshToken) error

	// FindByToken finds a refresh token by its value.
	FindByToken(ctx context.Context, token string) (*RefreshToken, error)

	// Revoke marks a refresh token as revoked.
	Revoke(ctx context.Context, token string) error

	// RevokeAllForUser revokes all refresh tokens for a user.
	RevokeAllForUser(ctx context.Context, userID string) error
}

// Service provides authentication operations.
type Service struct {
	siwaVerifier  *SIWAVerifier
	jwtService    *JWTService
	userRepo      UserRepository
	refreshRepo   RefreshTokenRepository
	defaultLocale string
}

// ServiceConfig holds configuration for the auth service.
type ServiceConfig struct {
	SIWAVerifier  *SIWAVerifier
	JWTService    *JWTService
	UserRepo      UserRepository
	RefreshRepo   RefreshTokenRepository
	DefaultLocale string
}

// NewService creates a new auth service.
func NewService(cfg ServiceConfig) *Service {
	locale := cfg.DefaultLocale
	if locale == "" {
		locale = "nl-NL"
	}

	return &Service{
		siwaVerifier:  cfg.SIWAVerifier,
		jwtService:    cfg.JWTService,
		userRepo:      cfg.UserRepo,
		refreshRepo:   cfg.RefreshRepo,
		defaultLocale: locale,
	}
}

// AuthenticateWithApple authenticates a user using Sign in with Apple.
// It verifies the Apple identity token, creates a user if needed, and returns API tokens.
func (s *Service) AuthenticateWithApple(ctx context.Context, req *SIWATokenRequest) (*TokenResponse, error) {
	// Validate request
	if errs := req.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("validation error: %s", errs[0].Message)
	}

	// Verify the Apple identity token
	claims, err := s.siwaVerifier.VerifyToken(ctx, req.IdentityToken, req.Nonce)
	if err != nil {
		return nil, fmt.Errorf("verifying Apple token: %w", err)
	}

	// Find or create user
	user, err := s.findOrCreateUser(ctx, claims)
	if err != nil {
		return nil, fmt.Errorf("finding or creating user: %w", err)
	}

	// Generate tokens
	return s.generateTokens(ctx, user)
}

// RefreshAccessToken refreshes an access token using a refresh token.
func (s *Service) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (*TokenResponse, error) {
	// Find the refresh token
	refreshToken, err := s.refreshRepo.FindByToken(ctx, refreshTokenStr)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Check if token is valid
	if refreshToken.RevokedAt != nil {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}

	// Get the user
	user, err := s.userRepo.FindByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Revoke the old refresh token (rotation)
	if err := s.refreshRepo.Revoke(ctx, refreshTokenStr); err != nil {
		return nil, fmt.Errorf("revoking old refresh token: %w", err)
	}

	// Generate new tokens
	return s.generateTokens(ctx, user)
}

// ValidateAccessToken validates an access token and returns the user ID.
func (s *Service) ValidateAccessToken(tokenString string) (string, error) {
	claims, err := s.jwtService.ValidateAccessToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, userID string) (*User, error) {
	return s.userRepo.FindByID(ctx, userID)
}

// RevokeRefreshToken revokes a specific refresh token.
func (s *Service) RevokeRefreshToken(ctx context.Context, refreshTokenStr string) error {
	return s.refreshRepo.Revoke(ctx, refreshTokenStr)
}

// RevokeAllTokens revokes all refresh tokens for a user (logout everywhere).
func (s *Service) RevokeAllTokens(ctx context.Context, userID string) error {
	return s.refreshRepo.RevokeAllForUser(ctx, userID)
}

// findOrCreateUser finds an existing user or creates a new one.
func (s *Service) findOrCreateUser(ctx context.Context, claims *AppleClaims) (*User, error) {
	// Try to find existing user
	user, err := s.userRepo.FindByAppleSub(ctx, claims.Subject)
	if err == nil {
		return user, nil
	}

	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	// Create new user
	now := time.Now()
	user = &User{
		ID:        generateUserID(),
		AppleSub:  claims.Subject,
		Email:     claims.Email,
		Locale:    s.defaultLocale,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return user, nil
}

// generateTokens generates both access and refresh tokens for a user.
func (s *Service) generateTokens(ctx context.Context, user *User) (*TokenResponse, error) {
	// Generate access token
	accessToken, expiresAt, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("generating access token: %w", err)
	}

	// Generate refresh token
	refreshTokenStr, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	// Store refresh token
	refreshToken := &RefreshToken{
		ID:        uuid.New().String(),
		Token:     refreshTokenStr,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(RefreshTokenExpiry),
		CreatedAt: time.Now(),
	}

	if err := s.refreshRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(expiresAt).Seconds()),
		RefreshToken: refreshTokenStr,
		User:         user,
	}, nil
}

// generateUserID generates a unique user ID with prefix.
func generateUserID() string {
	return "usr_" + uuid.New().String()[:22]
}

// DevAuthenticateRequest is the request for development authentication.
type DevAuthenticateRequest struct {
	// UserID is an optional user ID. If not provided, a new user is created.
	UserID string `json:"userId,omitempty"`
	// Email is an optional email for the test user.
	Email string `json:"email,omitempty"`
}

// DevAuthenticate creates or retrieves a test user and returns tokens.
// This is intended for local development only and should never be enabled in production.
func (s *Service) DevAuthenticate(ctx context.Context, req *DevAuthenticateRequest) (*TokenResponse, error) {
	var user *User

	if req.UserID != "" {
		// Try to find existing user
		var err error
		user, err = s.userRepo.FindByID(ctx, req.UserID)
		if err != nil && !errors.Is(err, ErrUserNotFound) {
			return nil, fmt.Errorf("finding user: %w", err)
		}
	}

	if user == nil {
		// Create a new test user
		now := time.Now()
		testSub := "dev_" + uuid.New().String()[:8]
		email := req.Email
		if email == "" {
			email = testSub + "@dev.local"
		}

		user = &User{
			ID:        generateUserID(),
			AppleSub:  testSub,
			Email:     email,
			Locale:    s.defaultLocale,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("creating test user: %w", err)
		}
	}

	// Generate tokens
	return s.generateTokens(ctx, user)
}
