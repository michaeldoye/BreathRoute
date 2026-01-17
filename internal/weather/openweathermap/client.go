package openweathermap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
	"github.com/breatheroute/breatheroute/internal/weather"
)

const (
	// ProviderName identifies this weather provider.
	ProviderName = "openweathermap"

	// DefaultBaseURL is the OpenWeatherMap API base URL.
	DefaultBaseURL = "https://api.openweathermap.org/data/2.5"

	// DefaultOneCallURL is the OpenWeatherMap OneCall API 3.0 base URL.
	DefaultOneCallURL = "https://api.openweathermap.org/data/3.0/onecall"
)

// ClientConfig holds configuration for the OpenWeatherMap client.
type ClientConfig struct {
	// APIKey is the OpenWeatherMap API key (required).
	APIKey string

	// BaseURL is the API base URL (optional, defaults to OpenWeatherMap API).
	BaseURL string

	// OneCallURL is the OneCall API URL (optional, defaults to OneCall 3.0).
	OneCallURL string

	// HTTPClient is the HTTP client to use (optional).
	// If nil, uses a resilient client with defaults.
	HTTPClient *resilience.Client

	// Logger for client operations.
	Logger zerolog.Logger
}

// Client is an OpenWeatherMap API client.
type Client struct {
	apiKey     string
	baseURL    string
	oneCallURL string
	httpClient *resilience.Client
	logger     zerolog.Logger
}

// NewClient creates a new OpenWeatherMap client.
func NewClient(cfg ClientConfig) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	oneCallURL := cfg.OneCallURL
	if oneCallURL == "" {
		oneCallURL = DefaultOneCallURL
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = resilience.NewClient(resilience.DefaultClientConfig("openweathermap"))
	}

	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    baseURL,
		oneCallURL: oneCallURL,
		httpClient: httpClient,
		logger:     cfg.Logger,
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return ProviderName
}

// GetCurrentWeather fetches current weather for a location.
func (c *Client) GetCurrentWeather(ctx context.Context, lat, lon float64) (*weather.Observation, error) {
	url := fmt.Sprintf("%s/weather?lat=%.6f&lon=%.6f&appid=%s&units=metric",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var owmResp currentWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&owmResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return c.toObservation(&owmResp), nil
}

// GetForecast fetches hourly forecast for a location.
func (c *Client) GetForecast(ctx context.Context, lat, lon float64) (*weather.Forecast, error) {
	url := fmt.Sprintf("%s?lat=%.6f&lon=%.6f&appid=%s&units=metric&exclude=minutely,daily,alerts",
		c.oneCallURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var owmResp oneCallResponse
	if err := json.NewDecoder(resp.Body).Decode(&owmResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return c.toForecast(&owmResp), nil
}

// toObservation converts OpenWeatherMap response to domain model.
func (c *Client) toObservation(resp *currentWeatherResponse) *weather.Observation {
	obs := &weather.Observation{
		Lat:           resp.Coord.Lat,
		Lon:           resp.Coord.Lon,
		Temperature:   resp.Main.Temp,
		Humidity:      resp.Main.Humidity,
		WindSpeed:     resp.Wind.Speed,
		WindDirection: resp.Wind.Deg,
		WindGust:      resp.Wind.Gust,
		Pressure:      resp.Main.Pressure,
		CloudCover:    resp.Clouds.All,
		Visibility:    float64(resp.Visibility),
		ObservedAt:    time.Unix(resp.Dt, 0),
		FetchedAt:     time.Now(),
	}

	if len(resp.Weather) > 0 {
		obs.Condition = mapCondition(resp.Weather[0].Main)
		obs.Description = resp.Weather[0].Description
	} else {
		obs.Condition = weather.ConditionUnknown
	}

	return obs
}

// toForecast converts OpenWeatherMap OneCall response to domain model.
func (c *Client) toForecast(resp *oneCallResponse) *weather.Forecast {
	forecast := &weather.Forecast{
		Lat:       resp.Lat,
		Lon:       resp.Lon,
		Hourly:    make([]weather.HourlyForecast, 0, len(resp.Hourly)),
		FetchedAt: time.Now(),
	}

	for _, h := range resp.Hourly {
		hourly := weather.HourlyForecast{
			Time:          time.Unix(h.Dt, 0),
			Temperature:   h.Temp,
			Humidity:      h.Humidity,
			WindSpeed:     h.WindSpeed,
			WindDirection: h.WindDeg,
			WindGust:      h.WindGust,
			CloudCover:    h.Clouds,
			Visibility:    float64(h.Visibility),
			PrecipProb:    h.Pop,
		}

		if len(h.Weather) > 0 {
			hourly.Condition = mapCondition(h.Weather[0].Main)
			hourly.Description = h.Weather[0].Description
		} else {
			hourly.Condition = weather.ConditionUnknown
		}

		forecast.Hourly = append(forecast.Hourly, hourly)
	}

	return forecast
}

// mapCondition maps OpenWeatherMap condition to domain condition.
func mapCondition(owmCondition string) weather.Condition {
	switch owmCondition {
	case "Clear":
		return weather.ConditionClear
	case "Clouds":
		return weather.ConditionClouds
	case "Rain":
		return weather.ConditionRain
	case "Drizzle":
		return weather.ConditionDrizzle
	case "Thunderstorm":
		return weather.ConditionThunderstorm
	case "Snow":
		return weather.ConditionSnow
	case "Mist":
		return weather.ConditionMist
	case "Fog":
		return weather.ConditionFog
	case "Haze", "Dust", "Sand", "Ash", "Squall", "Tornado":
		return weather.ConditionHaze
	default:
		return weather.ConditionUnknown
	}
}

// OpenWeatherMap API response structures.

type currentWeatherResponse struct {
	Coord struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coord"`
	Weather []struct {
		ID          int    `json:"id"`
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		TempMin   float64 `json:"temp_min"`
		TempMax   float64 `json:"temp_max"`
		Pressure  float64 `json:"pressure"`
		Humidity  float64 `json:"humidity"`
	} `json:"main"`
	Visibility int `json:"visibility"`
	Wind       struct {
		Speed float64 `json:"speed"`
		Deg   float64 `json:"deg"`
		Gust  float64 `json:"gust"`
	} `json:"wind"`
	Clouds struct {
		All float64 `json:"all"`
	} `json:"clouds"`
	Dt   int64  `json:"dt"`
	Name string `json:"name"`
}

type oneCallResponse struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Hourly []struct {
		Dt         int64   `json:"dt"`
		Temp       float64 `json:"temp"`
		FeelsLike  float64 `json:"feels_like"`
		Pressure   float64 `json:"pressure"`
		Humidity   float64 `json:"humidity"`
		Clouds     float64 `json:"clouds"`
		Visibility int     `json:"visibility"`
		WindSpeed  float64 `json:"wind_speed"`
		WindDeg    float64 `json:"wind_deg"`
		WindGust   float64 `json:"wind_gust"`
		Pop        float64 `json:"pop"` // Probability of precipitation
		Weather    []struct {
			ID          int    `json:"id"`
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
	} `json:"hourly"`
}
