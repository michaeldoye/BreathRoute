# Development Guide

This guide covers setting up and running BreatheRoute locally for development.

## Prerequisites

- [Go 1.22+](https://golang.org/dl/)
- [Docker](https://www.docker.com/products/docker-desktop) and Docker Compose
- [Make](https://www.gnu.org/software/make/) (usually pre-installed on macOS/Linux)

Optional but recommended:
- [golangci-lint](https://golangci-lint.run/usage/install/)
- [golang-migrate](https://github.com/golang-migrate/migrate)

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/your-org/breatheroute.git
cd breatheroute

# 2. Copy environment file
cp .env.example .env

# 3. Start dependencies (PostgreSQL + Redis)
make dev

# 4. Run database migrations
make migrate-up

# 5. Start the API server
make api
```

The API will be available at `http://localhost:8080`.

## Development Commands

Run `make help` to see all available commands:

```
Development:
  dev           Start local development dependencies (Postgres, Redis)
  dev-tools     Start development tools (pgAdmin, Redis Commander, MailHog)
  dev-down      Stop all development containers
  dev-clean     Stop containers and remove volumes (fresh start)
  dev-logs      Tail logs from development containers
  db-shell      Open PostgreSQL shell
  redis-shell   Open Redis CLI

Build & Test:
  build         Build all binaries
  api           Build and run the API service
  worker        Build and run the Worker service
  test          Run all tests
  test-coverage Run tests and show coverage report
  lint          Run linter
  fmt           Format code
  clean         Remove build artifacts

Database:
  migrate-up    Run database migrations
  migrate-down  Rollback last migration
  migrate-create Create a new migration (usage: make migrate-create name=create_users)

Docker:
  docker-build  Build Docker images
  docker-push   Push Docker images to registry
```

## Project Structure

```
breatheroute/
├── cmd/
│   ├── api/              # API service entrypoint
│   │   └── main.go
│   └── worker/           # Worker service entrypoint
│       └── main.go
├── internal/             # Private application code
│   ├── api/              # HTTP handlers, middleware
│   ├── domain/           # Business logic, entities
│   ├── repository/       # Database access
│   └── provider/         # External API integrations
├── pkg/                  # Public libraries (shared code)
├── migrations/           # Database migrations
├── scripts/              # Development and deployment scripts
├── infrastructure/       # Terraform configuration
├── ios/                  # iOS application (Swift)
├── docker-compose.yml    # Local development services
├── Dockerfile.api        # API container image
├── Dockerfile.worker     # Worker container image
├── Makefile              # Development commands
└── .env.example          # Environment template
```

## Local Services

### PostgreSQL + PostGIS

- **Host**: localhost
- **Port**: 5432
- **User**: breatheroute
- **Password**: localdev
- **Database**: breatheroute

Connect with psql:
```bash
make db-shell
# or
psql -h localhost -U breatheroute -d breatheroute
```

### Redis

- **Host**: localhost
- **Port**: 6379
- **Password**: (none)

Connect with redis-cli:
```bash
make redis-shell
# or
redis-cli -h localhost
```

## Development Tools

Start optional development tools:

```bash
make dev-tools
```

| Tool | URL | Credentials |
|------|-----|-------------|
| pgAdmin | http://localhost:8082 | admin@breatheroute.local / localdev |
| Redis Commander | http://localhost:8081 | - |
| MailHog | http://localhost:8025 | - |

## Database Migrations

We use [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations.

### Install migrate CLI

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Create a new migration

```bash
make migrate-create name=create_users_table
```

This creates two files in `migrations/`:
- `XXXXXX_create_users_table.up.sql`
- `XXXXXX_create_users_table.down.sql`

### Run migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down
```

## Testing

```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage

# Run specific package tests
go test -v ./internal/api/...

# Run tests with race detection
go test -race ./...
```

## Linting

```bash
# Install golangci-lint (one-time)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
make lint
```

## Environment Variables

Copy `.env.example` to `.env` and configure:

| Variable | Description | Default |
|----------|-------------|---------|
| `APP_ENV` | Environment (development/staging/production) | development |
| `APP_PORT` | HTTP server port | 8080 |
| `DB_HOST` | PostgreSQL host | localhost |
| `DB_PORT` | PostgreSQL port | 5432 |
| `REDIS_HOST` | Redis host | localhost |
| `REDIS_PORT` | Redis port | 6379 |

See `.env.example` for the complete list.

## Running Both Services

In separate terminals:

```bash
# Terminal 1: API service
make api

# Terminal 2: Worker service
make worker
```

Or use a process manager like [overmind](https://github.com/DarthSim/overmind) or [foreman](https://github.com/ddollar/foreman).

## Debugging

### VS Code

Add to `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "API",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/api",
      "envFile": "${workspaceFolder}/.env"
    },
    {
      "name": "Worker",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/worker",
      "envFile": "${workspaceFolder}/.env"
    }
  ]
}
```

### GoLand

1. Create a "Go Build" run configuration
2. Set package path to `./cmd/api` or `./cmd/worker`
3. Add environment file: `.env`

## Troubleshooting

### Port already in use

```bash
# Find process using port 5432
lsof -i :5432

# Kill it
kill -9 <PID>

# Or use different ports in docker-compose.yml
```

### Database connection refused

```bash
# Check if containers are running
docker compose ps

# Check container logs
docker compose logs postgres
```

### Permission denied on Docker

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
# Log out and back in
```

### Fresh start

```bash
# Remove all containers and volumes
make dev-clean

# Start fresh
make dev
make migrate-up
```

## IDE Setup

### GoLand / IntelliJ

1. Open the project folder
2. Go to Preferences → Go → GOROOT and set Go SDK
3. Enable Go Modules integration
4. Set `.env` as environment file for run configurations

### VS Code

Recommended extensions:
- Go (golang.go)
- Docker (ms-azuretools.vscode-docker)
- GitLens (eamodio.gitlens)
- Even Better TOML (tamasfe.even-better-toml)

## API Documentation

Once the API is running, OpenAPI documentation is available at:
- Swagger UI: http://localhost:8080/swagger/
- OpenAPI spec: http://localhost:8080/swagger/doc.json
