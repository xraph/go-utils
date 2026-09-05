package log

import (
	"flag"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// mode is what a constructed logger actually does.
type mode int

const (
	modePretty mode = iota
	modeJSON
	modeNoop
)

// resolveMode decides how a logger behaves, given everything observable at
// construction. It is pure so that every branch is table-testable.
//
// Precedence, highest first:
//
//  1. An explicit FORGE_LOG_FORMAT or LOG_FORMAT of "pretty" or "json". Someone
//     who set the variable meant it, including when they are debugging a single
//     failing test and want the output back.
//  2. Test silence. An unattended `go test` run stays quiet; `go test -v` opts
//     back into pretty output, matching what -v already means everywhere else.
//  3. TTY detection. A terminal gets pretty, anything else gets JSON.
func resolveMode(envFormat string, isTTY, underTest, testV bool) mode {
	switch strings.ToLower(envFormat) {
	case "pretty":
		return modePretty
	case "json":
		return modeJSON
	}

	if underTest {
		if testV {
			return modePretty
		}

		return modeNoop
	}

	if isTTY {
		return modePretty
	}

	return modeJSON
}

// resolveColor decides whether to emit ANSI escapes. Colour is layered on top
// of pretty and is never applied to JSON or noop.
func resolveColor(m mode, isTTY, noColor bool, termVar string) bool {
	if m != modePretty {
		return false
	}

	if noColor || termVar == "dumb" {
		return false
	}

	return isTTY
}

// isTerminal reports whether w is a terminal. Anything that is not an *os.File
// (a bytes.Buffer in a test, a pipe wrapper) is definitively not one.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}

	return term.IsTerminal(int(f.Fd())) //nolint:gosec // G115: fds are small, non-negative; term.IsTerminal needs an int
}

// underTest reports whether this binary was built by `go test`. It checks for
// the test.v flag that the generated test main registers, deliberately avoiding
// an import of the testing package, which would otherwise be linked into every
// production binary that uses this logger.
func underTest() bool {
	return flag.Lookup("test.v") != nil
}

// testVerbose reports whether `go test -v` was passed.
func testVerbose() bool {
	f := flag.Lookup("test.v")
	if f == nil {
		return false
	}

	return f.Value.String() == "true"
}

// envFormat returns the requested format from the environment, preferring the
// forge-specific variable over the generic one.
func envFormat() string {
	if v := os.Getenv("FORGE_LOG_FORMAT"); v != "" {
		return v
	}

	return os.Getenv("LOG_FORMAT")
}

// noColorSet reports whether the environment asks for uncoloured output.
func noColorSet() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0"
}
