package log

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Regression test for finding 8. LoggingConfig.Output was never read and the
// logger always wrote to os.Stdout, so it could not be captured.
func TestOutputIsHonoured(t *testing.T) {
	var buf bytes.Buffer
	l := New(Config{Format: FormatJSON, Output: &buf, Level: LevelInfo})

	l.Info("captured")

	if !strings.Contains(buf.String(), "captured") {
		t.Errorf("nothing reached the configured Output, buf = %q", buf.String())
	}
}

func TestNewNeverReturnsNil(t *testing.T) {
	for _, cfg := range []Config{
		{},
		{Format: FormatJSON},
		{Format: FormatPretty},
		{Format: "garbage"},
		{Level: "garbage"},
	} {
		if l := New(cfg); l == nil {
			t.Errorf("New(%+v) returned nil", cfg)
		}
	}
}

func TestNewWithNilOutputDoesNotPanic(t *testing.T) {
	l := New(Config{Format: FormatJSON, Output: nil})
	l.Info("should not panic") // goes to stderr
}

func TestFormatJSONProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	New(Config{Format: FormatJSON, Output: &buf}).Info("msg", String("k", "v"))

	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("FormatJSON did not produce JSON: %v\n%s", err, buf.String())
	}
}

func TestFormatPrettyProducesText(t *testing.T) {
	var buf bytes.Buffer
	New(Config{Format: FormatPretty, Output: &buf}).Info("msg", String("k", "v"))

	out := buf.String()
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("FormatPretty produced JSON: %q", out)
	}
	if !strings.Contains(out, "k=v") {
		t.Errorf("pretty output missing the field: %q", out)
	}
}

func TestLoggingConfigNameReachesTheOutput(t *testing.T) {
	// Name is a new field on LoggingConfig; confirm it actually lands in the
	// encoded line rather than being dropped on the way through New.
	var buf bytes.Buffer
	l := New(Config{Format: FormatJSON, Output: &buf, Name: "forge.http"})
	l.Info("msg")

	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatal(err)
	}
	if m["logger"] != "forge.http" {
		t.Errorf("logger = %v, want forge.http", m["logger"])
	}
}

func TestLoggingConfigStillWorks(t *testing.T) {
	// The old entry point, unchanged, must keep working.
	var buf bytes.Buffer
	l := NewLogger(LoggingConfig{Level: "debug", Format: "json", Output: "", Environment: "production"})
	if l == nil {
		t.Fatal("NewLogger returned nil")
	}
	_ = buf
	l.Debug("debug is enabled")
}

// Regression test for finding 3. SetGlobalLogger type-asserted to the
// unexported *logger and silently discarded everything else.
func TestSetGlobalLoggerAcceptsEveryImplementation(t *testing.T) {
	original := GetGlobalLogger()
	t.Cleanup(func() { SetGlobalLogger(original) })

	var buf bytes.Buffer
	candidates := []Logger{
		NewNoopLogger(),
		NewTestLogger(),
		New(Config{Format: FormatJSON, Output: &buf}),
	}

	for _, want := range candidates {
		SetGlobalLogger(want)
		got := GetGlobalLogger()
		if got != want {
			t.Errorf("SetGlobalLogger(%T) then GetGlobalLogger() returned %T, want the same value", want, got)
		}
	}
}

func TestGlobalLoggerDefaultsToNoopUnderTest(t *testing.T) {
	// The lazily created global must not flood the suite. Which logger that
	// means depends on how the suite was invoked, and this repository's
	// `make test` passes -v, so the test has to handle both branches or it
	// fails exactly when run the normal way.
	resetGlobalLogger()
	got := GetGlobalLogger()
	_, isNoop := got.(noopLogger)

	if testVerbose() {
		if isNoop {
			t.Error("under go test -v the global should be a real logger, got noop")
		}
		return
	}
	if !isNoop {
		t.Errorf("under plain go test the global logger is %T, want noopLogger", got)
	}
}

// Task 0 temporarily made LoggerFromContext return nil because GetGlobalLogger
// did not exist yet. Track and TrackWithFields dereference the result, so a nil
// return is a panic waiting to happen.
func TestLoggerFromContextNeverReturnsNil(t *testing.T) {
	if got := LoggerFromContext(nil); got == nil {
		t.Error("LoggerFromContext(nil) returned nil")
	}
	if got := LoggerFromContext(context.Background()); got == nil {
		t.Error("LoggerFromContext(empty ctx) returned nil")
	}

	var buf bytes.Buffer
	want := New(Config{Format: FormatJSON, Output: &buf})
	ctx := WithLogger(context.Background(), want)
	if got := LoggerFromContext(ctx); got != want {
		t.Errorf("LoggerFromContext returned %T, want the logger that was stored", got)
	}
}

// Track dereferences whatever LoggerFromContext returns.
func TestTrackDoesNotPanicWithoutALoggerInContext(t *testing.T) {
	done := Track(context.Background(), "op")
	done() // must not panic
}
