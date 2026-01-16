# -----------------------------------------------------------------------------
# BreatheRoute Infrastructure - Main Configuration
# -----------------------------------------------------------------------------

locals {
  name_prefix = "breatheroute-${var.environment}"

  labels = {
    app         = "breatheroute"
    environment = var.environment
    managed_by  = "terraform"
  }
}

# -----------------------------------------------------------------------------
# Enable Required APIs
# -----------------------------------------------------------------------------
resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "sqladmin.googleapis.com",
    "redis.googleapis.com",
    "secretmanager.googleapis.com",
    "pubsub.googleapis.com",
    "cloudscheduler.googleapis.com",
    "vpcaccess.googleapis.com",
    "servicenetworking.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "iam.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "cloudtrace.googleapis.com",
    "artifactregistry.googleapis.com",
  ])

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}

# -----------------------------------------------------------------------------
# Artifact Registry (Container Images)
# -----------------------------------------------------------------------------
resource "google_artifact_registry_repository" "containers" {
  project       = var.project_id
  location      = var.region
  repository_id = "breatheroute"
  description   = "BreatheRoute container images"
  format        = "DOCKER"
  labels        = local.labels

  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 10
    }
  }

  cleanup_policies {
    id     = "delete-old-untagged"
    action = "DELETE"
    condition {
      tag_state  = "UNTAGGED"
      older_than = "604800s" # 7 days
    }
  }

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Networking
# -----------------------------------------------------------------------------
module "networking" {
  source = "./modules/networking"

  project_id         = var.project_id
  region             = var.region
  name_prefix        = local.name_prefix
  vpc_connector_cidr = var.vpc_connector_cidr
  labels             = local.labels

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# IAM - Service Accounts
# -----------------------------------------------------------------------------
module "iam" {
  source = "./modules/iam"

  project_id  = var.project_id
  name_prefix = local.name_prefix
}

# -----------------------------------------------------------------------------
# Secret Manager
# -----------------------------------------------------------------------------
module "secrets" {
  source = "./modules/secrets"

  project_id  = var.project_id
  region      = var.region
  name_prefix = local.name_prefix
  labels      = local.labels

  # Service accounts that need secret access
  api_service_account_email    = module.iam.api_service_account_email
  worker_service_account_email = module.iam.worker_service_account_email

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Cloud SQL (Postgres + PostGIS)
# -----------------------------------------------------------------------------
module "cloud_sql" {
  source = "./modules/cloud-sql"

  project_id          = var.project_id
  region              = var.region
  name_prefix         = local.name_prefix
  labels              = local.labels
  network_id          = module.networking.vpc_id
  tier                = var.db_tier
  disk_size_gb        = var.db_disk_size_gb
  high_availability   = var.db_high_availability
  backup_enabled      = var.db_backup_enabled
  private_network     = module.networking.vpc_self_link

  depends_on = [
    google_project_service.apis,
    module.networking
  ]
}

# -----------------------------------------------------------------------------
# Memorystore Redis
# -----------------------------------------------------------------------------
module "redis" {
  source = "./modules/memorystore"

  project_id     = var.project_id
  region         = var.region
  name_prefix    = local.name_prefix
  labels         = local.labels
  memory_size_gb = var.redis_memory_size_gb
  tier           = var.redis_tier
  network_id     = module.networking.vpc_id

  depends_on = [
    google_project_service.apis,
    module.networking
  ]
}

# -----------------------------------------------------------------------------
# Pub/Sub Topics & Subscriptions
# -----------------------------------------------------------------------------
module "pubsub" {
  source = "./modules/pubsub"

  project_id  = var.project_id
  name_prefix = local.name_prefix
  labels      = local.labels

  # Service accounts for subscriptions
  worker_service_account_email = module.iam.worker_service_account_email

  depends_on = [google_project_service.apis]
}

# -----------------------------------------------------------------------------
# Cloud Scheduler Jobs
# -----------------------------------------------------------------------------
module "scheduler" {
  source = "./modules/scheduler"

  project_id  = var.project_id
  region      = var.region
  name_prefix = local.name_prefix

  # Pub/Sub topics to trigger
  provider_refresh_topic = module.pubsub.provider_refresh_topic_id
  alert_evaluation_topic = module.pubsub.alert_evaluation_topic_id

  depends_on = [
    google_project_service.apis,
    module.pubsub
  ]
}

# -----------------------------------------------------------------------------
# Cloud Run Services
# -----------------------------------------------------------------------------
module "cloud_run_api" {
  source = "./modules/cloud-run"

  project_id      = var.project_id
  region          = var.region
  name            = "${local.name_prefix}-api"
  labels          = local.labels
  service_account = module.iam.api_service_account_email
  vpc_connector   = module.networking.vpc_connector_id

  min_instances = var.cloud_run_min_instances
  max_instances = var.cloud_run_max_instances
  cpu           = var.cloud_run_cpu
  memory        = var.cloud_run_memory

  # Environment variables (non-secret)
  env_vars = {
    ENVIRONMENT          = var.environment
    GCP_PROJECT          = var.project_id
    REDIS_HOST           = module.redis.host
    REDIS_PORT           = tostring(module.redis.port)
    DB_HOST              = module.cloud_sql.private_ip
    DB_PORT              = "5432"
    DB_NAME              = module.cloud_sql.database_name
    PUBSUB_PROJECT       = var.project_id
  }

  # Secret references
  secret_env_vars = {
    DB_PASSWORD = {
      secret_id = module.secrets.db_password_secret_id
      version   = "latest"
    }
    JWT_SIGNING_KEY = {
      secret_id = module.secrets.jwt_signing_key_secret_id
      version   = "latest"
    }
  }

  # Allow unauthenticated access (public API)
  allow_unauthenticated = true

  depends_on = [
    google_project_service.apis,
    module.networking,
    module.cloud_sql,
    module.redis,
    module.secrets
  ]
}

module "cloud_run_worker" {
  source = "./modules/cloud-run"

  project_id      = var.project_id
  region          = var.region
  name            = "${local.name_prefix}-worker"
  labels          = local.labels
  service_account = module.iam.worker_service_account_email
  vpc_connector   = module.networking.vpc_connector_id

  min_instances = 0 # Always scale to zero when idle
  max_instances = 5
  cpu           = var.cloud_run_cpu
  memory        = var.cloud_run_memory

  env_vars = {
    ENVIRONMENT          = var.environment
    GCP_PROJECT          = var.project_id
    REDIS_HOST           = module.redis.host
    REDIS_PORT           = tostring(module.redis.port)
    DB_HOST              = module.cloud_sql.private_ip
    DB_PORT              = "5432"
    DB_NAME              = module.cloud_sql.database_name
    PUBSUB_PROJECT       = var.project_id
  }

  secret_env_vars = {
    DB_PASSWORD = {
      secret_id = module.secrets.db_password_secret_id
      version   = "latest"
    }
    APNS_KEY = {
      secret_id = module.secrets.apns_key_secret_id
      version   = "latest"
    }
  }

  # Worker is triggered by Pub/Sub, not public
  allow_unauthenticated = false

  depends_on = [
    google_project_service.apis,
    module.networking,
    module.cloud_sql,
    module.redis,
    module.secrets
  ]
}
