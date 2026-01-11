# go-utils

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/xraph/go-utils)](https://goreportcard.com/report/github.com/xraph/go-utils)
[![CI](https://github.com/xraph/go-utils/actions/workflows/ci.yml/badge.svg)](https://github.com/xraph/go-utils/actions/workflows/ci.yml)

A comprehensive collection of production-ready Go utilities for building robust applications with clean error handling, structured logging, dependency injection, HTTP request handling, validation, and metrics.

## Packages

### 📦 [errs](./errs) - Generic Error Handling

Interface-based error handling with full support for Go's standard error patterns, structured context, HTTP integration, and error chains.

**Key Features:**
- ✅ Interface-based design (ContextualError, CodedError, HTTPError, CausedError)
- ✅ Full `errors.Is` and `errors.As` compatibility
- ✅ Zero dependencies (standard library only)
- ✅ HTTP response mapping with status codes
- ✅ Structured error codes and metadata
- ✅ Error chain traversal and wrapping

```go
import "github.com/xraph/go-utils/errs"

// Create errors with context and metadata
err := errs.ErrNotFound("user").
    WithContext("user_id", "123").
    WithContext("query", "email=user@example.com")

// HTTP integration
if httpErr, ok := err.(errs.HTTPError); ok {
    statusCode := httpErr.StatusCode() // 404
    body := httpErr.ResponseBody()
}

// Check error types
if errs.IsNotFound(err) {
    // Handle not found
}
```

[**📖 Full Documentation →**](./errs/README.md)

---

### 📦 [log](./log) - Structured Logging

Production-grade structured logging built on uber/zap with beautiful console output, contextual fields, and performance optimization.

**Key Features:**
- ✅ Multiple logger implementations (Production, Development, Beautiful, Test)
- ✅ Structured logging with type-safe fields
- ✅ Colored console output for development
- ✅ JSON output for production
- ✅ Context-aware logging with request tracking
- ✅ Performance monitoring utilities
- ✅ Zero-allocation in hot paths

```go
import "github.com/xraph/go-utils/log"

// Beautiful console logger for development
logger := log.NewBeautifulLogger("myapp")

// Structured logging with fields
logger.Info("User logged in",
    log.String("user_id", "123"),
    log.String("ip", "192.168.1.1"),
    log.Duration("latency", time.Millisecond*45),
)

// Context-aware logging
ctx = log.WithRequestID(ctx, "req-abc-123")
contextLogger := logger.WithContext(ctx)
contextLogger.Info("Processing request") // Includes request_id automatically

// Production JSON logger
prodLogger := log.NewProductionLogger()
prodLogger.Error("Database connection failed",
    log.Error(err),
    log.String("database", "postgres"),
)
```

**Logger Types:**
- **Production** - JSON output, optimized for log aggregation
- **Development** - Human-readable console output
- **Beautiful** - Colored, emoji-enhanced output for CLI
- **Noop** - No-op logger for testing/benchmarking
- **Test** - Captures logs for test assertions

---

### 📦 [di](./di) - Dependency Injection

Lightweight dependency injection container with lifecycle management, scopes, and service health checks.

**Key Features:**
- ✅ Interface-based container design
- ✅ Service lifecycle management (Register, Start, Stop)
- ✅ Scoped services for request-level dependencies
- ✅ Dependency graph resolution with cycle detection
- ✅ Health check support for services
- ✅ ResolveReady for dependency initialization
- ✅ Zero reflection overhead after registration

```go
import "github.com/xraph/go-utils/di"

// Create container
container := di.NewContainer()

// Register services
container.Register("database", func(c di.Container) (any, error) {
    return &Database{}, nil
}, di.Singleton())

container.Register("userService", func(c di.Container) (any, error) {
    db, _ := c.Resolve("database")
    return &UserService{DB: db.(*Database)}, nil
}, di.Singleton())

// Start all services
ctx := context.Background()
container.Start(ctx)

// Resolve services
svc, _ := container.Resolve("userService")
userSvc := svc.(*UserService)

// Create request scope
scope := container.BeginScope()
defer scope.End()
```

---

### 📦 [http](./http) - HTTP Context & Request Handling

Feature-rich HTTP context for request handling with parameter binding, validation, and response helpers.

**Key Features:**
- ✅ Type-safe request binding (path, query, header, body)
- ✅ Integrated validation with go-playground/validator
- ✅ Custom validation tags (format, minLength, pattern, etc.)
- ✅ Fluent response API (JSON, XML, HTML, Stream)
- ✅ Cookie and session management
- ✅ DI container integration per request
- ✅ Sensitive data masking for logs
- ✅ Request/response metrics

```go
import "github.com/xraph/go-utils/http"

type UserRequest struct {
    ID    string `path:"id" validate:"required,uuid"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=0,lte=120"`
}

func handler(ctx http.Context) error {
    var req UserRequest
    
    // Bind and validate request
    if err := ctx.BindRequest(&req); err != nil {
        return ctx.Status(400).JSON(err)
    }
    
    // Fluent response
    return ctx.Status(200).JSON(map[string]any{
        "message": "User created",
        "user": req,
    })
}
```

**Validation Features:**
- Hybrid validation: go-playground/validator + custom tags
- Custom validators: `format`, `minLength`, `maxLength`, `pattern`, `enum`, `minimum`, `maximum`, `multipleOf`
- Detailed error messages with field names and error codes
- Support for nested structs and embedded fields

---

### 📦 [val](./val) - Validation Utilities

Validation error handling and field validation helpers.

**Key Features:**
- ✅ ValidationError with structured field errors
- ✅ HTTP error interface implementation (422 status)
- ✅ Field requirement detection (required, optional, omitempty)
- ✅ Format validation helpers (email, URL, UUID, ISO8601)
- ✅ Field name extraction from struct tags
- ✅ Type checking utilities

```go
import "github.com/xraph/go-utils/val"

// Create validation error
errors := &val.ValidationError{}

// Add field errors with codes
errors.AddWithCode("email", "must be a valid email", val.ErrCodeInvalidFormat, "not-an-email")
errors.AddWithCode("age", "must be at least 18", val.ErrCodeMinValue, 15)

// Check if field is required
field, _ := reflect.TypeOf(User{}).FieldByName("Email")
if val.IsFieldRequired(field) {
    // Field is required
}

// Validate formats
if !val.IsValidEmail("user@example.com") {
    // Invalid email
}

if !val.IsValidUUID("123e4567-e89b-12d3-a456-426614174000") {
    // Invalid UUID
}

// HTTP integration
return ctx.Status(errors.StatusCode()).JSON(errors.ResponseBody())
```

**Error Codes:**
- `ErrCodeRequired` - Field is required
- `ErrCodeInvalidType` - Invalid field type
- `ErrCodeInvalidFormat` - Invalid format
- `ErrCodeMinLength`, `ErrCodeMaxLength` - Length constraints
- `ErrCodeMinValue`, `ErrCodeMaxValue` - Numeric constraints
- `ErrCodePattern` - Pattern mismatch
- `ErrCodeEnum` - Invalid enum value

---

### 📦 [metrics](./metrics) - Application Metrics

Metrics collection and health monitoring for services.

**Key Features:**
- ✅ Multiple metric types (Counter, Gauge, Histogram, Timer)
- ✅ Export formats (Prometheus, JSON, InfluxDB, StatsD)
- ✅ System and runtime metrics collection
- ✅ HTTP metrics middleware integration
- ✅ Health check management
- ✅ Service health aggregation
- ✅ Configurable collection intervals

```go
import "github.com/xraph/go-utils/metrics"

// Create metrics collector
m := metrics.NewMetrics(metrics.MetricsConfig{
    Enabled: true,
    Features: metrics.MetricsFeatures{
        SystemMetrics:  true,
        RuntimeMetrics: true,
        HTTPMetrics:    true,
    },
})

// Record metrics
m.Increment("requests_total", map[string]string{
    "method": "GET",
    "path": "/api/users",
})

m.Gauge("active_connections", 42, nil)
m.Histogram("request_duration_ms", 123.45, nil)

// Health management
health := metrics.NewHealthManager()
health.RegisterCheck("database", func(ctx context.Context) error {
    return db.Ping(ctx)
})

status := health.Check(ctx)
if status.Status != "healthy" {
    // Handle unhealthy state
}
```

---

## Installation

```bash
go get github.com/xraph/go-utils
```

Or install specific packages:

```bash
go get github.com/xraph/go-utils/errs    # Error handling
go get github.com/xraph/go-utils/log     # Structured logging
go get github.com/xraph/go-utils/di      # Dependency injection
go get github.com/xraph/go-utils/http    # HTTP context & validation
go get github.com/xraph/go-utils/val     # Validation utilities
go get github.com/xraph/go-utils/metrics # Metrics & health checks
```

## Requirements

- Go 1.22 or higher (uses integer range loops)
- **errs**: No external dependencies (standard library only)
- **log**: `go.uber.org/zap` for structured logging
- **di**: No external dependencies
- **http**: `go-playground/validator` for validation
- **val**: `google/uuid` for UUID validation
- **metrics**: No external dependencies

## Quick Start

### Error Handling Example

```go
package main

import (
    "fmt"
    "github.com/xraph/go-utils/errs"
)

func GetUser(id string) error {
    if id == "" {
        return errs.BadRequest("user ID is required").
            WithContext("field", "id")
    }
    
    // Simulate user not found
    return errs.ErrNotFound("user").
        WithContext("user_id", id)
}

func main() {
    err := GetUser("")
    
    // Check specific error type
    if errs.IsBadRequest(err) {
        fmt.Println("Invalid input:", err)
    }
    
    // Extract HTTP status
    if httpErr, ok := err.(errs.HTTPError); ok {
        fmt.Printf("Status: %d\n", httpErr.StatusCode())
    }
}
```

### Logging Example

```go
package main

import (
    "context"
    "time"
    "github.com/xraph/go-utils/log"
)

func main() {
    // Create a beautiful logger for development
    logger := log.NewBeautifulLogger("myapp")
    
    // Basic logging
    logger.Info("Application started",
        log.String("version", "1.0.0"),
        log.String("environment", "production"),
    )
    
    // Context-aware logging
    ctx := context.Background()
    ctx = log.WithRequestID(ctx, "req-123")
    ctx = log.WithUserID(ctx, "user-456")
    
    reqLogger := logger.WithContext(ctx)
    reqLogger.Info("Processing request")
    
    // Performance monitoring
    pm := log.NewPerformanceMonitor(logger, "database_query")
    pm.WithField(log.String("query", "SELECT * FROM users"))
    
    // Simulate work
    time.Sleep(100 * time.Millisecond)
    pm.Finish()
}
```

### HTTP Request Handling Example

```go
package main

import (
    "github.com/xraph/go-utils/http"
)

type CreateUserRequest struct {
    Name  string `json:"name" validate:"required,min=3,max=50"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"required,gte=18,lte=120"`
    Role  string `json:"role" enum:"admin,user,guest"`
}

func CreateUserHandler(ctx http.Context) error {
    var req CreateUserRequest
    
    // Bind and validate in one step
    if err := ctx.BindRequest(&req); err != nil {
        return ctx.Status(400).JSON(err)
    }
    
    // Process request...
    user := createUser(req)
    
    // Return response
    return ctx.Status(201).JSON(map[string]any{
        "message": "User created successfully",
        "user": user,
    })
}
```

### Dependency Injection Example

```go
package main

import (
    "context"
    "github.com/xraph/go-utils/di"
)

type Database struct{}
func (d *Database) Start(ctx context.Context) error { return nil }
func (d *Database) Stop(ctx context.Context) error { return nil }

type UserService struct {
    DB *Database
}

func main() {
    container := di.NewContainer()
    
    // Register database
    container.Register("database", func(c di.Container) (any, error) {
        return &Database{}, nil
    }, di.Singleton())
    
    // Register user service with dependency
    container.Register("userService", func(c di.Container) (any, error) {
        db, _ := c.Resolve("database")
        return &UserService{DB: db.(*Database)}, nil
    }, di.Singleton())
    
    // Start all services
    ctx := context.Background()
    container.Start(ctx)
    defer container.Stop(ctx)
    
    // Use services
    svc, _ := container.Resolve("userService")
    userSvc := svc.(*UserService)
    _ = userSvc
}
```

## Package Philosophy

### Design Principles

1. **Interface-First Design** - All packages use interfaces for extensibility
2. **Zero Dependencies** - Core packages avoid external dependencies when possible
3. **Standard Library Compatible** - Full compatibility with Go's stdlib patterns
4. **Type Safety** - Leverage Go's type system for compile-time safety
5. **Performance** - Zero-allocation hot paths where possible
6. **Production Ready** - Battle-tested patterns and comprehensive testing

### Why This Library?

- **Consistent API** - Unified approach across error handling and logging
- **Rich Context** - Attach structured data to errors and logs
- **HTTP Ready** - Built-in HTTP status codes and response generation
- **Developer Experience** - Beautiful console output and helpful error messages
- **Testing Support** - First-class testing utilities included
- **No Lock-in** - Standard interfaces allow easy migration

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Run linter:

```bash
make lint
```

## Project Structure

```
go-utils/
├── errs/              # Error handling package
│   ├── errors.go      # Core error types and interfaces
│   ├── errors_test.go # Comprehensive tests
│   ├── examples_test.go # Usage examples
│   └── README.md      # Package documentation
├── log/               # Logging package
│   ├── logger.go      # Main logger implementation
│   ├── beautiful_logger.go # Beautiful console logger
│   ├── fields.go      # Type-safe field constructors
│   ├── interfaces.go  # Logger interfaces
│   ├── colors.go      # ANSI color codes
│   ├── perf.go        # Performance monitoring
│   ├── structured.go  # Structured logging utilities
│   ├── testing.go     # Test logger
│   └── *_test.go      # Tests
├── di/                # Dependency injection
│   ├── di.go          # Container interfaces
│   ├── dep.go         # Dependency resolution
│   ├── service.go     # Service lifecycle
│   └── di_opts.go     # Registration options
├── http/              # HTTP context & validation
│   ├── context.go     # HTTP context implementation
│   ├── binder.go      # Request binding
│   ├── validator.go   # Validation with go-playground
│   ├── sensitive.go   # Sensitive data masking
│   └── session.go     # Session management
├── val/               # Validation utilities
│   ├── validation.go  # ValidationError type
│   └── helpers.go     # Validation helpers
├── metrics/           # Metrics & health
│   ├── metrics.go     # Metrics collection
│   └── health.go      # Health checks
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Examples

For detailed examples, see:
- [errs package examples](./errs/examples_test.go)
- [log package examples](./log/logger_test.go)
- [di package examples](./di/di_test.go)
- [http package examples](./http/binder_test.go)
- [val package examples](./val/helpers_test.go)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development

1. Clone the repository
2. Make your changes
3. Run tests: `go test ./...`
4. Run linter: `make lint`
5. Submit a PR

## License

MIT License - see [LICENSE](LICENSE) for details

## Related Projects

- [uber-go/zap](https://github.com/uber-go/zap) - Blazing fast, structured logging
- [go-playground/validator](https://github.com/go-playground/validator) - Go struct and field validation
- [google/uuid](https://github.com/google/uuid) - UUID generation and parsing

## Features by Package

| Feature | errs | log | di | http | val | metrics |
|---------|------|-----|----|----|-----|---------|
| Zero Dependencies | ✅ | ❌ | ✅ | ❌ | ❌ | ✅ |
| HTTP Integration | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| Context Support | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| Structured Data | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| Testing Utilities | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Production Ready | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

## Roadmap

- [x] Error handling with context
- [x] Structured logging
- [x] Dependency injection
- [x] HTTP request binding and validation
- [x] Validation utilities
- [x] Metrics and health checks
- [ ] Add tracing support
- [ ] Add retry utilities
- [ ] Add rate limiting
- [ ] Add circuit breaker
- [ ] Add caching utilities

---

**Built with ❤️ for production Go applications**
