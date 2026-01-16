package models

// Health represents the health status of the service.
type Health struct {
	Status  HealthStatus           `json:"status"`
	Time    Timestamp              `json:"time"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SystemStatus represents the overall system status.
type SystemStatus struct {
	Status                 HealthStatus      `json:"status"`
	Time                   Timestamp         `json:"time"`
	Subsystems             []SubsystemStatus `json:"subsystems"`
	Providers              []ProviderStatus  `json:"providers"`
	ActiveDegradationFlags []string          `json:"activeDegradationFlags,omitempty"`
}

// SubsystemStatus represents the status of a subsystem.
type SubsystemStatus struct {
	Name   string       `json:"name"`
	Status HealthStatus `json:"status"`
	Detail *string      `json:"detail,omitempty"`
}

// ProviderStatus represents the status of an external provider.
type ProviderStatus struct {
	Provider      string       `json:"provider"`
	Status        HealthStatus `json:"status"`
	LastSuccessAt *Timestamp   `json:"lastSuccessAt,omitempty"`
	LastFailureAt *Timestamp   `json:"lastFailureAt,omitempty"`
	Message       *string      `json:"message,omitempty"`
}
