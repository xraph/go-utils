package errs

import (
	"errors"
	"net/http"
	"testing"
)

// TestInterfaceImplementations verifies that types implement expected interfaces.
func TestInterfaceImplementations(t *testing.T) {
	t.Run("Error implements ContextualError", func(t *testing.T) {
		var _ ContextualError = (*Error)(nil)
	})

	t.Run("Error implements CodedError", func(t *testing.T) {
		var _ CodedError = (*Error)(nil)
	})

	t.Run("Error implements HTTPError", func(t *testing.T) {
		var _ HTTPError = (*Error)(nil)
	})

	t.Run("httpError implements HTTPError", func(t *testing.T) {
		var _ HTTPError = (*httpError)(nil)
	})
}

// TestContextualError tests the ContextualError interface methods.
func TestContextualError(t *testing.T) {
	err := ErrNotFound("user")

	// Test WithContext
	contextual := err.WithContext("user_id", "123")
	if contextual == nil {
		t.Fatal("WithContext returned nil")
	}

	// Test GetContext
	ctx := contextual.GetContext()
	if ctx == nil {
		t.Fatal("GetContext returned nil")
	}

	if ctx["user_id"] != "123" {
		t.Errorf("context value = %v, want '123'", ctx["user_id"])
	}

	// Test chaining
	contextual.WithContext("request_id", "abc")

	ctx = contextual.GetContext()
	if ctx["request_id"] != "abc" {
		t.Errorf("chained context value = %v, want 'abc'", ctx["request_id"])
	}
}

// TestCodedError tests the CodedError interface methods.
func TestCodedError(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		wantCode string
	}{
		{
			name:     "not found error",
			err:      ErrNotFound("user"),
			wantCode: CodeNotFound,
		},
		{
			name:     "validation error",
			err:      ErrValidation("invalid input", nil),
			wantCode: CodeValidation,
		},
		{
			name:     "timeout error",
			err:      ErrTimeout("operation", 5000),
			wantCode: CodeTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.GetCode(); got != tt.wantCode {
				t.Errorf("GetCode() = %v, want %v", got, tt.wantCode)
			}
		})
	}
}

// TestCausedError tests the CausedError interface methods.
func TestCausedError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := ErrInternal("operation failed", innerErr)

	// Test Cause method
	if cause := err.Cause(); !errors.Is(cause, innerErr) {
		t.Errorf("Cause() = %v, want %v", cause, innerErr)
	}

	// Test Unwrap method
	if unwrapped := err.Unwrap(); !errors.Is(unwrapped, innerErr) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

// TestHTTPError tests the HTTPError interface methods.
func TestHTTPError(t *testing.T) {
	t.Run("Error implements HTTPError", func(t *testing.T) {
		err := ErrNotFound("user").WithContext("user_id", "123").(*Error)

		if statusCode := err.StatusCode(); statusCode != http.StatusInternalServerError {
			t.Errorf("StatusCode() = %d, want %d", statusCode, http.StatusInternalServerError)
		}

		body := err.ResponseBody()
		if body == nil {
			t.Fatal("ResponseBody() returned nil")
		}

		bodyMap, ok := body.(map[string]any)
		if !ok {
			t.Fatal("ResponseBody() did not return map[string]any")
		}

		if bodyMap["code"] != CodeNotFound {
			t.Errorf("response code = %v, want %v", bodyMap["code"], CodeNotFound)
		}
	})

	t.Run("httpError implements HTTPError", func(t *testing.T) {
		err := BadRequest("invalid input")

		if statusCode := err.StatusCode(); statusCode != http.StatusBadRequest {
			t.Errorf("StatusCode() = %d, want %d", statusCode, http.StatusBadRequest)
		}

		body := err.ResponseBody()
		if body == nil {
			t.Fatal("ResponseBody() returned nil")
		}
	})
}

