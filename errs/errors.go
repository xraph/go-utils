package errs

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// =============================================================================
// CORE INTERFACES
// =============================================================================

// ContextualError represents an error that can carry additional context.
// Implementations can add key-value pairs to enrich error information.
type ContextualError interface {
	error
	WithContext(key string, value any) ContextualError
	GetContext() map[string]any
}

// CodedError represents an error with a structured error code.
// This allows for programmatic error handling based on error type.
type CodedError interface {
	error
	GetCode() string
}

// HTTPError is an interface for errors that can be represented as HTTP responses.
// Any error implementing this interface can be automatically converted to an HTTP response.
// Both the Error type and httpError type implement this interface.
type HTTPError interface {
	error
	StatusCode() int
	ResponseBody() any
}

// CausedError represents an error that wraps an underlying cause.
// This provides explicit access to the error chain.
type CausedError interface {
	error
	Unwrap() error
	Cause() error
}

// =============================================================================
// ERROR CODES
// =============================================================================

// Error code constants for structured errors.
// These are generic codes that can be extended in consuming packages.
const (
	// CodeInternal represents an internal error.
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

// =============================================================================
// STRUCTURED ERROR
// =============================================================================

// Error represents a structured error with context.
// It implements error, ContextualError, CodedError, HTTPError, and CausedError interfaces.
type Error struct {
	Code      string
	Message   string
	Err       error // Underlying error (wrapped)
	Timestamp time.Time
	Ctx       map[string]any // Additional context as key-value pairs
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}

	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Is implements errors.Is interface for ForgeError
// Compares by error code, allowing matching against sentinel errors.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	// Match if codes are the same (and not empty)
	return e.Code != "" && e.Code == t.Code
}

// WithContext adds context to the error and returns the error for chaining.
// Implements ContextualError interface.
func (e *Error) WithContext(key string, value any) ContextualError {
	if e.Ctx == nil {
		e.Ctx = make(map[string]any)
	}

	e.Ctx[key] = value

	return e
}

// GetContext returns the error's context map.
// Implements ContextualError interface.
func (e *Error) GetContext() map[string]any {
	return e.Ctx
}

// GetCode returns the error's code.
// Implements CodedError interface.
func (e *Error) GetCode() string {
	return e.Code
}

// Cause returns the underlying error (same as Unwrap).
// Implements CausedError interface.
func (e *Error) Cause() error {
	return e.Err
}

// StatusCode returns 500 by default (implements HTTPError).
func (e *Error) StatusCode() int {
	return http.StatusInternalServerError
}

// ResponseBody returns the response body (implements HTTPError).
func (e *Error) ResponseBody() any {
	body := map[string]any{
		"error":     e.Message,
		"code":      e.Code,
		"timestamp": e.Timestamp,
	}

	// Include cause if present
	if e.Err != nil {
		body["cause"] = e.Err.Error()
	}

	// Include context if present and not empty
	if len(e.Ctx) > 0 {
		body["context"] = e.Ctx
	}

	return body
}

// =============================================================================
// ERROR CONSTRUCTORS
// =============================================================================

// NewError creates a new structured error with the given code, message, and optional cause.
func NewError(code, message string, cause error) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Err:       cause,
		Timestamp: time.Now(),
		Ctx:       make(map[string]any),
	}
}

// ErrValidation creates a validation error.
func ErrValidation(message string, cause error) *Error {
	return NewError(CodeValidation, message, cause)
}

// ErrNotFound creates a not found error.
func ErrNotFound(resource string) *Error {
	return NewError(CodeNotFound, resource+" not found", nil).
		WithContext("resource", resource).(*Error)
}

// ErrAlreadyExists creates an already exists error.
func ErrAlreadyExists(resource string) *Error {
	return NewError(CodeAlreadyExists, resource+" already exists", nil).
		WithContext("resource", resource).(*Error)
}

// ErrInvalidInput creates an invalid input error.
func ErrInvalidInput(field string, reason string) *Error {
	return NewError(CodeInvalidInput, fmt.Sprintf("invalid input for field '%s': %s", field, reason), nil).
		WithContext("field", field).(*Error)
}

// ErrTimeout creates a timeout error.
func ErrTimeout(operation string, duration time.Duration) *Error {
	return NewError(CodeTimeout, fmt.Sprintf("%s timed out after %s", operation, duration), nil).
		WithContext("operation", operation).
		WithContext("duration", duration.String()).(*Error)
}

// ErrCancelled creates a cancelled operation error.
func ErrCancelled(operation string) *Error {
	return NewError(CodeCancelled, operation+" was cancelled", nil).
		WithContext("operation", operation).(*Error)
}

// ErrInternal creates an internal error.
func ErrInternal(message string, cause error) *Error {
	return NewError(CodeInternal, message, cause)
}

// =============================================================================
// VALIDATION ERROR
// =============================================================================

