variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for secret replication"
  type        = string
}

variable "name_prefix" {
  description = "Prefix for resource names"
  type        = string
}

variable "labels" {
  description = "Labels to apply to resources"
  type        = map(string)
  default     = {}
}

variable "api_service_account_email" {
  description = "API service account email"
  type        = string
}

variable "worker_service_account_email" {
  description = "Worker service account email"
  type        = string
}
