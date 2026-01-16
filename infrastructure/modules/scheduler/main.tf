# -----------------------------------------------------------------------------
# Cloud Scheduler Module
# Scheduled triggers for background jobs
# -----------------------------------------------------------------------------

# Provider refresh job (hourly)
# Refreshes air quality and pollen data from external providers
resource "google_cloud_scheduler_job" "provider_refresh" {
  project     = var.project_id
  region      = var.region
  name        = "${var.name_prefix}-provider-refresh"
  description = "Hourly refresh of air quality and pollen data"
  schedule    = "0 * * * *" # Every hour at :00
  time_zone   = "Europe/Amsterdam"

  pubsub_target {
    topic_name = var.provider_refresh_topic
    data       = base64encode(jsonencode({
      job_type   = "provider_refresh"
      refresh_all = true
    }))
    attributes = {
      job_type = "provider_refresh"
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "30s"
    max_backoff_duration = "300s"
  }
}

# Alert evaluation job (daily, early morning)
# Evaluates alerts for commutes scheduled for the next 24 hours
resource "google_cloud_scheduler_job" "alert_evaluation" {
  project     = var.project_id
  region      = var.region
  name        = "${var.name_prefix}-alert-evaluation"
  description = "Daily evaluation of commute alerts"
  schedule    = "0 5 * * *" # 5 AM daily (before morning commute)
  time_zone   = "Europe/Amsterdam"

  pubsub_target {
    topic_name = var.alert_evaluation_topic
    data       = base64encode(jsonencode({
      job_type       = "alert_evaluation"
      lookahead_hours = 24
    }))
    attributes = {
      job_type = "alert_evaluation"
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "60s"
    max_backoff_duration = "600s"
  }
}

# Secondary alert evaluation (evening for next-day alerts)
resource "google_cloud_scheduler_job" "alert_evaluation_evening" {
  project     = var.project_id
  region      = var.region
  name        = "${var.name_prefix}-alert-evaluation-evening"
  description = "Evening evaluation for next-day commute alerts"
  schedule    = "0 20 * * *" # 8 PM daily
  time_zone   = "Europe/Amsterdam"

  pubsub_target {
    topic_name = var.alert_evaluation_topic
    data       = base64encode(jsonencode({
      job_type        = "alert_evaluation"
      lookahead_hours = 18 # Next morning
      priority        = "next_day"
    }))
    attributes = {
      job_type = "alert_evaluation"
      priority = "next_day"
    }
  }

  retry_config {
    retry_count          = 3
    min_backoff_duration = "60s"
    max_backoff_duration = "600s"
  }
}

# Provider health check (every 15 minutes)
resource "google_cloud_scheduler_job" "provider_health_check" {
  project     = var.project_id
  region      = var.region
  name        = "${var.name_prefix}-provider-health-check"
  description = "Periodic health check of external providers"
  schedule    = "*/15 * * * *" # Every 15 minutes
  time_zone   = "Europe/Amsterdam"

  pubsub_target {
    topic_name = var.provider_refresh_topic
    data       = base64encode(jsonencode({
      job_type    = "health_check"
      check_only  = true
    }))
    attributes = {
      job_type = "health_check"
    }
  }

  retry_config {
    retry_count          = 1
    min_backoff_duration = "10s"
    max_backoff_duration = "60s"
  }
}
