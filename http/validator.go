package http

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"github.com/xraph/go-utils/val"
)

var (
	// Global validator instance with caching.
	validatorInstance *validator.Validate
	validatorOnce     sync.Once
)

// getValidator returns a singleton validator instance.
func getValidator() *validator.Validate {
	validatorOnce.Do(func() {
		validatorInstance = validator.New()

		// Use JSON tag names for field names in error messages.
		validatorInstance.RegisterTagNameFunc(val.GetFieldName)

		// Register custom validators
		_ = validatorInstance.RegisterValidation("iso8601", validateISO8601)
	})

	return validatorInstance
}

// validateISO8601 is a custom validator for ISO 8601 datetime strings.
func validateISO8601(fl validator.FieldLevel) bool {
	return val.IsValidISO8601(fl.Field().String())
}

// validateStruct validates struct fields using go-playground/validator and custom tags.
func (c *Ctx) validateStruct(v any, rt reflect.Type, errs *val.ValidationError) error {
	validate := getValidator()

	// First, validate using go-playground/validator (only if validate tag exists).
	err := validate.Struct(v)
	if err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			c.mapValidationErrors(validationErrs, errs)
		}
	}

	// Then validate using our custom tags (format, minLength, pattern, etc.)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	c.validateCustomTags(rv, rt, errs)

	return nil
}

// mapValidationErrors maps validator.ValidationErrors to our ValidationError format.
func (c *Ctx) mapValidationErrors(validationErrs validator.ValidationErrors, errors *val.ValidationError) {
	for _, err := range validationErrs {
		fieldName := err.Field()
		message := c.formatValidationMessage(err)
		code := c.getErrorCode(err)
		actualValue := err.Value()

		errors.AddWithCode(fieldName, message, code, actualValue)
	}
}

// formatValidationMessage formats a validation error message.
func (c *Ctx) formatValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "url", "uri":
		return "must be a valid URL"
	case "uuid", "uuid4", "uuid5":
		return "must be a valid UUID"
	case "min":
		if err.Kind() == reflect.String {
			return fmt.Sprintf("must be at least %s characters", err.Param())
		}

		return "must be at least " + err.Param()
	case "max":
		if err.Kind() == reflect.String {
			return fmt.Sprintf("must be at most %s characters", err.Param())
		}

		return "must be at most " + err.Param()
	case "gte":
		return "must be at least " + err.Param()
	case "lte":
		return "must be at most " + err.Param()
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(err.Param(), " ", ", ")
	case "iso8601":
		return "must be a valid ISO 8601 date-time"
	case "datetime":
		return "must be a valid datetime in format " + err.Param()
	default:
		return fmt.Sprintf("validation failed on '%s'", err.Tag())
	}
}

// getErrorCode maps validator tags to our error codes.
func (c *Ctx) getErrorCode(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return val.ErrCodeRequired
	case "min":
		if err.Kind() == reflect.String {
			return val.ErrCodeMinLength
		}

		return val.ErrCodeMinValue
	case "max":
		if err.Kind() == reflect.String {
			return val.ErrCodeMaxLength
		}

		return val.ErrCodeMaxValue
	case "gte":
		return val.ErrCodeMinValue
	case "lte":
		return val.ErrCodeMaxValue
	case "email", "url", "uri", "uuid", "uuid4", "uuid5", "iso8601", "datetime":
		return val.ErrCodeInvalidFormat
	case "oneof":
		return val.ErrCodeEnum
	default:
		return val.ErrCodeInvalidType
	}
}

