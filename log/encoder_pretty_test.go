package log

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite golden files")

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden file %s (run: go test ./log/ -update): %v", path, err)
	}
	if string(got) != string(want) {
		t.Errorf("output does not match %s\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}

func prettyLines(enc *prettyEncoder, entries []entry, fields [][]Field) []byte {
	var out []byte
	for i := range entries {
		out = enc.encode(out, entries[i], fields[i])
	}
	return out
}

func TestPrettyEncoderGolden(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 5, 123000000, time.UTC)
	enc := newPrettyEncoder(false, 120)

	entries := []entry{
		{lvl: infoLevel, msg: "server listening", name: "forge.http", ts: ts},
		{lvl: debugLevel, msg: "acquired connection", name: "forge.db", ts: ts.Add(289 * time.Millisecond)},
		{lvl: warnLevel, msg: "eviction rate above target", name: "forge.cache", ts: ts.Add(1779 * time.Millisecond)},
		{lvl: errorLevel, msg: "request failed", name: "forge.http", ts: ts.Add(1896 * time.Millisecond)},
	}
	fields := [][]Field{
		{String("addr", ":8080"), Bool("tls", false)},
		{String("pool", "primary"), Int("idle", 9), Int("open", 1)},
		{Float64("rate", 0.42), Float64("target", 0.20)},
		{String("method", "POST"), String("path", "/v1/orders"), Int("status", 500)},
	}

	checkGolden(t, "pretty_basic.golden", prettyLines(enc, entries, fields))
}

func TestPrettyEncoderWrapsLongFieldSets(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 7, 1000000, time.UTC)
	enc := newPrettyEncoder(false, 100)

	e := entry{lvl: errorLevel, msg: "request failed", name: "forge.http", ts: ts}
	f := []Field{
		String("method", "POST"),
		String("path", "/v1/orders"),
		Int("status", 500),
		Duration("latency", 1200*time.Millisecond),
		Error(errors.New("dial tcp 10.0.0.4:5432: connect: connection refused")),
	}

	got := enc.encode(nil, e, f)
	checkGolden(t, "pretty_wrap.golden", got)

	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected the field set to wrap, got %d line(s):\n%s", len(lines), got)
	}
	// Continuation lines hang-indent to the message column, so they must start
	// with spaces and must not restate the timestamp.
	for i, ln := range lines[1:] {
		if !strings.HasPrefix(ln, " ") {
			t.Errorf("continuation line %d is not indented: %q", i+1, ln)
		}
		if strings.Contains(ln, "15:04:07") {
			t.Errorf("continuation line %d repeats the timestamp: %q", i+1, ln)
		}
	}
}

func TestPrettyEncoderColumnsGrowAndNeverShrink(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 5, 0, time.UTC)
	enc := newPrettyEncoder(false, 200)

	long := entry{lvl: infoLevel, msg: "m", name: "forge.dashboard.contract", ts: ts}
	short := entry{lvl: infoLevel, msg: "m", name: "a", ts: ts}

	_ = enc.encode(nil, long, nil)
	afterLong := enc.nameWidth
	_ = enc.encode(nil, short, nil)
	if enc.nameWidth < afterLong {
		t.Errorf("name column shrank from %d to %d", afterLong, enc.nameWidth)
	}
}

func TestPrettyEncoderTruncatesLongNamesFromTheLeft(t *testing.T) {
	enc := newPrettyEncoder(false, 120)
	got := truncateLeft("forge.dashboard.contract.dispatcher", 20)

	if len(got) != 20 {
		t.Errorf("truncateLeft returned %d chars, want 20: %q", len(got), got)
	}
	if !strings.HasSuffix(got, "dispatcher") {
		t.Errorf("truncation must keep the specific end, got %q", got)
	}
	if !strings.HasPrefix(got, "...") {
		t.Errorf("truncation must be marked with a leading ellipsis, got %q", got)
	}
	// A name that already fits is returned unchanged.
	if got := truncateLeft("forge.http", 20); got != "forge.http" {
		t.Errorf("short name was modified: %q", got)
	}
	_ = enc
}

func TestPrettyEncoderColorOnlyWhenEnabled(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 5, 0, time.UTC)
	e := entry{lvl: errorLevel, msg: "boom", name: "x", ts: ts}

	plain := string(newPrettyEncoder(false, 120).encode(nil, e, nil))
	if strings.Contains(plain, "\033[") {
		t.Errorf("colour disabled but ANSI escapes present: %q", plain)
	}

	coloured := string(newPrettyEncoder(true, 120).encode(nil, e, nil))
	if !strings.Contains(coloured, "\033[") {
		t.Errorf("colour enabled but no ANSI escapes: %q", coloured)
	}
}

