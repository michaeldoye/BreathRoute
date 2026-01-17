// Package luchtmeetnet provides a client for the Luchtmeetnet API.
package luchtmeetnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/breatheroute/breatheroute/internal/airquality"
	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

const (
	// DefaultBaseURL is the base URL for the Luchtmeetnet API.
	DefaultBaseURL = "https://api.luchtmeetnet.nl/open_api"

	// ProviderName identifies this provider.
	ProviderName = "luchtmeetnet"
)

// ClientConfig holds configuration for the Luchtmeetnet client.
type ClientConfig struct {
	// BaseURL is the API base URL (defaults to DefaultBaseURL).
	BaseURL string

	// HTTPClient is the HTTP client to use (must implement HTTPDoer).
	// If nil, a default resilient client will be created.
	HTTPClient HTTPDoer

	// Timeout for individual API requests (default: 10s).
	Timeout time.Duration
}

// HTTPDoer abstracts HTTP request execution.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a Luchtmeetnet API client.
type Client struct {
	baseURL    string
	httpClient HTTPDoer
}

// NewClient creates a new Luchtmeetnet client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = 10 * time.Second
		}
		httpClient = resilience.NewClient(resilience.ClientConfig{
			Name:            "luchtmeetnet",
			Timeout:         timeout,
			MaxRetries:      3,
			InitialInterval: 200 * time.Millisecond,
			MaxInterval:     5 * time.Second,
		})
	}

	return &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: httpClient,
	}
}

// API response types (from Luchtmeetnet API).

type stationsResponse struct {
	Pagination paginationInfo `json:"pagination"`
	Data       []stationData  `json:"data"`
}

type stationData struct {
	Number      string            `json:"number"`
	Location    string            `json:"location"`
	Coordinates locationData      `json:"geometry.coordinates"`
	Components  []string          `json:"components"`
}

type locationData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type paginationInfo struct {
	CurrentPage   int `json:"current_page"`
	LastPage      int `json:"last_page"`
	PerPage       int `json:"per_page"`
	TotalElements int `json:"total_elements"`
}

type measurementsResponse struct {
	Pagination paginationInfo    `json:"pagination"`
	Data       []measurementData `json:"data"`
}

type measurementData struct {
	StationNumber     string  `json:"station_number"`
	Formula           string  `json:"formula"`
	Value             float64 `json:"value"`
	TimestampMeasured string  `json:"timestamp_measured"`
}

// FetchStations retrieves all monitoring stations.
func (c *Client) FetchStations(ctx context.Context) ([]*airquality.Station, error) {
	var allStations []*airquality.Station
	page := 1

	for {
		stations, lastPage, err := c.fetchStationsPage(ctx, page)
		if err != nil {
			return nil, err
		}
		allStations = append(allStations, stations...)

		if page >= lastPage {
			break
		}
		page++
	}

	return allStations, nil
}

// fetchStationsPage fetches a single page of stations.
func (c *Client) fetchStationsPage(ctx context.Context, page int) ([]*airquality.Station, int, error) {
	url := fmt.Sprintf("%s/stations?page=%d", c.baseURL, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch stations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status %d from stations endpoint", resp.StatusCode)
	}

	var result stationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode stations response: %w", err)
	}

	stations := make([]*airquality.Station, 0, len(result.Data))
	for _, s := range result.Data {
		station := c.toStation(&s)
		stations = append(stations, station)
	}

	return stations, result.Pagination.LastPage, nil
}

// FetchLatestMeasurements retrieves the latest measurements for all stations.
func (c *Client) FetchLatestMeasurements(ctx context.Context) ([]*airquality.Measurement, error) {
	var allMeasurements []*airquality.Measurement
	page := 1

	for {
		measurements, lastPage, err := c.fetchMeasurementsPage(ctx, page)
		if err != nil {
			return nil, err
		}
		allMeasurements = append(allMeasurements, measurements...)

		if page >= lastPage {
			break
		}
		page++
	}

	return allMeasurements, nil
}

// fetchMeasurementsPage fetches a single page of measurements.
func (c *Client) fetchMeasurementsPage(ctx context.Context, page int) ([]*airquality.Measurement, int, error) {
	url := fmt.Sprintf("%s/measurements?page=%d", c.baseURL, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch measurements: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("unexpected status %d from measurements endpoint", resp.StatusCode)
	}

	var result measurementsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode measurements response: %w", err)
	}

	measurements := make([]*airquality.Measurement, 0, len(result.Data))
	for _, m := range result.Data {
		measurement := c.toMeasurement(&m)
		if measurement != nil {
			measurements = append(measurements, measurement)
		}
	}

	return measurements, result.Pagination.LastPage, nil
}

// FetchSnapshot fetches a complete AQ snapshot (stations + measurements).
func (c *Client) FetchSnapshot(ctx context.Context) (*airquality.AQSnapshot, error) {
	stations, err := c.FetchStations(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch stations: %w", err)
	}

	measurements, err := c.FetchLatestMeasurements(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch measurements: %w", err)
	}

	snapshot := airquality.NewAQSnapshot(ProviderName)
	snapshot.FetchedAt = time.Now()

	for _, s := range stations {
		snapshot.Stations[s.ID] = s
	}

	for _, m := range measurements {
		snapshot.SetMeasurement(m)
	}

	return snapshot, nil
}

// toStation converts API station data to domain Station.
func (c *Client) toStation(s *stationData) *airquality.Station {
	pollutants := make([]airquality.Pollutant, 0, len(s.Components))
	for _, comp := range s.Components {
		if p := toPollutant(comp); p != "" {
			pollutants = append(pollutants, p)
		}
	}

	return &airquality.Station{
		ID:         s.Number,
		Name:       s.Location,
		Lat:        s.Coordinates.Latitude,
		Lon:        s.Coordinates.Longitude,
		Pollutants: pollutants,
		UpdatedAt:  time.Now(),
	}
}

// toMeasurement converts API measurement data to domain Measurement.
func (c *Client) toMeasurement(m *measurementData) *airquality.Measurement {
	pollutant := toPollutant(m.Formula)
	if pollutant == "" {
		return nil // Skip unsupported pollutants
	}

	measuredAt, _ := time.Parse(time.RFC3339, m.TimestampMeasured)

	return &airquality.Measurement{
		StationID:  m.StationNumber,
		Pollutant:  pollutant,
		Value:      m.Value,
		Unit:       "µg/m³",
		MeasuredAt: measuredAt,
	}
}

// toPollutant converts a Luchtmeetnet formula string to our Pollutant type.
func toPollutant(formula string) airquality.Pollutant {
	switch strings.ToUpper(formula) {
	case "NO2":
		return airquality.PollutantNO2
	case "PM25":
		return airquality.PollutantPM25
	case "PM10":
		return airquality.PollutantPM10
	case "O3":
		return airquality.PollutantO3
	default:
		return ""
	}
}
