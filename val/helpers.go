package val

import (
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Compile regexes once at package level for better performance.
var (
	iso8601Regex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)
)

// IsFieldRequired determines if a field is required for validation.
// Uses consistent precedence order:
// 1. optional:"true" - explicitly optional (highest priority)
// 2. required:"true" - explicitly required
// 3. omitempty in json/query/header/body tags - optional
// 4. pointer type - optional
// 5. default: non-pointer types are required.
func IsFieldRequired(field reflect.StructField) bool {
	// 1. Explicit optional tag takes precedence (opt-out)
	if field.Tag.Get("optional") == "true" {
		return false
	}

	// 2. Explicit required tag
	if field.Tag.Get("required") == "true" {
		return true
	}

	// 3. Check for omitempty in various tags
	tags := []string{"json", "query", "header", "body"}
	for _, tagName := range tags {
		if tagValue := field.Tag.Get(tagName); tagValue != "" {
			if strings.Contains(tagValue, ",omitempty") {
				return false
			}
		}
	}

	// 4. Pointer types are optional by default
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	// 5. Non-pointer types without above markers are required
	return true
}

// GetFieldName extracts the field name from struct tags.
// Priority: path > query > header > json > field name.
func GetFieldName(field reflect.StructField) string {
	// Try tags in order of priority
	tagPriority := []string{"path", "query", "header", "json"}
	for _, tagName := range tagPriority {
		if tagValue := field.Tag.Get(tagName); tagValue != "" && tagValue != "-" {
			return parseTagName(tagValue)
		}
	}

	// Fallback to field name
	return field.Name
}

// parseTagName extracts the name part from a tag value (before comma).
func parseTagName(tagValue string) string {
	if idx := strings.Index(tagValue, ","); idx != -1 {
		return tagValue[:idx]
	}

	return tagValue
}

// IsNumericKind checks if a reflect.Kind is a numeric type.
func IsNumericKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// IsParameterField checks if a field is bound from query, header, or path parameters.
// These parameter types are validated during the binding phase where we can distinguish
// between missing values and explicit zero values (0, false, "").
func IsParameterField(field reflect.StructField) bool {
	return field.Tag.Get("query") != "" ||
		field.Tag.Get("header") != "" ||
		field.Tag.Get("path") != ""
}

// IsZeroValue checks if a reflect.Value is its zero value.
func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}

// IsValidEmail validates an email address using Go's standard library.
// This is more reliable than regex-based validation.
func IsValidEmail(email string) bool {
	if email == "" {
		return false
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Ensure the parsed address matches the input (no display name)
	return addr.Address == email
}

// IsValidUUID validates a UUID using the google/uuid package.
// Supports all UUID versions (v1, v3, v4, v5).
func IsValidUUID(uuidStr string) bool {
	if uuidStr == "" {
		return false
	}

	_, err := uuid.Parse(uuidStr)

	return err == nil
}

// IsValidURL validates a URL using Go's standard library.
// Checks for valid scheme (http/https) and proper URL structure.
func IsValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Ensure scheme is http or https and host is present
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	return u.Host != ""
}

// IsValidISO8601 validates an ISO 8601 date-time string.
// Supports formats: YYYY-MM-DDTHH:MM:SS, YYYY-MM-DDTHH:MM:SS.sss, with optional Z or timezone offset.
func IsValidISO8601(datetime string) bool {
	if datetime == "" {
		return false
	}

	return iso8601Regex.MatchString(datetime)
}