// TestErrorIs tests the Is implementation for Error type.
func TestErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same error code matches",
			err:    ErrNotFound("user"),
			target: ErrNotFoundSentinel,
			want:   true,
		},
		{
			name:   "different error code does not match",
			err:    ErrNotFound("user"),
			target: ErrAlreadyExistsSentinel,
			want:   false,
		},
		{
			name:   "wrapped error matches",
			err:    ErrInternal("operation failed", ErrNotFound("user")),
			target: ErrNotFoundSentinel,
			want:   true,
		},
		{
			name:   "nil target does not match",
			err:    ErrNotFound("user"),
			target: nil,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHTTPErrorIs tests the Is implementation for httpError.
func TestHTTPErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "same status code matches",
			err:    BadRequest("invalid input"),
			target: &httpError{Code: http.StatusBadRequest},
			want:   true,
		},
		{
			name:   "different status code does not match",
			err:    BadRequest("invalid input"),
			target: &httpError{Code: http.StatusNotFound},
			want:   false,
		},
		{
			name:   "wrapped http error matches",
			err:    InternalError(BadRequest("test").(error)),
			target: &httpError{Code: http.StatusBadRequest},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Is(tt.err, tt.target); got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorAs tests the As wrapper function.
func TestErrorAs(t *testing.T) {
	t.Run("extract Error", func(t *testing.T) {
		err := ErrNotFound("user")

		var structuredErr *Error
		if !As(err, &structuredErr) {
			t.Error("As() failed to extract Error")
		}

		if structuredErr.Code != CodeNotFound {
			t.Errorf("extracted error code = %s, want %s", structuredErr.Code, CodeNotFound)
		}
	})

	t.Run("extract HTTPError interface", func(t *testing.T) {
		err := BadRequest("invalid")

		var httpErr HTTPError
		if !As(err, &httpErr) {
			t.Error("As() failed to extract HTTPError interface")
		}

		if httpErr.StatusCode() != http.StatusBadRequest {
			t.Errorf("extracted status code = %d, want %d", httpErr.StatusCode(), http.StatusBadRequest)
		}
	})

	t.Run("extract from wrapped error", func(t *testing.T) {
		innerErr := BadRequest("test")
		wrappedErr := ErrInternal("operation failed", innerErr.(error))

		// When using As with HTTPError interface, it matches the first type
		// in the chain that implements HTTPError - which is the outer Error
		var httpErr HTTPError
		if !As(wrappedErr, &httpErr) {
			t.Error("As() failed to extract HTTPError from wrapped error")
		}

		// The outer Error returns 500, not the inner 400
		// To get the specific status code, use GetHTTPStatusCode
		if httpErr.StatusCode() != http.StatusInternalServerError {
			t.Errorf("extracted status code = %d, want %d", httpErr.StatusCode(), http.StatusInternalServerError)
		}

		// GetHTTPStatusCode will find the httpError in the chain
		if statusCode := GetHTTPStatusCode(wrappedErr); statusCode != http.StatusBadRequest {
			t.Errorf("GetHTTPStatusCode() = %d, want %d", statusCode, http.StatusBadRequest)
		}
	})
}

// TestHelperFunctions tests the convenience helper functions.
func TestHelperFunctions(t *testing.T) {
	t.Run("IsNotFound", func(t *testing.T) {
		err := ErrNotFound("user")
		if !IsNotFound(err) {
			t.Error("IsNotFound() failed to identify not found error")
		}
	})

	t.Run("IsValidation", func(t *testing.T) {
		err := ErrValidation("invalid email format", errors.New("missing @ symbol"))
		if !IsValidation(err) {
			t.Error("IsValidation() failed to identify validation error")
		}
	})

	t.Run("IsTimeout", func(t *testing.T) {
		err := ErrTimeout("database query", 5000)
		if !IsTimeout(err) {
			t.Error("IsTimeout() failed to identify timeout error")
		}
	})
}

// TestGetHTTPStatusCode tests the status code extraction helper.
func TestGetHTTPStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "extract from HTTPError",
			err:  BadRequest("test"),
			want: http.StatusBadRequest,
		},
		{
			name: "extract from wrapped HTTPError",
			err:  ErrInternal("operation failed", Unauthorized("not allowed")),
			want: http.StatusUnauthorized,
		},
		{
			name: "default to 500 for non-HTTP error",
			err:  ErrNotFound("user"),
			want: http.StatusInternalServerError,
		},
		{
			name: "default to 500 for nil",
			err:  nil,
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHTTPStatusCode(tt.err); got != tt.want {
				t.Errorf("GetHTTPStatusCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestUnwrap tests the Unwrap wrapper function.
func TestUnwrap(t *testing.T) {
	innerErr := errors.New("inner error")
	wrappedErr := ErrInternal("operation failed", innerErr)

	unwrapped := Unwrap(wrappedErr)
	if !errors.Is(unwrapped, innerErr) {
		t.Errorf("Unwrap() returned wrong error: got %v, want %v", unwrapped, innerErr)
	}
}

// TestJoin tests the Join wrapper function.
func TestJoin(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	joined := Join(err1, err2, err3)
	if joined == nil {
		t.Fatal("Join() returned nil")
	}

	// Check that all errors are in the chain
	if !errors.Is(joined, err1) {
		t.Error("joined error does not contain err1")
	}

	if !errors.Is(joined, err2) {
		t.Error("joined error does not contain err2")
	}

	if !errors.Is(joined, err3) {
		t.Error("joined error does not contain err3")
	}
}

// TestWithContext tests the WithContext method.
func TestWithContext(t *testing.T) {
	err := ErrNotFound("user").
		WithContext("user_id", "123").
		WithContext("request_id", "abc").(*Error)

	if err.Ctx["user_id"] != "123" {
		t.Error("context user_id not set correctly")
	}

	if err.Ctx["request_id"] != "abc" {
		t.Error("context request_id not set correctly")
	}
}

// Example usage demonstrating the Is functionality.
func ExampleIs() {
	// Create an error
	err := ErrNotFound("user")

	// Check using Is with sentinel error
	if Is(err, ErrNotFoundSentinel) {
		// Handle not found
	}

	// Or use the convenience helper
	if IsNotFound(err) {
		// Handle not found
	}
}

// Example showing error unwrapping.
func ExampleAs() {
	// Create a wrapped error
	innerErr := BadRequest("invalid input")
	wrappedErr := ErrInternal("operation failed", innerErr.(error))

	// Extract the HTTPError interface from the chain
	var httpErr HTTPError
	if As(wrappedErr, &httpErr) {
		// Use the extracted error
		statusCode := httpErr.StatusCode()
		_ = statusCode
	}
}
