package models

// ExportFormat represents the format for data export.
type ExportFormat string

// Export format values.
const (
	ExportFormatJSON ExportFormat = "JSON"
	ExportFormatZIP  ExportFormat = "ZIP"
)

// ExportRequestCreate is the request body for creating an export request.
type ExportRequestCreate struct {
	Format *ExportFormat `json:"format,omitempty" validate:"omitempty,oneof=JSON ZIP"`
}

// ExportRequest represents a GDPR data export request.
type ExportRequest struct {
	ID            string              `json:"id"`
	Status        ExportRequestStatus `json:"status"`
	CreatedAt     Timestamp           `json:"createdAt"`
	UpdatedAt     Timestamp           `json:"updatedAt"`
	DownloadURL   *string             `json:"downloadUrl,omitempty"`
	ExpiresAt     *Timestamp          `json:"expiresAt,omitempty"`
	FailureReason *string             `json:"failureReason,omitempty"`
}

// PagedExportRequests represents a paginated list of export requests.
type PagedExportRequests struct {
	Items []ExportRequest   `json:"items"`
	Meta  PagedResponseMeta `json:"meta"`
}

// DeletionRequestCreate is the request body for creating a deletion request.
type DeletionRequestCreate struct {
	Reason *string `json:"reason,omitempty" validate:"omitempty,max=500"`
}

// DeletionRequest represents a GDPR account deletion request.
type DeletionRequest struct {
	ID            string                `json:"id"`
	Status        DeletionRequestStatus `json:"status"`
	CreatedAt     Timestamp             `json:"createdAt"`
	UpdatedAt     Timestamp             `json:"updatedAt"`
	ScheduledFor  *Timestamp            `json:"scheduledFor,omitempty"`
	FailureReason *string               `json:"failureReason,omitempty"`
}

// PagedDeletionRequests represents a paginated list of deletion requests.
type PagedDeletionRequests struct {
	Items []DeletionRequest `json:"items"`
	Meta  PagedResponseMeta `json:"meta"`
}
