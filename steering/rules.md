# Go Microservice Development Rules

You are a Go expert. Write code that follows these team conventions.
Reply and write code comments in Chinese.

## Top Priority

- Go version: Go 1.26+ for all projects (services, apps, and public libraries).
- Logging: Initialize at entry point with gox/log, use log/slog in business code (see Logging section)
- JSON struct tags: snake_case (`json:"user_id"`)
- Protobuf fields: snake_case
- All external calls must have timeout (internal 10s, external 30s)

## Naming

- Package: lowercase, short, meaningful, no underscores
- Local vars: short names (ctx, err, cfg, buf, req, resp)
- Package vars: full names (DefaultTimeout, MaxRetryCount)
- Abbreviations stay uppercase: ID, URL, HTTP, JSON, XML
- No type prefixes: `userID` not `intUserID`

## Error Handling (MUST)

- Check and handle every error return
- Package errors: `Err` prefix, defined in `errors.go`
- Format: `"package: description"` (lowercase, no trailing punctuation)
- Wrap with `fmt.Errorf("pkg: action %q: %w", arg, err)`
- Compare with `errors.Is` / `errors.As`

## Logging (MUST)

- Entry point initialization: Use `github.com/chinayin/gox/log` to create logger and set global slog handler
  - Microservices: in `internal/bootstrap/` or `internal/adapter/bootstrap/`
  - CLI tools: in `cmd/<app>/main.go` assembly code
- Business code: Use standard library `log/slog` directly, since global handler is set by gox/log
- Contextual logging: `slog.With("key", value)` to create child logger
- Forbidden: directly importing zap/logrus or other third-party logging libraries

## Code Quality (MUST)

- Cyclomatic complexity <= 15
- Cognitive complexity <= 20
- Nesting depth <= 4
- After completing a feature or fixing multiple files, run `golangci-lint run --fix ./...` before presenting results. Fix auto-fixable issues silently; for warnings requiring judgment, analyze and fix properly (no blind nolint).

### Pattern: Early return to separate branches

```go
// Bad - dry-run and real logic mixed in one loop
func apply(items []Item, dryRun bool) {
    for _, item := range items {
        if dryRun {
            // dry-run logic...
            continue
        }
        // real logic... (nesting +1)
        result := doApply(item)
        switch result.Action {
            // nesting +1 again...
        }
    }
}

// Good - split into independent paths
func apply(items []Item, dryRun bool) {
    if dryRun {
        printDryRun(items)
        return
    }
    results := doApplyAll(items)
    printSummary(results)
}
```

### Pattern: Extract print/format helpers from loops

```go
// Bad - switch nested inside loop, function bloats
for _, item := range items {
    result := apply(item)
    switch result.Action {
    case "created":
        fmt.Printf(...)
        created++
    case "updated":
        fmt.Printf(...)
        for _, c := range result.Changes { ... }
        updated++
    }
}

// Good - extract display logic
for _, item := range items {
    result := apply(item)
    printResult(item, result)
    countByAction(result, &created, &updated)
}
```

### Pattern: Merge similar switch cases

```go
// Bad
case "updated":
    updated++
case "applied":
    updated++

// Good
case "updated", "applied":
    updated++
```

## Package File Organization (MUST)

Standard file layout:

```
doc.go      - package documentation
const.go    - constants shared across files (referenced by multiple files in same package)
errors.go   - package-level errors (Err prefix, with comments)
types.go    - shared types first, then constants
<impl>.go   - implementation, file-local constants declared at top of file
```

### Constant Placement Rules

- Used by multiple files in same package -> centralize in `const.go`
- Used only in current file -> declare at top of that file, keep colocation for decoupling
- AI judges constant scope and decides placement accordingly

### Code Examples

```go
// doc.go - package documentation
// Package storage provides multi-cloud storage abstraction.
package storage

// errors.go - package-level errors (only definitions, each with comment)
package storage

import "errors"

var (
    // ErrNotFound indicates resource does not exist
    ErrNotFound = errors.New("storage: resource not found")
)

// types.go - shared types first, then constants
package storage

type Client interface {
    Download(ctx context.Context, path string, w io.Writer) error
}

const (
    TypeS3  = "s3"
    TypeOSS = "oss"
)

// s3.go - implementation with file-local constants at top
package storage

const (
    s3DownloadTimeout = 30 * time.Minute
    s3UploadTimeout   = 30 * time.Minute
)

func (c *S3Client) Download(ctx context.Context, path string) error {
    obj, err := c.getObject(ctx, path)
    if err != nil {
        return fmt.Errorf("storage: get object %q: %w", path, err)
    }
    return nil
}
```

## Functional Options

Use when 3+ optional parameters:

```go
type Option func(*Client)

func WithTimeout(d time.Duration) Option {
    return func(c *Client) { if d > 0 { c.timeout = d } }
}

func NewClient(endpoint string, opts ...Option) *Client {
    c := &Client{endpoint: endpoint, timeout: 30 * time.Second}
    for _, opt := range opts { opt(c) }
    return c
}
```

## Constructor Conventions (MUST)

- Naming: `New<Type>` or `New<Type>From<Source>`
- Return pointer: `func NewClient() *Client`
- Return error if may fail: `func New() (*Client, error)`
- Use Functional Options for complex configuration

## Struct Design (SHOULD)

- Field order: embedded -> exported -> unexported
- Group related fields with blank lines
- Never store Context in structs
- Receiver name: type initial (`c *Client`, `s *Server`)
- Pointer receiver if mutating or has sync.Mutex

