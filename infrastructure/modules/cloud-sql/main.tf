# -----------------------------------------------------------------------------
# Cloud SQL Module
# PostgreSQL with PostGIS extension
# -----------------------------------------------------------------------------

resource "random_id" "db_suffix" {
  byte_length = 4
}

resource "google_sql_database_instance" "main" {
  project          = var.project_id
  name             = "${var.name_prefix}-db-${random_id.db_suffix.hex}"
  region           = var.region
  database_version = "POSTGRES_15"

  deletion_protection = var.deletion_protection

  settings {
    tier              = var.tier
    disk_size         = var.disk_size_gb
    disk_type         = "PD_SSD"
    disk_autoresize   = true
    availability_type = var.high_availability ? "REGIONAL" : "ZONAL"

    ip_configuration {
      ipv4_enabled    = false
      private_network = var.private_network
    }

    backup_configuration {
      enabled                        = var.backup_enabled
      point_in_time_recovery_enabled = var.backup_enabled
      start_time                     = "03:00" # 3 AM UTC
      location                       = var.region

      backup_retention_settings {
        retained_backups = 7
        retention_unit   = "COUNT"
      }
    }

    maintenance_window {
      day          = 7 # Sunday
      hour         = 4 # 4 AM UTC
      update_track = "stable"
    }

    database_flags {
      name  = "log_checkpoints"
      value = "on"
    }

    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
    }

    database_flags {
      name  = "log_lock_waits"
      value = "on"
    }

    insights_config {
      query_insights_enabled  = true
      query_string_length     = 1024
      record_application_tags = true
      record_client_address   = false
    }

    user_labels = var.labels
  }

  lifecycle {
    prevent_destroy = false # Set to true for prod
  }
}

# Application database
resource "google_sql_database" "app" {
  project  = var.project_id
  name     = "breatheroute"
  instance = google_sql_database_instance.main.name
}

# Application user
resource "random_password" "db_password" {
  length  = 32
  special = false
}

resource "google_sql_user" "app" {
  project  = var.project_id
  name     = "breatheroute_app"
  instance = google_sql_database_instance.main.name
  password = random_password.db_password.result
}

# Migrations user (for CI/CD)
resource "random_password" "migrations_password" {
  length  = 32
  special = false
}

resource "google_sql_user" "migrations" {
  project  = var.project_id
  name     = "breatheroute_migrations"
  instance = google_sql_database_instance.main.name
  password = random_password.migrations_password.result
}
