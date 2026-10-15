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
	spanIDKey
)

// Context helper functions

// WithLogger adds a logger to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext extracts a logger from the context.
func LoggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return GetGlobalLogger()
	}

	if l, ok := ctx.Value(loggerKey).(Logger); ok {
		return l
	}

	return GetGlobalLogger()
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
	return withID(ctx, traceIDKey, traceID)
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

// WithSpanID adds a span ID to the context.
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return withID(ctx, spanIDKey, spanID)
}

// idCtx carries a trace ID, a span ID or both in one context node. A tracing
// middleware sets both on every request, and two context.WithValue layers
// cost four allocations (each boxes its string); one idCtx costs one, plus
// one more when the second ID replaces the node.
type idCtx struct {
	context.Context //nolint:containedctx // idCtx is a context; it wraps its parent the way context.WithValue does

	traceID, spanID   string
	hasTrace, hasSpan bool
}

func (c *idCtx) Value(key any) any {
	switch key {
	case traceIDKey:
		if c.hasTrace {
			return c.traceID
		}
	case spanIDKey:
		if c.hasSpan {
			return c.spanID
		}
	}

	return c.Context.Value(key)
}

// withID sets one ID. When ctx is itself an idCtx, the result is a copy of it
// with the ID set, so the pair shares a node and the parent is left as it was.
func withID(ctx context.Context, key contextKey, id string) context.Context {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	c, ok := ctx.(*idCtx)
	if ok {
		cp := *c
		c = &cp
	} else {
		c = &idCtx{Context: ctx}
	}

	if key == traceIDKey {
		c.traceID, c.hasTrace = id, true
	} else {
		c.spanID, c.hasSpan = id, true
	}

	return c
}

// SpanIDFromContext extracts the span ID from the context.
func SpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if id, ok := ctx.Value(spanIDKey).(string); ok {
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
