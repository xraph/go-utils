package log

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestLoggerTo(buf *bytes.Buffer, lvl level) *logger {
	return newLogger(lvl, &jsonEncoder{}, newSyncWriter(buf), "", false)
}

func decodeLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var out []map[string]any

	for ln := range strings.SplitSeq(strings.TrimRight(buf.String(), "\n"), "\n") {
		if ln == "" {
			continue
		}

		var m map[string]any
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("bad line %q: %v", ln, err)
		}

		out = append(out, m)
	}

	return out
}

// Regression test for finding 2. zap.NewProductionConfig() enabled sampling at
// first-100-then-1-in-100, so 1,000 identical lines produced 109.
func TestNoSamplingEveryLineIsWritten(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	const n = 1000
	for i := range n {
		l.Info("repeated line", Int("i", i))
	}

	got := len(decodeLines(t, &buf))
	if got != n {
		t.Errorf("wrote %d lines, want %d (sampling must be off by default)", got, n)
	}
}

// Regression test for finding 1. The old getCaller added skip to
// SkipCallerPath and overshot into runtime internals, printing asm_arm64.s.
func TestCallerPointsAtTheCallSite(t *testing.T) {
	var buf bytes.Buffer

	l := newLogger(infoLevel, &jsonEncoder{}, newSyncWriter(&buf), "", true)

	l.Info("direct") // <- the caller must resolve to this file

	lines := decodeLines(t, &buf)

	caller, _ := lines[0]["caller"].(string)
	if !strings.HasPrefix(caller, "log/logger_test.go:") {
		t.Errorf("caller = %q, want a log/logger_test.go line", caller)
	}

	for _, bad := range []string{"asm_", "proc.go", "runtime"} {
		if strings.Contains(caller, bad) {
			t.Errorf("caller resolved into the runtime: %q", caller)
		}
	}
}

func TestCallerIsCorrectThroughWithAndNamed(t *testing.T) {
	var buf bytes.Buffer

	l := newLogger(infoLevel, &jsonEncoder{}, newSyncWriter(&buf), "", true)

	l.With(String("k", "v")).Info("through With")
	l.Named("child").Info("through Named")

	for _, ln := range decodeLines(t, &buf) {
		caller, _ := ln["caller"].(string)
		if !strings.HasPrefix(caller, "log/logger_test.go:") {
			t.Errorf("msg %q had caller %q, want a log/logger_test.go line", ln["msg"], caller)
		}
	}
}

func TestCallerIsCorrectThroughFVariantsAndSugar(t *testing.T) {
	var buf bytes.Buffer

	l := newLogger(infoLevel, &jsonEncoder{}, newSyncWriter(&buf), "", true)

	l.Infof("through Infof %d", 1)
	l.Sugar().Infow("through Infow", "k", "v")

	for _, ln := range decodeLines(t, &buf) {
		caller, _ := ln["caller"].(string)
		if !strings.HasPrefix(caller, "log/logger_test.go:") {
			t.Errorf("msg %q had caller %q, want a log/logger_test.go line", ln["msg"], caller)
		}
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, warnLevel)

	l.Debug("no")
	l.Info("no")
	l.Warn("yes")
	l.Error("yes")

	lines := decodeLines(t, &buf)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	if lines[0]["level"] != "WARN" || lines[1]["level"] != "ERROR" {
		t.Errorf("wrong lines survived: %v", lines)
	}
}

func TestWithAccumulatesFieldsInOrder(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	l.With(String("a", "1")).With(String("b", "2")).Info("msg", String("c", "3"))

	line := buf.String()

	ai, bi, ci := strings.Index(line, `"a"`), strings.Index(line, `"b"`), strings.Index(line, `"c"`)
	if ai < 0 || bi < 0 || ci < 0 {
		t.Fatalf("missing fields in %q", line)
	}

	if ai >= bi || bi >= ci {
		t.Errorf("fields out of order in %q", line)
	}
}