// The encoder carries mutable column state and a reused scratch buffer, and the
// logger calls encode outside the writer's lock, so a shared encoder must be
// safe for concurrent use. Without the mutex this fails under -race.
func TestPrettyEncoderIsSafeForConcurrentUse(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 5, 0, time.UTC)
	enc := newPrettyEncoder(false, 120)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	results := make([][]byte, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			e := entry{
				lvl:  infoLevel,
				msg:  "concurrent",
				name: fmt.Sprintf("forge.svc%02d", i),
				ts:   ts,
			}
			results[i] = enc.encode(nil, e, []Field{
				String("worker", fmt.Sprintf("w%02d", i)),
				Int("n", i),
			})
		}(i)
	}
	wg.Wait()

	for i, got := range results {
		line := string(got)
		if !strings.Contains(line, fmt.Sprintf("worker=w%02d", i)) {
			t.Errorf("goroutine %d lost or corrupted its field: %q", i, line)
		}
		if strings.Count(line, "\n") != 1 {
			t.Errorf("goroutine %d produced %d lines, want 1: %q", i, strings.Count(line, "\n"), line)
		}
	}
}

// Regression test for finding 5. Minimalist mode silently dropped every field.
func TestPrettyEncoderNeverDropsFields(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 5, 0, time.UTC)
	enc := newPrettyEncoder(false, 200)
	e := entry{lvl: infoLevel, msg: "minimal", name: "min", ts: ts}

	got := string(enc.encode(nil, e, []Field{String("critical_field", "DO NOT LOSE ME")}))
	if !strings.Contains(got, "critical_field=DO NOT LOSE ME") {
		t.Errorf("field was dropped from the output: %q", got)
	}
}

// The caller column has to be reserved in msgCol, or wrapped continuation
// lines hang-indent to the wrong column and the wrap check undercounts.
func TestPrettyEncoderAlignsWrapsUnderCaller(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 7, 1000000, time.UTC)
	enc := newPrettyEncoder(false, 100)

	e := entry{
		lvl:    errorLevel,
		msg:    "request failed",
		name:   "forge.http",
		caller: "http/handler.go:42",
		ts:     ts,
	}
	f := []Field{
		String("method", "POST"),
		String("path", "/v1/orders"),
		Int("status", 500),
		Duration("latency", 1200*time.Millisecond),
		Error(errors.New("connection refused")),
	}

	got := string(enc.encode(nil, e, f))
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected wrapping, got %d line(s):\n%s", len(lines), got)
	}

	msgCol := strings.Index(lines[0], "request failed")
	if msgCol < 0 {
		t.Fatalf("message not found in %q", lines[0])
	}
	for i, ln := range lines[1:] {
		indent := len(ln) - len(strings.TrimLeft(ln, " "))
		if indent != msgCol {
			t.Errorf("continuation %d indents to %d, want %d (the message column)\n%s",
				i+1, indent, msgCol, got)
		}
	}

	for i, ln := range lines {
		if len(ln) > 100 {
			t.Errorf("line %d is %d chars, over the 100 width:\n%s", i, len(ln), ln)
		}
	}
}

// A single field wider than the remaining budget cannot be wrapped, because
// the encoder never splits one field's key=value across lines. That is
// deliberate: a soft-wrapped error string or stack trace is still readable
// and still copy-pasteable, while a truncated or hyphenated one has lost
// data. So such a line is allowed to exceed the width, and what must hold
// is that the value survives intact and gets a line to itself.
func TestPrettyEncoderKeepsOversizedFieldsIntact(t *testing.T) {
	ts := time.Date(2026, 9, 5, 15, 4, 7, 1000000, time.UTC)
	enc := newPrettyEncoder(false, 100)

	long := "dial tcp 10.0.0.4:5432: connect: connection refused after 3 retries over 30s"
	e := entry{
		lvl:    errorLevel,
		msg:    "request failed",
		name:   "forge.http",
		caller: "http/handler.go:42",
		ts:     ts,
	}
	got := string(enc.encode(nil, e, []Field{
		String("method", "POST"),
		Error(errors.New(long)),
	}))

	if !strings.Contains(got, long) {
		t.Errorf("the oversized value was truncated or mangled:\n%s", got)
	}

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	var found bool
	for _, ln := range lines {
		trimmed := strings.TrimLeft(ln, " ")
		if strings.HasPrefix(trimmed, "error=") {
			found = true
			if trimmed != "error="+long {
				t.Errorf("the oversized field shares its line with other content: %q", trimmed)
			}
		}
	}
	if !found {
		t.Error("no line carried the error field")
	}
}
