package val

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name string
		ve   *ValidationError
		want string
	}{
		{
			name: "nil validation error",
			ve:   nil,
			want: ValidationFailedMessage,
		},
		{
			name: "empty errors",
			ve:   &ValidationError{},
			want: ValidationFailedMessage,
		},
		{
			name: "single error",
			ve: &ValidationError{
				Errors: []ValidationFieldError{
					{Field: "email", Message: "invalid email"},
				},
			},
			want: "email: invalid email",
		},
		{
			name: "multiple errors",
			ve: &ValidationError{
				Errors: []ValidationFieldError{
					{Field: "email", Message: "invalid email"},
					{Field: "age", Message: "must be positive"},
				},
			},
			want: "email: invalid email; age: must be positive",
		},
		{
			name: "error without field",
			ve: &ValidationError{
				Errors: []ValidationFieldError{
					{Message: "general validation error"},
				},
			},
			want: "general validation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ve.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidationError_Add(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid email", "test@")

	if len(ve.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(ve.Errors))
	}

	err := ve.Errors[0]
	if err.Field != "email" {
		t.Errorf("Field = %q, want %q", err.Field, "email")
	}

	if err.Message != "invalid email" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid email")
	}

	if err.Value != "test@" {
		t.Errorf("Value = %v, want %q", err.Value, "test@")
	}
}

func TestValidationError_Add_Nil(t *testing.T) {
	var ve *ValidationError
	ve.Add("email", "invalid", nil) // Should not panic
}

func TestValidationError_AddWithCode(t *testing.T) {
	ve := NewValidationError()
	ve.AddWithCode("email", "invalid format", ErrCodeInvalidFormat, "test@")

	if len(ve.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(ve.Errors))
	}

	err := ve.Errors[0]
	if err.Code != ErrCodeInvalidFormat {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeInvalidFormat)
	}
}

