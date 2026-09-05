package log

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// Format selects the output encoder. FormatAuto resolves once at construction.
type Format string

const (
	FormatAuto   Format = "auto"
	FormatPretty Format = "pretty"
	FormatJSON   Format = "json"
)

// Config is the full construction surface.
type Config struct {
	Level       LogLevel
	Format      Format
	Environment string    // retained for compatibility; does not pick the encoder
	Output      io.Writer // nil means os.Stderr
	Name        string
	AddCaller   bool
	Color       *bool // nil means auto
}

// LoggingConfig is the configuration-file-facing struct. Its shape and tags are
// unchanged so that existing YAML, mapstructure and env bindings keep working.
type LoggingConfig struct {
	Level       LogLevel `env:"LOG_LEVEL"   mapstructure:"level"       yaml:"level"`
	Format      string   `env:"LOG_FORMAT"  mapstructure:"format"      yaml:"format"`
	Environment string   `env:"ENVIRONMENT" mapstructure:"environment" yaml:"environment"`
	Output      string   `env:"LOG_OUTPUT"  mapstructure:"output"      yaml:"output"`
	// Name is the root logger name. Adding a field is additive: existing
	// YAML, mapstructure and env bindings for the four fields above are
	// unaffected, and callers that omit it get the empty name they had before.
	Name string `env:"LOG_NAME" mapstructure:"name" yaml:"name"`
}

// New builds a logger from a full Config.
func New(cfg Config) Logger {
	out := cfg.Output
	if out == nil {
		out = os.Stderr
	}

	format := cfg.Format
	if format == "" {
		format = FormatAuto
	}

	tty := isTerminal(out)

	var m mode

	switch format {
	case FormatPretty:
		m = modePretty
	case FormatJSON:
		m = modeJSON
	default: // FormatAuto and any unrecognised value
		m = resolveMode(envFormat(), tty, underTest(), testVerbose())
	}

	if m == modeNoop {
		return NewNoopLogger()
	}

	var enc encoder

	if m == modePretty {
		color := resolveColor(m, tty, noColorSet(), os.Getenv("TERM"))
		if cfg.Color != nil {
			color = *cfg.Color
		}

		enc = newPrettyEncoder(color, terminalWidth(out))
	} else {
		enc = &jsonEncoder{}
	}

	return newLogger(parseLevel(cfg.Level), enc, newSyncWriter(out), cfg.Name, cfg.AddCaller)
}

// terminalWidth returns the width of the output terminal, defaulting to a
// sensible fixed width when the size cannot be determined.
func terminalWidth(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 120
	}

	width, _, err := term.GetSize(int(f.Fd())) //nolint:gosec // G115: fds are small, non-negative; term.GetSize needs an int
	if err != nil || width <= 0 {
		return 120
	}

	return width
}

// NewLogger builds a logger from the configuration-file struct. This is the
// entry point every existing forge call site uses.
func NewLogger(cfg LoggingConfig) Logger {
	format := FormatAuto

	switch cfg.Format {
	case "json":
		format = FormatJSON
	case "pretty", "console", "text":
		format = FormatPretty
	}

	var out io.Writer

	switch cfg.Output {
	case "stdout":
		out = os.Stdout
	case "stderr", "":
		out = os.Stderr
	default:
		// A path. If it cannot be opened, fall back to stderr rather than
		// returning a logger that panics on first use.
		f, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			out = os.Stderr
		} else {
			out = f
		}
	}

	return New(Config{
		Level:       cfg.Level,
		Format:      format,
		Environment: cfg.Environment,
		Output:      out,
		Name:        cfg.Name,
		AddCaller:   true,
	})
}

var (
	globalMu     sync.RWMutex
	globalLogger Logger
)

// GetGlobalLogger returns the process-wide logger, creating a default on first
// use. Under a test binary that default is the noop logger, so an unconfigured
// package does not flood the test output.
func GetGlobalLogger() Logger {
	globalMu.RLock()

	l := globalLogger

	globalMu.RUnlock()

	if l != nil {
		return l
	}

	globalMu.Lock()
	defer globalMu.Unlock()

	if globalLogger == nil {
		globalLogger = New(Config{Name: "app"})
	}

	return globalLogger
}

// SetGlobalLogger installs the process-wide logger. It accepts every
// implementation; the previous version silently ignored all but one.
func SetGlobalLogger(l Logger) {
	if l == nil {
		return
	}

	globalMu.Lock()
	globalLogger = l
	globalMu.Unlock()
}

// resetGlobalLogger clears the cached global. Test-only.
func resetGlobalLogger() {
	globalMu.Lock()
	globalLogger = nil
	globalMu.Unlock()
}
