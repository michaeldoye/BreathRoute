# BreatheRoute Infrastructure

Terraform configuration for BreatheRoute GCP infrastructure.

## Prerequisites

1. [Terraform](https://www.terraform.io/downloads) >= 1.5.0
2. [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) (`gcloud`)
3. GCP project with billing enabled
4. Authenticated with GCP: `gcloud auth application-default login`

## Architecture

```
infrastructure/
├── main.tf                 # Root module - orchestrates all resources
├── variables.tf            # Input variables
├── outputs.tf              # Output values
├── versions.tf             # Provider versions
├── modules/
│   ├── networking/         # VPC, subnets, VPC connector
│   ├── cloud-sql/          # PostgreSQL + PostGIS
│   ├── memorystore/        # Redis cache
│   ├── iam/                # Service accounts & permissions
│   ├── secrets/            # Secret Manager secrets
│   ├── pubsub/             # Topics & subscriptions
│   ├── scheduler/          # Cloud Scheduler jobs
│   └── cloud-run/          # Cloud Run services
└── environments/
    └── prod/               # Production environment config
```

## Quick Start

### 1. Create GCP Project and Enable APIs

```bash
# Set your project ID
export PROJECT_ID=breatheroute

# Create the project (if not exists)
gcloud projects create $PROJECT_ID --name="BreatheRoute"

# Set as active project
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable \
  compute.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  run.googleapis.com \
  secretmanager.googleapis.com \
  pubsub.googleapis.com \
  cloudscheduler.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  vpcaccess.googleapis.com \
  servicenetworking.googleapis.com
```

### 2. Create Artifact Registry Repository

```bash
gcloud artifacts repositories create breatheroute \
  --repository-format=docker \
  --location=europe-west4 \
  --description="BreatheRoute container images"
```

### 3. Create GCS Bucket for Terraform State

```bash
gcloud storage buckets create gs://breatheroute-terraform-state \
  --location=europe-west4 \
  --uniform-bucket-level-access
```

### 4. Create Service Account for CI/CD

```bash
# Create service account
gcloud iam service-accounts create github-actions \
  --display-name="GitHub Actions"

# Grant required roles
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/editor"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.writer"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/run.admin"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:github-actions@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountUser"

gcloud storage buckets add-iam-policy-binding gs://breatheroute-terraform-state \
    --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"
    
      # Grant IAM admin permissions
  gcloud projects add-iam-policy-binding breatheroute \
    --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
    --role="roles/resourcemanager.projectIamAdmin"

  # Grant service networking permissions  
  gcloud projects add-iam-policy-binding breatheroute \
    --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
    --role="roles/servicenetworking.networksAdmin"

  # Grant secret manager admin
  gcloud projects add-iam-policy-binding breatheroute \
    --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
    --role="roles/secretmanager.admin"

  # Grant pubsub admin
  gcloud projects add-iam-policy-binding breatheroute \
    --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
    --role="roles/pubsub.admin"

# Create and download key
gcloud iam service-accounts keys create github-actions-key.json \
  --iam-account=github-actions@${PROJECT_ID}.iam.gserviceaccount.com

# Add key to GitHub Secrets as GCP_SA_KEY
echo "Add the contents of github-actions-key.json to GitHub Secrets as GCP_SA_KEY"
```

### 5. Deploy Infrastructure

```bash
cd environments/prod

# Initialize Terraform
terraform init

# Review the plan
terraform plan

# Apply the configuration
terraform apply
```

## Modules

### networking
- VPC network with private subnet
- VPC Access Connector for Cloud Run
- Private Service Access for Cloud SQL
- Firewall rules

### cloud-sql
- PostgreSQL 15 instance with PostGIS
- Private IP connectivity
- Automated backups with PITR
- Application and migration users

### memorystore
- Redis 7.0 instance
- RDB persistence
- Private connectivity

### iam
- Separate service accounts for API and Worker
- Least-privilege IAM bindings
- Cloud SQL, Secret Manager, Pub/Sub, Logging, Tracing permissions

### secrets
- Database credentials
- JWT signing key
- APNs key for push notifications
- External API keys (Luchtmeetnet, NS, Pollen)
- Webhook signing secret

### pubsub
- Provider refresh topic (hourly data refresh)
- Alert evaluation topic (daily alerts)
- Webhook delivery topic
- GDPR export/deletion topics
- Dead letter topic for failed messages

### scheduler
- Hourly provider refresh
- Daily alert evaluation (5 AM + 8 PM)
- 15-minute provider health checks

### cloud-run
- Generic Cloud Run service module
- VPC connector integration
- Secret Manager integration
- Health probes configured

## Populating Secrets

After Terraform creates the secret placeholders, populate them:

```bash
# Database password (generated by Terraform, stored automatically)

# JWT signing key
openssl rand -base64 32 | gcloud secrets versions add breatheroute-prod-jwt-signing-key --data-file=-

# APNs key
gcloud secrets versions add breatheroute-prod-apns-key --data-file=AuthKey_XXXXXXXXXX.p8

# External API keys
echo -n "your-ns-api-key" | gcloud secrets versions add breatheroute-prod-ns-api-key --data-file=-
echo -n "your-pollen-api-key" | gcloud secrets versions add breatheroute-prod-pollen-api-key --data-file=-
```

## CI/CD Integration

GitHub Actions workflows are configured in `.github/workflows/`:

| Workflow | Purpose | Trigger |
|----------|---------|---------|
| `ci.yml` | Tests, linting, build verification | PRs and pushes to `main` |
| `release.yml` | Build and deploy to production | Git tags `v*.*.*` |
| `terraform.yml` | Infrastructure deployment | Changes to `infrastructure/` |
| `ios.yml` | iOS build & TestFlight | Changes to `ios/` |
| `security.yml` | Security scanning | Weekly + PRs |

### Backend Deployment Flow

| Event | Actions |
|-------|---------|
| PR to `develop` | Run tests, linting, and build check (ci.yml) |
| Push to `develop` | Run tests, linting, and build check (ci.yml) |
| PR to `main` | Run tests, linting, and build check (ci.yml) |
| Push to `main` | Run tests, linting, and build check (ci.yml) |
| Git tag `v*.*.*` | Run tests → Build images → Deploy to Cloud Run (release.yml) |

### Creating a Release

```bash
# Tag a release (triggers production deployment)
git tag v1.0.0
git push origin v1.0.0
```

### Required GitHub Secrets

| Secret | Description |
|--------|-------------|
| `GCP_SA_KEY` | GCP service account JSON key |
| `APPLE_BUILD_CERTIFICATE_BASE64` | iOS signing certificate |
| `APPLE_P12_PASSWORD` | Certificate password |
| `APPLE_PROVISIONING_PROFILE_BASE64` | Provisioning profile |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `APP_STORE_CONNECT_API_KEY_ID` | App Store Connect API key ID |
| `APP_STORE_CONNECT_API_ISSUER_ID` | App Store Connect issuer ID |
| `APP_STORE_CONNECT_API_KEY_BASE64` | App Store Connect API key |

## Destroying Infrastructure

```bash
cd environments/prod
terraform destroy
```

## Future: Adding Staging Environment

Post-MVP, you can add a staging environment by:

1. Creating a new GCP project (`breatheroute-staging`)
2. Copying `environments/prod` to `environments/staging`
3. Updating the backend prefix and project ID
4. Updating workflows to support environment selection

## Jira Tickets Covered

This Terraform configuration covers the following MVP stories:

- **2002**: Set up Cloud SQL (Postgres) + migrations pipeline
- **2003**: Provision Redis cache (Memorystore) and connectivity from Cloud Run
- **2004**: Create service accounts + IAM least privilege for Cloud Run
- **2005**: Secret Manager integration + config conventions
- **2008**: Set up Pub/Sub topics + Cloud Scheduler triggers (jobs)
- **2006**: CI/CD pipeline (Cloud Run services ready for CI/CD deployment)
- **2007**: Observability (IAM permissions for logging/tracing configured)
