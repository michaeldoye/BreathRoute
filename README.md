# BreatheRoute

[![Backend CI](https://github.com/breatheroute/breatheroute/actions/workflows/deploy.yml/badge.svg)](https://github.com/breatheroute/breatheroute/actions/workflows/deploy.yml)
[![iOS CI](https://github.com/breatheroute/breatheroute/actions/workflows/ios.yml/badge.svg)](https://github.com/breatheroute/breatheroute/actions/workflows/ios.yml)
[![Terraform](https://github.com/breatheroute/breatheroute/actions/workflows/terraform.yml/badge.svg)](https://github.com/breatheroute/breatheroute/actions/workflows/terraform.yml)
[![Security](https://github.com/breatheroute/breatheroute/actions/workflows/security.yml/badge.svg)](https://github.com/breatheroute/breatheroute/actions/workflows/security.yml)

**BreatheRoute** is a commuter routing application for the Netherlands that optimizes routes based on **air quality exposure** (NO₂, PM2.5, O₃), **pollen levels**, and travel time. The app helps users find healthier commute options, particularly for cyclists and people with respiratory sensitivities.

## Features

- **Smart Route Planning** - Get 2-5 route alternatives with exposure scores
- **Air Quality Aware** - Real-time data from Luchtmeetnet stations
- **Time-Shift Recommendations** - "Leave 20 min earlier for 30% lower NO₂"
- **Commute Management** - Save recurring commutes with schedules
- **Push Alerts** - Get notified when air quality affects your commute
- **Pollen Forecasts** - Plan around high pollen days

## Tech Stack

| Component | Technology |
|-----------|------------|
| **Backend** | Go 1.22 on Cloud Run |
| **Database** | Cloud SQL PostgreSQL 15 + PostGIS |
| **Cache** | Memorystore Redis 7.0 |
| **Mobile** | iOS (Swift/SwiftUI) |
| **Auth** | Sign in with Apple |
| **Push** | APNs |
| **Infrastructure** | Terraform on GCP |
| **CI/CD** | GitHub Actions |

## Quick Start

### Prerequisites

- [Go 1.22+](https://golang.org/dl/)
- [Docker](https://www.docker.com/products/docker-desktop) and Docker Compose
- [Make](https://www.gnu.org/software/make/)

### Setup

```bash
# Clone the repository
git clone https://github.com/breatheroute/breatheroute.git
cd breatheroute

# Run setup (installs hooks, downloads dependencies)
make setup

# Start local development environment (PostgreSQL + Redis)
make dev

# Run the API server
make api
```

The API will be available at `http://localhost:8080`.

### Verify Installation

```bash
# Health check
curl http://localhost:8080/v1/health

# Expected response
{"status":"healthy","version":"dev"}
```

## Project Structure

```
breatheroute/
├── cmd/
│   ├── api/                  # API service entrypoint
│   └── worker/               # Background worker entrypoint
├── internal/                 # Private application code
├── pkg/                      # Public shared libraries
├── migrations/               # Database migrations
├── infrastructure/           # Terraform configuration
│   ├── modules/              # Reusable Terraform modules
│   └── environments/         # Staging and production configs
├── ios/                      # iOS application (Swift/SwiftUI)
├── docs/                     # Documentation
│   ├── GO_CODING_STANDARDS.md
│   ├── SWIFT_CODING_STANDARDS.md
│   └── CONTRIBUTING.md
├── scripts/                  # Development scripts
├── .github/
│   └── workflows/            # CI/CD pipelines
├── docker-compose.yml        # Local development services
├── Dockerfile.api            # API container image
├── Dockerfile.worker         # Worker container image
└── Makefile                  # Development commands
```

## Development Commands

Run `make help` to see all available commands:

```
Setup:
  setup          Complete project setup (hooks, dependencies)
  setup-hooks    Install git hooks for pre-commit linting

Development Environment:
  dev            Start local dependencies (Postgres, Redis)
  dev-tools      Start dev tools (pgAdmin, Redis Commander, MailHog)
  dev-down       Stop all development containers
  dev-clean      Stop containers and remove volumes
  db-shell       Open PostgreSQL shell
  redis-shell    Open Redis CLI

Build & Test:
  build          Build all binaries
  api            Build and run the API service
  worker         Build and run the Worker service
  test           Run all tests
  test-coverage  Run tests with coverage report
  lint           Run linter
  fmt            Format code
  check          Run all checks (fmt, lint, test)

Database:
  migrate-up     Run database migrations
  migrate-down   Rollback last migration
  migrate-create Create a new migration

Docker:
  docker-build   Build Docker images
  docker-push    Push images to registry
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              iOS App                                     │
│                         (Swift/SwiftUI)                                  │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                           Cloud Run                                      │
│  ┌─────────────────────┐          ┌─────────────────────┐               │
│  │     API Service     │          │   Worker Service    │               │
│  │    (Public API)     │          │  (Background Jobs)  │               │
│  └──────────┬──────────┘          └──────────┬──────────┘               │
└─────────────┼─────────────────────────────────┼─────────────────────────┘
              │                                 │
              ▼                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Internal Services                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │ Cloud SQL  │  │   Redis    │  │  Pub/Sub   │  │ Scheduler  │        │
│  │ (Postgres) │  │  (Cache)   │  │ (Messaging)│  │  (Cron)    │        │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘        │
└─────────────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        External Providers                                │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐        │
│  │Luchtmeetnet│  │   NS API   │  │BreezoMeter │  │ Weather API│        │
│  │(Air Quality)│ │  (Transit) │  │  (Pollen)  │  │            │        │
│  └────────────┘  └────────────┘  └────────────┘  └────────────┘        │
└─────────────────────────────────────────────────────────────────────────┘
```

## CI/CD Pipelines

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `deploy.yml` | Push to main/develop | Build & deploy backend |
| `ios.yml` | Changes to ios/ | Build & upload to TestFlight |
| `terraform.yml` | Changes to infrastructure/ | Deploy infrastructure |
| `release.yml` | Git tags (v*.*.*) | Coordinated releases |
| `security.yml` | Weekly + PRs | Security scanning |

### Deployment Flow

```
develop branch  ──────►  Staging
main branch     ──────►  Production
v*.*.* tag      ──────►  Release (Backend + iOS + GitHub Release)
```

## Infrastructure

Infrastructure is managed with Terraform. See [infrastructure/README.md](./infrastructure/README.md) for details.

### Environments

| Environment | Project ID | Purpose |
|-------------|------------|---------|
| Staging | `breatheroute-staging` | Development and testing |
| Production | `breatheroute-prod` | Live application |

### Key Resources

- **Cloud Run** - API and Worker services
- **Cloud SQL** - PostgreSQL with PostGIS
- **Memorystore** - Redis cache
- **Pub/Sub** - Async messaging
- **Cloud Scheduler** - Cron jobs
- **Secret Manager** - Credentials storage
- **Artifact Registry** - Container images

## API Documentation

The API follows the OpenAPI 3.0 specification. Key endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/health` | Health check |
| `POST` | `/v1/auth/apple` | Sign in with Apple |
| `GET` | `/v1/routes` | Calculate routes |
| `GET` | `/v1/commutes` | List saved commutes |
| `POST` | `/v1/commutes` | Create commute |
| `GET` | `/v1/alerts` | Get alert subscriptions |
| `POST` | `/v1/devices` | Register push device |

## Configuration

Copy `.env.example` to `.env` for local development:

```bash
cp .env.example .env
```

Key configuration:

| Variable | Description |
|----------|-------------|
| `APP_PORT` | HTTP server port (default: 8080) |
| `DB_HOST` | PostgreSQL host |
| `REDIS_HOST` | Redis host |
| `JWT_SIGNING_KEY` | JWT token signing key |
| `LUCHTMEETNET_API_URL` | Air quality API URL |

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test -v ./internal/routes/...
```

## Linting

Pre-commit hooks automatically run linting. To run manually:

```bash
# Go
make lint

# Swift (from ios directory)
swiftlint

# Terraform
terraform fmt -check -recursive infrastructure
```

## Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```bash
feat(routes): add exposure scoring algorithm
fix(auth): handle expired tokens correctly
docs: update API documentation
```

See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for details.

## Documentation

| Document | Description |
|----------|-------------|
| [DEVELOPMENT.md](./DEVELOPMENT.md) | Local development guide |
| [RELEASE.md](./RELEASE.md) | Release process |
| [CONTRIBUTING.md](./docs/CONTRIBUTING.md) | Contribution guidelines |
| [GO_CODING_STANDARDS.md](./docs/GO_CODING_STANDARDS.md) | Go best practices |
| [SWIFT_CODING_STANDARDS.md](./docs/SWIFT_CODING_STANDARDS.md) | Swift best practices |
| [infrastructure/README.md](./infrastructure/README.md) | Infrastructure guide |

## Roadmap

### MVP (Current)

- [x] Infrastructure setup (GCP, Terraform)
- [x] CI/CD pipelines
- [ ] Backend API implementation
- [ ] iOS app foundation
- [ ] Air quality integration
- [ ] Route calculation
- [ ] Push notifications

### Future

- [ ] Android app
- [ ] Web dashboard
- [ ] Premium subscriptions
- [ ] Family/employer tiers
- [ ] Precomputed exposure tiles

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feat/amazing-feature`)
5. Open a Pull Request

See [CONTRIBUTING.md](./docs/CONTRIBUTING.md) for detailed guidelines.

## License

This project is proprietary software. All rights reserved.

## Support

- **Issues**: [GitHub Issues](https://github.com/breatheroute/breatheroute/issues)
- **Email**: support@breatheroute.nl
