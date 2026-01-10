# errs - Generic Error Handling Package

A production-ready, extensible error handling package for Go with interface-based design and full support for Go's standard error patterns.

## Features

✅ **Interface-Based Design** - Extensible interfaces for custom error types  
✅ **Error Code Constants** - Generic, type-safe error codes  
✅ **`errs.Is` Support** - Pattern matching for error types  
✅ **`errs.As` Support** - Type extraction from error chains  
✅ **Structured Errors** - Rich context with timestamps and metadata  
✅ **Error Wrapping** - Full chain traversal support  
✅ **HTTP Integration** - Optional HTTP response mapping  
✅ **Sentinel Errors** - Predefined error instances for comparison  
✅ **Zero Dependencies** - Uses only Go standard library

## Core Interfaces

The package provides extensible interfaces that allow you to create custom error types:

```go
// ContextualError adds key-value context to errors
type ContextualError interface {
    error
    WithContext(key string, value any) ContextualError
    GetContext() map[string]any
}

// CodedError provides structured error codes
type CodedError interface {
    error
    GetCode() string
}

// HTTPError enables HTTP response generation
// This is the primary interface for HTTP-compatible errors
type HTTPError interface {
    error
    StatusCode() int
    ResponseBody() any
}

// CausedError provides explicit error chain access
type CausedError interface {
    error
    Unwrap() error
    Cause() error
}
```

## Error Code Constants

Generic error codes that can be extended in consuming packages:

```go
const (
    CodeInternal         = "INTERNAL_ERROR"
    CodeValidation       = "VALIDATION_ERROR"
    CodeNotFound         = "NOT_FOUND"
    CodeAlreadyExists    = "ALREADY_EXISTS"
    CodeInvalidInput     = "INVALID_INPUT"
    CodeTimeout          = "TIMEOUT"
    CodeCancelled        = "CANCELLED"
    CodeUnavailable      = "UNAVAILABLE"
    CodePermissionDenied = "PERMISSION_DENIED"
    CodeUnauthorized     = "UNAUTHORIZED"
    CodeConflict         = "CONFLICT"
)
```

## Quick Start

### Creating Errors

```go
// Using constructor functions (recommended)
err := errs.ErrNotFound("user")
err := errs.ErrValidation("invalid email format", fmt.Errorf("missing @ symbol"))
err := errs.ErrTimeout("database query", 5*time.Second)

// With additional context (fluent API)
err := errs.ErrNotFound("cache_entry").
    WithContext("key", "user:123").
    WithContext("namespace", "sessions")

// Create custom errors
err := errs.NewError("CUSTOM_CODE", "something went wrong", nil)
```

### Checking Error Types

```go
// Using sentinel errors (most concise)
if errs.Is(err, errs.ErrNotFoundSentinel) {
    // Handle not found error
}

// Using helper functions (clearest intent)
if errs.IsNotFound(err) {
    // Handle not found error
}

// Using error codes (most flexible)
var structuredErr *errs.Error
if errs.As(err, &structuredErr) {
    switch structuredErr.Code {
    case errs.CodeNotFound:
        // Handle not found
    case errs.CodeTimeout:
        // Handle timeout
    }
}

// Using interfaces (most extensible)
if codedErr, ok := err.(errs.CodedError); ok {
    log.Printf("ErrNotFound("user")
wrappedErr := errs.ErrInternal("database query failed", baseErr)

// Check if any error in the chain matches
if errs.Is(wrappedErr, errs.ErrNotFoundSentinel) {
    // This will match even though it's wrapped
}

// Extract specific error type from chain
var structuredErr *errs.Error
if errs.As(wrappedErr, &structuredErr) {
    log.Printf("Code: %s, Context: %v", structuredErr.Code, structuredErr.Ctx)
}

// Access context from errors
if contextualErr, ok := err.(errs.ContextualError); ok {
    ctx := contextualErr.GetContext()
    if userID, exists := ctx["user_id"]; exists {
        log.Printf("Error related to user: %v", userID)
    }
if errs.Is(wrappedErr, errs.ErrServiceNotFoundSentinel) {
    // This will match even though it's wrapped
}

// Extract specific error type from chain
var serviceErr *errs.ServiceError
if errs.As(wrappedErr, &serviceErr) {
    log.Printf("Service: %s, Operation: %s", serviceErr.Service, serviceErr.Operation)
}
```
Error

Structured error implementing ContextualError, CodedError, HTTPResponder, and CausedError interfaces.

```go
type Error struct {
    Code      string         // Error code constant
    Message   string         // Human-readable message
    Err       error          // Underlying error (can be nil)
    Timestamp time.Time      // When the error occurred
    Ctx       map[string]any // Additional context (key-value pairs)
}
```

**Implements:**
- `error` - Standard Go error interface
- `ContextualError` - Add context with `WithContext()`, retrieve with `GetContext()`
- `CodedError` - Get structured code with `GetCode()`
- `HTTPError` - Returns 500 status and JSON body
- `CausedError` - Access cause with `Cause()` or `Unwrap()`

### HTTPError

HTTP-compatible error interface. The package provides `httpError` as a concrete implementation.

