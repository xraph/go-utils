package log

import (
	"bytes"
	"os"
	"testing"
)

func TestResolveMode(t *testing.T) {
	cases := []struct {
		name      string
		envFormat string
		isTTY     bool
		underTest bool
		testV     bool
		want      mode
	}{
		// Plain runtime, no env override.
		{"tty, no env", "", true, false, false, modePretty},
		{"pipe, no env", "", false, false, false, modeJSON},

		// Explicit env wins over TTY detection in both directions.
		{"env pretty beats pipe", "pretty", false, false, false, modePretty},
		{"env json beats tty", "json", true, false, false, modeJSON},
		{"garbage env falls through to tty", "yaml", true, false, false, modePretty},
		{"garbage env falls through to pipe", "yaml", false, false, false, modeJSON},

		// Under test: silent by default, pretty when -v is passed. The
		// tty-without-v row matters: it is the only case that catches a
		// rewrite which checks isTTY before checking underTest.
		{"go test is silent", "", false, true, false, modeNoop},
		{"go test on a tty without -v is still silent", "", true, true, false, modeNoop},
		{"go test -v is pretty", "", false, true, true, modePretty},
		{"go test -v on a tty is pretty", "", true, true, true, modePretty},

		// Explicit env overrides test silence, so a developer debugging one
		// failing test can force output back on.
		{"env json beats test silence", "json", false, true, false, modeJSON},
		{"env pretty beats test silence", "pretty", false, true, false, modePretty},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveMode(c.envFormat, c.isTTY, c.underTest, c.testV)
			if got != c.want {
				t.Errorf("resolveMode(%q, tty=%v, test=%v, v=%v) = %v, want %v",
					c.envFormat, c.isTTY, c.underTest, c.testV, got, c.want)
			}
		})
	}
}

func TestResolveColor(t *testing.T) {
	cases := []struct {
		name    string
		m       mode
		isTTY   bool
		noColor bool
		termVar string
		want    bool
	}{
		{"pretty on a tty", modePretty, true, false, "xterm-256color", true},
		{"pretty but not a tty", modePretty, false, false, "xterm-256color", false},
		{"NO_COLOR wins", modePretty, true, true, "xterm-256color", false},
		{"TERM=dumb wins", modePretty, true, false, "dumb", false},
		{"json is never coloured", modeJSON, true, false, "xterm-256color", false},
		{"noop is never coloured", modeNoop, true, false, "xterm-256color", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveColor(c.m, c.isTTY, c.noColor, c.termVar)
			if got != c.want {
				t.Errorf("resolveColor = %v, want %v", got, c.want)
			}
		})
	}
}

func TestUnderTestIsTrueHere(t *testing.T) {
	// This test binary was built by `go test`, so the probe must say so.
	// If this ever fails, the detection mechanism has broken.
	if !underTest() {
		t.Error("underTest() = false inside a go test binary")
	}
}

func TestIsTerminalOnNonTerminal(t *testing.T) {
	if isTerminal(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer is not a terminal")
	}

	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()

	if isTerminal(devnull) {
		t.Error("/dev/null is not a terminal")
	}
}
