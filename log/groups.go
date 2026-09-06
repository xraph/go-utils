package log

import "time"

// FieldGroup represents a group of related fields.
type FieldGroup struct {
	fields []Field
}

// NewFieldGroup creates a new field group.
func NewFieldGroup(fields ...Field) *FieldGroup {
	return &FieldGroup{fields: fields}
}

// Add adds fields to the group.
func (fg *FieldGroup) Add(fields ...Field) *FieldGroup {
	fg.fields = append(fg.fields, fields...)

	return fg
}

// Fields returns all fields in the group.
func (fg *FieldGroup) Fields() []Field {
	return fg.fields
}

// Predefined field groups.
var (
	// HTTPRequestGroup creates a group of HTTP request fields.
	HTTPRequestGroup = func(method, path, userAgent string, status int) *FieldGroup {
		return NewFieldGroup(
			HTTPMethod(method),
			HTTPPath(path),
			HTTPUserAgent(userAgent),
			HTTPStatus(status),
		)
	}

	// DatabaseQueryGroup creates a group of database query fields.
	DatabaseQueryGroup = func(query, table string, rows int64, duration time.Duration) *FieldGroup {
		return NewFieldGroup(
			DatabaseQuery(query),
			DatabaseTable(table),
			DatabaseRows(rows),
			Duration("query_duration", duration),
		)
	}

	// ServiceInfoGroup creates a group of service information fields.
	ServiceInfoGroup = func(name, version, environment string) *FieldGroup {
		return NewFieldGroup(
			ServiceName(name),
			ServiceVersion(version),
			ServiceEnvironment(environment),
		)
	}
)
