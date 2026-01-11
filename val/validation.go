package val

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	// ValidationFailedMessage is the default error message for validation failures.
	ValidationFailedMessage = "Validation failed"
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
	if ve == nil || len(ve.Errors) == 0 {
		return ValidationFailedMessage
	}

	messages := make([]string, 0, len(ve.Errors))
	for _, err := range ve.Errors {
		if err.Field != "" {
			messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
		} else {
			messages = append(messages, err.Message)
		}
	}

	return strings.Join(messages, "; ")
}

// Add adds a validation error.
func (ve *ValidationError) Add(field, message string, value any) {
	if ve == nil {
		return
	}

	ve.Errors = append(ve.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// AddWithCode adds a validation error with a code.
func (ve *ValidationError) AddWithCode(field, message, code string, value any) {
	if ve == nil {
		return
	}

	ve.Errors = append(ve.Errors, ValidationFieldError{
		Field:   field,
		Message: message,
		Value:   value,
		Code:    code,
	})
}

// HasErrors returns true if there are validation errors.
func (ve *ValidationError) HasErrors() bool {
	return ve != nil && len(ve.Errors) > 0
}

// Count returns the number of validation errors.
func (ve *ValidationError) Count() int {
	if ve == nil {
		return 0
	}

	return len(ve.Errors)
}

// MarshalJSON implements json.Marshaler for custom JSON serialization.
func (ve *ValidationError) MarshalJSON() ([]byte, error) {
	if ve == nil {
		return json.Marshal(map[string]any{
			"error":            ValidationFailedMessage,
			"validationErrors": []ValidationFieldError{},
		})
	}

	return json.Marshal(map[string]any{
		"error":            ValidationFailedMessage,
		"validationErrors": ve.Errors,
	})
}

// NewValidationError creates a new ValidationError instance.
func NewValidationError() *ValidationError {
	return &ValidationError{}
}

// StatusCode returns 422 Unprocessable Entity for validation errors (implements HTTPError interface).
func (ve *ValidationError) StatusCode() int {
	return http.StatusUnprocessableEntity
}

// Unwrap returns nil since ValidationError doesn't wrap another error.
func (ve *ValidationError) Unwrap() error {
	return nil
}

// ResponseBody returns the response body (implements HTTPError interface).
func (ve *ValidationError) ResponseBody() any {
	errors := []ValidationFieldError{}
	if ve != nil && ve.Errors != nil {
		errors = ve.Errors
	}

	return map[string]any{
		"error":            ValidationFailedMessage,
		"code":             http.StatusUnprocessableEntity,
		"validationErrors": errors,
	}
}

// Headers returns custom headers for the HTTP response.
func (ve *ValidationError) Headers() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

// As attempts to convert the target error to *ValidationError.
func (ve *ValidationError) As(target any) bool {
	if t, ok := target.(**ValidationError); ok {
		*t = ve

		return true
	}

	return false
}

// GetFieldErrors returns all errors for a specific field.
func (ve *ValidationError) GetFieldErrors(field string) []ValidationFieldError {
	if ve == nil {
		return nil
	}

	var fieldErrors []ValidationFieldError

	for _, err := range ve.Errors {
		if err.Field == field {
			fieldErrors = append(fieldErrors, err)
		}
	}

	return fieldErrors
}

// HasFieldError checks if a specific field has validation errors.
func (ve *ValidationError) HasFieldError(field string) bool {
	return len(ve.GetFieldErrors(field)) > 0
}

// Merge combines errors from another ValidationError.
func (ve *ValidationError) Merge(other *ValidationError) {
	if ve == nil || other == nil {
		return
	}

	ve.Errors = append(ve.Errors, other.Errors...)
}

// ValidationErrorResponse is the HTTP response for validation errors.
type ValidationErrorResponse struct {
	Error            string                 `json:"error"`
	Code             int                    `json:"code"`
	ValidationErrors []ValidationFieldError `json:"validationErrors"`
}

// NewValidationErrorResponse creates a new validation error response.
func NewValidationErrorResponse(ve *ValidationError) *ValidationErrorResponse {
	validationErrors := []ValidationFieldError{}
	if ve != nil && ve.Errors != nil {
		validationErrors = ve.Errors
	}

	return &ValidationErrorResponse{
		Error:            ValidationFailedMessage,
		Code:             http.StatusUnprocessableEntity,
		ValidationErrors: validationErrors,
	}
}

// IsValidationError checks if an error is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError

	return errors.As(err, &ve)
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