// Severity represents the severity of a validation issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ValidationError represents a validation error.
type ValidationError struct {
	Key        string   `json:"key"`
	Value      any      `json:"value,omitempty"`
	Rule       string   `json:"rule"`
	Message    string   `json:"message"`
	Severity   Severity `json:"severity"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// =============================================================================
// HTTP ERRORS
// =============================================================================

// httpError is a concrete implementation of HTTPError for simple HTTP errors.
// This is unexported; users should use the constructor functions that return the HTTPError interface.
type httpError struct {
	Code    int
	Message string
	Err     error
}

func (e *httpError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return http.StatusText(e.Code)
}

func (e *httpError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is interface for httpError
// Compares by HTTP status code.
func (e *httpError) Is(target error) bool {
	t, ok := target.(*httpError)
	if !ok {
		return false
	}

	return e.Code == t.Code
}

// StatusCode returns the HTTP status code (implements HTTPError interface).
func (e *httpError) StatusCode() int {
	return e.Code
}

// ResponseBody returns the response body (implements HTTPError interface).
func (e *httpError) ResponseBody() any {
	body := map[string]any{
		"error": e.Message,
		"code":  e.Code,
	}

	// Include underlying error details if present
	if e.Err != nil {
		body["details"] = e.Err.Error()
	}

	return body
}

// HTTP error constructors.
// These return the HTTPError interface, allowing any type implementing the interface to be used.

// NewHTTPError creates a new HTTP error with the given status code and message.
func NewHTTPError(code int, message string) HTTPError {
	return &httpError{Code: code, Message: message}
}

// BadRequest creates a 400 Bad Request error.
func BadRequest(message string) HTTPError {
	return &httpError{Code: http.StatusBadRequest, Message: message}
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(message string) HTTPError {
	return &httpError{Code: http.StatusUnauthorized, Message: message}
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(message string) HTTPError {
	return &httpError{Code: http.StatusForbidden, Message: message}
}

// NotFound creates a 404 Not Found error.
func NotFound(message string) HTTPError {
	return &httpError{Code: http.StatusNotFound, Message: message}
}

// InternalError creates a 500 Internal Server Error.
func InternalError(err error) HTTPError {
	return &httpError{Code: http.StatusInternalServerError, Err: err}
}

// =============================================================================
// STANDARD ERRORS PACKAGE INTEGRATION
// =============================================================================

// Is reports whether any error in err's chain matches target.
// This is a convenience wrapper around errors.Is from the standard library.
//
// Example:
//
//	err := ErrNotFound("user")
//	if Is(err, ErrNotFoundSentinel) {
//	    // handle not found error
//	}
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target, and if so,
// sets target to that error value and returns true. Otherwise, it returns false.
// This is a convenience wrapper around errors.As from the standard library.
//
// Example:
//
//	var httpErr HTTPError
//	if As(err, &httpErr) {
//	    // handle HTTP error with httpErr.StatusCode()
//	}
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error. Otherwise, Unwrap returns nil.
// This is a convenience wrapper around errors.Unwrap from the standard library.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// New returns an error that formats as the given text.
// This is a convenience wrapper around errors.New from the standard library.
func New(text string) error {
	return errors.New(text)
}

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// This is a convenience wrapper around errors.Join from the standard library.
// Requires Go 1.20+.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// =============================================================================
// SENTINEL ERRORS (for use with Is)
// =============================================================================

// Sentinel errors that can be used with errors.Is comparisons.
// These allow for idiomatic error checking without string comparisons.
var (
	ErrValidationSentinel    = &Error{Code: CodeValidation}
	ErrNotFoundSentinel      = &Error{Code: CodeNotFound}
	ErrAlreadyExistsSentinel = &Error{Code: CodeAlreadyExists}
	ErrInvalidInputSentinel  = &Error{Code: CodeInvalidInput}
	ErrTimeoutSentinel       = &Error{Code: CodeTimeout}
	ErrCancelledSentinel     = &Error{Code: CodeCancelled}
	ErrInternalSentinel      = &Error{Code: CodeInternal}
)

// =============================================================================
// ERROR HELPERS
// =============================================================================

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	return Is(err, ErrValidationSentinel)
}

// IsNotFound checks if the error is a not found error.
func IsNotFound(err error) bool {
	return Is(err, ErrNotFoundSentinel)
}

// IsAlreadyExists checks if the error is an already exists error.
func IsAlreadyExists(err error) bool {
	return Is(err, ErrAlreadyExistsSentinel)
}

// IsTimeout checks if the error is a timeout error.
func IsTimeout(err error) bool {
	return Is(err, ErrTimeoutSentinel)
}

// IsCancelled checks if the error is a cancelled operation error.
func IsCancelled(err error) bool {
	return Is(err, ErrCancelledSentinel)
}

// GetHTTPStatusCode extracts HTTP status code from error, returns 500 if not found.
// This works with any error implementing the HTTPError interface.
// It checks for httpError in the chain first (for specific status codes),
// then falls back to checking any HTTPError implementation.
func GetHTTPStatusCode(err error) int {
	// First check if there's a httpError in the chain (has specific status code)
	var concreteHTTP *httpError
	if As(err, &concreteHTTP) {
		return concreteHTTP.Code
	}

	// Then check if the error itself implements HTTPError
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode()
	}

	return http.StatusInternalServerError
}