// With() must copy, not share, or two children corrupt each other.
func TestWithDoesNotAliasParentFields(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	base := l.With(String("base", "yes"))
	childA := base.With(String("a", "1"))
	childB := base.With(String("b", "2"))

	childA.Info("a")
	childB.Info("b")

	lines := decodeLines(t, &buf)
	if _, leaked := lines[0]["b"]; leaked {
		t.Error("childA saw childB's field")
	}

	if _, leaked := lines[1]["a"]; leaked {
		t.Error("childB saw childA's field")
	}

	for i := range lines {
		if lines[i]["base"] != "yes" {
			t.Errorf("line %d lost the parent field", i)
		}
	}
}

func TestNamedComposesWithDots(t *testing.T) {
	var buf bytes.Buffer

	l := newLogger(infoLevel, &jsonEncoder{}, newSyncWriter(&buf), "forge", false)

	l.Named("http").Named("router").Info("msg")

	if got := decodeLines(t, &buf)[0]["logger"]; got != "forge.http.router" {
		t.Errorf("logger = %v, want forge.http.router", got)
	}
}

func TestWithContextPullsIDs(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	ctx := WithUserID(WithTraceID(WithRequestID(context.Background(), "req1"), "trace1"), "user1")
	l.WithContext(ctx).Info("msg")

	got := decodeLines(t, &buf)[0]
	for k, want := range map[string]string{"request_id": "req1", "trace_id": "trace1", "user_id": "user1"} {
		if got[k] != want {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
}

func TestConcurrentLoggingProducesWholeLines(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)

	for i := range n {
		go func(i int) {
			defer wg.Done()

			l.Info("concurrent", Int("i", i), String("pad", strings.Repeat("x", 200)))
		}(i)
	}

	wg.Wait()

	if got := len(decodeLines(t, &buf)); got != n {
		t.Errorf("got %d well-formed lines, want %d", got, n)
	}
}

func TestDisabledLevelDoesNotAllocate(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	got := testing.AllocsPerRun(200, func() {
		l.Debug("cache lookup", String("key", "user:1234"), Int("shard", 7))
	})
	// The variadic slice may or may not escape depending on the compiler's
	// escape analysis. One allocation is acceptable; more means the level
	// check is not the first thing that runs.
	if got > 1 {
		t.Errorf("disabled Debug allocated %.0f times, want at most 1", got)
	}
}

// A logger built with With() is the common pattern, so the merge of its own
// fields with the call's fields must not allocate per line.
func TestWithLoggerDoesNotAllocatePerLine(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel).With(
		String("service", "api"), String("version", "1.2.3"))

	got := testing.AllocsPerRun(200, func() {
		buf.Reset()
		l.Info("request completed", String("method", "GET"), Int("status", 200))
	})
	// bytes.Buffer growth is the only allocation we permit here.
	if got > 1 {
		t.Errorf("With-logger Info allocated %.0f times per call, want at most 1", got)
	}
}

func TestSetLevelAtRuntime(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	l.Debug("hidden")
	l.setLevel(debugLevel)
	l.Debug("visible")

	lines := decodeLines(t, &buf)
	if len(lines) != 1 || lines[0]["msg"] != "visible" {
		t.Errorf("runtime level change did not take effect: %v", lines)
	}
}

func TestSugarLogger(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	l.Sugar().Infow("sugar", "k", "v", "n", 1)

	got := decodeLines(t, &buf)[0]
	if got["msg"] != "sugar" || got["k"] != "v" {
		t.Errorf("sugar output = %v", got)
	}
}

func TestSugarWithOddArgsDoesNotPanic(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	l.Sugar().Infow("odd", "k", "v", "dangling")

	if got := decodeLines(t, &buf)[0]["k"]; got != "v" {
		t.Errorf("k = %v, want v", got)
	}
}

func TestTimestampIsPopulated(t *testing.T) {
	var buf bytes.Buffer

	l := newTestLoggerTo(&buf, infoLevel)

	before := time.Now().Add(-time.Second)

	l.Info("msg")

	ts, _ := decodeLines(t, &buf)[0]["ts"].(string)

	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t.Fatalf("ts %q is not RFC3339Nano: %v", ts, err)
	}

	if parsed.Before(before) {
		t.Errorf("ts %v is implausibly old", parsed)
	}
}
