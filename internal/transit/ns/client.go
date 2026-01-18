package ns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
	"github.com/breatheroute/breatheroute/internal/transit"
)

const (
	// ProviderName identifies this transit provider.
	ProviderName = "ns"

	// DefaultBaseURL is the NS API base URL.
	DefaultBaseURL = "https://gateway.apiportal.ns.nl/reisinformatie-api/api/v3"
)

// ClientConfig holds configuration for the NS client.
type ClientConfig struct {
	// APIKey is the NS API key (required).
	APIKey string

	// BaseURL is the API base URL (optional, defaults to NS API).
	BaseURL string

	// HTTPClient is the HTTP client to use (optional).
	// If nil, uses a resilient client with defaults.
	HTTPClient *resilience.Client

	// Logger for client operations.
	Logger zerolog.Logger
}

// Client is an NS API client for transit disruption data.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *resilience.Client
	logger     zerolog.Logger
}

// NewClient creates a new NS client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = resilience.NewClient(resilience.DefaultClientConfig("ns"))
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

// GetAllDisruptions fetches all current disruptions from NS API.
func (c *Client) GetAllDisruptions(ctx context.Context) ([]*transit.Disruption, error) {
	url := fmt.Sprintf("%s/disruptions", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var nsResp disruptionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&nsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	disruptions := make([]*transit.Disruption, 0, len(nsResp))
	for i := range nsResp {
		disruptions = append(disruptions, c.toDisruption(&nsResp[i]))
	}

	return disruptions, nil
}

// GetDisruptionsForRoute fetches disruptions affecting a specific route.
func (c *Client) GetDisruptionsForRoute(ctx context.Context, origin, destination string) (*transit.RouteDisruptions, error) {
	// First get all disruptions
	allDisruptions, err := c.GetAllDisruptions(ctx)
	if err != nil {
		return nil, err
	}

	// Filter disruptions that affect the route
	relevant := make([]*transit.Disruption, 0)
	for _, d := range allDisruptions {
		if d.AffectsStation(origin) || d.AffectsStation(destination) {
			relevant = append(relevant, d)
			continue
		}
		// Also check if any affected route mentions these stations
		for _, route := range d.AffectedRoutes {
			routeLower := strings.ToLower(route)
			if strings.Contains(routeLower, strings.ToLower(origin)) ||
				strings.Contains(routeLower, strings.ToLower(destination)) {
				relevant = append(relevant, d)
				break
			}
		}
	}

	result := &transit.RouteDisruptions{
		Origin:         origin,
		Destination:    destination,
		Disruptions:    relevant,
		HasDisruptions: len(relevant) > 0,
		FetchedAt:      time.Now(),
	}

	if len(relevant) > 0 {
		result.OverallImpact = transit.CalculateOverallImpact(relevant)
		result.AdvisoryMessage = c.generateAdvisory(relevant)
	}

	return result, nil
}

// GetStations fetches the list of stations.
func (c *Client) GetStations(ctx context.Context) ([]*transit.Station, error) {
	url := fmt.Sprintf("%s/stations", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var nsResp stationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&nsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	stations := make([]*transit.Station, 0, len(nsResp.Payload))
	for i := range nsResp.Payload {
		stations = append(stations, c.toStation(&nsResp.Payload[i]))
	}

	return stations, nil
}

// setHeaders sets common request headers.
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Ocp-Apim-Subscription-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
}

// toDisruption converts NS API disruption to domain model.
func (c *Client) toDisruption(d *nsDisruption) *transit.Disruption {
	disruption := &transit.Disruption{
		ID:          d.ID,
		Type:        mapDisruptionType(d.Type),
		Title:       d.Title,
		Description: d.Description,
		Impact:      mapImpact(d.Impact),
		IsPlanned:   d.IsPlanned,
		Cause:       d.Cause,
		Provider:    ProviderName,
		LastUpdated: time.Now(),
	}

	// Parse timestamps
	if d.Start != "" {
		if parsed, err := time.Parse(time.RFC3339, d.Start); err == nil {
			disruption.Start = parsed
		}
	}
	if d.End != "" {
		if parsed, err := time.Parse(time.RFC3339, d.End); err == nil {
			disruption.End = parsed
		}
	}

	// Extract affected routes
	for _, section := range d.Sections {
		routeName := fmt.Sprintf("%s - %s", section.Station.Name, section.Direction)
		disruption.AffectedRoutes = append(disruption.AffectedRoutes, routeName)
		disruption.AffectedStations = append(disruption.AffectedStations, section.Station.Code)
	}

	// Alternative transport
	if d.AlternativeTransport != "" {
		disruption.AlternativeTransport = d.AlternativeTransport
	}

	// Expected duration from timespans
	if len(d.Timespans) > 0 && d.Timespans[0].ExpectedDuration > 0 {
		disruption.ExpectedDuration = d.Timespans[0].ExpectedDuration
	}

	return disruption
}

// toStation converts NS API station to domain model.
func (c *Client) toStation(s *nsStation) *transit.Station {
	return &transit.Station{
		Code:    s.Code,
		Name:    s.Name,
		Lat:     s.Lat,
		Lon:     s.Lng,
		Country: s.Country,
	}
}

// mapDisruptionType maps NS disruption type to domain type.
func mapDisruptionType(nsType string) transit.DisruptionType {
	switch strings.ToUpper(nsType) {
	case "MAINTENANCE", "WERKZAAMHEDEN":
		return transit.DisruptionMaintenance
	case "DISTURBANCE", "STORING":
		return transit.DisruptionDisturbance
	case "CONSTRUCTION", "BOUW":
		return transit.DisruptionConstruction
	case "STRIKE", "STAKING":
		return transit.DisruptionStrike
	case "WEATHER", "WEER":
		return transit.DisruptionWeather
	default:
		return transit.DisruptionUnknown
	}
}

// mapImpact maps NS impact level to domain impact.
func mapImpact(nsImpact string) transit.Impact {
	switch strings.ToUpper(nsImpact) {
	case "MINOR", "GERING":
		return transit.ImpactMinor
	case "MODERATE", "MATIG":
		return transit.ImpactModerate
	case "MAJOR", "GROOT":
		return transit.ImpactMajor
	case "SEVERE", "ERNSTIG", "NO_TRAINS":
		return transit.ImpactSevere
	default:
		return transit.ImpactMinor
	}
}

// generateAdvisory creates a user-friendly advisory message.
func (c *Client) generateAdvisory(disruptions []*transit.Disruption) string {
	if len(disruptions) == 0 {
		return ""
	}

	impact := transit.CalculateOverallImpact(disruptions)

	switch impact {
	case transit.ImpactSevere:
		return "Severe disruptions on your route. No train service available. Please use alternative transport."
	case transit.ImpactMajor:
		return "Major disruptions expected. Significant delays or cancellations possible. Plan extra travel time."
	case transit.ImpactModerate:
		return "Moderate disruptions on your route. Some delays expected. Check departure times before traveling."
	case transit.ImpactMinor:
		return "Minor disruptions reported. Slight delays possible."
	}

	return ""
}

// NS API response structures.

type disruptionsResponse []nsDisruption

type nsDisruption struct {
	ID                   string       `json:"id"`
	Type                 string       `json:"type"`
	Title                string       `json:"title"`
	Description          string       `json:"description"`
	Impact               string       `json:"impact"`
	IsPlanned            bool         `json:"isPlanned"`
	Cause                string       `json:"cause"`
	Start                string       `json:"start"`
	End                  string       `json:"end"`
	AlternativeTransport string       `json:"alternativeTransportTimesText"`
	Sections             []nsSection  `json:"trajectories"`
	Timespans            []nsTimespan `json:"timespans"`
}

type nsSection struct {
	Station struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"station"`
	Direction string `json:"direction"`
}

type nsTimespan struct {
	Start            string `json:"start"`
	End              string `json:"end"`
	ExpectedDuration int    `json:"expectedDurationMinutes"`
}

type stationsResponse struct {
	Payload []nsStation `json:"payload"`
}

type nsStation struct {
	Code    string  `json:"code"`
	Name    string  `json:"namen"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Country string  `json:"land"`
}
