# -----------------------------------------------------------------------------
# Cloud Run Module
# Generic Cloud Run service configuration
# -----------------------------------------------------------------------------

resource "google_cloud_run_v2_service" "main" {
  project  = var.project_id
  name     = var.name
  location = var.region
  ingress  = var.allow_unauthenticated ? "INGRESS_TRAFFIC_ALL" : "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    labels = var.labels

    service_account = var.service_account

    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    vpc_access {
      connector = var.vpc_connector
      egress    = "PRIVATE_RANGES_ONLY"
    }

    timeout = var.timeout

    containers {
      # Image will be updated by CI/CD, start with placeholder
      image = var.image != "" ? var.image : "gcr.io/cloudrun/hello"

      resources {
        limits = {
          cpu    = var.cpu
          memory = var.memory
        }
        cpu_idle          = var.cpu_idle
        startup_cpu_boost = var.startup_cpu_boost
      }

      # Non-secret environment variables
      dynamic "env" {
        for_each = var.env_vars
        content {
          name  = env.key
          value = env.value
        }
      }

      # Secret environment variables
      dynamic "env" {
        for_each = var.secret_env_vars
        content {
          name = env.key
          value_source {
            secret_key_ref {
              secret  = env.value.secret_id
              version = env.value.version
            }
          }
        }
      }

      # Health check
      startup_probe {
        http_get {
          path = "/v1/ops/health"
          port = 8080
        }
        initial_delay_seconds = 5
        timeout_seconds       = 3
        period_seconds        = 10
        failure_threshold     = 3
      }

      liveness_probe {
        http_get {
          path = "/v1/ops/health"
          port = 8080
        }
        timeout_seconds   = 3
        period_seconds    = 30
        failure_threshold = 3
      }

      ports {
        name           = "http1"
        container_port = 8080
      }
    }

    # Maximum concurrent requests per instance
    max_instance_request_concurrency = var.concurrency
  }

  traffic {
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
    percent = 100
  }

  lifecycle {
    ignore_changes = [
      # Ignore image changes - managed by CI/CD
      template[0].containers[0].image,
      # Ignore client info annotations
      client,
      client_version,
    ]
  }
}

# IAM binding for public access (if allowed)
resource "google_cloud_run_v2_service_iam_member" "public" {
  count = var.allow_unauthenticated ? 1 : 0

  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.main.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
