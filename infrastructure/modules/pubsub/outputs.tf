output "provider_refresh_topic_id" {
  description = "Provider refresh topic ID"
  value       = google_pubsub_topic.provider_refresh.id
}

output "provider_refresh_topic_name" {
  description = "Provider refresh topic name"
  value       = google_pubsub_topic.provider_refresh.name
}

output "alert_evaluation_topic_id" {
  description = "Alert evaluation topic ID"
  value       = google_pubsub_topic.alert_evaluation.id
}

output "alert_evaluation_topic_name" {
  description = "Alert evaluation topic name"
  value       = google_pubsub_topic.alert_evaluation.name
}

output "webhook_delivery_topic_id" {
  description = "Webhook delivery topic ID"
  value       = google_pubsub_topic.webhook_delivery.id
}

output "webhook_delivery_topic_name" {
  description = "Webhook delivery topic name"
  value       = google_pubsub_topic.webhook_delivery.name
}

output "gdpr_export_topic_id" {
  description = "GDPR export topic ID"
  value       = google_pubsub_topic.gdpr_export.id
}

output "gdpr_deletion_topic_id" {
  description = "GDPR deletion topic ID"
  value       = google_pubsub_topic.gdpr_deletion.id
}

output "dead_letter_topic_id" {
  description = "Dead letter topic ID"
  value       = google_pubsub_topic.dead_letter.id
}
