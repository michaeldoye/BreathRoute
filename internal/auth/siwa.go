package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

const (
	// AppleIssuer is the expected issuer for Apple identity tokens.
	AppleIssuer = "https://appleid.apple.com"

	// AppleKeysURL is the URL to fetch Apple's public keys.
	AppleKeysURL = "https://appleid.apple.com/auth/keys"

	// keyCacheRefreshInterval is how often to refresh the Apple public keys.
	keyCacheRefreshInterval = 24 * time.Hour
)

// Predefined errors for SIWA verification.
var (
	ErrInvalidToken      = errors.New("invalid identity token")
	ErrTokenExpired      = errors.New("identity token has expired")
	ErrInvalidIssuer     = errors.New("invalid token issuer")
	ErrInvalidAudience   = errors.New("invalid token audience")
	ErrNonceMismatch     = errors.New("nonce mismatch")
	ErrKeyNotFound       = errors.New("signing key not found")
	ErrFetchingAppleKeys = errors.New("failed to fetch Apple public keys")
	ErrInvalidKeyFormat  = errors.New("invalid key format")
)

// AppleJWK represents a single JSON Web Key from Apple.
type AppleJWK struct {
	Kty string `json:"kty"` // Key type (RSA)
	Kid string `json:"kid"` // Key ID
	Use string `json:"use"` // Key use (sig)
	Alg string `json:"alg"` // Algorithm (RS256)
	N   string `json:"n"`   // RSA modulus
	E   string `json:"e"`   // RSA exponent
}

// AppleJWKS represents Apple's JSON Web Key Set.
type AppleJWKS struct {
	Keys []AppleJWK `json:"keys"`
}

// HTTPDoer is an interface for making HTTP requests.
// Both *http.Client and *resilience.Client satisfy this interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// SIWAVerifier verifies Sign in with Apple identity tokens.
type SIWAVerifier struct {
	httpClient HTTPDoer
	bundleID   string // Your app's bundle ID (audience)

	// Key cache
	mu            sync.RWMutex
	keys          map[string]*rsa.PublicKey
	keysUpdatedAt time.Time
}

// SIWAConfig holds configuration for the SIWA verifier.
type SIWAConfig struct {
	// BundleID is your iOS app's bundle identifier.
	BundleID string

	// HTTPClient is an optional custom HTTP client for fetching keys.
	// Can be *http.Client or *resilience.Client.
	// If nil, a resilient client with circuit breaker is used.
	HTTPClient HTTPDoer
}

// NewSIWAVerifier creates a new Sign in with Apple token verifier.
func NewSIWAVerifier(cfg SIWAConfig) *SIWAVerifier {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		// Use a resilient client with circuit breaker and retry logic
		httpClient = resilience.NewClient(resilience.ClientConfig{
			Name:            "apple-siwa",
			Timeout:         10 * time.Second,
			MaxRetries:      3,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     2 * time.Second,
		})
	}

	return &SIWAVerifier{
		httpClient: httpClient,
		bundleID:   cfg.BundleID,
		keys:       make(map[string]*rsa.PublicKey),
	}
}

// VerifyToken verifies an Apple identity token and returns the claims.
func (v *SIWAVerifier) VerifyToken(ctx context.Context, tokenString, expectedNonce string) (*AppleClaims, error) {
	// Parse the token without verification first to get the key ID
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}

	// Get the key ID from the token header
	kid, ok := token.Header["kid"].(string)
	if !ok || kid == "" {
		return nil, fmt.Errorf("%w: missing key ID", ErrInvalidToken)
	}

	// Get the public key for verification
	publicKey, err := v.getPublicKey(ctx, kid)
	if err != nil {
		return nil, err
	}

	// Parse and verify the token
	token, err = jwt.NewParser(
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(AppleIssuer),
		jwt.WithAudience(v.bundleID),
		jwt.WithExpirationRequired(),
	).ParseWithClaims(tokenString, &appleClaims{}, func(t *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenInvalidIssuer) {
			return nil, ErrInvalidIssuer
		}
		if errors.Is(err, jwt.ErrTokenInvalidAudience) {
			return nil, ErrInvalidAudience
		}
		return nil, fmt.Errorf("%w: %s", ErrInvalidToken, err.Error())
	}

	// Extract claims
	ac, ok := token.Claims.(*appleClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify nonce if provided
	if expectedNonce != "" && ac.Nonce != expectedNonce {
		return nil, ErrNonceMismatch
	}

	// Convert to our claims type
	claims := &AppleClaims{
		Issuer:         ac.Issuer,
		Subject:        ac.Subject,
		Audience:       v.bundleID,
		IssuedAt:       ac.IssuedAt.Unix(),
		ExpiresAt:      ac.ExpiresAt.Unix(),
		Nonce:          ac.Nonce,
		NonceSupported: ac.NonceSupported,
		Email:          ac.Email,
		EmailVerified:  ac.EmailVerified,
		IsPrivateEmail: ac.IsPrivateEmail,
		RealUserStatus: ac.RealUserStatus,
		AuthTime:       ac.AuthTime,
	}

	return claims, nil
}

// appleClaims is an internal type implementing jwt.Claims.
type appleClaims struct {
	jwt.RegisteredClaims
	Nonce          string `json:"nonce,omitempty"`
	NonceSupported bool   `json:"nonce_supported,omitempty"`
	Email          string `json:"email,omitempty"`
	EmailVerified  string `json:"email_verified,omitempty"`
	IsPrivateEmail string `json:"is_private_email,omitempty"`
	RealUserStatus int    `json:"real_user_status,omitempty"`
	AuthTime       int64  `json:"auth_time,omitempty"`
}

// getPublicKey retrieves the public key for the given key ID.
func (v *SIWAVerifier) getPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Check cache first
	v.mu.RLock()
	key, ok := v.keys[kid]
	needsRefresh := time.Since(v.keysUpdatedAt) > keyCacheRefreshInterval
	v.mu.RUnlock()

	if ok && !needsRefresh {
		return key, nil
	}

	// Refresh keys
	if err := v.refreshKeys(ctx); err != nil {
		// If we have a cached key, use it even if refresh failed
		v.mu.RLock()
		key, ok = v.keys[kid]
		v.mu.RUnlock()
		if ok {
			return key, nil
		}
		return nil, err
	}

	// Get key from refreshed cache
	v.mu.RLock()
	key, ok = v.keys[kid]
	v.mu.RUnlock()

	if !ok {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

// refreshKeys fetches the latest public keys from Apple.
func (v *SIWAVerifier) refreshKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, AppleKeysURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFetchingAppleKeys, err.Error())
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFetchingAppleKeys, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrFetchingAppleKeys, resp.StatusCode)
	}

	var jwks AppleJWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("%w: %s", ErrFetchingAppleKeys, err.Error())
	}

	// Convert JWKs to RSA public keys
	newKeys := make(map[string]*rsa.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		key, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			continue // Skip invalid keys
		}
		newKeys[jwk.Kid] = key
	}

	// Update cache
	v.mu.Lock()
	v.keys = newKeys
	v.keysUpdatedAt = time.Now()
	v.mu.Unlock()

	return nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key.
func jwkToRSAPublicKey(jwk AppleJWK) (*rsa.PublicKey, error) {
	// Decode modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid modulus", ErrInvalidKeyFormat)
	}
	n := new(big.Int).SetBytes(nBytes)

	// Decode exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid exponent", ErrInvalidKeyFormat)
	}

	// Convert exponent bytes to int
	var e int
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}
