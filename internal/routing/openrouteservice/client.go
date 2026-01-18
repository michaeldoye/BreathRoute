// Package openrouteservice provides a client for the OpenRouteService directions API.
package openrouteservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
	"github.com/breatheroute/breatheroute/internal/routing"
)

const (
	// ProviderName identifies this routing provider.
	ProviderName = "openrouteservice"

	// DefaultBaseURL is the OpenRouteService API base URL.
	DefaultBaseURL = "https://api.openrouteservice.org"

	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 10 * time.Second
)

// HTTPDoer is an interface for executing HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// ClientConfig holds configuration for the OpenRouteService client.
type ClientConfig struct {
	// APIKey is the ORS API key (required).
	APIKey string

	// BaseURL is the API base URL (optional, defaults to ORS API).
	BaseURL string

	// HTTPClient is the HTTP client to use (optional).
	// If nil, uses a resilient client with defaults.
	HTTPClient HTTPDoer

	// Timeout is the request timeout (optional, defaults to 10s).
	Timeout time.Duration

	// Registry is the provider registry for health tracking (optional).
	Registry *resilience.Registry

	// Logger for client operations.
	Logger zerolog.Logger
}

// Client is an OpenRouteService API client.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient HTTPDoer
	logger     zerolog.Logger
}

// NewClient creates a new OpenRouteService client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		clientCfg := resilience.DefaultClientConfig(ProviderName)
		clientCfg.Timeout = timeout
		if cfg.Registry != nil {
			clientCfg.Registry = cfg.Registry
		}
		httpClient = resilience.NewClient(clientCfg)
	}

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     cfg.Logger,
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return ProviderName
}

// SupportedProfiles returns the supported routing profiles.
func (c *Client) SupportedProfiles() []routing.RouteProfile {
	return []routing.RouteProfile{
		routing.ProfileWalk,
		routing.ProfileBike,
	}
}

