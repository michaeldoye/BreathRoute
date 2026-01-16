output "id" {
  description = "Redis instance ID"
  value       = google_redis_instance.main.id
}

output "host" {
  description = "Redis host IP"
  value       = google_redis_instance.main.host
}

output "port" {
  description = "Redis port"
  value       = google_redis_instance.main.port
}

output "current_location_id" {
  description = "Current zone where the Redis instance is located"
  value       = google_redis_instance.main.current_location_id
}
