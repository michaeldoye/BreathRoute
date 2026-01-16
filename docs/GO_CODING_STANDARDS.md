# Go Coding Standards

This document defines the coding standards and best practices for Go development in the BreatheRoute project. These standards are based on official Go guidelines, industry best practices, and lessons learned from production systems.

## Table of Contents

1. [Formatting and Style](#formatting-and-style)
2. [Naming Conventions](#naming-conventions)
3. [Package Design](#package-design)
4. [Error Handling](#error-handling)
5. [Testing](#testing)
6. [Concurrency](#concurrency)
7. [Documentation](#documentation)
8. [Security](#security)
9. [Performance](#performance)
10. [Project Structure](#project-structure)
11. [Dependencies](#dependencies)
12. [Code Review Checklist](#code-review-checklist)

---

## Formatting and Style

### Use Standard Tools

All code must pass these checks before merge:

```bash
# Format code
gofmt -s -w .

# Or use goimports (preferred - also manages imports)
goimports -w .

# Lint code
golangci-lint run
```

### Line Length

- Aim for 100 characters per line
- Hard limit at 120 characters
- Break long function signatures across multiple lines

```go
// Good: Multi-line function signature
func ProcessRouteRequest(
    ctx context.Context,
    origin, destination Coordinate,
    options RouteOptions,
) (*RouteResponse, error) {
    // ...
}

// Avoid: Single long line
func ProcessRouteRequest(ctx context.Context, origin, destination Coordinate, options RouteOptions) (*RouteResponse, error) {
```

### Import Organization

Group imports in this order, separated by blank lines:

1. Standard library
2. External packages
3. Internal packages

```go
import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/rs/zerolog"
    "go.opentelemetry.io/otel"

    "github.com/breatheroute/breatheroute/internal/domain"
    "github.com/breatheroute/breatheroute/pkg/geo"
)
```

### Struct Field Alignment

Align struct tags for readability:

```go
type User struct {
    ID        uuid.UUID `json:"id"         db:"id"`
    Email     string    `json:"email"      db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

---

## Naming Conventions

### General Rules

- Use **MixedCaps** or **mixedCaps**, never underscores
- Acronyms should be all caps: `HTTPHandler`, `XMLParser`, `userID`
- Keep names short but descriptive
- Avoid stuttering: `user.User` → `user.Profile`

### Variables

```go
// Good: Short, clear names
var (
    ctx  context.Context
    err  error
    resp *http.Response
    buf  bytes.Buffer
)

// Good: Descriptive for complex types
var (
    routeCalculator  *RouteCalculator
    exposureScorer   ExposureScorer
    providerRegistry *ProviderRegistry
)

// Avoid: Overly abbreviated
var r *http.Response  // What is r?
var u *User           // Acceptable in small scope

// Avoid: Overly verbose
var httpResponseFromServer *http.Response
```

### Functions and Methods

```go
// Good: Verb + Noun for actions
func CreateUser(ctx context.Context, u User) error
func GetRouteOptions(origin, dest Coordinate) ([]Route, error)
func ValidateRequest(r *http.Request) error

// Good: Noun for getters (no Get prefix needed)
func (u *User) Email() string
func (r *Route) Distance() float64

// Good: Is/Has/Can for boolean returns
func (u *User) IsActive() bool
func (r *Route) HasTransit() bool
func (p *Permission) CanEdit() bool

// Avoid: Redundant prefixes
func GetUserEmail(u *User) string  // Just use u.Email()
```

### Interfaces

- Use `-er` suffix for single-method interfaces
- Name describes behavior, not implementation

```go
// Good: Behavior-focused names
type Reader interface {
    Read(p []byte) (n int, err error)
}

type RouteScorer interface {
    Score(ctx context.Context, route Route) (float64, error)
}

type ProviderHealthChecker interface {
    CheckHealth(ctx context.Context) error
}

// Avoid: Implementation-focused names
type RouteScoreCalculatorInterface interface { ... }  // Too verbose
type IRouteScorer interface { ... }                   // No I prefix
```

### Constants and Enums

```go
// Good: Typed constants with clear prefix
type RouteMode string

const (
    RouteModeWalk    RouteMode = "walk"
    RouteModeBike    RouteMode = "bike"
    RouteModeTransit RouteMode = "transit"
)

// Good: Iota for sequential values
type AlertSeverity int

const (
    AlertSeverityInfo AlertSeverity = iota
    AlertSeverityWarning
    AlertSeverityCritical
)

// Avoid: Untyped constants without context
const (
    Walk    = "walk"
    Bike    = "bike"
    Transit = "transit"
)
```

### Packages

```go
// Good: Short, lowercase, no underscores
package user
package routes
package provider
package geo

// Avoid
package userService      // No camelCase
package route_calculator // No underscores
package util            // Too generic
package common          // Too generic
package helpers         // Too generic
```

---

## Package Design

### Single Responsibility

Each package should have one clear purpose:

```
internal/
├── api/           # HTTP handlers and middleware
├── domain/        # Business logic and entities
├── repository/    # Database access
├── provider/      # External API integrations
│   ├── luchtmeetnet/
│   ├── ns/
│   └── weather/
└── service/       # Application services
```

### Avoid Circular Dependencies

```go
// Bad: Circular dependency
// package a imports package b
// package b imports package a

// Good: Use interfaces to break cycles
// package a defines interface
type UserRepository interface {
    Find(id string) (*User, error)
}

// package b implements interface
type PostgresUserRepository struct { ... }
```

### Package Initialization

- Avoid `init()` functions when possible
- If needed, keep them simple and fast
- Never do I/O or network calls in `init()`

```go
// Acceptable: Registering handlers
func init() {
    prometheus.MustRegister(requestCounter)
}

// Avoid: Complex initialization
func init() {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)  // Don't crash in init
    }
}
```

---

## Error Handling

### Error Creation

```go
// Use errors.New for static errors
var ErrUserNotFound = errors.New("user not found")
var ErrInvalidCoordinate = errors.New("invalid coordinate")

// Use fmt.Errorf with %w for wrapping
func GetUser(id string) (*User, error) {
    user, err := repo.Find(id)
    if err != nil {
        return nil, fmt.Errorf("getting user %s: %w", id, err)
    }
    return user, nil
}

// Use custom error types for rich errors
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}
```

### Error Handling Patterns

```go
// Always check errors
result, err := doSomething()
if err != nil {
    return err  // or handle appropriately
}

// Use errors.Is for sentinel errors
if errors.Is(err, ErrUserNotFound) {
    return nil, status.NotFound("user not found")
}

// Use errors.As for type assertions
var validationErr *ValidationError
if errors.As(err, &validationErr) {
    return nil, status.BadRequest(validationErr.Message)
}

// Don't ignore errors silently
_ = file.Close()  // Bad: ignoring error

// If intentionally ignoring, document why
// Close error ignored: we're in cleanup after successful write
_ = file.Close()

// Or handle it
if err := file.Close(); err != nil {
    log.Warn().Err(err).Msg("failed to close file")
}
```

### Error Messages

```go
// Good: Lowercase, no punctuation, add context
return fmt.Errorf("fetching route from provider: %w", err)
return fmt.Errorf("invalid latitude %f: must be between -90 and 90", lat)

// Avoid: Starting with capital, ending with punctuation
return fmt.Errorf("Failed to fetch route: %w.", err)
return errors.New("Something went wrong!")
```

### Don't Panic

```go
// Avoid: Panic in library code
func MustParse(s string) Config {
    c, err := Parse(s)
    if err != nil {
        panic(err)  // Don't do this
    }
    return c
}

// Good: Return error and let caller decide
func Parse(s string) (Config, error) {
    // ...
}

// Acceptable: Panic only for programmer errors in main
func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal().Err(err).Msg("failed to load config")
    }
}
```

---

## Testing

### Test File Organization

```
user/
├── user.go
├── user_test.go          # Unit tests
├── user_integration_test.go  # Integration tests (use build tags)
└── testdata/             # Test fixtures
    └── valid_user.json
```

### Test Naming

```go
// Pattern: Test<Function>_<Scenario>_<ExpectedBehavior>
func TestCreateUser_ValidInput_ReturnsUser(t *testing.T)
func TestCreateUser_DuplicateEmail_ReturnsError(t *testing.T)
func TestCalculateExposure_NoStations_ReturnsLowConfidence(t *testing.T)
```

### Table-Driven Tests

```go
func TestCalculateDistance(t *testing.T) {
    tests := []struct {
        name     string
        origin   Coordinate
        dest     Coordinate
        expected float64
    }{
        {
            name:     "same point",
            origin:   Coordinate{Lat: 52.0, Lon: 4.0},
            dest:     Coordinate{Lat: 52.0, Lon: 4.0},
            expected: 0,
        },
        {
            name:     "amsterdam to rotterdam",
            origin:   Coordinate{Lat: 52.3676, Lon: 4.9041},
            dest:     Coordinate{Lat: 51.9244, Lon: 4.4777},
            expected: 57000, // approximately 57km
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateDistance(tt.origin, tt.dest)
            if math.Abs(got-tt.expected) > 1000 { // 1km tolerance
                t.Errorf("CalculateDistance() = %v, want %v", got, tt.expected)
            }
        })
    }
}
```

### Use testify for Assertions (Optional)

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUserCreation(t *testing.T) {
    user, err := CreateUser(ctx, input)

    // require stops test on failure
    require.NoError(t, err)
    require.NotNil(t, user)

    // assert continues after failure
    assert.Equal(t, "test@example.com", user.Email)
    assert.True(t, user.IsActive())
}
```

### Mocking

Use interfaces for testability:

```go
// Define interface
type UserRepository interface {
    Find(ctx context.Context, id string) (*User, error)
    Create(ctx context.Context, user *User) error
}

// Production implementation
type PostgresUserRepository struct {
    db *sql.DB
}

// Test mock
type MockUserRepository struct {
    FindFunc   func(ctx context.Context, id string) (*User, error)
    CreateFunc func(ctx context.Context, user *User) error
}

func (m *MockUserRepository) Find(ctx context.Context, id string) (*User, error) {
    return m.FindFunc(ctx, id)
}

// In test
func TestUserService_GetUser(t *testing.T) {
    mockRepo := &MockUserRepository{
        FindFunc: func(ctx context.Context, id string) (*User, error) {
            return &User{ID: id, Email: "test@example.com"}, nil
        },
    }

    svc := NewUserService(mockRepo)
    user, err := svc.GetUser(ctx, "123")

    require.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
}
```

### Test Coverage

- Aim for 80%+ coverage on business logic
- Don't chase 100% - focus on critical paths
- Use coverage to find untested code, not as a metric

```bash
# Run tests with coverage
go test -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Integration Tests

Use build tags to separate integration tests:

```go
//go:build integration

package repository_test

func TestPostgresUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup real database connection
    db := setupTestDB(t)
    defer teardownTestDB(t, db)

    repo := NewPostgresUserRepository(db)
    // ... test with real database
}
```

Run with: `go test -tags=integration ./...`

---

## Concurrency

### Goroutine Lifecycle

Always ensure goroutines can be stopped:

```go
// Good: Goroutine with cancellation
func StartWorker(ctx context.Context) {
    go func() {
        ticker := time.NewTicker(time.Minute)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                doWork()
            }
        }
    }()
}

// Bad: Goroutine that can't be stopped
func StartWorker() {
    go func() {
        for {
            time.Sleep(time.Minute)
            doWork()
        }
    }()
}
```

### Channel Patterns

```go
// Close channels from sender side only
func producer(ctx context.Context) <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch)  // Sender closes
        for i := 0; ; i++ {
            select {
            case <-ctx.Done():
                return
            case ch <- i:
            }
        }
    }()
    return ch
}

// Use buffered channels to prevent blocking
results := make(chan Result, 10)

// Don't communicate by sharing memory; share memory by communicating
// Bad
var counter int
var mu sync.Mutex
go func() { mu.Lock(); counter++; mu.Unlock() }()

// Good
counterCh := make(chan int, 1)
counterCh <- 0
go func() {
    val := <-counterCh
    counterCh <- val + 1
}()
```

### Sync Primitives

```go
// Use sync.Once for one-time initialization
var (
    instance *Client
    once     sync.Once
)

func GetClient() *Client {
    once.Do(func() {
        instance = &Client{}
    })
    return instance
}

// Use sync.WaitGroup to wait for goroutines
func processItems(items []Item) {
    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(item Item) {
            defer wg.Done()
            process(item)
        }(item)
    }
    wg.Wait()
}

// Use errgroup for error handling with goroutines
import "golang.org/x/sync/errgroup"

func fetchAll(ctx context.Context, urls []string) error {
    g, ctx := errgroup.WithContext(ctx)
    for _, url := range urls {
        url := url  // Capture loop variable
        g.Go(func() error {
            return fetch(ctx, url)
        })
    }
    return g.Wait()
}
```

### Mutex Best Practices

```go
type SafeCounter struct {
    mu    sync.RWMutex  // Use RWMutex for read-heavy workloads
    count int
}

// Use defer for unlock to handle panics
func (c *SafeCounter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

// Use RLock for read operations
func (c *SafeCounter) Value() int {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.count
}

// Keep critical sections small
func (c *SafeCounter) ProcessAndIncrement(data []byte) error {
    // Process outside lock
    result, err := process(data)
    if err != nil {
        return err
    }

    // Only lock for the mutation
    c.mu.Lock()
    c.count += result
    c.mu.Unlock()

    return nil
}
```

---

## Documentation

### Package Documentation

```go
// Package routes provides route calculation and exposure scoring
// for commuter journeys in the Netherlands.
//
// The package supports multiple routing modes (walk, bike, transit)
// and calculates air quality exposure scores using data from
// Luchtmeetnet stations.
//
// Basic usage:
//
//     calculator := routes.NewCalculator(providers)
//     options, err := calculator.Calculate(ctx, origin, dest)
//     if err != nil {
//         return err
//     }
//     for _, opt := range options {
//         fmt.Printf("Route: %s, Exposure: %.2f\n", opt.Mode, opt.ExposureScore)
//     }
package routes
```

### Function Documentation

```go
// CalculateExposure computes the air quality exposure score for a route.
//
// The score is calculated using inverse-distance weighted interpolation
// from nearby Luchtmeetnet stations. The algorithm samples points along
// the route polyline at regular intervals and aggregates pollution values.
//
// The returned score ranges from 0.0 (best) to 100.0 (worst).
// A confidence level indicates data quality based on station coverage.
//
// Returns ErrNoStationsInRange if no stations are within 50km of any
// route point.
func CalculateExposure(
    ctx context.Context,
    route Route,
    stations []Station,
) (score float64, confidence Confidence, err error) {
    // ...
}
```

### Struct Documentation

```go
// RouteOption represents a single route alternative with its
// associated metadata and exposure scoring.
type RouteOption struct {
    // ID uniquely identifies this route option within a request.
    ID string

    // Mode indicates the transportation mode (walk, bike, transit).
    Mode RouteMode

    // Polyline is the encoded route geometry in Google Polyline format.
    Polyline string

    // DurationSeconds is the estimated travel time.
    DurationSeconds int

    // DistanceMeters is the total route distance.
    DistanceMeters int

    // ExposureScore ranges from 0.0 (best) to 100.0 (worst).
    ExposureScore float64

    // Confidence indicates data quality (low, medium, high).
    Confidence Confidence

    // Segments breaks down the route into logical parts with
    // per-segment exposure data.
    Segments []RouteSegment
}
```

---

## Security

### Input Validation

```go
// Validate all external input
func (h *Handler) CreateCommute(w http.ResponseWriter, r *http.Request) {
    var req CreateCommuteRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if err := req.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Proceed with validated input
}

func (r *CreateCommuteRequest) Validate() error {
    if r.Label == "" || len(r.Label) > 100 {
        return errors.New("label must be 1-100 characters")
    }
    if !isValidLatitude(r.Origin.Lat) || !isValidLongitude(r.Origin.Lon) {
        return errors.New("invalid origin coordinates")
    }
    return nil
}
```

### SQL Injection Prevention

```go
// Good: Parameterized queries
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
    query := `SELECT id, email, created_at FROM users WHERE email = $1`
    row := r.db.QueryRowContext(ctx, query, email)
    // ...
}

// Bad: String concatenation
query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)  // NEVER DO THIS
```

### Secrets Management

```go
// Never log secrets
log.Info().
    Str("user_id", userID).
    Str("api_key", apiKey).  // BAD: logging secret
    Msg("authenticated")

// Redact sensitive fields
log.Info().
    Str("user_id", userID).
    Str("api_key", redact(apiKey)).  // Good: redacted
    Msg("authenticated")

func redact(s string) string {
    if len(s) <= 8 {
        return "***"
    }
    return s[:4] + "***" + s[len(s)-4:]
}

// Use environment variables or secret manager
apiKey := os.Getenv("API_KEY")  // Good

// Never hardcode secrets
const apiKey = "sk-1234..."  // NEVER DO THIS
```

### HTTP Security Headers

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
}
```

---

## Performance

### Avoid Premature Optimization

1. Write clear, correct code first
2. Measure with benchmarks and profiling
3. Optimize only what matters

### Memory Allocation

```go
// Pre-allocate slices when size is known
users := make([]User, 0, len(userIDs))  // Known capacity
for _, id := range userIDs {
    users = append(users, fetchUser(id))
}

// Use sync.Pool for frequently allocated objects
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func process(data []byte) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()
    // Use buf...
}

// Avoid unnecessary string conversions
// Bad
for _, b := range []byte(s) { ... }

// Good
for i := 0; i < len(s); i++ {
    b := s[i]
    ...
}
```

### String Building

```go
// Good: strings.Builder for concatenation
var b strings.Builder
for _, s := range parts {
    b.WriteString(s)
}
result := b.String()

// Bad: String concatenation in loop
var result string
for _, s := range parts {
    result += s  // Creates new string each iteration
}
```

### Benchmarking

```go
func BenchmarkCalculateExposure(b *testing.B) {
    route := createTestRoute()
    stations := createTestStations()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        CalculateExposure(context.Background(), route, stations)
    }
}

// Run with: go test -bench=. -benchmem
```

### Context Timeouts

```go
// Always set timeouts for external calls
func (c *Client) FetchData(ctx context.Context) ([]byte, error) {
    ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    // ...
}
```

---

## Project Structure

### Standard Layout

```
breatheroute/
├── cmd/
│   ├── api/
│   │   └── main.go           # API entrypoint
│   └── worker/
│       └── main.go           # Worker entrypoint
├── internal/                  # Private application code
│   ├── api/                   # HTTP layer
│   │   ├── handler/           # Request handlers
│   │   ├── middleware/        # HTTP middleware
│   │   └── response/          # Response helpers
│   ├── domain/                # Business logic
│   │   ├── user/
│   │   ├── commute/
│   │   └── route/
│   ├── repository/            # Data access
│   │   ├── postgres/
│   │   └── redis/
│   ├── provider/              # External APIs
│   │   ├── luchtmeetnet/
│   │   ├── ns/
│   │   └── weather/
│   └── config/                # Configuration
├── pkg/                       # Public libraries
│   ├── geo/                   # Geometry utilities
│   ├── polyline/              # Polyline encoding
│   └── validate/              # Validation helpers
├── migrations/                # Database migrations
├── scripts/                   # Development scripts
├── docs/                      # Documentation
└── test/                      # Additional test utilities
    ├── fixtures/
    └── integration/
```

### Internal vs Pkg

- `internal/` - Private to this module, cannot be imported externally
- `pkg/` - Public libraries that could be used by other projects

---

## Dependencies

### Go Modules

```bash
# Initialize module
go mod init github.com/breatheroute/breatheroute

# Add dependency
go get github.com/go-chi/chi/v5

# Update dependencies
go get -u ./...

# Tidy (remove unused, add missing)
go mod tidy

# Vendor dependencies (optional)
go mod vendor
```

### Dependency Guidelines

1. **Minimize dependencies** - Each adds maintenance burden
2. **Prefer standard library** - It's well-tested and stable
3. **Vet dependencies** - Check maintenance status, security history
4. **Pin versions** - Use go.mod, consider go.sum in version control
5. **Review updates** - Don't blindly update, check changelogs

### Recommended Libraries

| Purpose | Library |
|---------|---------|
| HTTP Router | `github.com/go-chi/chi/v5` |
| Logging | `github.com/rs/zerolog` |
| Database | `github.com/jackc/pgx/v5` |
| Redis | `github.com/redis/go-redis/v9` |
| Testing | `github.com/stretchr/testify` |
| Validation | `github.com/go-playground/validator/v10` |
| Config | `github.com/spf13/viper` |
| Tracing | `go.opentelemetry.io/otel` |
| UUID | `github.com/google/uuid` |

---

## Code Review Checklist

Use this checklist when reviewing Go code:

### Correctness
- [ ] Does the code do what it claims?
- [ ] Are edge cases handled?
- [ ] Are errors checked and handled appropriately?
- [ ] Is there potential for nil pointer dereference?

### Design
- [ ] Is the code simple and readable?
- [ ] Are functions focused and reasonably sized?
- [ ] Are interfaces used appropriately?
- [ ] Is there unnecessary abstraction?

### Concurrency
- [ ] Are shared resources protected?
- [ ] Can goroutines be properly terminated?
- [ ] Are channels used correctly (closing, buffering)?
- [ ] Is there potential for deadlock or race conditions?

### Testing
- [ ] Are there adequate tests for new code?
- [ ] Do tests cover error cases?
- [ ] Are tests deterministic (no flakiness)?
- [ ] Is test code clean and maintainable?

### Security
- [ ] Is user input validated?
- [ ] Are queries parameterized?
- [ ] Are secrets handled properly?
- [ ] Are there potential injection vulnerabilities?

### Performance
- [ ] Are there unnecessary allocations?
- [ ] Are context timeouts set for I/O?
- [ ] Is logging appropriate (not excessive)?
- [ ] Are database queries efficient?

### Style
- [ ] Does code pass `gofmt` and `golangci-lint`?
- [ ] Are names clear and consistent?
- [ ] Is documentation adequate?
- [ ] Are magic numbers explained?

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [Practical Go](https://dave.cheney.net/practical-go/presentations/qcon-china.html)
