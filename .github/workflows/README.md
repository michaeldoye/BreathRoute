# GitHub Actions Workflows

This document describes all CI/CD workflows for the BreatheRoute project.

## Workflow Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Development Flow                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   Feature ──► PR to develop ──► develop ──► PR to main ──► main ──► Tag │
│      │             │               │             │           │       │   │
│      ▼             ▼               ▼             ▼           ▼       ▼   │
│   (local)      [ci.yml]        [ci.yml]     [ci.yml]    [ci.yml] [release]│
│                - Tests         - Tests      - Tests     - Tests  - Build │
│                - Lint          - Lint       - Lint      - Lint   - Deploy│
│                - Build check   - Build      - Build     - Build          │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Workflows

### ci.yml - Continuous Integration

**Trigger:** PRs and pushes to `main` and `develop`

**Purpose:** Validate code quality on every change

**Jobs:**
1. `test` - Run linter and tests
2. `build-check` - Verify code compiles

**No deployment occurs** - this workflow only validates code.

---

### release.yml - Production Release

**Trigger:** Git tags matching `v*.*.*` (e.g., `v1.0.0`)

**Purpose:** Build and deploy to production

**Jobs:**
1. `prepare` - Extract version from tag
2. `release-backend` - Build Docker images, push to Artifact Registry, deploy to Cloud Run
3. `release-ios` - Build iOS app and submit to App Store (if applicable)
4. `create-release` - Create GitHub Release with changelog
5. `notify` - Post release summary

**Manual Trigger:** Can also be triggered manually via workflow_dispatch with version input.

#### Creating a Release

```bash
# 1. Ensure all changes are merged to main
git checkout main
git pull

# 2. Create and push a tag
git tag v1.0.0
git push origin v1.0.0

# This triggers the release workflow automatically
```

---

### terraform.yml - Infrastructure Deployment

**Trigger:** Changes to `infrastructure/**`, Push to `main`

**Purpose:** Plan and apply Terraform infrastructure changes

**Jobs:**
1. `terraform` - Format check, init, validate, plan, (apply on main)

**Behavior:**
- On PR: Runs plan and posts result as PR comment
- On push to main: Runs plan and applies changes

---

### ios.yml - iOS Build

**Trigger:** Changes to `ios/**`, Called by release.yml

**Purpose:** Build iOS app and deploy to TestFlight/App Store

---

### security.yml - Security Scanning

**Trigger:** Weekly schedule, Pull Requests

**Purpose:** Scan for security vulnerabilities

## Required GitHub Secrets

| Secret | Description | Used By |
|--------|-------------|---------|
| `GCP_SA_KEY` | GCP service account JSON key | ci.yml, release.yml, terraform.yml |
| `APPLE_BUILD_CERTIFICATE_BASE64` | iOS signing certificate | ios.yml |
| `APPLE_P12_PASSWORD` | Certificate password | ios.yml |
| `APPLE_PROVISIONING_PROFILE_BASE64` | Provisioning profile | ios.yml |
| `APPLE_TEAM_ID` | Apple Developer Team ID | ios.yml |
| `APP_STORE_CONNECT_API_KEY_ID` | App Store Connect API key ID | ios.yml |
| `APP_STORE_CONNECT_API_ISSUER_ID` | App Store Connect issuer ID | ios.yml |
| `APP_STORE_CONNECT_API_KEY_BASE64` | App Store Connect API key | ios.yml |

## Manual Steps

### First-Time Setup

1. **Create GCP Project**
   ```bash
   gcloud projects create breatheroute --name="BreatheRoute"
   gcloud config set project breatheroute
   ```

2. **Enable Required APIs**
   ```bash
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

3. **Create Artifact Registry Repository**
   ```bash
   gcloud artifacts repositories create breatheroute \
     --repository-format=docker \
     --location=europe-west4 \
     --description="BreatheRoute container images"
   ```

4. **Create Service Account for CI/CD**
   ```bash
   # Create service account
   gcloud iam service-accounts create github-actions \
     --display-name="GitHub Actions"

   # Grant required roles
   for role in roles/editor roles/artifactregistry.writer roles/run.admin roles/iam.serviceAccountUser; do
     gcloud projects add-iam-policy-binding breatheroute \
       --member="serviceAccount:github-actions@breatheroute.iam.gserviceaccount.com" \
       --role="$role"
   done

   # Create key
   gcloud iam service-accounts keys create github-actions-key.json \
     --iam-account=github-actions@breatheroute.iam.gserviceaccount.com
   ```

5. **Add Secrets to GitHub**
   - Go to Repository Settings → Secrets and variables → Actions
   - Add `GCP_SA_KEY` with contents of `github-actions-key.json`

6. **Create Terraform State Bucket**
   ```bash
   gcloud storage buckets create gs://breatheroute-terraform-state \
     --location=europe-west4 \
     --uniform-bucket-level-access
   ```

### Deploying a New Version

1. Merge your changes to `main`
2. Wait for CI to pass
3. Create a tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
4. Monitor the release workflow in GitHub Actions

### Rolling Back

To rollback to a previous version:

```bash
# Option 1: Deploy a previous image tag via gcloud
gcloud run deploy breatheroute-api \
  --image=europe-west4-docker.pkg.dev/breatheroute/breatheroute/api:v0.9.0 \
  --region=europe-west4

# Option 2: Create a new tag pointing to old commit
git tag v1.0.1 <old-commit-sha>
git push origin v1.0.1
```

### Debugging Failed Deployments

1. Check GitHub Actions logs for the failed job
2. For Cloud Run issues:
   ```bash
   gcloud run services describe breatheroute-api --region=europe-west4
   gcloud logging read "resource.type=cloud_run_revision" --limit=50
   ```
3. For Artifact Registry issues:
   ```bash
   gcloud artifacts docker images list europe-west4-docker.pkg.dev/breatheroute/breatheroute
   ```
