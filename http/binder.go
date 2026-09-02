package http

import (
	"encoding"
	"errors"
	"fmt"
	gohttp "net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/xraph/go-utils/val"
)

// BindRequest binds and validates request data from all sources (path, query, header, body).
// This method provides comprehensive request binding that:
//   - Binds path parameters from URL path segments (path:"name")
//   - Binds query parameters from URL query string (query:"name")
//   - Binds headers from HTTP headers (header:"name")
//   - Binds body fields from request body (json:"name" or body:"")
//   - Validates all fields using validation tags (required, minLength, etc.)
//
// Example:
//
//	type CreateUserRequest struct {
//	    TenantID string `path:"tenantId" description:"Tenant ID"`
//	    DryRun   bool   `query:"dryRun" default:"false"`
//	    APIKey   string `header:"X-API-Key" required:"true"`
//	    Name     string `json:"name" minLength:"1" maxLength:"100"`
//	}
//
//	func handler(ctx forge.Context) error {
//	    var req CreateUserRequest
//	    if err := ctx.BindRequest(&req); err != nil {
//	        return err // Returns ValidationError if validation fails
//	    }
//	    // All fields are now bound and validated
//	}
func (c *Ctx) BindRequest(v any) error {
	// Get reflection value
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("BindRequest requires non-nil pointer")
	}

	rv = rv.Elem()
	rt := rv.Type()

	if rt.Kind() != reflect.Struct {
		// Not a struct, just bind body using regular Bind
		return c.Bind(v)
	}

	// Track validation errors.
	//
	// Declared as a value and passed by address so escape analysis can keep it
	// on the stack: the overwhelming majority of requests validate cleanly and
	// have no use for a heap object. Only the failure path below allocates,
	// by returning a copy.
	var validationError val.ValidationError

	// Bind struct fields recursively (handles embedded structs)
	if err := c.bindStructFields(rv, rt, &validationError); err != nil {
		return err
	}

	// Bind body fields (if any) - this handles json/body tagged fields
	if err := c.bindBodyFields(v, rt); err != nil {
		// Don't fail on body binding for GET requests without body
		if c.request.Method != gohttp.MethodGet && c.request.Method != gohttp.MethodHead && c.request.Method != gohttp.MethodDelete {
			return fmt.Errorf("failed to bind body: %w", err)
		}
	}

	// Validate all fields using their validation tags
	if err := c.validateStruct(v, rt, &validationError); err != nil {
		return err
	}

	// Return validation errors if any. The copy is what lets the value above
	// stay on the stack for the successful path.
	if validationError.HasErrors() {
		failed := validationError

		return &failed
	}

	return nil
}

// bindStructFields recursively binds struct fields, handling embedded structs.
func (c *Ctx) bindStructFields(rv reflect.Value, rt reflect.Type, errors *val.ValidationError) error {
	for i := range rt.NumField() {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		// Skip unexported fields
		if !field.IsExported() || !fieldValue.CanSet() {
			continue
		}

		// Handle embedded/anonymous struct fields - flatten them
		if field.Anonymous {
			// Check if the embedded field has explicit tags (would mean it's not truly flattened)
			hasExplicitTag := field.Tag.Get("path") != "" ||
				field.Tag.Get("query") != "" ||
				field.Tag.Get("header") != "" ||
				field.Tag.Get("form") != ""

			if !hasExplicitTag {
				// Get the embedded struct type
				embeddedType := field.Type
				embeddedValue := fieldValue

				// Handle pointer to struct
				if embeddedType.Kind() == reflect.Pointer {
					embeddedType = embeddedType.Elem()
					if embeddedValue.IsNil() {
						embeddedValue.Set(reflect.New(embeddedType))
					}

					embeddedValue = embeddedValue.Elem()
				}

				// Only recurse if it's a struct
				if embeddedType.Kind() == reflect.Struct {
					if err := c.bindStructFields(embeddedValue, embeddedType, errors); err != nil {
						return err
					}

					continue
				}
			}
		}

		// Bind based on tag priority: path -> query -> header -> form -> body/json
		if err := c.bindField(field, fieldValue, errors); err != nil {
			return err
		}
	}

	return nil
}

