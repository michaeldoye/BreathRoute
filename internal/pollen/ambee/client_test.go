package ambee_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/breatheroute/breatheroute/internal/pollen"
	"github.com/breatheroute/breatheroute/internal/pollen/ambee"
	"github.com/breatheroute/breatheroute/internal/provider/resilience"
)

func TestClient_GetRegionalPollen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/latest/pollen/by-lat-lng", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("lat"), "52.370")
		assert.Contains(t, r.URL.Query().Get("lng"), "4.895")
		assert.Equal(t, "****", r.Header.Get("x-api-key"))

		response := map[string]interface{}{
			"message": "success",
			"data": []map[string]interface{}{
				{
					"Count": map[string]int{
						"grass_index": 3,
						"tree_index":  1,
						"weed_index":  2,
					},
					"Risk": map[string]string{
						"grass_index": "High",
						"tree_index":  "Low",
						"weed_index":  "Moderate",
					},
					"Species": map[string][]string{
						"Grass": {"Timothy", "Rye"},
						"Tree":  {"Birch", "Oak"},
						"Weed":  {"Ragweed"},
					},
					"updatedAt": "2024-01-15T14:00:00Z",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ambee.NewClient(ambee.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	data, err := client.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, data)

	assert.Equal(t, "NL", data.Region)
	assert.Equal(t, "Netherlands", data.RegionName)
	assert.Equal(t, 52.370, data.Lat)
	assert.Equal(t, 4.895, data.Lon)
	assert.Equal(t, "ambee", data.Provider)

	// Check grass pollen
	grass := data.Readings[pollen.PollenGrass]
	require.NotNil(t, grass)
	assert.Equal(t, 3.0, grass.Index)
	assert.Equal(t, pollen.RiskHigh, grass.Risk)
	assert.Contains(t, grass.Species, "Timothy")

	// Check tree pollen
	tree := data.Readings[pollen.PollenTree]
	require.NotNil(t, tree)
	assert.Equal(t, 1.0, tree.Index)
	assert.Equal(t, pollen.RiskLow, tree.Risk)

	// Check weed pollen
	weed := data.Readings[pollen.PollenWeed]
	require.NotNil(t, weed)
	assert.Equal(t, 2.0, weed.Index)
	assert.Equal(t, pollen.RiskModerate, weed.Risk)

	// Overall risk should be highest
	assert.Equal(t, pollen.RiskHigh, data.OverallRisk)
}

func TestClient_GetRegionalPollen_AllRiskLevels(t *testing.T) {
	risks := []struct {
		ambeeRisk string
		expected  pollen.RiskLevel
	}{
		{"Low", pollen.RiskLow},
		{"Moderate", pollen.RiskModerate},
		{"High", pollen.RiskHigh},
		{"Very High", pollen.RiskVeryHigh},
		{"", pollen.RiskNone},
		{"Unknown", pollen.RiskNone},
	}

	for _, tc := range risks {
		t.Run(tc.ambeeRisk, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := map[string]interface{}{
					"message": "success",
					"data": []map[string]interface{}{
						{
							"Count": map[string]int{
								"grass_index": 2,
								"tree_index":  0,
								"weed_index":  0,
							},
							"Risk": map[string]string{
								"grass_index": tc.ambeeRisk,
								"tree_index":  "",
								"weed_index":  "",
							},
							"Species": map[string][]string{
								"Grass": {},
								"Tree":  {},
								"Weed":  {},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := ambee.NewClient(ambee.ClientConfig{
				APIKey:     "****",
				BaseURL:    server.URL,
				HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
			})

			data, err := client.GetRegionalPollen(context.Background(), 52.0, 4.0)
			require.NoError(t, err)

			grass := data.Readings[pollen.PollenGrass]
			require.NotNil(t, grass)
			assert.Equal(t, tc.expected, grass.Risk)
		})
	}
}

func TestClient_GetForecast(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/forecast/pollen/by-lat-lng", r.URL.Path)

		response := map[string]interface{}{
			"message": "success",
			"data": []map[string]interface{}{
				{
					"time": "2024-01-16",
					"Count": map[string]int{
						"grass_index": 3,
						"tree_index":  2,
						"weed_index":  1,
					},
					"Risk": map[string]string{
						"grass_index": "High",
						"tree_index":  "Moderate",
						"weed_index":  "Low",
					},
					"Species": map[string][]string{
						"Grass": {"Timothy"},
						"Tree":  {"Birch"},
						"Weed":  {},
					},
				},
				{
					"time": "2024-01-17",
					"Count": map[string]int{
						"grass_index": 2,
						"tree_index":  1,
						"weed_index":  0,
					},
					"Risk": map[string]string{
						"grass_index": "Moderate",
						"tree_index":  "Low",
						"weed_index":  "",
					},
					"Species": map[string][]string{
						"Grass": {"Rye"},
						"Tree":  {"Oak"},
						"Weed":  {},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ambee.NewClient(ambee.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	forecast, err := client.GetForecast(context.Background(), 52.370, 4.895)
	require.NoError(t, err)
	require.NotNil(t, forecast)

	assert.Equal(t, "NL", forecast.Region)
	assert.Len(t, forecast.Daily, 2)

	// First day
	day1 := forecast.Daily[0]
	assert.Equal(t, pollen.RiskHigh, day1.OverallRisk)
	assert.NotNil(t, day1.Readings[pollen.PollenGrass])

	// Second day
	day2 := forecast.Daily[1]
	assert.Equal(t, pollen.RiskModerate, day2.OverallRisk)
}

func TestClient_GetRegionalPollen_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"message": "success",
			"data":    []interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := ambee.NewClient(ambee.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	_, err := client.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.ErrorIs(t, err, pollen.ErrNoDataForRegion)
}

func TestClient_GetRegionalPollen_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := resilience.DefaultClientConfig("test")
	cfg.MaxRetries = 0

	client := ambee.NewClient(ambee.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(cfg),
	})

	_, err := client.GetRegionalPollen(context.Background(), 52.370, 4.895)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestClient_GetRegionalPollen_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := ambee.NewClient(ambee.ClientConfig{
		APIKey:     "****",
		BaseURL:    server.URL,
		HTTPClient: resilience.NewClient(resilience.DefaultClientConfig("test")),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetRegionalPollen(ctx, 52.370, 4.895)
	require.Error(t, err)
}

func TestClient_Name(t *testing.T) {
	client := ambee.NewClient(ambee.ClientConfig{
		APIKey: "****",
	})

	assert.Equal(t, "ambee", client.Name())
}
