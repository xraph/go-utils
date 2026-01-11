package log

// F creates a new field.
func F(key string, value any) Field {
	return Any(key, value)
}
