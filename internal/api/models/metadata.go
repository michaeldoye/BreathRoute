package models

// Station represents an air quality monitoring station.
type Station struct {
	StationID  string      `json:"stationId"`
	Name       string      `json:"name"`
	Point      Point       `json:"point"`
	Pollutants []Pollutant `json:"pollutants,omitempty"`
	UpdatedAt  Timestamp   `json:"updatedAt"`
}

// PagedStations represents a paginated list of stations.
type PagedStations struct {
	Items []Station         `json:"items"`
	Meta  PagedResponseMeta `json:"meta"`
}

// Enums represents the enum values used by the API.
type Enums struct {
	Modes      []Mode       `json:"modes"`
	Objectives []Objective  `json:"objectives"`
	Confidence []Confidence `json:"confidence"`
	Pollutants []Pollutant  `json:"pollutants"`
}
