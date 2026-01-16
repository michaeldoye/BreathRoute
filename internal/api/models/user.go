package models

// Me represents the authenticated user's account summary.
type Me struct {
	UserID    string    `json:"userId"`
	Locale    string    `json:"locale"`
	CreatedAt Timestamp `json:"createdAt"`
}

// Consents represents the user's consent states.
type Consents struct {
	Analytics         bool      `json:"analytics"`
	Marketing         bool      `json:"marketing"`
	PushNotifications bool      `json:"pushNotifications"`
	UpdatedAt         Timestamp `json:"updatedAt"`
}

// ConsentsInput is the request body for updating consents.
type ConsentsInput struct {
	Analytics         *bool `json:"analytics,omitempty"`
	Marketing         *bool `json:"marketing,omitempty"`
	PushNotifications *bool `json:"pushNotifications,omitempty"`
}

// Profile represents the user's sensitivity profile.
type Profile struct {
	Weights     ExposureWeights  `json:"weights"`
	Constraints RouteConstraints `json:"constraints"`
	CreatedAt   Timestamp        `json:"createdAt"`
	UpdatedAt   Timestamp        `json:"updatedAt"`
}

// ProfileInput is the request body for creating or updating a profile.
type ProfileInput struct {
	Weights     ExposureWeights  `json:"weights" validate:"required"`
	Constraints RouteConstraints `json:"constraints" validate:"required"`
}

// ExposureWeights represents the relative importance of pollutant factors.
type ExposureWeights struct {
	NO2    float64 `json:"no2" validate:"gte=0,lte=1"`
	PM25   float64 `json:"pm25" validate:"gte=0,lte=1"`
	O3     float64 `json:"o3" validate:"gte=0,lte=1"`
	Pollen float64 `json:"pollen" validate:"gte=0,lte=1"`
}

// RouteConstraints represents route generation preferences.
type RouteConstraints struct {
	AvoidMajorRoads          bool  `json:"avoidMajorRoads"`
	PreferParks              *bool `json:"preferParks,omitempty"`
	MaxExtraMinutesVsFastest *int  `json:"maxExtraMinutesVsFastest,omitempty" validate:"omitempty,gte=0,lte=120"`
	MaxTransfers             *int  `json:"maxTransfers,omitempty" validate:"omitempty,gte=0,lte=10"`
}
