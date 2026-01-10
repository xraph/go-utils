# go-utils

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/xraph/go-utils)](https://goreportcard.com/report/github.com/xraph/go-utils)
[![CI](https://github.com/xraph/go-utils/actions/workflows/ci.yml/badge.svg)](https://github.com/xraph/go-utils/actions/workflows/ci.yml)

A collection of production-ready Go utilities for building robust applications with clean error handling and structured logging.

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

## Installation

```bash
go get github.com/xraph/go-utils
```

Or install specific packages:

```bash
go get github.com/xraph/go-utils/errs
go get github.com/xraph/go-utils/log
```

## Requirements

- Go 1.25 or higher
- No external dependencies for `errs` package
- `go.uber.org/zap` for `log` package

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
    
    // Error logging with fields
    logger.Error("Failed to connect",
        log.String("host", "db.example.com"),
        log.Int("port", 5432),
        log.Error(err),
    )
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
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Examples

For detailed examples, see:
- [errs package examples](./errs/examples_test.go)
- [log package examples](./log/logger_test.go)

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
- [pkg/errors](https://github.com/pkg/errors) - Simple error handling primitives

## Roadmap

- [ ] Add metrics integration
- [ ] Add tracing support
- [ ] Add retry utilities
- [ ] Add validation helpers
- [ ] Add HTTP middleware utilities

---

**Built with ❤️ for production Go applications**
