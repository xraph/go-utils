package log

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func testEntry() entry {
	return entry{
		lvl:    infoLevel,
		msg:    "request completed",
		name:   "forge.http",
		ts:     time.Date(2026, 9, 5, 15, 4, 5, 123000000, time.UTC),
		caller: "http/server.go:42",
	}
}

func testFields() []Field {
	return []Field{
		String("method", "GET"),
		String("path", "/v1/users"),
		Int("status", 200),
		Duration("latency", 3*time.Millisecond),
		Error(errors.New("boom")),
	}
}

// Regression test for finding 6. NewBeautifulLoggerJSON claimed to emit JSON
// and emitted the same text format as every other preset.
func TestJSONEncoderProducesParseableJSON(t *testing.T) {
	out := (&jsonEncoder{}).encode(nil, testEntry(), testFields())

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	if got["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", got["level"])
	}

	if got["msg"] != "request completed" {
		t.Errorf("msg = %v, want request completed", got["msg"])
	}

	if got["logger"] != "forge.http" {
		t.Errorf("logger = %v, want forge.http", got["logger"])
	}

	if got["method"] != "GET" {
		t.Errorf("method = %v, want GET", got["method"])
	}

	if got["status"] != float64(200) {
		t.Errorf("status = %v, want 200", got["status"])
	}

	if got["error"] != "boom" {
		t.Errorf("error = %v, want boom", got["error"])
	}
}

func TestJSONEncoderEndsWithNewline(t *testing.T) {
	out := (&jsonEncoder{}).encode(nil, testEntry(), nil)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Errorf("encoded line must end with a newline, got %q", out)
	}
}

// Regression test for finding 4. The old logger stored fields in a map and
// iterated it, so the same call produced a different order every time.
func TestFieldOrderIsDeterministic(t *testing.T) {
	enc := &jsonEncoder{}

	first := string(enc.encode(nil, testEntry(), testFields()))
	for i := range 50 {
		got := string(enc.encode(nil, testEntry(), testFields()))
		if got != first {
			t.Fatalf("encoding is not deterministic\niteration %d: %s\nfirst:       %s", i, got, first)
		}
	}
}

func TestJSONEncoderEscapesControlCharacters(t *testing.T) {
	e := testEntry()
	e.msg = "line one\nline \"two\"\ttabbed\\slash"
	out := (&jsonEncoder{}).encode(nil, e, []Field{String("k", "a\"b\nc")})

	if strings.Count(string(out), "\n") != 1 {
		t.Errorf("a raw newline leaked into the JSON line: %q", out)
	}

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("escaping produced invalid JSON: %v\n%s", err, out)
	}

	if got["msg"] != e.msg {
		t.Errorf("msg round trip = %q, want %q", got["msg"], e.msg)
	}

	if got["k"] != "a\"b\nc" {
		t.Errorf("field round trip = %q", got["k"])
	}
}

func TestJSONEncoderHandlesNilError(t *testing.T) {
	out := (&jsonEncoder{}).encode(nil, testEntry(), []Field{Error(nil)})

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("nil error produced invalid JSON: %v\n%s", err, out)
	}

	if got["error"] != nil {
		t.Errorf("error = %v, want null", got["error"])
	}
}

// Regression test: Time used to store every value as UnixNano unconditionally,
// which wraps outside roughly 1678-2262 and rendered the zero time.Time as
// 1754-08-30T22:43:41Z instead of a recognisable zero value.
func TestJSONEncoderRendersZeroTimeRecognisably(t *testing.T) {
	out := (&jsonEncoder{}).encode(nil, testEntry(), []Field{Time("t", time.Time{})})

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}

	ts, _ := got["t"].(string)
	if !strings.HasPrefix(ts, "0001-01-01") {
		t.Errorf("zero time rendered as %q, want a 0001-01-01 prefix", ts)
	}
}

func TestJSONEncoderEvaluatesLazyFields(t *testing.T) {
	out := (&jsonEncoder{}).encode(nil, testEntry(), []Field{Lazy("k", func() any { return "late" })})

	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}

	if got["k"] != "late" {
		t.Errorf("lazy field = %v, want late", got["k"])
	}
}

func TestJSONEncoderSkipsUnknownFields(t *testing.T) {
	// Conditional(false, ...) produces an unknownType field that must not
	// appear in the output at all.
	out := (&jsonEncoder{}).encode(nil, testEntry(), []Field{Conditional(false, "hidden", 1)})
	if strings.Contains(string(out), "hidden") {
		t.Errorf("a skipped field leaked into the output: %s", out)
	}
}

func TestJSONEncodeAllocationBudget(t *testing.T) {
	skipUnderRace(t)

	enc := &jsonEncoder{}
	e := testEntry()
	f := testFields()
	buf := make([]byte, 0, 1024)

	got := testing.AllocsPerRun(200, func() {
		_ = enc.encode(buf[:0], e, f)
	})
	if got > 0 {
		t.Errorf("encode allocated %.0f times per call, want 0", got)
	}

	bare := testing.AllocsPerRun(200, func() {
		_ = enc.encode(buf[:0], e, nil)
	})
	if bare > 0 {
		t.Errorf("encode with no fields allocated %.0f times per call, want 0", bare)
	}
}
