package log

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

const fatalChildEnv = "GO_UTILS_FATAL_CHILD"

// stdoutFlusher is a Sink whose Flush leaves a mark the parent test can see.
type stdoutFlusher struct{}

func (stdoutFlusher) Emit(Record) {}

func (stdoutFlusher) Flush() { _, _ = os.Stdout.WriteString("flushed\n") }

type panicFlusher struct{}

func (panicFlusher) Emit(Record) {}

func (panicFlusher) Flush() { panic("flush exploded") }

// plainSink has no Flush method and must be skipped.
type plainSink struct{}

func (plainSink) Emit(Record) {}

// runFatalChild is the body of the re-executed test binary. It never returns:
// every variant ends in a Fatal call, which exits the process.
func runFatalChild(mode string) {
	l := New(Config{Format: FormatJSON, Output: io.Discard, Level: LevelDebug})

	if mode == "p" {
		// A sink that panics in Flush must not stop the next one.
		_, _ = AttachSink(l, panicFlusher{})
	}

	_, _ = AttachSink(l, plainSink{})
	_, _ = AttachSink(l, stdoutFlusher{})

	switch mode {
	case "f":
		l.Fatalf("bye %d", 1)
	case "w":
		l.Sugar().Fatalw("bye", "k", "v")
	default:
		l.Fatal("bye")
	}
}

func TestFatalFlushesSinks(t *testing.T) {
	if mode := os.Getenv(fatalChildEnv); mode != "" {
		runFatalChild(mode)

		return
	}

	cases := []struct {
		name string
		mode string
	}{
		{"Fatal", "1"},
		{"Fatalf", "f"},
		{"Fatalw", "w"},
		{"Fatal after a panicking Flush", "p"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFatalFlushesSinks$")

			cmd.Env = append(os.Environ(), fatalChildEnv+"="+tc.mode)

			var stdout bytes.Buffer

			cmd.Stdout = &stdout

			err := cmd.Run()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("child err = %v, want exit status 1", err)
			}

			if !strings.Contains(stdout.String(), "flushed") {
				t.Errorf("child stdout = %q, want it to contain %q", stdout.String(), "flushed")
			}
		})
	}
}
