output "instance_name" {
  description = "Cloud SQL instance name"
  value       = google_sql_database_instance.main.name
}

output "connection_name" {
  description = "Cloud SQL connection name"
  value       = google_sql_database_instance.main.connection_name
}

output "private_ip" {
  description = "Private IP address"
  value       = google_sql_database_instance.main.private_ip_address
  sensitive   = true
}

output "database_name" {
  description = "Database name"
  value       = google_sql_database.app.name
}

output "app_user" {
  description = "Application database user"
  value       = google_sql_user.app.name
}

output "app_password" {
  description = "Application database password"
  value       = random_password.db_password.result
  sensitive   = true
}

output "migrations_user" {
  description = "Migrations database user"
  value       = google_sql_user.migrations.name
}

output "migrations_password" {
  description = "Migrations database password"
  value       = random_password.migrations_password.result
  sensitive   = true
}
