package log

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"
)

// stringerVal is a comparable fmt.Stringer used to exercise the Stringer
// field type without dragging in a full test double.
type stringerVal string

func (s stringerVal) String() string { return string(s) }

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
		{"stringer", Stringer("k", stringerVal("hi")), stringerVal("hi")},
		{"any", Any("k", 99), 99},
		{"conditional true", Conditional(true, "k", 5), 5},
		{"conditional false", Conditional(false, "k", 5), nil},
		{"nullable nil", Nullable("k", nil), "null"},
		{"nullable value", Nullable("k", 7), 7},
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

	// Strings holds a slice, which is not comparable with !=, so it gets its
	// own assertion rather than a slot in the table above.
	gotStrings, ok := Strings("k", []string{"a", "b"}).Value().([]string)
	if !ok || len(gotStrings) != 2 || gotStrings[0] != "a" || gotStrings[1] != "b" {
		t.Errorf("Strings round trip = %#v, want [a b]", gotStrings)
	}
}

func TestTimeSurvivesOutOfRangeValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   time.Time
	}{
		{"zero", time.Time{}},
		{"far future", time.Date(2300, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"far past", time.Date(1500, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"in range", time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Time("t", tc.in).Value().(time.Time)
			if !ok {
				t.Fatalf("Value() is not a time.Time: %T", Time("t", tc.in).Value())
			}

			if !got.Equal(tc.in) {
				t.Errorf("round trip = %v, want %v", got, tc.in)
			}
		})
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

// TestFloat32WidensToFloat64 pins down current behaviour rather than asserting
// a round trip: Float32 is implemented in terms of Float64(key, float64(val)),
// so a float32 that is not exactly representable widens with the extra binary
// digits float32->float64 conversion introduces. zap kept a distinct float32
// wire type and avoided this; this package does not, trading that precision
// for one fewer field type. If this assertion ever needs to change, that is a
// deliberate behaviour change, not a bug fix.
func TestFloat32WidensToFloat64(t *testing.T) {
	const want = 1.100000023841858

	got := Float32("r", 1.1).Value().(float64)
	if got != want {
		t.Errorf("Float32(1.1).Value() = %v, want %v", got, want)
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
	skipUnderRace(t)

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
