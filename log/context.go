package log

import (
	"context"
)

// Context keys.
type contextKey int

const (
	loggerKey contextKey = iota
	requestIDKey
	traceIDKey
	userIDKey
)

// Context helper functions

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext extracts a logger from the context.
//
// The global-logger fallback (GetGlobalLogger) was removed with the rest of
// the zap-backed implementation in this task; a later task reintroduces a
// default logger fallback. Until then this returns nil when no logger has
// been attached to the context.
func LoggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return nil
	}

	if l, ok := ctx.Value(loggerKey).(Logger); ok {
		return l
	}

	return nil
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}

	return ""
}

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext extracts the trace ID from the context.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}

	return ""
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}

	return ""
}