// bindField binds a single struct field from the appropriate source.
func (c *Ctx) bindField(field reflect.StructField, fieldValue reflect.Value, errors *val.ValidationError) error {
	// Check tags in priority order
	if pathTag := field.Tag.Get("path"); pathTag != "" {
		return c.bindPathParam(field, fieldValue, pathTag, errors)
	}

	if queryTag := field.Tag.Get("query"); queryTag != "" {
		return c.bindQueryParam(field, fieldValue, queryTag, errors)
	}

	if headerTag := field.Tag.Get("header"); headerTag != "" {
		return c.bindHeaderParam(field, fieldValue, headerTag, errors)
	}

	// Form fields only bind when the request actually carries a form-encoded
	// body. A struct may tag a field both `json:"x" form:"x"` so one endpoint
	// accepts either encoding; when the body is JSON this branch must stand
	// aside and let bindBodyFields decode it.
	if formTag := field.Tag.Get("form"); formTag != "" && formTag != "-" && c.hasFormBody() {
		return c.bindFormParam(field, fieldValue, formTag, errors)
	}

	// Body fields are handled separately in bindBodyFields
	return nil
}

// bindPathParam binds a path parameter.
func (c *Ctx) bindPathParam(field reflect.StructField, fieldValue reflect.Value, tag string, errors *val.ValidationError) error {
	paramName := parseTagName(tag)
	if paramName == "" {
		paramName = field.Name
	}

	value := c.Param(paramName)

	// Path params are always required
	if value == "" {
		errors.AddWithCode(paramName, "path parameter is required", val.ErrCodeRequired, nil)

		return nil
	}

	return setFieldValue(fieldValue, value, paramName, errors)
}

// bindQueryParam binds a query parameter.
func (c *Ctx) bindQueryParam(field reflect.StructField, fieldValue reflect.Value, tag string, errors *val.ValidationError) error {
	paramName := parseTagName(tag)
	if paramName == "" {
		paramName = field.Name
	}

	// Determine if field is required using consistent precedence:
	// 1. optional:"true" - explicitly optional (highest priority)
	// 2. required:"true" - explicitly required
	// 3. omitempty in tag - optional
	// 4. pointer type - optional
	// 5. default: non-pointer types are required
	required := isBindFieldRequired(field, tag)

	// Repeated parameters (resource=a&resource=b) fill a slice field. Reading
	// a single value through Query would keep the first occurrence and drop
	// the rest, which for something like an RFC 8707 resource indicator
	// silently narrows what the caller asked for.
	if isMultiValueTarget(fieldValue) {
		present := c.queryAll(paramName)
		if len(present) == 0 {
			if required {
				errors.AddWithCode(paramName, "query parameter is required", val.ErrCodeRequired, nil)

				return nil
			}

			if defaultVal := field.Tag.Get("default"); defaultVal != "" {
				present = strings.Split(defaultVal, ",")
			}
		}

		switch len(present) {
		case 0:
			return nil
		case 1:
			// A single occurrence goes through setFieldValue so that a
			// comma-separated value (scope=openid,profile) expands the same way
			// it always has. Splitting only ever applies to a lone value;
			// repeated parameters are taken verbatim.
			return setFieldValue(fieldValue, present[0], paramName, errors)
		default:
			return setSliceFieldValue(fieldValue, present, paramName, errors)
		}
	}

	value := c.Query(paramName)

	if required && value == "" {
		errors.AddWithCode(paramName, "query parameter is required", val.ErrCodeRequired, nil)

		return nil
	}

	// Use default if provided and value is empty
	if value == "" {
		if defaultVal := field.Tag.Get("default"); defaultVal != "" {
			value = defaultVal
		}
	}

	if value != "" {
		return setFieldValue(fieldValue, value, paramName, errors)
	}

	return nil
}

// bindHeaderParam binds a header parameter.
func (c *Ctx) bindHeaderParam(field reflect.StructField, fieldValue reflect.Value, tag string, errors *val.ValidationError) error {
	headerName := parseTagName(tag)
	if headerName == "" {
		headerName = field.Name
	}

	value := c.Header(headerName)

	// Determine if field is required using consistent precedence:
	// 1. optional:"true" - explicitly optional (highest priority)
	// 2. required:"true" - explicitly required
	// 3. omitempty in tag - optional
	// 4. pointer type - optional
	// 5. default: non-pointer types are required
	required := isBindFieldRequired(field, tag)

	if required && value == "" {
		errors.AddWithCode(headerName, "header is required", val.ErrCodeRequired, nil)

		return nil
	}

	// Use default if provided
	if value == "" {
		if defaultVal := field.Tag.Get("default"); defaultVal != "" {
			value = defaultVal
		}
	}

	if value != "" {
		return setFieldValue(fieldValue, value, headerName, errors)
	}

	return nil
}

