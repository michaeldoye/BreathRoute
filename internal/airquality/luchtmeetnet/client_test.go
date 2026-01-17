package luchtmeetnet_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/airquality"
	"github.com/breatheroute/breatheroute/internal/airquality/luchtmeetnet"
)

func TestClient_FetchStations(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/stations", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("page"))

		response := map[string]interface{}{
			"pagination": map[string]int{
				"current_page":   1,
				"last_page":      1,
				"per_page":       100,
				"total_elements": 2,
			},
			"data": []map[string]interface{}{
				{
					"number":   "NL10938",
					"location": "Amsterdam-Einsteinweg",
					"geometry.coordinates": map[string]float64{
						"latitude":  52.370216,
						"longitude": 4.895168,
					},
					"components": []string{"NO2", "PM10", "PM25"},
				},
				{
					"number":   "NL10636",
					"location": "Rotterdam-Schiedamsevest",
					"geometry.coordinates": map[string]float64{
						"latitude":  51.9225,
						"longitude": 4.47917,
					},
					"components": []string{"NO2", "O3"},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	stations, err := client.FetchStations(context.Background())
	require.NoError(t, err)
	require.Len(t, stations, 2)

	// Verify first station
	assert.Equal(t, "NL10938", stations[0].ID)
	assert.Equal(t, "Amsterdam-Einsteinweg", stations[0].Name)
	assert.Equal(t, 52.370216, stations[0].Lat)
	assert.Equal(t, 4.895168, stations[0].Lon)
	assert.Contains(t, stations[0].Pollutants, airquality.PollutantNO2)
	assert.Contains(t, stations[0].Pollutants, airquality.PollutantPM25)
}

func TestClient_FetchStations_Pagination(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		page := r.URL.Query().Get("page")

		var response map[string]interface{}
		if page == "1" {
			response = map[string]interface{}{
				"pagination": map[string]int{
					"current_page":   1,
					"last_page":      2,
					"per_page":       1,
					"total_elements": 2,
				},
				"data": []map[string]interface{}{
					{
						"number":   "NL10001",
						"location": "Station 1",
						"geometry.coordinates": map[string]float64{
							"latitude":  52.0,
							"longitude": 4.0,
						},
						"components": []string{"NO2"},
					},
				},
			}
		} else {
			response = map[string]interface{}{
				"pagination": map[string]int{
					"current_page":   2,
					"last_page":      2,
					"per_page":       1,
					"total_elements": 2,
				},
				"data": []map[string]interface{}{
					{
						"number":   "NL10002",
						"location": "Station 2",
						"geometry.coordinates": map[string]float64{
							"latitude":  51.0,
							"longitude": 5.0,
						},
						"components": []string{"PM25"},
					},
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	stations, err := client.FetchStations(context.Background())
	require.NoError(t, err)
	assert.Len(t, stations, 2)
	assert.Equal(t, 2, pageCount) // Both pages were fetched
}

func TestClient_FetchLatestMeasurements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/measurements", r.URL.Path)

		response := map[string]interface{}{
			"pagination": map[string]int{
				"current_page":   1,
				"last_page":      1,
				"per_page":       100,
				"total_elements": 3,
			},
			"data": []map[string]interface{}{
				{
					"station_number":     "NL10938",
					"formula":            "NO2",
					"value":              32.5,
					"timestamp_measured": "2024-01-15T14:00:00+01:00",
				},
				{
					"station_number":     "NL10938",
					"formula":            "PM25",
					"value":              12.3,
					"timestamp_measured": "2024-01-15T14:00:00+01:00",
				},
				{
					"station_number":     "NL10636",
					"formula":            "O3",
					"value":              45.8,
					"timestamp_measured": "2024-01-15T14:00:00+01:00",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	measurements, err := client.FetchLatestMeasurements(context.Background())
	require.NoError(t, err)
	require.Len(t, measurements, 3)

	// Verify first measurement
	assert.Equal(t, "NL10938", measurements[0].StationID)
	assert.Equal(t, airquality.PollutantNO2, measurements[0].Pollutant)
	assert.Equal(t, 32.5, measurements[0].Value)
	assert.Equal(t, "µg/m³", measurements[0].Unit)
}

func TestClient_FetchLatestMeasurements_SkipsUnknownPollutants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"pagination": map[string]int{
				"current_page":   1,
				"last_page":      1,
				"per_page":       100,
				"total_elements": 2,
			},
			"data": []map[string]interface{}{
				{
					"station_number":     "NL10938",
					"formula":            "NO2",
					"value":              32.5,
					"timestamp_measured": "2024-01-15T14:00:00+01:00",
				},
				{
					"station_number":     "NL10938",
					"formula":            "SO2", // Unknown pollutant
					"value":              5.0,
					"timestamp_measured": "2024-01-15T14:00:00+01:00",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	measurements, err := client.FetchLatestMeasurements(context.Background())
	require.NoError(t, err)
	assert.Len(t, measurements, 1) // Only NO2, SO2 is skipped
}

func TestClient_FetchSnapshot(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var response map[string]interface{}

		if r.URL.Path == "/stations" {
			response = map[string]interface{}{
				"pagination": map[string]int{
					"current_page":   1,
					"last_page":      1,
					"per_page":       100,
					"total_elements": 1,
				},
				"data": []map[string]interface{}{
					{
						"number":   "NL10938",
						"location": "Amsterdam-Einsteinweg",
						"geometry.coordinates": map[string]float64{
							"latitude":  52.370216,
							"longitude": 4.895168,
						},
						"components": []string{"NO2"},
					},
				},
			}
		} else if r.URL.Path == "/measurements" {
			response = map[string]interface{}{
				"pagination": map[string]int{
					"current_page":   1,
					"last_page":      1,
					"per_page":       100,
					"total_elements": 1,
				},
				"data": []map[string]interface{}{
					{
						"station_number":     "NL10938",
						"formula":            "NO2",
						"value":              32.5,
						"timestamp_measured": "2024-01-15T14:00:00+01:00",
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	snapshot, err := client.FetchSnapshot(context.Background())
	require.NoError(t, err)

	// Verify snapshot contains both stations and measurements
	assert.Len(t, snapshot.Stations, 1)
	assert.Equal(t, "NL10938", snapshot.Stations["NL10938"].ID)

	m := snapshot.GetMeasurement("NL10938", airquality.PollutantNO2)
	require.NotNil(t, m)
	assert.Equal(t, 32.5, m.Value)

	assert.Equal(t, luchtmeetnet.ProviderName, snapshot.Provider)
	assert.Equal(t, 2, callCount) // Stations + Measurements
}

func TestClient_FetchStations_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	_, err := client.FetchStations(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_FetchStations_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context to be done
		<-r.Context().Done()
	}))
	defer server.Close()

	client := luchtmeetnet.NewClient(luchtmeetnet.ClientConfig{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.FetchStations(ctx)
	require.Error(t, err)
}
