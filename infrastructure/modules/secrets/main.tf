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

# Weather API key (OpenWeatherMap)
resource "google_secret_manager_secret" "weather_api_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-weather-api-key"

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

# OpenRouteService API key (for bike/walk routing)
resource "google_secret_manager_secret" "openrouteservice_api_key" {
  project   = var.project_id
  secret_id = "${var.name_prefix}-openrouteservice-api-key"

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
# Initial Secret Versions (placeholders - update with real values)
# -----------------------------------------------------------------------------

resource "google_secret_manager_secret_version" "db_password" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data] # Don't overwrite after manual update
  }
}

resource "google_secret_manager_secret_version" "jwt_signing_key" {
  secret      = google_secret_manager_secret.jwt_signing_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "apns_key" {
  secret      = google_secret_manager_secret.apns_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "luchtmeetnet_api_key" {
  secret      = google_secret_manager_secret.luchtmeetnet_api_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "ns_api_key" {
  secret      = google_secret_manager_secret.ns_api_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "pollen_api_key" {
  secret      = google_secret_manager_secret.pollen_api_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "weather_api_key" {
  secret      = google_secret_manager_secret.weather_api_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "webhook_signing_secret" {
  secret      = google_secret_manager_secret.webhook_signing_secret.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret_version" "openrouteservice_api_key" {
  secret      = google_secret_manager_secret.openrouteservice_api_key.id
  secret_data = "PLACEHOLDER_UPDATE_ME"

  lifecycle {
    ignore_changes = [secret_data]
  }
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

resource "google_secret_manager_secret_iam_member" "api_weather" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.weather_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.api_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "api_openrouteservice" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.openrouteservice_api_key.secret_id
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

resource "google_secret_manager_secret_iam_member" "worker_weather" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.weather_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_ns" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.ns_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_openrouteservice" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.openrouteservice_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}

resource "google_secret_manager_secret_iam_member" "worker_webhook" {
  project   = var.project_id
  secret_id = google_secret_manager_secret.webhook_signing_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${var.worker_service_account_email}"
}
