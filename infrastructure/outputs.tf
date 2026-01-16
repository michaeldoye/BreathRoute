# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------

# Networking
output "vpc_id" {
  description = "VPC network ID"
  value       = module.networking.vpc_id
}

output "vpc_connector_id" {
  description = "VPC Access Connector ID"
  value       = module.networking.vpc_connector_id
}

# Cloud SQL
output "cloud_sql_instance_name" {
  description = "Cloud SQL instance name"
  value       = module.cloud_sql.instance_name
}

output "cloud_sql_connection_name" {
  description = "Cloud SQL connection name for Cloud Run"
  value       = module.cloud_sql.connection_name
}

output "cloud_sql_private_ip" {
  description = "Cloud SQL private IP address"
  value       = module.cloud_sql.private_ip
  sensitive   = true
}

output "database_name" {
  description = "PostgreSQL database name"
  value       = module.cloud_sql.database_name
}

# Redis
output "redis_host" {
  description = "Redis host"
  value       = module.redis.host
  sensitive   = true
}

output "redis_port" {
  description = "Redis port"
  value       = module.redis.port
}

# Cloud Run
output "api_url" {
  description = "Cloud Run API service URL"
  value       = module.cloud_run_api.url
}

output "worker_url" {
  description = "Cloud Run Worker service URL"
  value       = module.cloud_run_worker.url
}

# Service Accounts
output "api_service_account" {
  description = "API service account email"
  value       = module.iam.api_service_account_email
}

output "worker_service_account" {
  description = "Worker service account email"
  value       = module.iam.worker_service_account_email
}

# Artifact Registry
output "container_registry_url" {
  description = "Artifact Registry URL for container images"
  value       = "${google_artifact_registry_repository.containers.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.containers.repository_id}"
}

# Pub/Sub Topics
output "provider_refresh_topic" {
  description = "Pub/Sub topic for provider refresh jobs"
  value       = module.pubsub.provider_refresh_topic_id
}

output "alert_evaluation_topic" {
  description = "Pub/Sub topic for alert evaluation jobs"
  value       = module.pubsub.alert_evaluation_topic_id
}