// GetDirections retrieves route directions between two points.
func (c *Client) GetDirections(ctx context.Context, req routing.DirectionsRequest) (*routing.DirectionsResponse, error) {
	// Validate coordinates
	if err := validateCoordinates(req.Origin); err != nil {
		return nil, &routing.Error{
			Provider: ProviderName,
			Code:     "INVALID_ORIGIN",
			Message:  "invalid origin coordinates",
			Err:      routing.ErrInvalidCoordinates,
		}
	}
	if err := validateCoordinates(req.Destination); err != nil {
		return nil, &routing.Error{
			Provider: ProviderName,
			Code:     "INVALID_DESTINATION",
			Message:  "invalid destination coordinates",
			Err:      routing.ErrInvalidCoordinates,
		}
	}

	// Default max alternatives
	maxAlts := req.MaxAlternatives
	if maxAlts <= 0 {
		maxAlts = 2
	}

	// Build request body
	orsReq := orsRequest{
		// ORS uses [lon, lat] order (GeoJSON)
		Coordinates: [][]float64{
			{req.Origin.Lon, req.Origin.Lat},
			{req.Destination.Lon, req.Destination.Lat},
		},
		AlternativeRoutes: &alternativeRoutesOpts{
			TargetCount: maxAlts + 1, // +1 because the first route is not counted as alternative
		},
		Instructions: true,
		Geometry:     true,
		Units:        "m",
		Language:     "en",
	}

	body, err := json.Marshal(orsReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	// Build HTTP request
	url := fmt.Sprintf("%s/v2/directions/%s", c.baseURL, req.Profile)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", c.apiKey)
	httpReq.Header.Set("Accept", "application/json, application/geo+json")

	c.logger.Debug().
		Str("profile", string(req.Profile)).
		Float64("origin_lat", req.Origin.Lat).
		Float64("origin_lon", req.Origin.Lon).
		Float64("dest_lat", req.Destination.Lat).
		Float64("dest_lon", req.Destination.Lon).
		Msg("requesting directions from ORS")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, &routing.Error{
			Provider: ProviderName,
			Code:     "REQUEST_FAILED",
			Message:  "failed to reach routing provider",
			Err:      routing.ErrProviderUnavailable,
		}
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		return nil, c.handleErrorResponse(resp.StatusCode, respBody)
	}

	// Parse successful response
	var orsResp orsResponse
	if err := json.Unmarshal(respBody, &orsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	// Convert to domain model
	result := c.toDirectionsResponse(&orsResp)

	c.logger.Debug().
		Int("route_count", len(result.Routes)).
		Msg("received directions from ORS")

	return result, nil
}

// handleErrorResponse maps ORS error responses to domain errors.
func (c *Client) handleErrorResponse(statusCode int, body []byte) error {
	var orsErr orsErrorResponse
	if err := json.Unmarshal(body, &orsErr); err != nil {
		// Fall back to generic error if we can't parse
		return &routing.Error{
			Provider: ProviderName,
			Code:     fmt.Sprintf("HTTP_%d", statusCode),
			Message:  fmt.Sprintf("routing provider returned status %d", statusCode),
			Err:      routing.ErrProviderUnavailable,
		}
	}

	switch statusCode {
	case http.StatusTooManyRequests:
		return &routing.Error{
			Provider: ProviderName,
			Code:     "RATE_LIMIT",
			Message:  "API rate limit exceeded, please try again later",
			Err:      routing.ErrRateLimitExceeded,
		}
	case http.StatusForbidden:
		return &routing.Error{
			Provider: ProviderName,
			Code:     "FORBIDDEN",
			Message:  "API access denied - check API key configuration",
			Err:      routing.ErrProviderUnavailable,
		}
	case http.StatusNotFound:
		return &routing.Error{
			Provider: ProviderName,
			Code:     "NO_ROUTE",
			Message:  "no route found between the given points",
			Err:      routing.ErrNoRouteFound,
		}
	case http.StatusBadRequest:
		// Check for specific ORS error codes
		if orsErr.Error.Code == orsErrorCodeNotFound {
			return &routing.Error{
				Provider: ProviderName,
				Code:     "NO_ROUTE",
				Message:  orsErr.Error.Message,
				Err:      routing.ErrNoRouteFound,
			}
		}
		return &routing.Error{
			Provider: ProviderName,
			Code:     "BAD_REQUEST",
			Message:  orsErr.Error.Message,
			Err:      routing.ErrInvalidCoordinates,
		}
	default:
		if statusCode >= 500 {
			return &routing.Error{
				Provider: ProviderName,
				Code:     fmt.Sprintf("SERVER_%d", statusCode),
				Message:  "routing provider is temporarily unavailable",
				Err:      routing.ErrProviderUnavailable,
			}
		}
		return &routing.Error{
			Provider: ProviderName,
			Code:     fmt.Sprintf("HTTP_%d", statusCode),
			Message:  orsErr.Error.Message,
			Err:      routing.ErrProviderUnavailable,
		}
	}
}

// toDirectionsResponse converts ORS response to domain model.
func (c *Client) toDirectionsResponse(resp *orsResponse) *routing.DirectionsResponse {
	routes := make([]routing.Route, 0, len(resp.Routes))

	for i := range resp.Routes {
		orsRoute := &resp.Routes[i]
		route := routing.Route{
			GeometryPolyline: orsRoute.Geometry,
			DistanceMeters:   int(orsRoute.Summary.Distance),
			DurationSeconds:  int(orsRoute.Summary.Duration),
		}

		// Extract bounding box if available
		if len(orsRoute.BBox) >= 4 {
			route.BoundingBox = &routing.BoundingBox{
				MinLon: orsRoute.BBox[0],
				MinLat: orsRoute.BBox[1],
				MaxLon: orsRoute.BBox[2],
				MaxLat: orsRoute.BBox[3],
			}
		}

		// Extract instructions from segments
		for j := range orsRoute.Segments {
			segment := &orsRoute.Segments[j]
			for k := range segment.Steps {
				step := &segment.Steps[k]
				route.Instructions = append(route.Instructions, routing.Instruction{
					Text:           step.Instruction,
					DistanceMeters: int(step.Distance),
					DurationSecs:   int(step.Duration),
					Type:           step.Type,
				})
			}
		}

		// Generate summary from first and last instruction
		if len(route.Instructions) > 0 {
			route.Summary = generateRouteSummary(route.Instructions)
		}

		routes = append(routes, route)
	}

	return &routing.DirectionsResponse{
		Routes:    routes,
		Provider:  ProviderName,
		FetchedAt: time.Now(),
	}
}

// generateRouteSummary creates a human-readable route summary.
func generateRouteSummary(instructions []routing.Instruction) string {
	if len(instructions) == 0 {
		return ""
	}

	// Find main roads/paths in the route
	var mainRoads []string
	for _, inst := range instructions {
		// Type 1 and 6 are typically major turns or road changes
		if inst.DistanceMeters > 500 && len(mainRoads) < 3 {
			// Extract road name if present
			if inst.Text != "" {
				mainRoads = append(mainRoads, inst.Text)
			}
		}
	}

	if len(mainRoads) > 0 {
		return mainRoads[0]
	}

	return ""
}

// validateCoordinates checks if coordinates are within valid ranges.
func validateCoordinates(c routing.Coordinate) error {
	if c.Lat < -90 || c.Lat > 90 {
		return fmt.Errorf("latitude %f out of range [-90, 90]", c.Lat)
	}
	if c.Lon < -180 || c.Lon > 180 {
		return fmt.Errorf("longitude %f out of range [-180, 180]", c.Lon)
	}
	return nil
}