```go
type Client struct {
    *BaseClient  // embedded

    endpoint string  // core config
    bucket   string

    timeout time.Duration  // connection config
    mu      sync.RWMutex   // internal state
}
```

## Interface Design (SHOULD)

- Small and focused (single responsibility)
- Name with `-er` suffix: Reader, Writer, Closer
- Define at consumer side, not implementor
- Prefer stdlib interfaces: io.Reader, io.Writer

```go
// Good - small and focused
type Downloader interface {
    Download(ctx context.Context, path string, w io.Writer) error
}

// Bad - too large
type Storage interface {
    Download(...) error
    Upload(...) error
    Delete(...) error
    List(...) ([]string, error)
    Copy(...) error
    Move(...) error
}
```

## Generics (SHOULD)

- Use for generic data structures: Stack, Queue, Set
- Use for collection operations: Map, Filter, Reduce
- Avoid over-genericizing business logic

## Concurrency (MUST)

- Context as first parameter, always
- Explicit goroutine lifecycle with Context
- Sender closes channel
- Use WaitGroup or errgroup
- Limit goroutine count

## Standard Library

- Use `slices` package: Sort, Index, Contains
- Use `maps` package: Clone, Copy, Equal
- Use `cmp` package: Compare, Or

## Testing (MUST)

- Name: `Test<Component>_<Method>_<Scenario>`
- Table-driven with `[]struct` + `t.Run`
- Each test independent, no shared state
- Use testify (assert + require)
- AAA pattern: Arrange, Act, Assert
- Use gomock for mocking, `defer ctrl.Finish()` to verify expectations

### Coverage Requirements

- Utility functions: 100%
- Domain models: > 90%
- Service layer: > 80%

## Protobuf (MUST)

- Fields: snake_case (never camelCase)
- Enum values: UPPER_SNAKE_CASE
- Zero value: `STATUS_UNSPECIFIED = 0`
- Never delete/rename fields or change field numbers
- Use `reserved` to preserve deleted field numbers

## gRPC Client (MUST)

- Set timeout via `context.WithTimeout`
- Check status codes with `status.Code(err)`
- Convert gRPC status to business error codes
- Retry only idempotent ops, max 3 times, exponential backoff
- POST operations use idempotency key
- Use circuit breaker to prevent cascade failures (SHOULD)
- Propagate OpenTelemetry trace context (SHOULD)

## API Design (MUST for HTTP services)

- RESTful style, resources use plural nouns: /users, /orders
- Version prefix: /v1, /v2
- JSON field names: snake_case
- Unified response: `{code, message, data, metadata}`
- Error response: `{code, message, errors, request_id}`
- Timestamps: ISO 8601 UTC

## Microservice Governance (MUST)

- Service ports: HTTP `:8000` (PORT), gRPC `:9000` (GRPC_PORT)
- Health checks: `/health/live` (liveness), `/health/ready` (readiness)
- Tracing: propagate Trace ID across services
- Rate limiting and circuit breaker for all inbound endpoints

## Project Structure

CLI tools use standard layout:

```
cmd/<app>/main.go     - entry point, assembly only
internal/<app>/       - core business logic
  const.go            - shared constants across files
  errors.go           - package errors
  types.go            - shared types
config/               - config files (config.local.yaml gitignored)
docs/                 - project documentation (UPPER_SNAKE_CASE.md)
bin/                  - build output (gitignored)
```

Business microservices use DDD layout (organize by aggregate root, not tech layer):

```
internal/
  <aggregate>/        - domain aggregate
    domain.go         - entity, value object, repository interface
    service.go        - domain service (business logic)
    repository.go     - repository interface (defined in domain layer)
    events.go         - domain events
  adapter/            - ports and adapters
    grpc/             - gRPC handlers
    http/             - HTTP handlers
    repository/       - repository implementations (GORM/Redis)
  bootstrap/          - app init and DI
```

For CLI-specific conventions (framework, subcommands, flags, version injection, local config repository), see the `cli` steering file.

## Common Pitfalls

### Design for zero-value usability

```go
// Good - zero value is usable
type Buffer struct {
    buf []byte
}

func (b *Buffer) Write(p []byte) (int, error) {
    b.buf = append(b.buf, p...)  // nil slice can append
    return len(p), nil
}

var buf Buffer  // usable without initialization
buf.Write([]byte("hello"))
```

### Avoid package-level mutable state

```go
// Bad - package-level mutable state
var globalClient *http.Client

// Good - dependency injection
type Service struct {
    client *http.Client
}

func NewService(client *http.Client) *Service {
    return &Service{client: client}
}
```

### Context passing

- Bad: store Context in structs
- Good: pass as first function parameter

## gox Library

Use github.com/chinayin/gox for:
log, config, discovery, trace, metrics, middleware, transport, utils

## Scaffold Files (MUST)

Every Go project root must have: `.editorconfig`, `.gitignore`, `Makefile`.
Refer to the `scaffold` steering file for standard templates.

## Documentation (MUST)

- All docs in `docs/` directory
- Filename: UPPER_SNAKE_CASE.md
- Must contain: title, overview, main content

### Naming Conventions

- Architecture: ARCHITECTURE.md
- API: API_REFERENCE.md, API_GUIDE.md
- Testing: TESTING_*.md, E2E_TEST_*.md
- Migration: *_MIGRATION_GUIDE.md, *_MIGRATION_STATUS.md
- Integration: *_INTEGRATION_GUIDE.md, *_INTEGRATION_SUMMARY.md
- Status: *_STATUS.md, *_SUMMARY.md
