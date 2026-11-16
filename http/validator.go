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
	if rv.Kind() == reflect.Pointer {
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

// The custom tags (minLength, pattern, minimum and the rest) are properties of
// the request type, not of the request. They used to be read and parsed on
// every request: each pattern went through regexp.MatchString, which compiles
// it, and each numeric bound went through fmt.Sscanf. Two pattern fields
// alone cost over a hundred allocations a request.
//
// They are now parsed once per type into a customTagPlan, and compiled
// patterns are shared across types through patternCache. The parse is the
// same one as before (fmt.Sscanf with the same verbs), so a tag that failed to
// parse and was ignored is still ignored, and a pattern that fails to compile
// still fails every value it checks, as regexp.MatchString's error did.

// customTagPlan is the parsed form of one struct level's custom tags.
type customTagPlan struct {
	fields []customFieldPlan
}

// customFieldPlan holds everything validateCustomTags needs to know about one
// field. Fields that need no checking and are not flattened embedded structs
// are left out of the plan entirely.
type customFieldPlan struct {
	// Flattened embedded struct handling.
	sub         *customTagPlan // set when the field is flattened and is a struct
	flatten     bool           // anonymous with no path, query, header or json tag
	flattenPtr  bool
	check       bool // has custom tags or is required
	required    bool
	index       int
	name        string
	minLength   int
	maxLength   int
	minimum     float64
	maximum     float64
	multipleOf  float64
	hasMinLen   bool
	hasMaxLen   bool
	hasMinimum  bool
	hasMaximum  bool
	hasMultiple bool
	pattern     string
	patternRe   *regexp.Regexp // nil with a non-empty pattern means it did not compile
	format      string
	enumValues  []string
	enumMessage string
}

var (
	// customTagPlans caches *customTagPlan by reflect.Type.
	customTagPlans sync.Map

	// patternCache caches compiled pattern:"..." expressions by source. An
	// expression that fails to compile is stored as a nil *regexp.Regexp.
	patternCache sync.Map

	// dateFormatRe backs format:"date".
	dateFormatRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// compiledPattern returns the compiled form of pattern, or nil when it does
// not compile.
func compiledPattern(pattern string) *regexp.Regexp {
	if cached, ok := patternCache.Load(pattern); ok {
		return cached.(*regexp.Regexp) //nolint:forcetypeassert // only *regexp.Regexp is stored
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		re = nil
	}

	actual, _ := patternCache.LoadOrStore(pattern, re)

	return actual.(*regexp.Regexp) //nolint:forcetypeassert // only *regexp.Regexp is stored
}

// customTagPlanFor returns the cached plan for rt, building it on first use.
func customTagPlanFor(rt reflect.Type) *customTagPlan {
	if cached, ok := customTagPlans.Load(rt); ok {
		return cached.(*customTagPlan) //nolint:forcetypeassert // only *customTagPlan is stored
	}

	plan := buildCustomTagPlan(rt, map[reflect.Type]*customTagPlan{})

	actual, _ := customTagPlans.LoadOrStore(rt, plan)

	return actual.(*customTagPlan) //nolint:forcetypeassert // only *customTagPlan is stored
}

// buildCustomTagPlan parses rt's custom tags. building holds the plans under
// construction so a type that embeds a pointer to itself resolves to the same
// plan instead of recursing forever.
func buildCustomTagPlan(rt reflect.Type, building map[reflect.Type]*customTagPlan) *customTagPlan {
	if plan, ok := building[rt]; ok {
		return plan
	}

	plan := &customTagPlan{}
	building[rt] = plan

	for i := range rt.NumField() {
		field := rt.Field(i)

		if !field.IsExported() {
			continue
		}

		fp := customFieldPlan{index: i}

		if field.Anonymous {
			hasExplicitTag := field.Tag.Get("path") != "" ||
				field.Tag.Get("query") != "" ||
				field.Tag.Get("header") != "" ||
				field.Tag.Get("json") != ""

			if !hasExplicitTag {
				fp.flatten = true

				embeddedType := field.Type
				if embeddedType.Kind() == reflect.Pointer {
					fp.flattenPtr = true
					embeddedType = embeddedType.Elem()
				}

				if embeddedType.Kind() == reflect.Struct {
					fp.sub = buildCustomTagPlan(embeddedType, building)
					plan.fields = append(plan.fields, fp)

					continue
				}
			}
		}

		parseCustomFieldTags(field, &fp)

		if fp.check || fp.flatten {
			plan.fields = append(plan.fields, fp)
		}
	}

	return plan
}

// parseCustomFieldTags fills fp from field's tags, parsing them exactly as the
// per-request code used to.
func parseCustomFieldTags(field reflect.StructField, fp *customFieldPlan) {
	tag := field.Tag

	hasCustomTags := tag.Get("format") != "" ||
		tag.Get("minLength") != "" ||
		tag.Get("maxLength") != "" ||
		tag.Get("pattern") != "" ||
		tag.Get("minimum") != "" ||
		tag.Get("maximum") != "" ||
		tag.Get("multipleOf") != "" ||
		tag.Get("enum") != ""

	fp.required = val.IsFieldRequired(field)
	fp.check = hasCustomTags || fp.required

	if !fp.check {
		return
	}

	fp.name = val.GetFieldName(field)

	if s := tag.Get("minLength"); s != "" {
		_, err := fmt.Sscanf(s, "%d", &fp.minLength)
		fp.hasMinLen = err == nil
	}

	if s := tag.Get("maxLength"); s != "" {
		_, err := fmt.Sscanf(s, "%d", &fp.maxLength)
		fp.hasMaxLen = err == nil
	}

	if s := tag.Get("pattern"); s != "" {
		fp.pattern = s
		fp.patternRe = compiledPattern(s)
	}

	fp.format = tag.Get("format")

	if s := tag.Get("minimum"); s != "" {
		_, err := fmt.Sscanf(s, "%f", &fp.minimum)
		fp.hasMinimum = err == nil
	}

	if s := tag.Get("maximum"); s != "" {
		_, err := fmt.Sscanf(s, "%f", &fp.maximum)
		fp.hasMaximum = err == nil
	}

	if s := tag.Get("multipleOf"); s != "" {
		_, err := fmt.Sscanf(s, "%f", &fp.multipleOf)
		fp.hasMultiple = err == nil && fp.multipleOf != 0
	}

	if s := tag.Get("enum"); s != "" {
		fp.enumValues = strings.Split(s, ",")
		for i, v := range fp.enumValues {
			fp.enumValues[i] = strings.TrimSpace(v)
		}

		fp.enumMessage = "must be one of: " + strings.Join(fp.enumValues, ", ")
	}
}

// validateCustomTags validates fields with our custom validation tags.
func (c *Ctx) validateCustomTags(rv reflect.Value, rt reflect.Type, errors *val.ValidationError) {
	c.validateCustomPlan(rv, customTagPlanFor(rt), errors)
}

func (c *Ctx) validateCustomPlan(rv reflect.Value, plan *customTagPlan, errors *val.ValidationError) {
	for i := range plan.fields {
		fp := &plan.fields[i]
		fieldValue := rv.Field(fp.index)

		// Handle embedded structs
		if fp.flatten {
			if fp.flattenPtr && fieldValue.IsNil() {
				continue
			}

			if fp.sub != nil {
				if fp.flattenPtr {
					fieldValue = fieldValue.Elem()
				}

				c.validateCustomPlan(fieldValue, fp.sub, errors)

				continue
			}
		}

		// Skip if no validation needed: no custom tags AND (not required OR is optional)
		if !fp.check {
			continue
		}

		// Handle pointer fields
		wasPointer := fieldValue.Kind() == reflect.Pointer
		if wasPointer {
			if fieldValue.IsNil() {
				// Applies to body fields as well as parameters. Restricting
				// this to parameters meant a required *T in a body was
				// silently accepted when absent, while the generated OpenAPI
				// still advertised it as required — the published contract
				// and the runtime disagreed. A pointer is how a caller says
				// "I need to tell absent from zero"; it is not a statement
				// about whether the field may be omitted.
				if fp.required {
					errors.AddWithCode(fp.name, "field is required", val.ErrCodeRequired, nil)
				}

				continue
			}

			fieldValue = fieldValue.Elem()
		}

		// Value types only, and "" is a proxy for absence rather than a
		// judgement about emptiness: once JSON is decoded into a non-pointer
		// string, {"name":""} and {} are indistinguishable, so this is the
		// only signal available at this layer.
		//
		// It must not extend to pointers. A non-nil pointer was demonstrably
		// supplied — presence was already established above — so "" behind
		// one is a real value. A field that must be present yet may
		// legitimately be empty is therefore expressed as a pointer.
		//
		// The parameter and body cases were separate branches doing the same
		// thing; they are one check now.
		if fp.required && !wasPointer && fieldValue.Kind() == reflect.String && fieldValue.String() == "" {
			errors.AddWithCode(fp.name, "field is required", val.ErrCodeRequired, "")

			continue
		}

		// Custom tag validation
		c.validateFieldCustomTags(fp, fieldValue, errors)
	}
}

// validateFieldCustomTags validates a field using its parsed custom tags.
func (c *Ctx) validateFieldCustomTags(fp *customFieldPlan, fieldValue reflect.Value, errors *val.ValidationError) {
	fieldName := fp.name
	isOptional := !fp.required
	isEmpty := val.IsZeroValue(fieldValue)

	// String validations
	if fieldValue.Kind() == reflect.String {
		value := fieldValue.String()

		// MinLength
		if fp.hasMinLen && (!isOptional || !isEmpty) && len(value) < fp.minLength {
			errors.AddWithCode(fieldName, fmt.Sprintf("must be at least %d characters", fp.minLength), val.ErrCodeMinLength, value)
		}

		// MaxLength
		if fp.hasMaxLen && len(value) > fp.maxLength {
			errors.AddWithCode(fieldName, fmt.Sprintf("must be at most %d characters", fp.maxLength), val.ErrCodeMaxLength, value)
		}

		// Pattern. A pattern that does not compile matches nothing, as it
		// did when regexp.MatchString reported the compile error as false.
		if fp.pattern != "" && (!isOptional || !isEmpty) {
			if fp.patternRe == nil || !fp.patternRe.MatchString(value) {
				errors.AddWithCode(fieldName, "does not match required pattern", val.ErrCodePattern, value)
			}
		}

		// Format
		if fp.format != "" && (!isOptional || !isEmpty) {
			c.validateFormat(fp.format, value, fieldName, errors)
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
		if fp.hasMinimum && (!isOptional || !isZero || fp.minimum == 0) && numValue < fp.minimum {
			errors.AddWithCode(fieldName, fmt.Sprintf("must be at least %v", fp.minimum), val.ErrCodeMinValue, numValue)
		}

		// Maximum
		if fp.hasMaximum && numValue > fp.maximum {
			errors.AddWithCode(fieldName, fmt.Sprintf("must be at most %v", fp.maximum), val.ErrCodeMaxValue, numValue)
		}

		// MultipleOf
		if fp.hasMultiple && (!isOptional || !isZero) && int(numValue)%int(fp.multipleOf) != 0 {
			errors.AddWithCode(fieldName, fmt.Sprintf("must be a multiple of %v", fp.multipleOf), val.ErrCodeInvalidType, numValue)
		}
	}

	// Enum validation
	if fp.enumValues != nil && (!isOptional || !isEmpty) {
		c.validateEnumTag(fieldValue, fieldName, fp, errors)
	}
}

// validateEnumTag validates enum constraints.
func (c *Ctx) validateEnumTag(fieldValue reflect.Value, fieldName string, fp *customFieldPlan, errors *val.ValidationError) {
	var strValue string

	switch fieldValue.Kind() {
	case reflect.String:
		strValue = fieldValue.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		strValue = strconv.FormatInt(fieldValue.Int(), 10)
	default:
		strValue = fmt.Sprintf("%v", fieldValue.Interface())
	}

	if !slices.Contains(fp.enumValues, strValue) {
		errors.AddWithCode(fieldName, fp.enumMessage, val.ErrCodeEnum, strValue)
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
		if !dateFormatRe.MatchString(value) {
			errors.AddWithCode(fieldName, "must be a valid date (YYYY-MM-DD)", val.ErrCodeInvalidFormat, value)
		}
	case "date-time":
		if !val.IsValidISO8601(value) {
			errors.AddWithCode(fieldName, "must be a valid ISO 8601 date-time", val.ErrCodeInvalidFormat, value)
		}
	}
}