// validateCustomTags validates fields with our custom validation tags.
func (c *Ctx) validateCustomTags(rv reflect.Value, rt reflect.Type, errors *val.ValidationError) {
	for i := range rt.NumField() {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs
		if field.Anonymous {
			hasExplicitTag := field.Tag.Get("path") != "" ||
				field.Tag.Get("query") != "" ||
				field.Tag.Get("header") != "" ||
				field.Tag.Get("json") != ""

			if !hasExplicitTag {
				embeddedType := field.Type
				embeddedValue := fieldValue

				if embeddedType.Kind() == reflect.Ptr {
					if embeddedValue.IsNil() {
						continue
					}

					embeddedType = embeddedType.Elem()
					embeddedValue = embeddedValue.Elem()
				}

				if embeddedType.Kind() == reflect.Struct {
					c.validateCustomTags(embeddedValue, embeddedType, errors)

					continue
				}
			}
		}

		// Check if field has any of our custom validation tags
		hasCustomTags := field.Tag.Get("format") != "" ||
			field.Tag.Get("minLength") != "" ||
			field.Tag.Get("maxLength") != "" ||
			field.Tag.Get("pattern") != "" ||
			field.Tag.Get("minimum") != "" ||
			field.Tag.Get("maximum") != "" ||
			field.Tag.Get("multipleOf") != "" ||
			field.Tag.Get("enum") != ""

		// Also check for required validation on parameter fields
		isParamField := val.IsParameterField(field)
		fieldRequired := val.IsFieldRequired(field)

		// Skip if no validation needed: no custom tags AND (not required OR is optional)
		if !hasCustomTags && !fieldRequired {
			continue
		}

		fieldName := val.GetFieldName(field)

		// Handle pointer fields
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				if fieldRequired && isParamField {
					errors.AddWithCode(fieldName, "field is required", val.ErrCodeRequired, nil)
				}

				continue
			}

			fieldValue = fieldValue.Elem()
		}

		// Required validation for parameter string fields
		if fieldRequired && isParamField && fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			errors.AddWithCode(fieldName, "field is required", val.ErrCodeRequired, "")

			continue
		}

		// Required validation for non-parameter (body) string fields
		// go-playground/validator doesn't catch empty strings for required fields
		if fieldRequired && !isParamField && fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			errors.AddWithCode(fieldName, "field is required", val.ErrCodeRequired, "")

			continue
		}

		// Custom tag validation
		c.validateFieldCustomTags(field, fieldValue, fieldName, errors)
	}
}