```go
// HTTPError is the interface
type HTTPError interface {
    error
    StatusCode() int
    ResponseBody() any
}

// httpError is the concrete implementation (unexported)
// Use constructor functions like BadRequest(), NotFound(), etc.
```

**Constructors:**
- `NewHTTPError(code, message)` - Custom status code
- `BadRequest(message)` - 400 Bad Request
- `Unauthorized(message)` - 401 Unauthorized  
- `Forbidden(message)` - 403 Forbidden
- `NotFound(message)` - 404 Not Found
- `InternalError(err)` - 500 Internal Server Error

**Features:**
- All constructors return the `HTTPError` interface
- Both `Error` and `httpError` types implement `HTTPError`
- Use `GetHTTPStatusCode()` to extract status codes from error chains Service   string // Service name
    Operation string // Operation that failed
    Err       error  // Underlying error
}ors.Is`:

- `ErrValidationSentinel` - Validation errors
- `ErrNotFoundSentinel` - Resource not found
- `ErrAlreadyExistsSentinel` - Resource already exists
- `ErrInvalidInputSentinel` - Invalid input provided
- `ErrTimeoutSentinel` - Operation timeout
- `ErrCancelledSentinel` - Operation cancelled
- `ErrInternalSentinel` - Internal errors

## Helper Functions

Convenience functions for common error checks:

```go
IsValidation(err)    bool
IsNotFound(err)      bool
IsAlreadyExists(err) bool
IsTimeout(err)       bool
IsCancelled(err)     bool
GetHTTPStatusCode(err)el`
- `ErrLifecycleErrorSentinel`
- `ErrContextCancelledSentinel`
- `ErrTimeoutErrorSentinel`
- `ErrConfigErrorSentinel`

## Helper Functions

Convenience functions for common error checks:

```go
IsServiceNotFound(err)      bool
IsServiceAlreadyExists(err) bool
IsCircularDependency(err)   bool
IsValidationError(err)      bool
IsContextCancelled(err)     bool
IsTimeout(err)              bool
GetHTTPStatusCode(err)      int
```Extending the Package

### Creating Custom Error Types

Implement the provided interfaces to create domain-specific errors:

```go
// Custom error for your domain
type DatabaseError struct {
    Query     string
    Table     string
    Operation string
    Err       error
}

func (e *DatabaseError) Error() string {
    return fmt.Sprintf("%s on %s failed: %v", e.Operation, e.Table, e.Err)
}

func (e *DatabaseError) Unwrap() error { return e.Err }
func (e *DatabaseError) Cause() error  { return e.Err }

// Implement CodedError for structured error handling
func (e *DatabaseError) GetCode() string {
    return "DATABASE_ERROR"
}

// Optionally implement HTTPResponder
func (e *DatabaseError) StatusCode() int {
    return http.StatusInternalServerError
}

func (e *DatabaseError) ResponseBody() any {
    return map[string]any{
        "error":     e.Error(),
        "table":     e.Table,
        "operation": e.Operation,
    }
}
```

### Adding Custom Error Codes

Define your own error codes in consuming packages:

```go
package myapp

const (
    CodeDatabaseError    = "DATABASE_ERROR"
    CodeCacheError       = "CACHE_ERROR"
    CodeExternalAPIError = "EXTERNAL_API_ERROR"
)

// Create constructors using errs.NewError
func ErrDatabaseError(operation string, cause error) *errs.Error {
    return errs.NewError(CodeDatabaseError, 
        fmt.Sprintf("database %s failed", operation), 
        cause).WithContext("operation", operation).(*errs.Error)
}
```

## Best Practices

1. **Use interfaces** - Program against interfaces for maximum flexibility
2. **Use constructor functions** - They ensure consistent error codes and structure
3. **Add context liberally** - Use `WithContext()` for debugging information
4. **Check error chains** - Always use `Is()` and `As()` for type checking
5. **Log at boundaries** - Don't log at every layer, only at system boundaries
6. **Preserve causes** - Always wrap underlying errors, don't discard them
7. **Extend with custom codes** - Add domain-specific error codes in your packages
8. **Implement interfaces** - Create custom error types that implement the provided interfaces

## Examples

See `examples_test.go` for comprehensive usage examples including:
- Error creation and wrapping
- Context enrichment
- Error chain traversal
- Sentinel error matching
- HTTP error handling
- Custom error type implementation

## Testing

Comprehensive test coverage including:
- Interface implementation verification
- Error matching with `Is()`
- Type extraction with `As()`
- Error chain traversal
- HTTP status code extraction
- Context management

1. **Use constructor functions** - They ensure consistent error codes
2. **Add context** - Use `WithContext()` for debugging information
3. **Check error chains** - Always use `Is()` and `As()` for type checking
4. **Log at boundaries** - Don't log at every layer, only at boundaries
5. **Preserve causes** - Always wrap underlying errors, don't discard them

## Testing

Comprehensive test coverage including:
- Error matching with `Is()`
- Type extraction with `As()`
- Error chain traversal
- HTTP status code extraction
- Context preservation

Run tests:
```bash
go test -v ./v2/internal/errors/...
```

## Examples

See `examples_test.go` for detailed usage examples.

