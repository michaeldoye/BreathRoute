variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Cloud Scheduler"
  type        = string
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "provider_refresh_topic" {
  description = "Pub/Sub topic ID for provider refresh jobs"
  type        = string
}

variable "alert_evaluation_topic" {
  description = "Pub/Sub topic ID for alert evaluation jobs"
  type        = string
}
