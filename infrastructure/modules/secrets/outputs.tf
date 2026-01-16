output "db_password_secret_id" {
  description = "Database password secret ID"
  value       = google_secret_manager_secret.db_password.secret_id
}

output "jwt_signing_key_secret_id" {
  description = "JWT signing key secret ID"
  value       = google_secret_manager_secret.jwt_signing_key.secret_id
}

output "apns_key_secret_id" {
  description = "APNs key secret ID"
  value       = google_secret_manager_secret.apns_key.secret_id
}

output "luchtmeetnet_api_key_secret_id" {
  description = "Luchtmeetnet API key secret ID"
  value       = google_secret_manager_secret.luchtmeetnet_api_key.secret_id
}

output "ns_api_key_secret_id" {
  description = "NS API key secret ID"
  value       = google_secret_manager_secret.ns_api_key.secret_id
}

output "pollen_api_key_secret_id" {
  description = "Pollen API key secret ID"
  value       = google_secret_manager_secret.pollen_api_key.secret_id
}

output "webhook_signing_secret_id" {
  description = "Webhook signing secret ID"
  value       = google_secret_manager_secret.webhook_signing_secret.secret_id
}
