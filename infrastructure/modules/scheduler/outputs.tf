output "provider_refresh_job_name" {
  description = "Provider refresh scheduler job name"
  value       = google_cloud_scheduler_job.provider_refresh.name
}

output "alert_evaluation_job_name" {
  description = "Alert evaluation scheduler job name"
  value       = google_cloud_scheduler_job.alert_evaluation.name
}

output "provider_health_check_job_name" {
  description = "Provider health check scheduler job name"
  value       = google_cloud_scheduler_job.provider_health_check.name
}
