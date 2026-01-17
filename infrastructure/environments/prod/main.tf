# -----------------------------------------------------------------------------
# BreatheRoute - Production Environment
# -----------------------------------------------------------------------------

terraform {
  backend "gcs" {
    bucket = "breatheroute-terraform-state"
    prefix = "prod"
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
  environment = "prod"

  # Networking
  vpc_connector_cidr = "10.8.0.0/28"

  # Cloud SQL - production sized
  db_tier              = "db-custom-2-4096" # 2 vCPUs, 4GB RAM
  db_disk_size_gb      = 50
  db_high_availability = true
  db_backup_enabled    = true

  # Redis - production sized
  redis_memory_size_gb = 2
  redis_tier           = "STANDARD_HA" # High availability

  # Cloud Run - production limits
  cloud_run_min_instances = 1 # Always warm
  cloud_run_max_instances = 20
  cloud_run_cpu           = "2"
  cloud_run_memory        = "1Gi"

  # Custom domain (optional)
  # domain = "api.breatheroute.nl"
}
