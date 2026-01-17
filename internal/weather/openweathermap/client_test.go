package openweathermap_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
	"github.com/breatheroute/breatheroute/internal/weather"
	"github.com/breatheroute/breatheroute/internal/weather/openweathermap"
)

func TestClient_GetCurrentWeather(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/weather", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("lat"), "52.370")
		assert.Contains(t, r.URL.Query().Get("lon"), "4.895")
		assert.Equal(t, "****", r.URL.Query().Get("appid"))
		assert.Equal(t, "metric", r.URL.Query().Get("units"))

		response := map[string]interface{}{
			"coord": map[string]float64{
				"lat": 52.370,
				"lon": 4.895,
			},
			"weather": []map[string]interface{}{
				{
					"id":          800,
					"main":        "Clear",
					"description": "clear sky",
				},
			},
			"main": map[string]float64{
				"temp":       18.5,
				"feels_like": 17.8,
				"temp_min":   17.0,
				"temp_max":   20.0,
				"pressure":   1015.0,
				"humidity":   72.0,
			},
			"visibility": 10000,
			"wind": map[string]float64{
				"speed": 4.5,
				"deg":   220.0,
				"gust":  7.2,
			},
			"clouds": map[string]float64{
				"all": 10.0,
			},
			"dt":   time.Now().Unix(),
			"name": "Amsterdam",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := openweathermap.NewClient(openweathermap.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	obs, err := client.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, obs)

	assert.Equal(t, 52.370, obs.Lat)
	assert.Equal(t, 4.895, obs.Lon)
	assert.Equal(t, 18.5, obs.Temperature)
	assert.Equal(t, 72.0, obs.Humidity)
	assert.Equal(t, 4.5, obs.WindSpeed)
	assert.Equal(t, 220.0, obs.WindDirection)
	assert.Equal(t, 7.2, obs.WindGust)
	assert.Equal(t, 1015.0, obs.Pressure)
	assert.Equal(t, 10.0, obs.CloudCover)
	assert.Equal(t, 10000.0, obs.Visibility)
	assert.Equal(t, weather.ConditionClear, obs.Condition)
	assert.Equal(t, "clear sky", obs.Description)
}

func TestClient_GetCurrentWeather_AllConditions(t *testing.T) {
	conditions := []struct {
		owmMain  string
		expected weather.Condition
	}{
		{"Clear", weather.ConditionClear},
		{"Clouds", weather.ConditionClouds},
		{"Rain", weather.ConditionRain},
		{"Drizzle", weather.ConditionDrizzle},
		{"Thunderstorm", weather.ConditionThunderstorm},
		{"Snow", weather.ConditionSnow},
		{"Mist", weather.ConditionMist},
		{"Fog", weather.ConditionFog},
		{"Haze", weather.ConditionHaze},
		{"Dust", weather.ConditionHaze},
		{"Unknown", weather.ConditionUnknown},
	}

	for _, tc := range conditions {
		t.Run(tc.owmMain, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"coord": map[string]float64{"lat": 52.0, "lon": 4.0},
					"weather": []map[string]interface{}{
						{"main": tc.owmMain, "description": "test"},
					},
					"main":       map[string]float64{"temp": 20.0, "humidity": 50.0, "pressure": 1013.0},
					"visibility": 10000,
					"wind":       map[string]float64{"speed": 5.0, "deg": 180.0},
					"clouds":     map[string]float64{"all": 50.0},
					"dt":         time.Now().Unix(),
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := openweathermap.NewClient(openweathermap.ClientConfig{
				APIKey:     "****",
				BaseURL:    server.URL,
				HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
			})

			obs, err := client.GetCurrentWeather(context.Background(), 52.0, 4.0)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, obs.Condition)
		})
	}
}

func TestClient_GetForecast(t *testing.T) {
	now := time.Now()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "onecall")
		assert.Contains(t, r.URL.Query().Get("lat"), "52.370")
		assert.Contains(t, r.URL.Query().Get("lon"), "4.895")
		assert.Contains(t, r.URL.Query().Get("exclude"), "minutely")

		response := map[string]interface{}{
			"lat": 52.370,
			"lon": 4.895,
			"hourly": []map[string]interface{}{
				{
					"dt":         now.Add(1 * time.Hour).Unix(),
					"temp":       19.0,
					"feels_like": 18.5,
					"pressure":   1015.0,
					"humidity":   70.0,
					"clouds":     20.0,
					"visibility": 10000,
					"wind_speed": 5.0,
					"wind_deg":   200.0,
					"wind_gust":  8.0,
					"pop":        0.1,
					"weather": []map[string]interface{}{
						{"main": "Clouds", "description": "few clouds"},
					},
				},
				{
					"dt":         now.Add(2 * time.Hour).Unix(),
					"temp":       20.0,
					"feels_like": 19.5,
					"pressure":   1014.0,
					"humidity":   65.0,
					"clouds":     30.0,
					"visibility": 10000,
					"wind_speed": 6.0,
					"wind_deg":   210.0,
					"wind_gust":  9.0,
					"pop":        0.2,
					"weather": []map[string]interface{}{
						{"main": "Clouds", "description": "scattered clouds"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := openweathermap.NewClient(openweathermap.ClientConfig{
		APIKey:     "****",
		OneCallURL: server.URL + "/onecall",
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	forecast, err := client.GetForecast(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, forecast)

	assert.Equal(t, 52.370, forecast.Lat)
	assert.Equal(t, 4.895, forecast.Lon)
	assert.Len(t, forecast.Hourly, 2)

	// Verify first hour
	h1 := forecast.Hourly[0]
	assert.Equal(t, 19.0, h1.Temperature)
	assert.Equal(t, 70.0, h1.Humidity)
	assert.Equal(t, 5.0, h1.WindSpeed)
	assert.Equal(t, 200.0, h1.WindDirection)
	assert.Equal(t, 8.0, h1.WindGust)
	assert.Equal(t, 0.1, h1.PrecipProb)
	assert.Equal(t, weather.ConditionClouds, h1.Condition)
}

func TestClient_GetCurrentWeather_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Use a client with minimal retries for faster tests
	cfg := resilience.DefaultClientConfig("test")
	cfg.MaxRetries = 0

	client := openweathermap.NewClient(openweathermap.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(cfg),
	})

	_, err := client.GetCurrentWeather(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_GetCurrentWeather_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := openweathermap.NewClient(openweathermap.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetCurrentWeather(ctx, 52.370, 4.895)
	require.Error(t, err)
}

func TestClient_Name(t *testing.T) {
	client := openweathermap.NewClient(openweathermap.ClientConfig{
		APIKey: "****",
	})

	assert.Equal(t, "openweathermap", client.Name())
}
