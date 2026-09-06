package log

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Regression test for finding 8. LoggingConfig.Output was never read and the
// logger always wrote to os.Stdout, so it could not be captured.
// forceFormat pins the resolved format for one test. Format resolution consults
// the test binary's own -v flag, so a test that builds a logger through
// FormatAuto is measuring how the suite was invoked: noop under `go test`,
// pretty under `go test -v`. Every test that cares which logger it gets must
// pin the format, or it changes meaning depending on who runs it. This
// repository's `make test` passes -v, which is the invocation least likely to
// reveal the problem.
func forceFormat(t *testing.T, f Format) {
	t.Helper()
	t.Setenv("FORGE_LOG_FORMAT", string(f))
}

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

// NewLogger falls back to stderr when the configured path cannot be opened,
// rather than returning a logger that panics on first use.
// A log file holds request paths, user ids and error text, so it must not be
// created world-readable. gosec flags this as G302; the test pins the property
// rather than relying on the linter to keep noticing.
func TestNewLoggerCreatesTheLogFilePrivate(t *testing.T) {
	// Without this the logger is a noop under plain `go test`, which has no
	// file to check the permissions of and no Close to call.
	forceFormat(t, FormatJSON)

	path := filepath.Join(t.TempDir(), "app.log")

	l := NewLogger(LoggingConfig{Level: "info", Output: path})
	l.Info("written")

	// Release the file before t.TempDir's cleanup runs. POSIX happily unlinks
	// an open file; Windows refuses, so without this the cleanup fails with
	// "The process cannot access the file because it is being used by another
	// process" and the test goes red for a reason unrelated to what it asserts.
	if c, ok := l.(io.Closer); !ok {
		t.Fatal("a file-backed logger must be closeable, got no io.Closer")
	} else if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("log file was not created: %v", err)
	}

	// Windows has no POSIX permission bits. OpenFile's perm argument only
	// decides whether the read-only attribute gets set, and Stat reports a
	// synthesised 0666 for any writable file and 0444 for a read-only one, so
	// the 0600 this asserts can never be observed there. Access is governed by
	// ACLs instead, which this library does not touch. Assert the property on
	// the platforms where it exists rather than weakening it everywhere.
	if runtime.GOOS == "windows" {
		if info.Mode().Perm()&0o200 == 0 {
			t.Errorf("log file is not writable: mode %04o", info.Mode().Perm())
		}

		t.Skip("file permission bits are not meaningful on Windows")
	}

	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("log file mode = %04o, want 0600", got)
	}
}

// Close must release only what this package opened. A writer the caller handed
// in, and os.Stdout/os.Stderr, stay open: closing them would be a surprise, and
// closing os.Stderr would take out the process's error reporting.
func TestCloseDoesNotTouchAWriterTheCallerOwns(t *testing.T) {
	caller := &closeSpy{}

	l := New(Config{Format: FormatJSON, Output: caller})
	l.Info("written")

	c, ok := l.(io.Closer)
	if !ok {
		t.Fatal("logger is not an io.Closer")
	}

	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if caller.closed {
		t.Error("Close closed a writer the caller supplied and still owns")
	}
}

// A logger whose format resolves to noop must not sit on the file handle it
// opened, or a test binary that never logs still blocks deleting the file.
func TestNoopModeReleasesTheFileItOpened(t *testing.T) {
	// The noop outcome only happens in a test binary WITHOUT -v. An explicit
	// FORGE_LOG_FORMAT outranks test silence and -v resolves to pretty, so
	// there is no way to force this branch: say so rather than let the test
	// quietly stop exercising it under `make test`, which passes -v.
	if testVerbose() {
		t.Skip("format resolves to pretty under -v; this test targets the noop branch")
	}

	t.Setenv("FORGE_LOG_FORMAT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "unused.log")

	// Under a test binary with no -v and no explicit format, this resolves to
	// noop, so the file is opened and immediately unneeded.
	NewLogger(LoggingConfig{Level: "info", Output: path})

	if err := os.Remove(path); err != nil {
		t.Errorf("noop-mode logger left the file open: %v", err)
	}
}

type closeSpy struct {
	bytes.Buffer

	closed bool
}

func (c *closeSpy) Close() error {
	c.closed = true

	return nil
}

func TestNewLoggerFallsBackToStderrOnUnopenablePath(t *testing.T) {
	// A path inside a directory that does not exist, so O_CREATE fails on every
	// platform. A hardcoded "/nonexistent-dir-xyz/..." would resolve to the
	// current drive root on Windows, and would silently stop testing the
	// fallback the day somebody happened to create that directory.
	path := filepath.Join(t.TempDir(), "no-such-dir", "app.log")

	l := NewLogger(LoggingConfig{Level: "info", Output: path})
	if l == nil {
		t.Fatal("NewLogger returned nil for an unopenable path")
	}

	l.Info("must not panic")
}

// Regression test for finding 3. SetGlobalLogger type-asserted to the
// unexported *logger and silently discarded everything else.
func TestSetGlobalLoggerAcceptsEveryImplementation(t *testing.T) {
	original := GetGlobalLogger()

	t.Cleanup(func() { SetGlobalLogger(original) })

	var buf bytes.Buffer
	// NewTestLogger deliberately is not in this list: it arrives in the next
	// task, and referencing it here would make this commit's test binary
	// uncompilable on its own, breaking git bisect inside the branch. The
	// capture logger gets its own SetGlobalLogger assertion alongside it.
	candidates := []Logger{
		NewNoopLogger(),
		New(Config{Format: FormatJSON, Output: &buf}),
		New(Config{Format: FormatPretty, Output: &buf}),
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
	//nolint:staticcheck // SA1012: nil is the case under test, not an unsure caller
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
