package shared

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ValidationFieldError represents a single field validation error.
type ValidationFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
	Code    string `json:"code,omitempty"`
}

// ValidationError is a collection of validation errors that implements error and HTTP error interfaces.
type ValidationError struct {
	Errors []ValidationFieldError `json:"errors"`
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}

	return strings.Join(messages, "; ")
}

// Add adds a validation error.
func (ve *ValidationError) Add(field, message string, value any) {
	ve.Errors = append(ve.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// AddWithCode adds a validation error with a code.
func (ve *ValidationError) AddWithCode(field, message, code string, value any) {
	ve.Errors = append(ve.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Value:   value,
		Code:    code,
	})
}

// HasErrors returns true if there are validation errors.
func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

// Count returns the number of validation errors.
func (ve *ValidationError) Count() int {
	return len(ve.Errors)
}

// ToJSON converts validation errors to JSON bytes.
func (ve *ValidationError) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"error":            "Validation failed",
		"validationErrors": ve.Errors,
	})
}

// NewValidationError creates a new ValidationError instance.
func NewValidationError() *ValidationError {
	return &ValidationError{
		Errors: make([]ValidationFieldError, 0),
	}
}

// StatusCode returns 422 Unprocessable Entity for validation errors (implements HTTPError interface).
func (ve *ValidationError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// ResponseBody returns the response body (implements HTTPError interface).
func (ve *ValidationError) ResponseBody() any {
	return map[string]any{
		"error":            "Validation failed",
		"code":             http.StatusUnprocessableEntity,
		"validationErrors": ve.Errors,
	}
}

// Headers returns custom headers for the HTTP response.
func (ve *ValidationError) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// HTTPError returns the HTTP status code and response body.
func (ve *ValidationError) HTTPError() (int, any) {
	return ve.StatusCode(), ve.ResponseBody()
}

// Is checks if the target error is a ValidationError.
func (ve *ValidationError) Is(target error) bool {
	_, ok := target.(*ValidationError)

	return ok
}

// ValidationErrorResponse is the HTTP response for validation errors.
type ValidationErrorResponse struct {
	Error            string                 `json:"error"`
	Code             int                    `json:"code"`
	ValidationErrors []ValidationFieldError `json:"validationErrors"`
}

// NewValidationErrorResponse creates a new validation error response.
func NewValidationErrorResponse(errors *ValidationError) *ValidationErrorResponse {
	return &ValidationErrorResponse{
		Error:            "Validation failed",
		Code:             http.StatusUnprocessableEntity,
		ValidationErrors: errors.Errors,
	}
}

// Common validation error codes.
const (
	ErrCodeRequired      = "REQUIRED"
	ErrCodeInvalidType   = "INVALID_TYPE"
	ErrCodeInvalidFormat = "INVALID_FORMAT"
	ErrCodeMinLength     = "MIN_LENGTH"
	ErrCodeMaxLength     = "MAX_LENGTH"
	ErrCodeMinValue      = "MIN_VALUE"
	ErrCodeMaxValue      = "MAX_VALUE"
	ErrCodePattern       = "PATTERN"
	ErrCodeEnum          = "ENUM"
	ErrCodeMinItems      = "MIN_ITEMS"
	ErrCodeMaxItems      = "MAX_ITEMS"
	ErrCodeUniqueItems   = "UNIQUE_ITEMS"
)
