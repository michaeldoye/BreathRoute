package models

// CommuteLocation represents a location for a commute endpoint.
type CommuteLocation struct {
	Point   Point   `json:"point" validate:"required"`
	Geohash *string `json:"geohash,omitempty"`
}

// CommuteSchedule represents the normalized schedule for a commute.
type CommuteSchedule struct {
	// DaysOfWeek contains ISO weekday numbers (1=Monday, 7=Sunday)
	DaysOfWeek []int `json:"daysOfWeek"`
	// DayNames contains human-readable day names for convenience
	DayNames []string `json:"dayNames"`
	// ArrivalTime is the preferred arrival time in HH:mm format
	ArrivalTime string `json:"arrivalTime"`
	// Timezone is the IANA timezone identifier (e.g., "Europe/Amsterdam")
	Timezone string `json:"timezone"`
	// NextOccurrence is the next scheduled commute time (if within 7 days), in RFC3339 format
	NextOccurrence *string `json:"nextOccurrence,omitempty"`
	// IsActiveToday indicates if the commute is scheduled for today
	IsActiveToday bool `json:"isActiveToday"`
}

// Commute represents a saved commute.
type Commute struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Origin      CommuteLocation `json:"origin"`
	Destination CommuteLocation `json:"destination"`
	Schedule    CommuteSchedule `json:"schedule"`
	Notes       *string         `json:"notes,omitempty"`
	CreatedAt   Timestamp       `json:"createdAt"`
	UpdatedAt   Timestamp       `json:"updatedAt"`
}

// CommuteCreateRequest is the request body for creating a commute.
type CommuteCreateRequest struct {
	Label                     string          `json:"label" validate:"required,min=1,max=80"`
	Origin                    CommuteLocation `json:"origin" validate:"required"`
	Destination               CommuteLocation `json:"destination" validate:"required"`
	DaysOfWeek                []int           `json:"daysOfWeek" validate:"required,dive,gte=1,lte=7"`
	PreferredArrivalTimeLocal string          `json:"preferredArrivalTimeLocal" validate:"required,time_hhmm"`
	Timezone                  *string         `json:"timezone,omitempty" validate:"omitempty,timezone"`
	Notes                     *string         `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// CommuteUpdateRequest is the request body for updating a commute.
type CommuteUpdateRequest struct {
	Label                     *string          `json:"label,omitempty" validate:"omitempty,min=1,max=80"`
	Origin                    *CommuteLocation `json:"origin,omitempty"`
	Destination               *CommuteLocation `json:"destination,omitempty"`
	DaysOfWeek                []int            `json:"daysOfWeek,omitempty" validate:"omitempty,dive,gte=1,lte=7"`
	PreferredArrivalTimeLocal *string          `json:"preferredArrivalTimeLocal,omitempty" validate:"omitempty,time_hhmm"`
	Timezone                  *string          `json:"timezone,omitempty" validate:"omitempty,timezone"`
	Notes                     *string          `json:"notes,omitempty" validate:"omitempty,max=500"`
}

// PagedCommutes represents a paginated list of commutes.
type PagedCommutes struct {
	Items []Commute         `json:"items"`
	Meta  PagedResponseMeta `json:"meta"`
}
