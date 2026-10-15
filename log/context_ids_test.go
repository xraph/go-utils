package log

import (
	"context"
	"strings"
	"testing"
	"time"
)

var idSink context.Context

// A tracing middleware sets both IDs on every request. Setting the span ID
// right after the trace ID shares one context node, so the pair costs two
// allocations, not four.
func TestTraceAndSpanIDsAllocateOnce(t *testing.T) {
	ctx := context.Background()

	// Runtime strings and an escaping context, as in a real middleware:
	// constants box for free and a context that never escapes stays on the
	// stack, which would hide the cost.
	traceID := strings.Repeat("a", 32)
	spanID := strings.Repeat("b", 16)

	allocs := testing.AllocsPerRun(100, func() {
		idSink = WithSpanID(WithTraceID(ctx, traceID), spanID) //nolint:fatcontext // the sink makes the context escape, which is the point
	})

	if allocs > 2 {
		t.Fatalf("WithTraceID then WithSpanID allocated %.0f times, want at most 2", allocs)
	}
}

func TestTraceAndSpanIDsRoundTrip(t *testing.T) {
	ctx := WithSpanID(WithTraceID(context.Background(), "trace-1"), "span-1")

	if got := TraceIDFromContext(ctx); got != "trace-1" {
		t.Fatalf("trace ID = %q, want trace-1", got)
	}

	if got := SpanIDFromContext(ctx); got != "span-1" {
		t.Fatalf("span ID = %q, want span-1", got)
	}

	// The parent keeps its own view: no span ID was ever set on it.
	parent := WithTraceID(context.Background(), "trace-2")
	_ = WithSpanID(parent, "span-2")

	if got := SpanIDFromContext(parent); got != "" {
		t.Fatalf("setting a span ID leaked into the parent context: %q", got)
	}
}

// The innermost setting wins, whichever helper set it and in whatever order.
func TestTraceAndSpanIDsInnermostWins(t *testing.T) {
	ctx := WithTraceID(context.Background(), "outer-trace")
	ctx = WithSpanID(ctx, "outer-span")
	ctx = context.WithValue(ctx, requestIDKey, "req")
	ctx = WithSpanID(ctx, "inner-span")
	ctx = WithTraceID(ctx, "inner-trace")

	if got := TraceIDFromContext(ctx); got != "inner-trace" {
		t.Fatalf("trace ID = %q, want inner-trace", got)
	}

	if got := SpanIDFromContext(ctx); got != "inner-span" {
		t.Fatalf("span ID = %q, want inner-span", got)
	}

	if got := RequestIDFromContext(ctx); got != "req" {
		t.Fatalf("request ID = %q, want req", got)
	}
}

// The ID context still cancels, carries deadlines and passes other values
// through from its parent.
func TestTraceAndSpanIDsKeepParentBehaviour(t *testing.T) {
	parent, cancel := context.WithTimeout(context.WithValue(context.Background(), userIDKey, "u1"), time.Hour)
	ctx := WithSpanID(WithTraceID(parent, "t"), "s")

	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("deadline lost")
	}

	if got := UserIDFromContext(ctx); got != "u1" {
		t.Fatalf("user ID = %q, want u1", got)
	}

	child, childCancel := context.WithCancel(ctx)
	defer childCancel()

	cancel()

	select {
	case <-child.Done():
	case <-time.After(time.Second):
		t.Fatal("cancelling the parent did not reach a context derived from the ID context")
	}
}