// maxFormMemory bounds the in-memory portion of a parsed multipart form.
const maxFormMemory = 32 << 20

// hasFormBody reports whether the request carries a form-encoded body that
// form:"..." tagged fields can be read from.
func (c *Ctx) hasFormBody() bool {
	switch mediaType(c.request.Header.Get("Content-Type")) {
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return true
	default:
		return false
	}
}

// formValues parses the request form and returns the values that form:"..."
// tagged fields bind from.
//
// On methods that carry a body the values come from PostForm, the body alone.
// Form would additionally merge in the URL query, which would let
// POST /token?client_secret=... supply a credential the body never sent.
// RFC 6749 §4.1.3 requires token parameters in the body, so the merge is both
// a spec violation and a parameter-pollution surface. Bodyless methods fall
// back to the merged set so form: still resolves there.
//
// ParseForm and ParseMultipartForm are both idempotent, so calling this once
// per field is cheap.
func (c *Ctx) formValues() (url.Values, error) {
	if mediaType(c.request.Header.Get("Content-Type")) == "multipart/form-data" {
		if c.request.MultipartForm == nil {
			if err := c.request.ParseMultipartForm(maxFormMemory); err != nil {
				return nil, fmt.Errorf("failed to parse multipart form: %w", err)
			}
		}
	} else if err := c.request.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	switch c.request.Method {
	case gohttp.MethodPost, gohttp.MethodPut, gohttp.MethodPatch:
		return c.request.PostForm, nil
	default:
		return c.request.Form, nil
	}
}

// bindFormParam binds a field from an application/x-www-form-urlencoded or
// multipart/form-data body.
func (c *Ctx) bindFormParam(field reflect.StructField, fieldValue reflect.Value, tag string, errors *val.ValidationError) error {
	fieldName := parseTagName(tag)
	if fieldName == "" {
		fieldName = field.Name
	}

	values, err := c.formValues()
	if err != nil {
		return err
	}

	// Determine if field is required using consistent precedence:
	// 1. optional:"true" - explicitly optional (highest priority)
	// 2. required:"true" - explicitly required
	// 3. omitempty in tag - optional
	// 4. pointer type - optional
	// 5. default: non-pointer types are required
	required := isBindFieldRequired(field, tag)

	// Repeated parameters (scope=openid&scope=profile) fill a slice field.
	if isMultiValueTarget(fieldValue) {
		present := values[fieldName]
		if len(present) == 0 {
			if required {
				errors.AddWithCode(fieldName, "form field is required", val.ErrCodeRequired, nil)

				return nil
			}

			if defaultVal := field.Tag.Get("default"); defaultVal != "" {
				present = strings.Split(defaultVal, ",")
			}
		}

		switch len(present) {
		case 0:
			return nil
		case 1:
			// A single occurrence goes through setFieldValue so that a
			// comma-separated value (scope=openid,profile) expands the same way
			// it does for a query or header parameter. Splitting only ever
			// applies to a lone value; repeated parameters are taken verbatim.
			return setFieldValue(fieldValue, present[0], fieldName, errors)
		default:
			return setSliceFieldValue(fieldValue, present, fieldName, errors)
		}
	}

	value := values.Get(fieldName)

	if required && value == "" {
		errors.AddWithCode(fieldName, "form field is required", val.ErrCodeRequired, nil)

		return nil
	}

	// Use default if provided and value is empty
	if value == "" {
		if defaultVal := field.Tag.Get("default"); defaultVal != "" {
			value = defaultVal
		}
	}

	if value != "" {
		return setFieldValue(fieldValue, value, fieldName, errors)
	}

	return nil
}

// structHasFormFields reports whether any field of rt, including fields of
// flattened embedded structs, binds from the form body.
func structHasFormFields(rt reflect.Type) bool {
	for i := range rt.NumField() {
		field := rt.Field(i)

		if tag := field.Tag.Get("form"); tag != "" && tag != "-" {
			return true
		}

		if !field.Anonymous {
			continue
		}

		embedded := field.Type
		if embedded.Kind() == reflect.Pointer {
			embedded = embedded.Elem()
		}

		if embedded.Kind() == reflect.Struct && structHasFormFields(embedded) {
			return true
		}
	}

	return false
}

