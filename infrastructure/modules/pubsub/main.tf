# -----------------------------------------------------------------------------
# Pub/Sub Module
# Topics and subscriptions for background job processing
# -----------------------------------------------------------------------------

# Provider refresh topic (hourly AQ/pollen refresh)
resource "google_pubsub_topic" "provider_refresh" {
  project = var.project_id
  name    = "${var.name_prefix}-provider-refresh"
  labels  = var.labels

  message_retention_duration = "86400s" # 24 hours
}

# Alert evaluation topic (daily alert evaluation)
resource "google_pubsub_topic" "alert_evaluation" {
  project = var.project_id
  name    = "${var.name_prefix}-alert-evaluation"
  labels  = var.labels

  message_retention_duration = "86400s"
}

# Webhook delivery topic
resource "google_pubsub_topic" "webhook_delivery" {
  project = var.project_id
  name    = "${var.name_prefix}-webhook-delivery"
  labels  = var.labels

  message_retention_duration = "86400s"
}

# GDPR export topic
resource "google_pubsub_topic" "gdpr_export" {
  project = var.project_id
  name    = "${var.name_prefix}-gdpr-export"
  labels  = var.labels

  message_retention_duration = "86400s"
}

# GDPR deletion topic
resource "google_pubsub_topic" "gdpr_deletion" {
  project = var.project_id
  name    = "${var.name_prefix}-gdpr-deletion"
  labels  = var.labels

  message_retention_duration = "86400s"
}

# Dead letter topic (for failed messages)
resource "google_pubsub_topic" "dead_letter" {
  project = var.project_id
  name    = "${var.name_prefix}-dead-letter"
  labels  = var.labels

  message_retention_duration = "604800s" # 7 days
}

# -----------------------------------------------------------------------------
# Subscriptions
# -----------------------------------------------------------------------------

resource "google_pubsub_subscription" "provider_refresh" {
  project = var.project_id
  name    = "${var.name_prefix}-provider-refresh-sub"
  topic   = google_pubsub_topic.provider_refresh.name
  labels  = var.labels

  ack_deadline_seconds       = 120
  message_retention_duration = "86400s"

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }

  expiration_policy {
    ttl = "" # Never expire
  }
}

resource "google_pubsub_subscription" "alert_evaluation" {
  project = var.project_id
  name    = "${var.name_prefix}-alert-evaluation-sub"
  topic   = google_pubsub_topic.alert_evaluation.name
  labels  = var.labels

  ack_deadline_seconds       = 300 # 5 minutes for heavy processing
  message_retention_duration = "86400s"

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 5
  }

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_subscription" "webhook_delivery" {
  project = var.project_id
  name    = "${var.name_prefix}-webhook-delivery-sub"
  topic   = google_pubsub_topic.webhook_delivery.name
  labels  = var.labels

  ack_deadline_seconds       = 60
  message_retention_duration = "86400s"

  retry_policy {
    minimum_backoff = "30s"
    maximum_backoff = "600s" # Max allowed is 10 minutes # Up to 1 hour for webhook retries
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 12 # Match webhook retry policy
  }

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_subscription" "gdpr_export" {
  project = var.project_id
  name    = "${var.name_prefix}-gdpr-export-sub"
  topic   = google_pubsub_topic.gdpr_export.name
  labels  = var.labels

  ack_deadline_seconds       = 600 # 10 minutes for export jobs
  message_retention_duration = "86400s"

  retry_policy {
    minimum_backoff = "60s"
    maximum_backoff = "600s" # Max allowed is 10 minutes
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 3
  }

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_subscription" "gdpr_deletion" {
  project = var.project_id
  name    = "${var.name_prefix}-gdpr-deletion-sub"
  topic   = google_pubsub_topic.gdpr_deletion.name
  labels  = var.labels

  ack_deadline_seconds       = 600
  message_retention_duration = "86400s"

  retry_policy {
    minimum_backoff = "60s"
    maximum_backoff = "600s" # Max allowed is 10 minutes
  }

  dead_letter_policy {
    dead_letter_topic     = google_pubsub_topic.dead_letter.id
    max_delivery_attempts = 3
  }

  expiration_policy {
    ttl = ""
  }
}

resource "google_pubsub_subscription" "dead_letter" {
  project = var.project_id
  name    = "${var.name_prefix}-dead-letter-sub"
  topic   = google_pubsub_topic.dead_letter.name
  labels  = var.labels

  ack_deadline_seconds       = 60
  message_retention_duration = "604800s" # 7 days

  expiration_policy {
    ttl = ""
  }
}

# -----------------------------------------------------------------------------
# IAM for subscriptions
# -----------------------------------------------------------------------------

resource "google_pubsub_subscription_iam_member" "worker_provider_refresh" {
  project      = var.project_id
  subscription = google_pubsub_subscription.provider_refresh.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_pubsub_subscription_iam_member" "worker_alert_evaluation" {
  project      = var.project_id
  subscription = google_pubsub_subscription.alert_evaluation.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_pubsub_subscription_iam_member" "worker_webhook_delivery" {
  project      = var.project_id
  subscription = google_pubsub_subscription.webhook_delivery.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_pubsub_subscription_iam_member" "worker_gdpr_export" {
  project      = var.project_id
  subscription = google_pubsub_subscription.gdpr_export.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_pubsub_subscription_iam_member" "worker_gdpr_deletion" {
  project      = var.project_id
  subscription = google_pubsub_subscription.gdpr_deletion.name
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:${var.worker_service_account_email}"
}
