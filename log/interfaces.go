package log

import (
	"context"
)

// Logger represents the logging interface.
type Logger interface {
	// Logging levels
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// Formatted logging
	Debugf(template string, args ...any)
	Infof(template string, args ...any)
	Warnf(template string, args ...any)
	Errorf(template string, args ...any)
	Fatalf(template string, args ...any)

	// Context and enrichment
	With(fields ...Field) Logger
	WithContext(ctx context.Context) Logger
	Named(name string) Logger

	// Sugar logger
	Sugar() SugarLogger

	// Utilities
	Sync() error
}

// SugarLogger provides a more flexible API.
type SugarLogger interface {
	Debugw(msg string, keysAndValues ...any)
	Infow(msg string, keysAndValues ...any)
	Warnw(msg string, keysAndValues ...any)
	Errorw(msg string, keysAndValues ...any)
	Fatalw(msg string, keysAndValues ...any)

	With(args ...any) SugarLogger
}
