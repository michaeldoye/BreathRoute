# -----------------------------------------------------------------------------
# IAM Module
# Service accounts with least-privilege permissions
# -----------------------------------------------------------------------------

# API Service Account
resource "google_service_account" "api" {
  project      = var.project_id
  account_id   = "${var.name_prefix}-api-sa"
  display_name = "BreatheRoute API Service Account"
  description  = "Service account for the public API Cloud Run service"
}

# Worker Service Account
resource "google_service_account" "worker" {
  project      = var.project_id
  account_id   = "${var.name_prefix}-worker-sa"
  display_name = "BreatheRoute Worker Service Account"
  description  = "Service account for the background worker Cloud Run service"
}

# -----------------------------------------------------------------------------
# API Service Account Permissions
# -----------------------------------------------------------------------------

# Cloud SQL Client (connect to database)
resource "google_project_iam_member" "api_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Secret Manager Secret Accessor
resource "google_project_iam_member" "api_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Pub/Sub Publisher (for enqueueing jobs)
resource "google_project_iam_member" "api_pubsub_publisher" {
  project = var.project_id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Cloud Trace Agent
resource "google_project_iam_member" "api_trace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Cloud Logging Writer
resource "google_project_iam_member" "api_logging_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# Cloud Monitoring Metric Writer
resource "google_project_iam_member" "api_monitoring_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.api.email}"
}

# -----------------------------------------------------------------------------
# Worker Service Account Permissions
# -----------------------------------------------------------------------------

# Cloud SQL Client
resource "google_project_iam_member" "worker_sql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Secret Manager Secret Accessor
resource "google_project_iam_member" "worker_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Pub/Sub Subscriber (for consuming jobs)
resource "google_project_iam_member" "worker_pubsub_subscriber" {
  project = var.project_id
  role    = "roles/pubsub.subscriber"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Pub/Sub Publisher (for re-enqueueing failed jobs)
resource "google_project_iam_member" "worker_pubsub_publisher" {
  project = var.project_id
  role    = "roles/pubsub.publisher"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Cloud Trace Agent
resource "google_project_iam_member" "worker_trace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Cloud Logging Writer
resource "google_project_iam_member" "worker_logging_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.worker.email}"
}

# Cloud Monitoring Metric Writer
resource "google_project_iam_member" "worker_monitoring_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.worker.email}"
}
