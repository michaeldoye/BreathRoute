output "api_service_account_email" {
  description = "API service account email"
  value       = google_service_account.api.email
}

output "api_service_account_id" {
  description = "API service account ID"
  value       = google_service_account.api.id
}

output "worker_service_account_email" {
  description = "Worker service account email"
  value       = google_service_account.worker.email
}

output "worker_service_account_id" {
  description = "Worker service account ID"
  value       = google_service_account.worker.id
}
