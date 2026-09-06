package log

import (
	"fmt"
	"strings"
	"testing"
)

// nilDerefStringer implements fmt.Stringer by dereferencing a field, so
// calling String() on a nil *nilDerefStringer panics with a nil pointer
// dereference. This is the shape of bug that reaches production: a typed-nil
// value satisfies the interface, so nothing at the call site catches it
// before the logger does.
type nilDerefStringer struct {
	n int
}

func (s *nilDerefStringer) String() string {
	return fmt.Sprintf("n=%d", s.n)
}

// panickyError implements error by always panicking, standing in for an
// Error() method that blows up on bad internal state rather than a nil
// receiver specifically.
type panickyError struct{}

func (panickyError) Error() string {
	panic("boom-error")
}

func TestFormatAnyRecoversFromPanickingStringer(t *testing.T) {
	var s *nilDerefStringer // typed nil; satisfies fmt.Stringer, panics on use

	got := formatAny(s)
	if !strings.Contains(got, "PANIC") {
		t.Errorf("formatAny(nil stringer) = %q, want a PANIC marker", got)
	}
}

func TestFormatAnyRecoversFromPanickingError(t *testing.T) {
	got := formatAny(panickyError{})
	if !strings.Contains(got, "PANIC") {
		t.Errorf("formatAny(panicking error) = %q, want a PANIC marker", got)
	}
}

func TestSafeErrorStringRecoversFromPanic(t *testing.T) {
	got := safeErrorString(panickyError{})
	if !strings.Contains(got, "PANIC") {
		t.Errorf("safeErrorString(panicking error) = %q, want a PANIC marker", got)
	}
}

// TestJSONEncoderSurvivesPanickingValues is the end-to-end regression: a
// logging call with a bad Stringer or error must produce a line and must not
// crash the caller. Reproduced before the fix: logger.Info("x", Any("user",
// u)) with u a typed-nil *User whose String() dereferences took the process
// down.
func TestJSONEncoderSurvivesPanickingValues(t *testing.T) {
	var nilStringer *nilDerefStringer

	fields := []Field{
		Any("user", nilStringer),
		Stringer("named", nilStringer),
		Error(panickyError{}),
	}

	var out []byte

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("encode panicked: %v", r)
			}
		}()

		out = (&jsonEncoder{}).encode(nil, testEntry(), fields)
	}()

	if !strings.Contains(string(out), "PANIC") {
		t.Errorf("output has no PANIC marker for the bad values: %s", out)
	}

	if !strings.Contains(string(out), `"msg":"request completed"`) {
		t.Errorf("encoder dropped the rest of the line: %s", out)
	}
}

func TestPrettyEncoderSurvivesPanickingValues(t *testing.T) {
	var nilStringer *nilDerefStringer

	fields := []Field{
		Any("user", nilStringer),
		Stringer("named", nilStringer),
		Error(panickyError{}),
	}

	enc := newPrettyEncoder(false, 120)

	var out []byte

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("encode panicked: %v", r)
			}
		}()

		out = enc.encode(nil, testEntry(), fields)
	}()

	if !strings.Contains(string(out), "PANIC") {
		t.Errorf("output has no PANIC marker for the bad values: %s", out)
	}

	if !strings.Contains(string(out), "request completed") {
		t.Errorf("encoder dropped the rest of the line: %s", out)
	}
}