// mediaType strips any parameters (charset, boundary) from a Content-Type so
// the bare type can be named in an error.
func mediaType(contentType string) string {
	base, _, _ := strings.Cut(contentType, ";")

	return strings.TrimSpace(base)
}

// bindBodyFields binds body/json tagged fields.
func (c *Ctx) bindBodyFields(v any, rt reflect.Type) error {
	// Check if struct has body fields
	hasBodyFields := false

	for i := range rt.NumField() {
		field := rt.Field(i)
		if field.Tag.Get("path") == "" &&
			field.Tag.Get("query") == "" &&
			field.Tag.Get("header") == "" {
			// Check if has json or body tag
			if field.Tag.Get("json") != "" && field.Tag.Get("json") != "-" {
				hasBodyFields = true

				break
			}

			if field.Tag.Get("body") != "" && field.Tag.Get("body") != "-" {
				hasBodyFields = true

				break
			}
		}
	}

	if !hasBodyFields {
		return nil
	}

	if c.hasFormBody() {
		// Fields tagged form:"..." were already read by bindFormParam, and a
		// form body holds nothing else for Bind to decode.
		if structHasFormFields(rt) {
			return nil
		}

		// The struct declares body fields, but none of them opted into form
		// binding, so not one would be populated. Reporting success here is how
		// a form-encoded POST used to reach its handler holding an entirely
		// zero request, failing later with an error that named the wrong cause.
		return fmt.Errorf("cannot bind %s body into %s: no field is tagged `form:\"...\"`",
			mediaType(c.request.Header.Get("Content-Type")), rt.String())
	}

	// Bind body content using existing Bind method
	return c.Bind(v)
}

// parseTagName extracts the parameter name from a tag value
// Handles formats like: "paramName", "paramName,omitempty".
func parseTagName(tag string) string {
	if name, _, ok := strings.Cut(tag, ","); ok {
		return strings.TrimSpace(name)
	}

	return strings.TrimSpace(tag)
}

// isBindFieldRequired determines if a field is required for binding.
// Uses consistent precedence order:
// 1. optional:"true" - explicitly optional (highest priority)
// 2. required:"true" - explicitly required
// 3. default:"..." - fields with defaults are implicitly optional
// 4. omitempty in tag - optional
// 5. pointer type - optional
// 6. default: non-pointer types are required.
func isBindFieldRequired(field reflect.StructField, tag string) bool {
	// 1. Explicit optional tag takes precedence (opt-out)
	if field.Tag.Get("optional") == "true" {
		return false
	}

	// 2. Explicit required tag
	if field.Tag.Get("required") == "true" {
		return true
	}

	// 3. Fields with default values are implicitly optional
	if field.Tag.Get("default") != "" {
		return false
	}

	// 4. Check for omitempty in the parameter tag (query, header, etc.)
	if strings.Contains(tag, ",omitempty") {
		return false
	}

	// 5. Check JSON tag for omitempty (for body fields)
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		if strings.Contains(jsonTag, ",omitempty") {
			return false
		}
	}

	// 6. Pointer types are optional by default
	if field.Type.Kind() == reflect.Pointer {
		return false
	}

	// 7. Non-pointer types without above markers are required
	return true
}

