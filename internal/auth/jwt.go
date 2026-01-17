package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token Expiry and Session Policy
//
// This package implements a dual-token authentication strategy:
//
// 1. ACCESS TOKENS (Short-lived JWTs)
//    - Expiry: 1 hour
//    - Purpose: Authenticate API requests via Bearer token in Authorization header
//    - Storage: Should be stored in memory only on the client (not persisted)
//    - On expiry: Client should use refresh token to obtain a new access token
//    - Claims: Contains user ID, issuer, audience, issued-at, and expiry
//
// 2. REFRESH TOKENS (Long-lived opaque tokens)
//    - Expiry: 30 days
//    - Purpose: Obtain new access tokens without re-authenticating with Apple
//    - Storage: Should be stored securely on the client (Keychain on iOS)
//    - Rotation: Each use generates a new refresh token (old one is revoked)
//    - Revocation: Can be explicitly revoked via logout endpoints
//
// Token Refresh Flow:
//    1. Client detects access token expired (401 response or local expiry check)
//    2. Client calls POST /v1/auth/refresh with refresh token
//    3. Server validates refresh token, revokes it, and issues new token pair
//    4. Client stores new tokens and retries the failed request
//
// Security Considerations:
//    - Refresh token rotation prevents token theft from being persistent
//    - Short access token expiry limits damage from token leakage
//    - All refresh tokens can be revoked via POST /v1/auth/logout-all
//    - Tokens are signed with HS256 using a server-side secret key
//
// Session Termination:
//    - POST /v1/auth/logout: Revokes a specific refresh token
//    - POST /v1/auth/logout-all: Revokes all refresh tokens for the user
//    - Access tokens remain valid until expiry (consider a token blacklist for immediate revocation)

// Token expiry constants.
const (
	// AccessTokenExpiry is how long access tokens are valid.
	// Short expiry (1 hour) limits exposure if a token is compromised.
	AccessTokenExpiry = 1 * time.Hour

	// RefreshTokenExpiry is how long refresh tokens are valid.
	// 30 days provides a balance between security and user convenience.
	// Users won't need to re-authenticate with Apple frequently.
	RefreshTokenExpiry = 30 * 24 * time.Hour // 30 days

	// RefreshTokenLength is the byte length of refresh tokens.
	// 32 bytes = 256 bits of entropy, providing strong security.
	RefreshTokenLength = 32
)

// Predefined JWT errors.
var (
	ErrInvalidAccessToken  = errors.New("invalid access token")
	ErrAccessTokenExpired  = errors.New("access token has expired")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token has expired")
)

// JWTClaims represents the claims in our API access tokens.
type JWTClaims struct {
	jwt.RegisteredClaims

	// UserID is the authenticated user's ID.
	UserID string `json:"uid"`
}

// JWTService handles JWT creation and validation.
type JWTService struct {
	signingKey []byte
	issuer     string
	audience   string
}

// JWTConfig holds configuration for the JWT service.
type JWTConfig struct {
	// SigningKey is the secret key used to sign JWTs.
	SigningKey string

	// Issuer is the issuer claim for tokens (e.g., "https://api.breatheroute.nl").
	Issuer string

	// Audience is the audience claim for tokens (e.g., "breatheroute-api").
	Audience string
}

// NewJWTService creates a new JWT service.
func NewJWTService(cfg JWTConfig) *JWTService {
	return &JWTService{
		signingKey: []byte(cfg.SigningKey),
		issuer:     cfg.Issuer,
		audience:   cfg.Audience,
	}
}

// GenerateAccessToken creates a new access token for the given user.
func (s *JWTService) GenerateAccessToken(user *User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(AccessTokenExpiry)

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		UserID: user.ID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing access token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// ValidateAccessToken validates an access token and returns the claims.
func (s *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.signingKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(s.audience),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrAccessTokenExpired
		}
		return nil, fmt.Errorf("%w: %s", ErrInvalidAccessToken, err.Error())
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidAccessToken
	}

	return claims, nil
}

// RefreshToken represents a refresh token stored in the database.
type RefreshToken struct {
	ID        string
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// GenerateRefreshToken creates a new opaque refresh token.
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, RefreshTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateTokenID generates a unique token ID.
func generateTokenID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}
