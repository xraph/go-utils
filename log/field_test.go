package log

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

func TestFieldValueRoundTrip(t *testing.T) {
	err := errors.New("boom")
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name string
		f    Field
		want any
	}{
		{"string", String("k", "v"), "v"},
		{"int", Int("k", 42), int64(42)},
		{"int64", Int64("k", -7), int64(-7)},
		{"uint64", Uint64("k", 9), uint64(9)},
		{"float64", Float64("k", 1.5), 1.5},
		{"bool true", Bool("k", true), true},
		{"bool false", Bool("k", false), false},
		{"duration", Duration("k", 3*time.Second), 3 * time.Second},
		{"error", Error(err), err},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.f.Value(); got != c.want {
				t.Errorf("Value() = %#v, want %#v", got, c.want)
			}
			if c.f.Key() == "" {
				t.Error("Key() must not be empty")
			}
		})
	}

	if got := Time("k", now).Value().(time.Time); !got.Equal(now) {
		t.Errorf("Time round trip = %v, want %v", got, now)
	}
}

func TestFieldFloatPrecision(t *testing.T) {
	// Floats are stored in the int64 payload via math.Float64bits, so the
	// round trip has to be exact, including for values that are awkward in
	// binary.
	for _, v := range []float64{0, -0.1, 1.0 / 3.0, math.MaxFloat64, math.SmallestNonzeroFloat64} {
		if got := Float64("k", v).Value().(float64); got != v {
			t.Errorf("Float64 round trip = %v, want %v", got, v)
		}
	}
}

func TestLazyFieldEvaluatesOnRead(t *testing.T) {
	calls := 0
	f := Lazy("k", func() any {
		calls++
		return "computed"
	})
	if calls != 0 {
		t.Fatalf("Lazy must not evaluate at construction, got %d calls", calls)
	}
	if got := f.Value(); got != "computed" {
		t.Errorf("Value() = %v, want computed", got)
	}
	if calls != 1 {
		t.Errorf("evaluated %d times, want 1", calls)
	}
}

func TestErrorFieldKeyIsError(t *testing.T) {
	if got := Error(errors.New("x")).Key(); got != "error" {
		t.Errorf("Error().Key() = %q, want error", got)
	}
}

func TestFieldDoesNotAllocateForScalars(t *testing.T) {
	// The whole point of the struct: scalar fields never touch the any slot,
	// so building one must not allocate.
	got := testing.AllocsPerRun(100, func() {
		_ = String("method", "GET")
		_ = Int("status", 200)
		_ = Bool("tls", false)
		_ = Duration("latency", time.Second)
	})
	if got != 0 {
		t.Errorf("scalar field construction allocated %.0f times, want 0", got)
	}
}

func TestFieldKeysMatchTheHistoricalNames(t *testing.T) {
	cases := map[string]Field{
		"http.method":         HTTPMethod("GET"),
		"http.status":         HTTPStatus(200),
		"http.path":           HTTPPath("/v1"),
		"http.user_agent":     HTTPUserAgent("curl"),
		"db.query":            DatabaseQuery("SELECT 1"),
		"db.table":            DatabaseTable("users"),
		"db.rows":             DatabaseRows(3),
		"service.name":        ServiceName("api"),
		"service.version":     ServiceVersion("1.0"),
		"service.environment": ServiceEnvironment("prod"),
		"latency.ms":          LatencyMs(time.Second),
		"memory.usage":        MemoryUsage(1024),
	}
	for want, f := range cases {
		if got := f.Key(); got != want {
			t.Errorf("key = %q, want %q", got, want)
		}
	}
	if got := HTTPURL(nil).Key(); got != "http.url" {
		t.Errorf("HTTPURL key = %q, want http.url", got)
	}
}

func TestContextFieldHelpersSkipWhenAbsent(t *testing.T) {
	ctx := context.Background()
	for _, f := range []Field{RequestID(ctx), TraceID(ctx), UserID(ctx)} {
		if f.typ != unknownType {
			t.Errorf("%s on an empty context has typ %v, want unknownType", f.Key(), f.typ)
		}
	}
	withID := WithRequestID(ctx, "req1")
	if got := RequestID(withID).Value(); got != "req1" {
		t.Errorf("RequestID value = %v, want req1", got)
	}
}