// setFieldValue sets a field value from a string, converting to the appropriate type.
// Supports types that implement encoding.TextUnmarshaler (e.g., xid.ID, uuid.UUID).
func setFieldValue(fieldValue reflect.Value, value string, fieldName string, errors *val.ValidationError) error {
	// Handle pointer types first - create the value if nil, then recurse
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}

		return setFieldValue(fieldValue.Elem(), value, fieldName, errors)
	}

	// Check if the type implements encoding.TextUnmarshaler
	// This handles types like xid.ID, uuid.UUID, time.Time, etc.
	if handled := tryTextUnmarshaler(fieldValue, value, fieldName, errors); handled {
		return nil
	}

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			errors.AddWithCode(fieldName, "invalid integer value", val.ErrCodeInvalidType, value)

			return err
		}

		fieldValue.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			errors.AddWithCode(fieldName, "invalid unsigned integer value", val.ErrCodeInvalidType, value)

			return err
		}

		fieldValue.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			errors.AddWithCode(fieldName, "invalid float value", val.ErrCodeInvalidType, value)

			return err
		}

		fieldValue.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			errors.AddWithCode(fieldName, "invalid boolean value", val.ErrCodeInvalidType, value)

			return err
		}

		fieldValue.SetBool(boolVal)

	case reflect.Slice:
		// []byte is a slice by type but a single scalar in practice.
		if fieldValue.Type().Elem().Kind() == reflect.Uint8 {
			fieldValue.SetBytes([]byte(value))

			return nil
		}

		// A lone comma-separated value (scopes=read,write) expands into a
		// slice. Repeated parameters take the setSliceFieldValue path instead.
		return setSliceFieldValue(fieldValue, strings.Split(value, ","), fieldName, errors)

	default:
		errors.AddWithCode(fieldName, fmt.Sprintf("unsupported field type: %s", fieldValue.Kind()), val.ErrCodeInvalidType, value)
	}

	return nil
}

// isMultiValueTarget reports whether a field should be filled from every
// occurrence of a repeated parameter rather than from the first one only.
// []byte is excluded, since it stands for a single scalar value.
func isMultiValueTarget(fieldValue reflect.Value) bool {
	t := fieldValue.Type()
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	return t.Kind() == reflect.Slice && t.Elem().Kind() != reflect.Uint8
}

// setSliceFieldValue fills a slice field from a repeated parameter
// (scope=openid&scope=profile), converting each element through setFieldValue.
func setSliceFieldValue(fieldValue reflect.Value, values []string, fieldName string, errors *val.ValidationError) error {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}

		return setSliceFieldValue(fieldValue.Elem(), values, fieldName, errors)
	}

	// Nested slices have no unambiguous wire form, and allowing them here would
	// let setFieldValue and this function recurse into each other.
	if elemKind := fieldValue.Type().Elem().Kind(); elemKind == reflect.Slice || elemKind == reflect.Array {
		errors.AddWithCode(fieldName, fmt.Sprintf("unsupported field type: %s", fieldValue.Type()), val.ErrCodeInvalidType, nil)

		return nil
	}

	slice := reflect.MakeSlice(fieldValue.Type(), len(values), len(values))

	for i, value := range values {
		if err := setFieldValue(slice.Index(i), value, fieldName, errors); err != nil {
			return err
		}
	}

	fieldValue.Set(slice)

	return nil
}

// tryTextUnmarshaler attempts to use encoding.TextUnmarshaler if the type implements it.
// Returns true if the type was handled (either successfully or with an error added).
// textUnmarshalerType is looked up once. Whether a type implements
// encoding.TextUnmarshaler is a property of the TYPE, so it is answered with
// reflect.Type.Implements rather than by boxing a value into an interface.
//
// The boxing version cost an allocation per field per request: .Interface() on
// an addressable field allocates to build the any, and it did that for every
// field just to discover that string does not implement TextUnmarshaler. On a
// five field struct that was five allocations a request, and reflect.unsafe_New
// was 39% of the bind path's allocations.
var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

func tryTextUnmarshaler(fieldValue reflect.Value, value string, fieldName string, errors *val.ValidationError) bool {
	fieldType := fieldValue.Type()

	// Pointer receiver implementations, which is the common case.
	if fieldValue.CanAddr() && reflect.PointerTo(fieldType).Implements(textUnmarshalerType) {
		unmarshaler, ok := fieldValue.Addr().Interface().(encoding.TextUnmarshaler)
		if ok {
			if err := unmarshaler.UnmarshalText([]byte(value)); err != nil {
				errors.AddWithCode(fieldName, fmt.Sprintf("invalid value: %v", err), val.ErrCodeInvalidType, value)
			}

			return true
		}
	}

	// Value receiver implementations, which are rare.
	if fieldType.Implements(textUnmarshalerType) && fieldValue.CanInterface() {
		unmarshaler, ok := fieldValue.Interface().(encoding.TextUnmarshaler)
		if ok {
			if err := unmarshaler.UnmarshalText([]byte(value)); err != nil {
				errors.AddWithCode(fieldName, fmt.Sprintf("invalid value: %v", err), val.ErrCodeInvalidType, value)
			}

			return true
		}
	}

	return false
}
