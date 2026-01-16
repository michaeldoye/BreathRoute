# Contributing to BreatheRoute

This guide covers the development workflow, commit conventions, and contribution guidelines.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- Xcode 15+ (for iOS development)
- Git 2.9+

### Setup

```bash
# Clone the repository
git clone https://github.com/breatheroute/breatheroute.git
cd breatheroute

# Run setup (installs hooks, downloads dependencies)
make setup

# Start local development environment
make dev
```

## Git Hooks

We use git hooks to enforce code quality before commits. The hooks are automatically installed when you run `make setup`.

### Pre-commit Hook

Runs automatically before each commit:

- **Go**: gofmt, go vet, golangci-lint, tests
- **Swift**: SwiftLint, SwiftFormat
- **Terraform**: terraform fmt, terraform validate, tfsec
- **General**: Secrets detection, large file warnings

### Commit Message Hook

Enforces [Conventional Commits](https://www.conventionalcommits.org/) format.

## Commit Message Convention

All commits must follow the Conventional Commits format:

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | Description |
|------|-------------|
| `feat` | A new feature |
| `fix` | A bug fix |
| `docs` | Documentation only changes |
| `style` | Formatting, white-space, etc. (no code change) |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `perf` | A code change that improves performance |
| `test` | Adding or updating tests |
| `build` | Changes to build process or dependencies |
| `ci` | Changes to CI/CD configuration |
| `chore` | Other changes that don't modify src or test files |
| `revert` | Reverts a previous commit |

### Scope (Optional)

The scope provides additional context about what part of the codebase is affected:

- `api` - Backend API changes
- `worker` - Background worker changes
- `ios` - iOS app changes
- `routes` - Routing feature
- `commutes` - Commutes feature
- `alerts` - Alerts/notifications feature
- `auth` - Authentication
- `infra` - Infrastructure/Terraform
- `deps` - Dependencies

### Examples

```bash
# Feature
feat(routes): add exposure score calculation

# Bug fix
fix(auth): handle expired tokens correctly

# Documentation
docs: update API documentation

# Refactoring
refactor(api): extract validation logic into separate package

# Performance improvement
perf(routes): cache station data to reduce API calls

# Tests
test(commutes): add unit tests for schedule validation

# Build/Dependencies
build(deps): upgrade go-chi to v5.0.12

# CI/CD
ci: add CodeQL security scanning

# Chore
chore: update .gitignore for IDE files
```

### Breaking Changes

For breaking changes, add `!` after the type/scope or add `BREAKING CHANGE:` in the footer:

```bash
# With exclamation mark
feat(api)!: change route response format

# With footer
feat(api): change route response format

BREAKING CHANGE: Route response now uses camelCase for all fields
```

### Referencing Issues

Reference Jira tickets in the commit body:

```bash
feat(commutes): add schedule editing

Implements the ability to edit commute schedules after creation.
Users can now modify days of week and preferred arrival time.

Refs: BR-123
```

## Pull Request Process

### 1. Create a Branch

```bash
# Feature branch
git checkout -b feat/route-exposure-scoring

# Bug fix branch
git checkout -b fix/auth-token-expiry

# Use kebab-case for branch names
```

### 2. Make Changes

- Write code following our [Go Coding Standards](./GO_CODING_STANDARDS.md) or [Swift Coding Standards](./SWIFT_CODING_STANDARDS.md)
- Add tests for new functionality
- Update documentation as needed

### 3. Commit Changes

```bash
# Stage changes
git add .

# Commit with conventional message
git commit -m "feat(routes): add exposure scoring algorithm"
```

### 4. Push and Create PR

```bash
# Push branch
git push origin feat/route-exposure-scoring

# Create PR via GitHub CLI or web
gh pr create --title "feat(routes): add exposure scoring algorithm" --body "..."
```

### 5. PR Requirements

Before a PR can be merged:

- [ ] All CI checks pass
- [ ] Code review approved
- [ ] No merge conflicts
- [ ] Tests added/updated
- [ ] Documentation updated (if needed)

## Code Review Guidelines

### For Authors

- Keep PRs small and focused
- Write clear PR descriptions
- Respond to feedback promptly
- Don't take feedback personally

### For Reviewers

- Be constructive and respectful
- Explain the "why" behind suggestions
- Approve when good enough, not perfect
- Use these prefixes:
  - `nit:` - Minor suggestion, optional
  - `suggestion:` - Recommended but not required
  - `blocking:` - Must be addressed before merge

## Testing

### Running Tests

```bash
# Go tests
make test

# With coverage
make test-coverage

# Swift tests (from ios directory)
xcodebuild test -scheme BreatheRoute -destination 'platform=iOS Simulator,name=iPhone 15'
```

### Test Requirements

- Unit tests for business logic
- Integration tests for API endpoints
- UI tests for critical user flows
- Aim for 80%+ coverage on new code

## Linting

### Go

```bash
# Run linter
make lint

# Format code
make fmt
```

### Swift

```bash
cd ios

# Run linter
swiftlint

# Auto-fix
swiftlint --fix
```

### Terraform

```bash
cd infrastructure

# Format
terraform fmt -recursive

# Validate
terraform validate
```

## Bypassing Hooks

In rare cases where you need to bypass pre-commit hooks:

```bash
# Skip pre-commit hook
git commit --no-verify -m "wip: temporary commit"

# This should be rare and temporary
```

## Release Process

See [RELEASE.md](../RELEASE.md) for the full release process.

Quick version:

```bash
# Create a release tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## Getting Help

- Check existing issues and PRs
- Ask in the team Slack channel
- Consult the coding standards docs
- Reach out to maintainers

## Code of Conduct

- Be respectful and inclusive
- Assume good intentions
- Focus on the code, not the person
- Help others learn and grow
