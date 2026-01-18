package ambee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/pollen"
	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

const (
	// ProviderName identifies this pollen provider.
	ProviderName = "ambee"

	// DefaultBaseURL is the Ambee API base URL.
	DefaultBaseURL = "https://api.ambeedata.com"
)

// ClientConfig holds configuration for the Ambee client.
type ClientConfig struct {
	// APIKey is the Ambee API key (required).
	APIKey string

	// BaseURL is the API base URL (optional, defaults to Ambee API).
	BaseURL string

	// HTTPClient is the HTTP client to use (optional).
	// If nil, uses a resilient client with defaults.
	HTTPClient *resilience.Client

	// Logger for client operations.
	Logger zerolog.Logger
}

// Client is an Ambee API client for pollen data.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *resilience.Client
	logger     zerolog.Logger
}

// NewClient creates a new Ambee client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = resilience.NewClient(resilience.DefaultClientConfig("ambee"))
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

// GetRegionalPollen fetches current pollen data for a location.
func (c *Client) GetRegionalPollen(ctx context.Context, lat, lon float64) (*pollen.RegionalPollen, error) {
	url := fmt.Sprintf("%s/latest/pollen/by-lat-lng?lat=%.6f&lng=%.6f",
		c.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var ambeeResp pollenResponse
	if err := json.NewDecoder(resp.Body).Decode(&ambeeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(ambeeResp.Data) == 0 {
		return nil, pollen.ErrNoDataForRegion
	}

	return c.toRegionalPollen(&ambeeResp.Data[0], lat, lon), nil
}

// GetForecast fetches pollen forecast for a location.
func (c *Client) GetForecast(ctx context.Context, lat, lon float64) (*pollen.Forecast, error) {
	url := fmt.Sprintf("%s/forecast/pollen/by-lat-lng?lat=%.6f&lng=%.6f",
		c.baseURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var ambeeResp forecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&ambeeResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return c.toForecast(&ambeeResp), nil
}

// toRegionalPollen converts Ambee response to domain model.
func (c *Client) toRegionalPollen(data *pollenData, lat, lon float64) *pollen.RegionalPollen {
	readings := make(map[pollen.Type]*pollen.Reading)

	// Grass pollen
	if data.Count.GrassIndex > 0 || data.Risk.GrassIndex != "" {
		readings[pollen.PollenGrass] = &pollen.Reading{
			Type:    pollen.PollenGrass,
			Index:   float64(data.Count.GrassIndex),
			Risk:    mapRiskLevel(data.Risk.GrassIndex),
			Species: data.Species.Grass,
		}
	}

	// Tree pollen
	if data.Count.TreeIndex > 0 || data.Risk.TreeIndex != "" {
		readings[pollen.PollenTree] = &pollen.Reading{
			Type:    pollen.PollenTree,
			Index:   float64(data.Count.TreeIndex),
			Risk:    mapRiskLevel(data.Risk.TreeIndex),
			Species: data.Species.Tree,
		}
	}

	// Weed pollen
	if data.Count.WeedIndex > 0 || data.Risk.WeedIndex != "" {
		readings[pollen.PollenWeed] = &pollen.Reading{
			Type:    pollen.PollenWeed,
			Index:   float64(data.Count.WeedIndex),
			Risk:    mapRiskLevel(data.Risk.WeedIndex),
			Species: data.Species.Weed,
		}
	}

	overallRisk := calculateOverallRisk(readings)
	overallIndex := calculateOverallIndex(readings)

	validFor := time.Now()
	if data.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, data.UpdatedAt); err == nil {
			validFor = parsed
		}
	}

	return &pollen.RegionalPollen{
		Region:       "NL", // Region is determined by coordinates
		RegionName:   "Netherlands",
		Lat:          lat,
		Lon:          lon,
		Readings:     readings,
		OverallRisk:  overallRisk,
		OverallIndex: overallIndex,
		ValidFor:     validFor,
		FetchedAt:    time.Now(),
		Provider:     ProviderName,
	}
}

// toForecast converts Ambee forecast response to domain model.
func (c *Client) toForecast(resp *forecastResponse) *pollen.Forecast {
	forecast := &pollen.Forecast{
		Region:    "NL",
		Daily:     make([]pollen.DailyForecast, 0, len(resp.Data)),
		FetchedAt: time.Now(),
	}

	for i := range resp.Data {
		day := &resp.Data[i]
		readings := make(map[pollen.Type]*pollen.Reading)

		if day.Count.GrassIndex > 0 || day.Risk.GrassIndex != "" {
			readings[pollen.PollenGrass] = &pollen.Reading{
				Type:    pollen.PollenGrass,
				Index:   float64(day.Count.GrassIndex),
				Risk:    mapRiskLevel(day.Risk.GrassIndex),
				Species: day.Species.Grass,
			}
		}

		if day.Count.TreeIndex > 0 || day.Risk.TreeIndex != "" {
			readings[pollen.PollenTree] = &pollen.Reading{
				Type:    pollen.PollenTree,
				Index:   float64(day.Count.TreeIndex),
				Risk:    mapRiskLevel(day.Risk.TreeIndex),
				Species: day.Species.Tree,
			}
		}

		if day.Count.WeedIndex > 0 || day.Risk.WeedIndex != "" {
			readings[pollen.PollenWeed] = &pollen.Reading{
				Type:    pollen.PollenWeed,
				Index:   float64(day.Count.WeedIndex),
				Risk:    mapRiskLevel(day.Risk.WeedIndex),
				Species: day.Species.Weed,
			}
		}

		date := time.Now()
		if day.Time != "" {
			if parsed, err := time.Parse("2006-01-02", day.Time); err == nil {
				date = parsed
			}
		}

		forecast.Daily = append(forecast.Daily, pollen.DailyForecast{
			Date:         date,
			Readings:     readings,
			OverallRisk:  calculateOverallRisk(readings),
			OverallIndex: calculateOverallIndex(readings),
		})
	}

	return forecast
}

// mapRiskLevel maps Ambee risk string to domain risk level.
func mapRiskLevel(risk string) pollen.RiskLevel {
	switch risk {
	case "Low":
		return pollen.RiskLow
	case "Moderate":
		return pollen.RiskModerate
	case "High":
		return pollen.RiskHigh
	case "Very High":
		return pollen.RiskVeryHigh
	default:
		return pollen.RiskNone
	}
}

// calculateOverallRisk determines the highest risk level from readings.
func calculateOverallRisk(readings map[pollen.Type]*pollen.Reading) pollen.RiskLevel {
	highest := pollen.RiskNone
	riskOrder := map[pollen.RiskLevel]int{
		pollen.RiskNone:     0,
		pollen.RiskLow:      1,
		pollen.RiskModerate: 2,
		pollen.RiskHigh:     3,
		pollen.RiskVeryHigh: 4,
	}

	for _, reading := range readings {
		if reading != nil && riskOrder[reading.Risk] > riskOrder[highest] {
			highest = reading.Risk
		}
	}

	return highest
}

// calculateOverallIndex calculates the average index from readings.
func calculateOverallIndex(readings map[pollen.Type]*pollen.Reading) float64 {
	if len(readings) == 0 {
		return 0
	}

	var sum float64
	var count int
	for _, reading := range readings {
		if reading != nil {
			sum += reading.Index
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// Ambee API response structures.

type pollenResponse struct {
	Message string       `json:"message"`
	Data    []pollenData `json:"data"`
}

type pollenData struct {
	Count struct {
		GrassIndex int `json:"grass_index"`
		TreeIndex  int `json:"tree_index"`
		WeedIndex  int `json:"weed_index"`
	} `json:"Count"`
	Risk struct {
		GrassIndex string `json:"grass_index"`
		TreeIndex  string `json:"tree_index"`
		WeedIndex  string `json:"weed_index"`
	} `json:"Risk"`
	Species struct {
		Grass []string `json:"Grass"`
		Tree  []string `json:"Tree"`
		Weed  []string `json:"Weed"`
	} `json:"Species"`
	UpdatedAt string `json:"updatedAt"`
	Time      string `json:"time"` // Used in forecast responses
}

type forecastResponse struct {
	Message string       `json:"message"`
	Data    []pollenData `json:"data"`
}
