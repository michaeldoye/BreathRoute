# Release Process

This document describes the release process for BreatheRoute, covering both backend (Go) and iOS (Swift) components.

## Overview

BreatheRoute uses GitHub Actions for automated releases. The release process is triggered by git tags and coordinates deployment across all components.

```
┌─────────────────────────────────────────────────────────────────┐
│                        Release Flow                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   git tag v1.0.0                                                │
│        │                                                         │
│        ▼                                                         │
│   ┌─────────────┐                                               │
│   │   GitHub    │                                               │
│   │   Actions   │                                               │
│   └──────┬──────┘                                               │
│          │                                                       │
│    ┌─────┴─────┐                                                │
│    ▼           ▼                                                │
│ ┌──────┐  ┌──────┐                                              │
│ │ API  │  │ iOS  │                                              │
│ │Worker│  │ App  │                                              │
│ └──┬───┘  └──┬───┘                                              │
│    │         │                                                   │
│    ▼         ▼                                                   │
│ ┌──────┐  ┌──────────┐                                          │
│ │Cloud │  │App Store │                                          │
│ │ Run  │  │ Connect  │                                          │
│ └──────┘  └──────────┘                                          │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Environments

| Environment | Branch | Backend | iOS |
|-------------|--------|---------|-----|
| Staging | `develop` | Auto-deploy | TestFlight (internal) |
| Production | `main` | Auto-deploy | TestFlight (external) |
| Release | `v*.*.*` tag | Versioned deploy | App Store |

## Pre-Release Checklist

Before creating a release, ensure:

- [ ] All tests pass on `main` branch
- [ ] No critical security vulnerabilities (check Security tab)
- [ ] Changelog/release notes prepared
- [ ] iOS version number updated in Xcode (`MARKETING_VERSION`)
- [ ] Any required database migrations are ready
- [ ] Feature flags configured for new features

## Creating a Release

### Step 1: Prepare the Release Branch

```bash
# Ensure you're on main and up to date
git checkout main
git pull origin main

# Verify all tests pass
go test ./...
```

### Step 2: Update Version Numbers

**iOS (required for App Store):**
1. Open `ios/BreatheRoute.xcodeproj` in Xcode
2. Update `MARKETING_VERSION` (e.g., `1.0.0`)
3. Commit the change:
   ```bash
   git add ios/
   git commit -m "chore: bump iOS version to 1.0.0"
   git push origin main
   ```

**Backend (optional - version injected at build time):**
The backend version is automatically set from the git tag during the build process.

### Step 3: Create and Push the Tag

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag
git push origin v1.0.0
```

### Step 4: Monitor the Release

1. Go to **Actions** tab in GitHub
2. Watch the "Release" workflow
3. Verify all jobs complete successfully:
   - `release-backend` → Cloud Run deployment
   - `release-ios` → App Store Connect upload
   - `create-release` → GitHub Release creation

### Step 5: Post-Release Tasks

**Backend:**
- Verify Cloud Run services are healthy
- Check logs for any startup errors
- Validate API endpoints respond correctly

