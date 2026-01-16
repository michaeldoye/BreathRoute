output "api_url" {
  description = "Staging API URL"
  value       = module.infrastructure.api_url
}

output "cloud_sql_connection_name" {
  description = "Cloud SQL connection name"
  value       = module.infrastructure.cloud_sql_connection_name
}

output "api_service_account" {
  description = "API service account"
  value       = module.infrastructure.api_service_account
}

output "worker_service_account" {
  description = "Worker service account"
  value       = module.infrastructure.worker_service_account
}
