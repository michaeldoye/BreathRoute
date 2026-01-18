package ns_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/provider/resilience"
	"github.com/breatheroute/breatheroute/internal/transit"
	"github.com/breatheroute/breatheroute/internal/transit/ns"
)

func TestClient_Name(t *testing.T) {
	client := ns.NewClient(ns.ClientConfig{
		APIKey: "****",
		Logger: zerolog.Nop(),
	})

	assert.Equal(t, "ns", client.Name())
}

func TestClient_GetAllDisruptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/disruptions", r.URL.Path)
		assert.Equal(t, "****", r.Header.Get("Ocp-Apim-Subscription-Key"))

		resp := []map[string]interface{}{
			{
				"id":          "disruption-1",
				"type":        "MAINTENANCE",
				"title":       "Track maintenance Amsterdam-Utrecht",
				"description": "Planned maintenance work on tracks",
				"impact":      "MODERATE",
				"isPlanned":   true,
				"cause":       "Maintenance",
				"start":       "2024-01-15T08:00:00Z",
				"end":         "2024-01-15T18:00:00Z",
				"trajectories": []map[string]interface{}{
					{
						"station": map[string]string{
							"code": "ASD",
							"name": "Amsterdam Centraal",
						},
						"direction": "Utrecht Centraal",
					},
				},
				"timespans": []map[string]interface{}{
					{
						"expectedDurationMinutes": 30,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	disruptions, err := client.GetAllDisruptions(context.Background())
	require.NoError(t, err)
	require.Len(t, disruptions, 1)

	d := disruptions[0]
	assert.Equal(t, "disruption-1", d.ID)
	assert.Equal(t, transit.DisruptionMaintenance, d.Type)
	assert.Equal(t, "Track maintenance Amsterdam-Utrecht", d.Title)
	assert.Equal(t, transit.ImpactModerate, d.Impact)
	assert.True(t, d.IsPlanned)
	assert.Equal(t, 30, d.ExpectedDuration)
	assert.Contains(t, d.AffectedStations, "ASD")
}

func TestClient_GetAllDisruptions_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	disruptions, err := client.GetAllDisruptions(context.Background())
	require.NoError(t, err)
	assert.Empty(t, disruptions)
}

func TestClient_GetAllDisruptions_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	_, err := client.GetAllDisruptions(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestClient_GetDisruptionsForRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := []map[string]interface{}{
			{
				"id":          "disruption-1",
				"type":        "DISTURBANCE",
				"title":       "Signal failure",
				"description": "Signal failure between Amsterdam and Utrecht",
				"impact":      "MAJOR",
				"isPlanned":   false,
				"trajectories": []map[string]interface{}{
					{
						"station": map[string]string{
							"code": "ASD",
							"name": "Amsterdam Centraal",
						},
						"direction": "Utrecht Centraal",
					},
				},
			},
			{
				"id":          "disruption-2",
				"type":        "MAINTENANCE",
				"title":       "Track work Rotterdam",
				"description": "Maintenance in Rotterdam area",
				"impact":      "MINOR",
				"isPlanned":   true,
				"trajectories": []map[string]interface{}{
					{
						"station": map[string]string{
							"code": "RTD",
							"name": "Rotterdam Centraal",
						},
						"direction": "Den Haag",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	result, err := client.GetDisruptionsForRoute(context.Background(), "ASD", "UT")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "ASD", result.Origin)
	assert.Equal(t, "UT", result.Destination)
	assert.True(t, result.HasDisruptions)
	assert.Len(t, result.Disruptions, 1) // Only disruption-1 affects ASD
	assert.Equal(t, transit.ImpactMajor, result.OverallImpact)
	assert.NotEmpty(t, result.AdvisoryMessage)
}

func TestClient_GetDisruptionsForRoute_NoDisruptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	result, err := client.GetDisruptionsForRoute(context.Background(), "ASD", "UT")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.HasDisruptions)
	assert.Empty(t, result.Disruptions)
	assert.Empty(t, result.AdvisoryMessage)
}

func TestClient_GetStations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/stations", r.URL.Path)

		resp := map[string]interface{}{
			"payload": []map[string]interface{}{
				{
					"code":  "ASD",
					"namen": "Amsterdam Centraal",
					"lat":   52.378901,
					"lng":   4.900272,
					"land":  "NL",
				},
				{
					"code":  "UT",
					"namen": "Utrecht Centraal",
					"lat":   52.089444,
					"lng":   5.110278,
					"land":  "NL",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	stations, err := client.GetStations(context.Background())
	require.NoError(t, err)
	require.Len(t, stations, 2)

	assert.Equal(t, "ASD", stations[0].Code)
	assert.Equal(t, "Amsterdam Centraal", stations[0].Name)
	assert.Equal(t, "NL", stations[0].Country)
	assert.InDelta(t, 52.378901, stations[0].Lat, 0.0001)
}

func TestClient_GetStations_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := ns.NewClient(ns.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
		Logger:     zerolog.Nop(),
	})

	_, err := client.GetStations(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 401")
}

func TestMapDisruptionType(t *testing.T) {
	// Test via GetAllDisruptions since mapDisruptionType is not exported
	tests := []struct {
		nsType   string
		expected transit.DisruptionType
	}{
		{"MAINTENANCE", transit.DisruptionMaintenance},
		{"WERKZAAMHEDEN", transit.DisruptionMaintenance},
		{"DISTURBANCE", transit.DisruptionDisturbance},
		{"STORING", transit.DisruptionDisturbance},
		{"STRIKE", transit.DisruptionStrike},
		{"WEATHER", transit.DisruptionWeather},
		{"UNKNOWN_TYPE", transit.DisruptionUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.nsType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				resp := []map[string]interface{}{
					{
						"id":    "test-1",
						"type":  tt.nsType,
						"title": "Test disruption",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := ns.NewClient(ns.ClientConfig{
				APIKey:     "****",
				BaseURL:    server.URL,
				HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
				Logger:     zerolog.Nop(),
			})

			disruptions, err := client.GetAllDisruptions(context.Background())
			require.NoError(t, err)
			require.Len(t, disruptions, 1)
			assert.Equal(t, tt.expected, disruptions[0].Type)
		})
	}
}

func TestMapImpact(t *testing.T) {
	tests := []struct {
		nsImpact string
		expected transit.Impact
	}{
		{"MINOR", transit.ImpactMinor},
		{"GERING", transit.ImpactMinor},
		{"MODERATE", transit.ImpactModerate},
		{"MATIG", transit.ImpactModerate},
		{"MAJOR", transit.ImpactMajor},
		{"GROOT", transit.ImpactMajor},
		{"SEVERE", transit.ImpactSevere},
		{"NO_TRAINS", transit.ImpactSevere},
		{"UNKNOWN", transit.ImpactMinor}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.nsImpact, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				resp := []map[string]interface{}{
					{
						"id":     "test-1",
						"type":   "DISTURBANCE",
						"title":  "Test disruption",
						"impact": tt.nsImpact,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
			}))
			defer server.Close()

			client := ns.NewClient(ns.ClientConfig{
				APIKey:     "****",
				BaseURL:    server.URL,
				HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("ns-test")),
				Logger:     zerolog.Nop(),
			})

			disruptions, err := client.GetAllDisruptions(context.Background())
			require.NoError(t, err)
			require.Len(t, disruptions, 1)
			assert.Equal(t, tt.expected, disruptions[0].Impact)
		})
	}
}
