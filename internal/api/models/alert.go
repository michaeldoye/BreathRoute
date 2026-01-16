package models

// AlertPreviewRequest is the request body for previewing departure windows.
type AlertPreviewRequest struct {
	CommuteID           *string       `json:"commuteId,omitempty"`
	Origin              *Point        `json:"origin,omitempty"`
	Destination         *Point        `json:"destination,omitempty"`
	TargetArrivalTime   *string       `json:"targetArrivalTime,omitempty"`
	TargetDepartureTime *string       `json:"targetDepartureTime,omitempty"`
	WindowMinutes       *int          `json:"windowMinutes,omitempty" validate:"omitempty,gte=10,lte=360"`
	StepMinutes         *int          `json:"stepMinutes,omitempty" validate:"omitempty,gte=5,lte=60"`
	Objective           Objective     `json:"objective" validate:"required,oneof=FASTEST LOWEST_EXPOSURE BALANCED"`
	Modes               []Mode        `json:"modes,omitempty"`
	ProfileOverride     *ProfileInput `json:"profileOverride,omitempty"`
}

// AlertPreviewResponse is the response for departure window preview.
type AlertPreviewResponse struct {
	Recommended    []DepartureRecommendation `json:"recommended"`
	EvaluatedCount *int                      `json:"evaluatedCount,omitempty"`
	Objective      *Objective                `json:"objective,omitempty"`
}

// DepartureRecommendation represents a recommended departure time.
type DepartureRecommendation struct {
	DepartureTime   Timestamp  `json:"departureTime"`
	DurationSeconds int        `json:"durationSeconds"`
	ExposureScore   float64    `json:"exposureScore"`
	Confidence      Confidence `json:"confidence"`
	Rationale       string     `json:"rationale"`
}

// AlertSubscription represents an alert subscription for a commute.
type AlertSubscription struct {
	ID         string         `json:"id"`
	CommuteID  string         `json:"commuteId"`
	Enabled    bool           `json:"enabled"`
	Objective  Objective      `json:"objective"`
	Threshold  AlertThreshold `json:"threshold"`
	QuietHours QuietHours     `json:"quietHours"`
	Schedule   *AlertSchedule `json:"schedule,omitempty"`
	CreatedAt  Timestamp      `json:"createdAt"`
	UpdatedAt  Timestamp      `json:"updatedAt"`
}

// AlertThreshold determines when an alert triggers.
type AlertThreshold struct {
	Type                    AlertThresholdType `json:"type" validate:"required,oneof=ABSOLUTE_SCORE PERCENT_WORSE_THAN_BASELINE"`
	AbsoluteScore           *float64           `json:"absoluteScore,omitempty"`
	PercentWorseThanBaseline *float64           `json:"percentWorseThanBaseline,omitempty"`
}

// QuietHours defines when alerts should not be sent.
type QuietHours struct {
	StartLocal string `json:"startLocal" validate:"required,time_hhmm"`
	EndLocal   string `json:"endLocal" validate:"required,time_hhmm"`
}

// AlertSchedule defines when alerts are evaluated.
type AlertSchedule struct {
	DaysOfWeek          []int  `json:"daysOfWeek" validate:"required,dive,gte=1,lte=7"`
	EvaluationTimeLocal string `json:"evaluationTimeLocal" validate:"required,time_hhmm"`
}

// AlertSubscriptionCreateRequest is the request body for creating an alert subscription.
type AlertSubscriptionCreateRequest struct {
	CommuteID  string         `json:"commuteId" validate:"required"`
	Enabled    *bool          `json:"enabled,omitempty"`
	Objective  Objective      `json:"objective" validate:"required,oneof=FASTEST LOWEST_EXPOSURE BALANCED"`
	Threshold  AlertThreshold `json:"threshold" validate:"required"`
	QuietHours *QuietHours    `json:"quietHours,omitempty"`
	Schedule   AlertSchedule  `json:"schedule" validate:"required"`
}

// AlertSubscriptionUpdateRequest is the request body for updating an alert subscription.
type AlertSubscriptionUpdateRequest struct {
	Enabled    *bool           `json:"enabled,omitempty"`
	Objective  *Objective      `json:"objective,omitempty" validate:"omitempty,oneof=FASTEST LOWEST_EXPOSURE BALANCED"`
	Threshold  *AlertThreshold `json:"threshold,omitempty"`
	QuietHours *QuietHours     `json:"quietHours,omitempty"`
	Schedule   *AlertSchedule  `json:"schedule,omitempty"`
}

// PagedAlertSubscriptions represents a paginated list of alert subscriptions.
type PagedAlertSubscriptions struct {
	Items []AlertSubscription `json:"items"`
	Meta  PagedResponseMeta   `json:"meta"`
}