**iOS:**
- Go to [App Store Connect](https://appstoreconnect.apple.com)
- Wait for build processing (5-30 minutes)
- Add release notes
- Submit for review (if App Store release)

## Manual Release Options

### Backend Only

```bash
# Via GitHub Actions UI
# Go to Actions → Release → Run workflow
# Select "backend" for components
```

Or deploy directly:
```bash
gcloud run deploy breatheroute-api \
  --image=europe-west4-docker.pkg.dev/breatheroute-prod/breatheroute/api:v1.0.0 \
  --region=europe-west4 \
  --project=breatheroute-prod
```

### iOS Only

```bash
# Via GitHub Actions UI
# Go to Actions → iOS Build & Release → Run workflow
# Select "app-store" for destination
```

## Rollback Procedures

### Backend Rollback

Cloud Run maintains revision history. To rollback:

```bash
# List revisions
gcloud run revisions list \
  --service=breatheroute-api \
  --region=europe-west4 \
  --project=breatheroute-prod

# Route traffic to previous revision
gcloud run services update-traffic breatheroute-api \
  --to-revisions=breatheroute-api-PREVIOUS_REVISION=100 \
  --region=europe-west4 \
  --project=breatheroute-prod
```

Or via Console:
1. Go to [Cloud Run Console](https://console.cloud.google.com/run)
2. Select the service
3. Click "Manage Traffic"
4. Route 100% to the previous revision

### iOS Rollback

iOS apps cannot be directly rolled back. Options:

1. **Expedited Review**: Submit a hotfix and request expedited review
2. **Remove from Sale**: Temporarily remove the app (extreme cases)
3. **TestFlight**: Push a fixed build to TestFlight immediately

## Hotfix Process

For urgent production fixes:

```bash
# Create hotfix branch from the release tag
git checkout -b hotfix/1.0.1 v1.0.0

# Make fixes
# ... edit files ...

# Commit and push
git add .
git commit -m "fix: critical bug description"
git push origin hotfix/1.0.1

# Create PR to main, get review, merge

# Tag the hotfix release
git checkout main
git pull origin main
git tag -a v1.0.1 -m "Hotfix v1.0.1"
git push origin v1.0.1
```

## Version Numbering

We follow [Semantic Versioning](https://semver.org/):

```
MAJOR.MINOR.PATCH

1.0.0 → 1.0.1  (patch: bug fixes)
1.0.1 → 1.1.0  (minor: new features, backward compatible)
1.1.0 → 2.0.0  (major: breaking changes)
```

**iOS Build Numbers:**
- Automatically generated as `YYYYMMDDHHMM`
- Example: `202501161430` for Jan 16, 2025 at 14:30

## Release Artifacts

Each release produces:

| Artifact | Location | Retention |
|----------|----------|-----------|
| Docker images | Artifact Registry | 10 versions |
| iOS IPA | GitHub Actions | 90 days |
| iOS dSYM | GitHub Actions | 90 days |
| GitHub Release | Repository | Permanent |

## Monitoring a Release

### Backend Health Checks

```bash
# Check API health
curl https://api.breatheroute.nl/v1/health

# Check Cloud Run status
gcloud run services describe breatheroute-api \
  --region=europe-west4 \
  --project=breatheroute-prod
```

### Key Metrics to Watch

After a release, monitor for 30-60 minutes:

- **Error rate**: Should remain < 1%
- **Latency p95**: Should remain < 500ms
- **Instance count**: Watch for unexpected scaling
- **Memory usage**: Check for leaks

### Logs

```bash
# Stream API logs
gcloud logging tail \
  "resource.type=cloud_run_revision AND resource.labels.service_name=breatheroute-api" \
  --project=breatheroute-prod

# Stream Worker logs
gcloud logging tail \
  "resource.type=cloud_run_revision AND resource.labels.service_name=breatheroute-worker" \
  --project=breatheroute-prod
```

## Troubleshooting

### Release Workflow Failed

1. Check the failed job in GitHub Actions
2. Common issues:
   - **GCP auth failed**: Verify `GCP_SA_KEY` secret is valid
   - **iOS signing failed**: Check certificate expiry
   - **Tests failed**: Fix tests before re-releasing

### Build Stuck in App Store Processing

- Processing typically takes 5-30 minutes
- If stuck > 1 hour, check App Store Connect for errors
- Ensure the build number is unique

### Cloud Run Deployment Failed

```bash
# Check deployment status
gcloud run services describe breatheroute-api \
  --region=europe-west4 \
  --project=breatheroute-prod \
  --format="value(status.conditions)"
```

Common issues:
- **Container failed to start**: Check health endpoint
- **Permission denied**: Verify service account permissions
- **Resource exhausted**: Check quotas

## Emergency Contacts

| Role | Contact |
|------|---------|
| Backend Lead | TBD |
| iOS Lead | TBD |
| DevOps | TBD |

## Related Documentation

- [Infrastructure README](./infrastructure/README.md)
- [API Documentation](./docs/api.md)
- [Runbook](./docs/runbook.md)
