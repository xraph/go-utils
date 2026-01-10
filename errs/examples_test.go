package errs

import (
	"fmt"
)

// Example_errorCodes demonstrates using error code constants.
func Example_errorCodes() {
	// Create errors using constructors (they use the constants internally)
	err := ErrNotFound("user")

	// You can access the code constant for custom logic
	var structuredErr *Error
	if As(err, &structuredErr) {
		switch structuredErr.Code {
		case CodeNotFound:
			fmt.Println("Resource not found")
		case CodeAlreadyExists:
			fmt.Println("Resource already exists")
		case CodeValidation:
			fmt.Println("Validation error")
		}
	}

	// Using sentinel errors for comparison
	if Is(err, ErrNotFoundSentinel) {
		fmt.Println("Matched using sentinel")
	}

	// Using helper functions
	if IsNotFound(err) {
		fmt.Println("Matched using helper")
	}

	// Output:
	// Resource not found
	// Matched using sentinel
	// Matched using helper
}

// Example_errorConstants demonstrates the available error code constants.
func Example_errorConstants() {
	// All available generic error code constants:
	codes := []string{
		CodeInternal,         // "INTERNAL_ERROR"
		CodeValidation,       // "VALIDATION_ERROR"
		CodeNotFound,         // "NOT_FOUND"
		CodeAlreadyExists,    // "ALREADY_EXISTS"
		CodeInvalidInput,     // "INVALID_INPUT"
		CodeTimeout,          // "TIMEOUT"
		CodeCancelled,        // "CANCELLED"
		CodeUnavailable,      // "UNAVAILABLE"
		CodePermissionDenied, // "PERMISSION_DENIED"
		CodeUnauthorized,     // "UNAUTHORIZED"
		CodeConflict,         // "CONFLICT"
	}

	for _, code := range codes {
		fmt.Printf("Code: %s\n", code)
	}

	// Output:
	// Code: INTERNAL_ERROR
	// Code: VALIDATION_ERROR
	// Code: NOT_FOUND
	// Code: ALREADY_EXISTS
	// Code: INVALID_INPUT
	// Code: TIMEOUT
	// Code: CANCELLED
	// Code: UNAVAILABLE
	// Code: PERMISSION_DENIED
	// Code: UNAUTHORIZED
	// Code: CONFLICT
}

// Example_structuredErrorWithConstants shows creating custom errors with constants.
func Example_structuredErrorWithConstants() {
	// When creating custom errors, use NewError with constants
	customErr := NewError(CodeValidation, "custom validation failed", nil)

	// Add context using the fluent API
	customErr.WithContext("field", "email").WithContext("rule", "format")

	// Check error type
	if Is(customErr, ErrValidationSentinel) {
		fmt.Println("This is a validation error")
	}

	// Output:
	// This is a validation error
}

// Example_errorMatching demonstrates various error matching patterns.
func Example_errorMatching() {
	// Create a wrapped error chain
	baseErr := ErrNotFound("user")
	wrappedErr := ErrInternal("database query failed", baseErr)

	// Match by code using Is with sentinel
	if Is(wrappedErr, ErrNotFoundSentinel) {
		fmt.Println("Found NOT_FOUND in chain")
	}

	// Match by code using Is with a new error instance
	if Is(wrappedErr, &Error{Code: CodeInternal}) {
		fmt.Println("Found INTERNAL_ERROR in chain")
	}

	// Extract specific error type using As
	var internalErr *Error
	if As(wrappedErr, &internalErr) {
		if internalErr.Code == CodeInternal {
			fmt.Println("Top-level error is INTERNAL_ERROR")
		}
	}

	// Output:
	// Found NOT_FOUND in chain
	// Found INTERNAL_ERROR in chain
	// Top-level error is INTERNAL_ERROR
}

// Example_interfaceUsage demonstrates using the package interfaces.
func Example_interfaceUsage() {
	// Create an error
	err := ErrNotFound("user").WithContext("user_id", "123")

	// Use CodedError interface
	if codedErr, ok := err.(CodedError); ok {
		fmt.Printf("Error code: %s\n", codedErr.GetCode())
	}

	// Use ContextualError interface
	if contextualErr, ok := err.(ContextualError); ok {
		ctx := contextualErr.GetContext()
		fmt.Printf("Context: %v\n", ctx["user_id"])
	}

	// Use HTTPError interface
	if httpErr, ok := err.(HTTPError); ok {
		fmt.Printf("HTTP Status: %d\n", httpErr.StatusCode())
	}

	// Output:
	// Error code: NOT_FOUND
	// Context: 123
	// HTTP Status: 500
}

// Example_customErrorType shows how to extend the package with custom types.
func Example_customErrorType() {
	// Define your own error codes
	const CodeDatabaseError = "DATABASE_ERROR"

	// Create a constructor
	ErrDatabaseError := func(operation string, cause error) *Error {
		return NewError(CodeDatabaseError,
			fmt.Sprintf("database %s failed", operation),
			cause).WithContext("operation", operation).(*Error)
	}

	// Use your custom error
	err := ErrDatabaseError("query", nil)
	fmt.Printf("Code: %s\n", err.Code)
	fmt.Printf("Context: %v\n", err.Ctx["operation"])

	// Output:
	// Code: DATABASE_ERROR
	// Context: query
}