func TestValidationError_HasErrors(t *testing.T) {
	tests := []struct {
		name string
		ve   *ValidationError
		want bool
	}{
		{
			name: "nil validation error",
			ve:   nil,
			want: false,
		},
		{
			name: "empty errors",
			ve:   NewValidationError(),
			want: false,
		},
		{
			name: "with errors",
			ve: &ValidationError{
				Errors: []ValidationFieldError{
					{Field: "email", Message: "invalid"},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ve.HasErrors()
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationError_Count(t *testing.T) {
	tests := []struct {
		name string
		ve   *ValidationError
		want int
	}{
		{
			name: "nil validation error",
			ve:   nil,
			want: 0,
		},
		{
			name: "empty errors",
			ve:   NewValidationError(),
			want: 0,
		},
		{
			name: "multiple errors",
			ve: &ValidationError{
				Errors: []ValidationFieldError{
					{Field: "email", Message: "invalid"},
					{Field: "age", Message: "required"},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ve.Count()
			if got != tt.want {
				t.Errorf("Count() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidationError_MarshalJSON(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid email", "test@")

	data, err := json.Marshal(ve)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if result["error"] != ValidationFailedMessage {
		t.Errorf("error = %q, want %q", result["error"], ValidationFailedMessage)
	}

	validationErrors, ok := result["validationErrors"].([]any)
	if !ok {
		t.Fatalf("validationErrors is not a slice")
	}

	if len(validationErrors) != 1 {
		t.Errorf("len(validationErrors) = %d, want 1", len(validationErrors))
	}
}

func TestValidationError_MarshalJSON_Nil(t *testing.T) {
	var ve *ValidationError

	data, err := json.Marshal(ve)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// A nil pointer marshals to "null" in JSON
	expected := "null"
	if string(data) != expected {
		t.Errorf("Marshal(nil) = %q, want %q", string(data), expected)
	}
}

func TestValidationError_StatusCode(t *testing.T) {
	ve := NewValidationError()
	if got := ve.StatusCode(); got != http.StatusUnprocessableEntity {
		t.Errorf("StatusCode() = %d, want %d", got, http.StatusUnprocessableEntity)
	}
}

func TestValidationError_ResponseBody(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", "test@")

	body := ve.ResponseBody()

	bodyMap, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("ResponseBody() is not a map")
	}

	if bodyMap["error"] != ValidationFailedMessage {
		t.Errorf("error = %q, want %q", bodyMap["error"], ValidationFailedMessage)
	}

	if bodyMap["code"] != http.StatusUnprocessableEntity {
		t.Errorf("code = %d, want %d", bodyMap["code"], http.StatusUnprocessableEntity)
	}
}

func TestValidationError_ResponseBody_Nil(t *testing.T) {
	var ve *ValidationError

	body := ve.ResponseBody()

	bodyMap, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("ResponseBody() is not a map")
	}

	validationErrors, ok := bodyMap["validationErrors"].([]ValidationFieldError)
	if !ok {
		t.Fatalf("validationErrors is not []ValidationFieldError")
	}

	if len(validationErrors) != 0 {
		t.Errorf("len(validationErrors) = %d, want 0", len(validationErrors))
	}
}

func TestValidationError_Headers(t *testing.T) {
	ve := NewValidationError()
	headers := ve.Headers()

	if headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want %q", headers["Content-Type"], "application/json")
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	ve := NewValidationError()
	if ve.Unwrap() != nil {
		t.Error("Unwrap() should return nil")
	}
}

func TestValidationError_As(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", nil)

	var target *ValidationError
	if !ve.As(&target) {
		t.Error("As() should return true for *ValidationError")
	}

	if target == nil {
		t.Error("target should not be nil")
	}

	if target != ve {
		t.Error("target should be the same as ve")
	}
}

func TestValidationError_GetFieldErrors(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid format", "test@")
	ve.Add("email", "too short", "a@b")
	ve.Add("age", "required", nil)

	emailErrors := ve.GetFieldErrors("email")
	if len(emailErrors) != 2 {
		t.Errorf("len(emailErrors) = %d, want 2", len(emailErrors))
	}

	ageErrors := ve.GetFieldErrors("age")
	if len(ageErrors) != 1 {
		t.Errorf("len(ageErrors) = %d, want 1", len(ageErrors))
	}

	nonExistent := ve.GetFieldErrors("nonexistent")
	if len(nonExistent) != 0 {
		t.Errorf("len(nonExistent) = %d, want 0", len(nonExistent))
	}
}

func TestValidationError_GetFieldErrors_Nil(t *testing.T) {
	var ve *ValidationError

	result := ve.GetFieldErrors("email")

	if result != nil {
		t.Error("GetFieldErrors() on nil should return nil")
	}
}

func TestValidationError_HasFieldError(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", nil)

	if !ve.HasFieldError("email") {
		t.Error("HasFieldError(email) should be true")
	}

	if ve.HasFieldError("age") {
		t.Error("HasFieldError(age) should be false")
	}
}

func TestValidationError_Merge(t *testing.T) {
	ve1 := NewValidationError()
	ve1.Add("email", "invalid", nil)

	ve2 := NewValidationError()
	ve2.Add("age", "required", nil)

	ve1.Merge(ve2)

	if ve1.Count() != 2 {
		t.Errorf("Count() = %d, want 2", ve1.Count())
	}

	if !ve1.HasFieldError("email") {
		t.Error("should have email error")
	}

	if !ve1.HasFieldError("age") {
		t.Error("should have age error")
	}
}

func TestValidationError_Merge_Nil(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", nil)

	ve.Merge(nil) // Should not panic

	if ve.Count() != 1 {
		t.Errorf("Count() = %d, want 1", ve.Count())
	}

	var nilVE *ValidationError
	nilVE.Merge(ve) // Should not panic
}

func TestNewValidationErrorResponse(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", "test@")

	response := NewValidationErrorResponse(ve)

	if response.Error != ValidationFailedMessage {
		t.Errorf("Error = %q, want %q", response.Error, ValidationFailedMessage)
	}

	if response.Code != http.StatusUnprocessableEntity {
		t.Errorf("Code = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}

	if len(response.ValidationErrors) != 1 {
		t.Errorf("len(ValidationErrors) = %d, want 1", len(response.ValidationErrors))
	}
}

func TestNewValidationErrorResponse_Nil(t *testing.T) {
	response := NewValidationErrorResponse(nil)

	if len(response.ValidationErrors) != 0 {
		t.Errorf("len(ValidationErrors) = %d, want 0", len(response.ValidationErrors))
	}
}

func TestIsValidationError(t *testing.T) {
	ve := NewValidationError()
	ve.Add("email", "invalid", nil)

	if !IsValidationError(ve) {
		t.Error("IsValidationError() should return true for ValidationError")
	}

	regularErr := errors.New("regular error")
	if IsValidationError(regularErr) {
		t.Error("IsValidationError() should return false for regular error")
	}

	if IsValidationError(nil) {
		t.Error("IsValidationError() should return false for nil")
	}
}

func TestValidationErrorConstants(t *testing.T) {
	constants := []string{
		ErrCodeRequired,
		ErrCodeInvalidType,
		ErrCodeInvalidFormat,
		ErrCodeMinLength,
		ErrCodeMaxLength,
		ErrCodeMinValue,
		ErrCodeMaxValue,
		ErrCodePattern,
		ErrCodeEnum,
		ErrCodeMinItems,
		ErrCodeMaxItems,
		ErrCodeUniqueItems,
	}

	for i, code := range constants {
		if code == "" {
			t.Errorf("constant at index %d is empty", i)
		}
	}
}

func BenchmarkValidationError_Add(b *testing.B) {
	ve := NewValidationError()

	for b.Loop() {
		ve.Add("field", "message", nil)
	}
}

func BenchmarkValidationError_HasFieldError(b *testing.B) {
	ve := NewValidationError()
	for range 100 {
		ve.Add("field", "message", nil)
	}

	for b.Loop() {
		_ = ve.HasFieldError("field")
	}
}
