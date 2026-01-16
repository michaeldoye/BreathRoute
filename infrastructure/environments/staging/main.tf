# -----------------------------------------------------------------------------
# BreatheRoute - Staging Environment
# -----------------------------------------------------------------------------

terraform {
  backend "gcs" {
    bucket = "breatheroute-terraform-state"
    prefix = "staging"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

module "infrastructure" {
  source = "../../"

  project_id  = var.project_id
  region      = var.region
  environment = "staging"

  # Networking
  vpc_connector_cidr = "10.8.0.0/28"

  # Cloud SQL - smaller for staging
  db_tier              = "db-f1-micro"
  db_disk_size_gb      = 10
  db_high_availability = false
  db_backup_enabled    = true

  # Redis - smaller for staging
  redis_memory_size_gb = 1
  redis_tier           = "BASIC"

  # Cloud Run - lower limits for staging
  cloud_run_min_instances = 0
  cloud_run_max_instances = 5
  cloud_run_cpu           = "1"
  cloud_run_memory        = "512Mi"
}
