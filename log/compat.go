package log

// Deprecated constructors. They all forward to New and return the Logger
// interface. The old *BeautifulLogger concrete type and its chainable
// WithXxx options are gone; use Config instead.

// NewDevelopmentLogger returns a pretty logger at debug level.
//
// Deprecated: use New(Config{Format: FormatPretty, Level: LevelDebug}).
func NewDevelopmentLogger() Logger {
	return New(Config{Format: FormatPretty, Level: LevelDebug, AddCaller: true})
}

// NewProductionLogger returns a JSON logger at info level.
//
// Deprecated: use New(Config{Format: FormatJSON}).
func NewProductionLogger() Logger {
	return New(Config{Format: FormatJSON, Level: LevelInfo, AddCaller: true})
}

// NewBeautifulLogger returns a logger that picks its format automatically.
//
// Deprecated: use New(Config{Name: name}).
func NewBeautifulLogger(name string) Logger {
	return New(Config{Name: name, AddCaller: true})
}

// NewBeautifulLoggerCompact returns a logger that picks its format
// automatically and omits caller information.
//
// Deprecated: use New(Config{Name: name}).
func NewBeautifulLoggerCompact(name string) Logger {
	return New(Config{Name: name})
}

// NewBeautifulLoggerMinimal returns a logger that picks its format
// automatically and omits caller information.
//
// Deprecated: use New(Config{Name: name}). Unlike the previous minimal mode,
// this one does not discard fields.
func NewBeautifulLoggerMinimal(name string) Logger {
	return New(Config{Name: name})
}

// NewBeautifulLoggerJSON returns a JSON logger.
//
// Deprecated: use New(Config{Name: name, Format: FormatJSON}). Unlike the
// previous version, this one really does emit JSON.
func NewBeautifulLoggerJSON(name string) Logger {
	return New(Config{Name: name, Format: FormatJSON, AddCaller: true})
}
