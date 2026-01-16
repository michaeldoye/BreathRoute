output "id" {
  description = "Cloud Run service ID"
  value       = google_cloud_run_v2_service.main.id
}

output "name" {
  description = "Cloud Run service name"
  value       = google_cloud_run_v2_service.main.name
}

output "url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.main.uri
}

output "latest_revision" {
  description = "Latest revision name"
  value       = google_cloud_run_v2_service.main.latest_ready_revision
}
