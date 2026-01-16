# -----------------------------------------------------------------------------
# Secret Manager Module
# Secrets for database credentials, API keys, signing keys
# -----------------------------------------------------------------------------

# Database password secret
resource "google_secret_manager_secret" "db_password" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-db-password"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# JWT signing key
resource "google_secret_manager_secret" "jwt_signing_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-jwt-signing-key"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# APNs key for push notifications
resource "google_secret_manager_secret" "apns_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-apns-key"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# Luchtmeetnet API key (if needed for authenticated endpoints)
resource "google_secret_manager_secret" "luchtmeetnet_api_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-luchtmeetnet-api-key"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# NS API key
resource "google_secret_manager_secret" "ns_api_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-ns-api-key"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# Pollen API key (BreezoMeter)
resource "google_secret_manager_secret" "pollen_api_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-pollen-api-key"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# Webhook signing secret (for partner webhooks)
resource "google_secret_manager_secret" "webhook_signing_secret" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-webhook-signing-secret"

  replication {
    user_managed {
      replicas {
        location = var.region
      }
    }
  }

  labels = var.labels
}

# -----------------------------------------------------------------------------
# IAM bindings for secret access
# -----------------------------------------------------------------------------

# API service account access
resource "google_secret_manager_secret_iam_member" "api_db_password" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.db_password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_jwt_key" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.jwt_signing_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_luchtmeetnet" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.luchtmeetnet_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_ns" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.ns_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_pollen" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.pollen_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

# Worker service account access
resource "google_secret_manager_secret_iam_member" "worker_db_password" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.db_password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_apns" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.apns_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_luchtmeetnet" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.luchtmeetnet_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_pollen" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.pollen_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_webhook" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.webhook_signing_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}