// validateFieldCustomTags validates a field using our custom tags.
func (c *Ctx) validateFieldCustomTags(field reflect.StructField, fieldValue reflect.Value, fieldName string, errors *val.ValidationError) {
	isOptional := !val.IsFieldRequired(field)
	isEmpty := val.IsZeroValue(fieldValue)

	// String validations
	if fieldValue.Kind() == reflect.String {
		value := fieldValue.String()

		// MinLength
		if minLengthTag := field.Tag.Get("minLength"); minLengthTag != "" {
			var minLen int
			if _, err := fmt.Sscanf(minLengthTag, "%d", &minLen); err == nil && (!isOptional || !isEmpty) {
				if len(value) < minLen {
					errors.AddWithCode(fieldName, fmt.Sprintf("must be at least %d characters", minLen), val.ErrCodeMinLength, value)
				}
			}
		}

		// MaxLength
		if maxLengthTag := field.Tag.Get("maxLength"); maxLengthTag != "" {
			var maxLen int
			if _, err := fmt.Sscanf(maxLengthTag, "%d", &maxLen); err == nil {
				if len(value) > maxLen {
					errors.AddWithCode(fieldName, fmt.Sprintf("must be at most %d characters", maxLen), val.ErrCodeMaxLength, value)
				}
			}
		}

		// Pattern
		if pattern := field.Tag.Get("pattern"); pattern != "" && (!isOptional || !isEmpty) {
			if matched, _ := regexp.MatchString(pattern, value); !matched {
				errors.AddWithCode(fieldName, "does not match required pattern", val.ErrCodePattern, value)
			}
		}

		// Format
		if format := field.Tag.Get("format"); format != "" && (!isOptional || !isEmpty) {
			c.validateFormat(format, value, fieldName, errors)
		}
	}

	// Numeric validations
	if val.IsNumericKind(fieldValue.Kind()) {
		var numValue float64

		switch fieldValue.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			numValue = float64(fieldValue.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			numValue = float64(fieldValue.Uint())
		case reflect.Float32, reflect.Float64:
			numValue = fieldValue.Float()
		}

		isZero := numValue == 0

		// Minimum
		if minTag := field.Tag.Get("minimum"); minTag != "" {
			var minValue float64
			if _, err := fmt.Sscanf(minTag, "%f", &minValue); err == nil {
				if !isOptional || !isZero || minValue == 0 {
					if numValue < minValue {
						errors.AddWithCode(fieldName, fmt.Sprintf("must be at least %v", minValue), val.ErrCodeMinValue, numValue)
					}
				}
			}
		}

		// Maximum
		if maxTag := field.Tag.Get("maximum"); maxTag != "" {
			var maxValue float64
			if _, err := fmt.Sscanf(maxTag, "%f", &maxValue); err == nil {
				if numValue > maxValue {
					errors.AddWithCode(fieldName, fmt.Sprintf("must be at most %v", maxValue), val.ErrCodeMaxValue, numValue)
				}
			}
		}

		// MultipleOf
		if multipleOfTag := field.Tag.Get("multipleOf"); multipleOfTag != "" && (!isOptional || !isZero) {
			var multipleOf float64
			if _, err := fmt.Sscanf(multipleOfTag, "%f", &multipleOf); err == nil && multipleOf != 0 {
				if int(numValue)%int(multipleOf) != 0 {
					errors.AddWithCode(fieldName, fmt.Sprintf("must be a multiple of %v", multipleOf), val.ErrCodeInvalidType, numValue)
				}
			}
		}
	}

	// Enum validation
	if enumTag := field.Tag.Get("enum"); enumTag != "" && (!isOptional || !isEmpty) {
		c.validateEnumTag(fieldValue, fieldName, enumTag, errors)
	}
}

// validateEnumTag validates enum constraints.
func (c *Ctx) validateEnumTag(fieldValue reflect.Value, fieldName string, enumTag string, errors *val.ValidationError) {
	enumValues := strings.Split(enumTag, ",")
	for i, v := range enumValues {
		enumValues[i] = strings.TrimSpace(v)
	}

	var strValue string

	switch fieldValue.Kind() {
	case reflect.String:
		strValue = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue = strconv.FormatInt(fieldValue.Int(), 10)
	default:
		strValue = fmt.Sprintf("%v", fieldValue.Interface())
	}

	found := slices.Contains(enumValues, strValue)

	if !found {
		errors.AddWithCode(fieldName, "must be one of: "+strings.Join(enumValues, ", "), val.ErrCodeEnum, strValue)
	}
}

// validateFormat validates format constraints.
func (c *Ctx) validateFormat(format string, value string, fieldName string, errors *val.ValidationError) {
	switch format {
	case "email":
		if !val.IsValidEmail(value) {
			errors.AddWithCode(fieldName, "must be a valid email address", val.ErrCodeInvalidFormat, value)
		}
	case "uuid":
		if !val.IsValidUUID(value) {
			errors.AddWithCode(fieldName, "must be a valid UUID", val.ErrCodeInvalidFormat, value)
		}
	case "uri", "url":
		if !val.IsValidURL(value) {
			errors.AddWithCode(fieldName, "must be a valid URL", val.ErrCodeInvalidFormat, value)
		}
	case "date":
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value)
		if !matched {
			errors.AddWithCode(fieldName, "must be a valid date (YYYY-MM-DD)", val.ErrCodeInvalidFormat, value)
		}
	case "date-time":
		if !val.IsValidISO8601(value) {
			errors.AddWithCode(fieldName, "must be a valid ISO 8601 date-time", val.ErrCodeInvalidFormat, value)
		}
	}
}
