variable "project_id" {
  description = "GCP project ID for staging"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "europe-west4"
}
