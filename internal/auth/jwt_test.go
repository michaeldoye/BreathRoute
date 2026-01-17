package auth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/auth"
)

func TestJWTService_GenerateAndValidateAccessToken(t *testing.T) {
	svc := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	user := &auth.User{
		ID:        "usr_test123",
		AppleSub:  "apple.sub.123",
		Email:     "test@example.com",
		Locale:    "nl-NL",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate token
	token, expiresAt, err := svc.GenerateAccessToken(user)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))

	// Validate token
	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, user.ID, claims.UserID)
	assert.Equal(t, user.ID, claims.Subject)
	assert.Equal(t, "https://api.breatheroute.nl", claims.Issuer)
}

func TestJWTService_InvalidToken(t *testing.T) {
	svc := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-secret-key-for-testing-only",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"malformed token", "not.a.valid.jwt"},
		{"invalid base64", "xxx.yyy.zzz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateAccessToken(tt.token)
			assert.Error(t, err)
		})
	}
}

func TestJWTService_WrongSigningKey(t *testing.T) {
	// Generate with one key
	svc1 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "key-one",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	user := &auth.User{ID: "usr_test123"}
	token, _, err := svc1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Validate with different key
	svc2 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "key-two",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "breatheroute-api",
	})

	_, err = svc2.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.ErrorIs(t, err, auth.ErrInvalidAccessToken)
}

func TestJWTService_WrongIssuer(t *testing.T) {
	// Generate with one issuer
	svc1 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-key",
		Issuer:     "issuer-one",
		Audience:   "breatheroute-api",
	})

	user := &auth.User{ID: "usr_test123"}
	token, _, err := svc1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Validate with different issuer
	svc2 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-key",
		Issuer:     "issuer-two",
		Audience:   "breatheroute-api",
	})

	_, err = svc2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestJWTService_WrongAudience(t *testing.T) {
	// Generate with one audience
	svc1 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-key",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "audience-one",
	})

	user := &auth.User{ID: "usr_test123"}
	token, _, err := svc1.GenerateAccessToken(user)
	require.NoError(t, err)

	// Validate with different audience
	svc2 := auth.NewJWTService(auth.JWTConfig{
		SigningKey: "test-key",
		Issuer:     "https://api.breatheroute.nl",
		Audience:   "audience-two",
	})

	_, err = svc2.ValidateAccessToken(token)
	assert.Error(t, err)
}

func TestGenerateRefreshToken(t *testing.T) {
	token1, err := auth.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	token2, err := auth.GenerateRefreshToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token2)

	// Tokens should be unique
	assert.NotEqual(t, token1, token2)

	// Tokens should be URL-safe base64
	assert.Regexp(t, `^[A-Za-z0-9_-]+$`, token1)
}
