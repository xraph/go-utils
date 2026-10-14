package log

import (
	"context"
	"testing"
)

func TestSpanIDRoundTrip(t *testing.T) {
	ctx := WithSpanID(context.Background(), "00f067aa0ba902b7")

	if got := SpanIDFromContext(ctx); got != "00f067aa0ba902b7" {
		t.Fatalf("SpanIDFromContext = %q, want 00f067aa0ba902b7", got)
	}
}

func TestSpanIDFromNilContext(t *testing.T) {
	//nolint:staticcheck // a nil context is the case under test
	if got := SpanIDFromContext(nil); got != "" {
		t.Fatalf("SpanIDFromContext(nil) = %q, want empty", got)
	}
}

func TestContextFieldsOrderWithSpanID(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTraceID(ctx, "4bf92f3577b34da6a3ce929d0e0e4736")
	ctx = WithSpanID(ctx, "00f067aa0ba902b7")
	ctx = WithUserID(ctx, "user-9")

	fields := ContextFields(ctx)

	want := []string{"request_id", "trace_id", "span_id", "user_id"}
	if len(fields) != len(want) {
		t.Fatalf("got %d fields, want %d", len(fields), len(want))
	}

	for i, f := range fields {
		if f.Key() != want[i] {
			t.Fatalf("field %d key = %q, want %q", i, f.Key(), want[i])
		}
	}
}

func TestContextFieldsSpanIDWithoutTraceID(t *testing.T) {
	ctx := WithSpanID(context.Background(), "00f067aa0ba902b7")

	fields := ContextFields(ctx)
	if len(fields) != 1 || fields[0].Key() != "span_id" {
		t.Fatalf("want a single span_id field, got %v", fields)
	}
}

func TestSpanIDFieldEmptyMatchesTraceIDShape(t *testing.T) {
	empty := context.Background()

	span := SpanID(empty)
	trace := TraceID(empty)

	// Field has no Type accessor, so compare the unexported type tag directly.
	if span.typ != trace.typ {
		t.Fatalf("empty SpanID field type = %v, want %v (same as TraceID)", span.typ, trace.typ)
	}

	if span.Key() != "span_id" {
		t.Fatalf("SpanID key = %q, want span_id", span.Key())
	}
}
