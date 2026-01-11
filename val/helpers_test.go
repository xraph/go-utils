package val

import (
	"reflect"
	"testing"
)

func TestIsFieldRequired(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected bool
	}{
		{
			name: "explicit optional tag",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `optional:"true"`,
			},
			expected: false,
		},
		{
			name: "explicit required tag",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `required:"true"`,
			},
			expected: true,
		},
		{
			name: "json omitempty",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `json:"field,omitempty"`,
			},
			expected: false,
		},
		{
			name: "query omitempty",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `query:"field,omitempty"`,
			},
			expected: false,
		},
		{
			name: "pointer type",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[*string](),
			},
			expected: false,
		},
		{
			name: "non-pointer type without tags",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
			},
			expected: true,
		},
		{
			name: "required overrides omitempty",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `json:"field,omitempty" required:"true"`,
			},
			expected: true,
		},
		{
			name: "optional overrides required",
			field: reflect.StructField{
				Name: "Field",
				Type: reflect.TypeFor[string](),
				Tag:  `required:"true" optional:"true"`,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFieldRequired(tt.field)
			if got != tt.expected {
				t.Errorf("IsFieldRequired() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetFieldName(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected string
	}{
		{
			name: "path tag priority",
			field: reflect.StructField{
				Name: "ID",
				Tag:  `path:"id" json:"identifier"`,
			},
			expected: "id",
		},
		{
			name: "query tag",
			field: reflect.StructField{
				Name: "Page",
				Tag:  `query:"page" json:"pageNumber"`,
			},
			expected: "page",
		},
		{
			name: "header tag",
			field: reflect.StructField{
				Name: "Auth",
				Tag:  `header:"Authorization" json:"auth"`,
			},
			expected: "Authorization",
		},
		{
			name: "json tag",
			field: reflect.StructField{
				Name: "Email",
				Tag:  `json:"email"`,
			},
			expected: "email",
		},
		{
			name: "json tag with options",
			field: reflect.StructField{
				Name: "Email",
				Tag:  `json:"email,omitempty"`,
			},
			expected: "email",
		},
		{
			name: "json dash ignored",
			field: reflect.StructField{
				Name: "Internal",
				Tag:  `json:"-"`,
			},
			expected: "Internal",
		},
		{
			name: "no tags",
			field: reflect.StructField{
				Name: "Username",
			},
			expected: "Username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFieldName(tt.field)
			if got != tt.expected {
				t.Errorf("GetFieldName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsNumericKind(t *testing.T) {
	numericKinds := []reflect.Kind{
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
	}

	for _, kind := range numericKinds {
		t.Run(kind.String(), func(t *testing.T) {
			if !IsNumericKind(kind) {
				t.Errorf("IsNumericKind(%v) = false, want true", kind)
			}
		})
	}

	nonNumericKinds := []reflect.Kind{
		reflect.String, reflect.Bool, reflect.Array, reflect.Slice,
		reflect.Map, reflect.Struct, reflect.Ptr, reflect.Chan,
	}

	for _, kind := range nonNumericKinds {
		t.Run(kind.String(), func(t *testing.T) {
			if IsNumericKind(kind) {
				t.Errorf("IsNumericKind(%v) = true, want false", kind)
			}
		})
	}
}

func TestIsParameterField(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		expected bool
	}{
		{
			name: "query parameter",
			field: reflect.StructField{
				Name: "Page",
				Tag:  `query:"page"`,
			},
			expected: true,
		},
		{
			name: "header parameter",
			field: reflect.StructField{
				Name: "Auth",
				Tag:  `header:"Authorization"`,
			},
			expected: true,
		},
		{
			name: "path parameter",
			field: reflect.StructField{
				Name: "ID",
				Tag:  `path:"id"`,
			},
			expected: true,
		},
		{
			name: "body field",
			field: reflect.StructField{
				Name: "Email",
				Tag:  `json:"email"`,
			},
			expected: false,
		},
		{
			name: "no tags",
			field: reflect.StructField{
				Name: "Field",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsParameterField(tt.field)
			if got != tt.expected {
				t.Errorf("IsParameterField() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsZeroValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"empty string", "", true},
		{"non-empty string", "hello", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"nil pointer", (*string)(nil), true},
		{"non-nil pointer", func() *string {
			s := "test"

			return &s
		}(), false},
		{"empty slice", []int{}, true},
		{"non-empty slice", []int{1}, false},
		{"empty map", map[string]int{}, true},
		{"non-empty map", map[string]int{"a": 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)

			got := IsZeroValue(v)
			if got != tt.expected {
				t.Errorf("IsZeroValue(%v) = %v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name+tag@example.co.uk", true},
		{"test@sub.example.com", true},
		{"", false},
		{"invalid", false},
		{"@example.com", false},
		{"test@", false},
		{"test @example.com", false},
		{"test@example", true},                  // net/mail.ParseAddress accepts this
		{"Test User <test@example.com>", false}, // Display name not allowed
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := IsValidEmail(tt.email)
			if got != tt.valid {
				t.Errorf("IsValidEmail(%q) = %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		uuid  string
		valid bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"6ba7b810-9dad-11d1-80b4-00c04fd430c8", true},
		{"6ba7b811-9dad-11d1-80b4-00c04fd430c8", true},
		{"", false},
		{"invalid-uuid", false},
		{"550e8400-e29b-41d4-a716", false},
		{"550e8400e29b41d4a716446655440000", true},      // google/uuid accepts without hyphens
		{"550e8400-e29b-41d4-a716-44665544000g", false}, // Invalid character
	}

	for _, tt := range tests {
		t.Run(tt.uuid, func(t *testing.T) {
			got := IsValidUUID(tt.uuid)
			if got != tt.valid {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tt.uuid, got, tt.valid)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://sub.example.com/path", true},
		{"https://example.com:8080/path?query=value", true},
		{"", false},
		{"not-a-url", false},
		{"ftp://example.com", false}, // Only http/https allowed
		{"example.com", false},       // No scheme
		{"http://", false},           // No host
		{"https://", false},          // No host
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsValidURL(tt.url)
			if got != tt.valid {
				t.Errorf("IsValidURL(%q) = %v, want %v", tt.url, got, tt.valid)
			}
		})
	}
}

func TestIsValidISO8601(t *testing.T) {
	tests := []struct {
		datetime string
		valid    bool
	}{
		{"2023-12-25T10:30:45Z", true},
		{"2023-12-25T10:30:45+00:00", true},
		{"2023-12-25T10:30:45-05:00", true},
		{"2023-12-25T10:30:45.123Z", true},
		{"2023-12-25T10:30:45.123456Z", true},
		{"", false},
		{"2023-12-25", false},           // Date only
		{"10:30:45", false},             // Time only
		{"2023/12/25T10:30:45Z", false}, // Wrong separator
		{"not-a-date", false},
	}

	for _, tt := range tests {
		t.Run(tt.datetime, func(t *testing.T) {
			got := IsValidISO8601(tt.datetime)
			if got != tt.valid {
				t.Errorf("IsValidISO8601(%q) = %v, want %v", tt.datetime, got, tt.valid)
			}
		})
	}
}

func BenchmarkIsValidEmail(b *testing.B) {
	email := "test@example.com"
	for b.Loop() {
		IsValidEmail(email)
	}
}

func BenchmarkIsValidUUID(b *testing.B) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	for b.Loop() {
		IsValidUUID(uuid)
	}
}

func BenchmarkIsValidURL(b *testing.B) {
	url := "https://example.com/path"
	for b.Loop() {
		IsValidURL(url)
	}
}

func BenchmarkIsValidISO8601(b *testing.B) {
	datetime := "2023-12-25T10:30:45Z"
	for b.Loop() {
		IsValidISO8601(datetime)
	}
}
