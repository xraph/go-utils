package log

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LogEntry is one captured log event.
type LogEntry struct {
	Level   string
	Message string
	Fields  []Field
	Time    time.Time
}

// Field looks up a captured field by key and reports whether it was present.
func (e LogEntry) Field(name string) (any, bool) {
	for i := range e.Fields {
		if e.Fields[i].Key() == name {
			return e.Fields[i].Value(), true
		}
	}

	return nil, false
}

// captureState is the buffer every TestLogger in a With/Named family writes
// into. The mutex lives here, alongside the slice it guards. Giving each clone
// its own mutex while sharing the slice would be a data race: a parent and its
// child would append to the same backing array under different locks.
type captureState struct {
	mu   sync.RWMutex
	logs []LogEntry
}

// TestLogger captures log events in memory for assertions.
type TestLogger struct {
	state  *captureState // shared with every With/Named descendant
	name   string
	fields []Field
}

// NewTestLogger returns a logger that records everything it is given.
func NewTestLogger() Logger {
	return &TestLogger{state: &captureState{logs: make([]LogEntry, 0, 8)}}
}

func (tl *TestLogger) add(level, msg string, fields []Field) {
	all := make([]Field, 0, len(tl.fields)+len(fields))
	all = append(all, tl.fields...)
	all = append(all, fields...)

	tl.state.mu.Lock()
	defer tl.state.mu.Unlock()

	tl.state.logs = append(tl.state.logs, LogEntry{
		Level:   level,
		Message: msg,
		Fields:  all,
		Time:    time.Now(),
	})
}

func (tl *TestLogger) Debug(msg string, f ...Field) { tl.add("DEBUG", msg, f) }
func (tl *TestLogger) Info(msg string, f ...Field)  { tl.add("INFO", msg, f) }
func (tl *TestLogger) Warn(msg string, f ...Field)  { tl.add("WARN", msg, f) }
func (tl *TestLogger) Error(msg string, f ...Field) { tl.add("ERROR", msg, f) }
func (tl *TestLogger) Fatal(msg string, f ...Field) { tl.add("FATAL", msg, f) }

func (tl *TestLogger) Debugf(t string, a ...any) { tl.add("DEBUG", fmt.Sprintf(t, a...), nil) }
func (tl *TestLogger) Infof(t string, a ...any)  { tl.add("INFO", fmt.Sprintf(t, a...), nil) }
func (tl *TestLogger) Warnf(t string, a ...any)  { tl.add("WARN", fmt.Sprintf(t, a...), nil) }
func (tl *TestLogger) Errorf(t string, a ...any) { tl.add("ERROR", fmt.Sprintf(t, a...), nil) }
func (tl *TestLogger) Fatalf(t string, a ...any) { tl.add("FATAL", fmt.Sprintf(t, a...), nil) }

func (tl *TestLogger) clone() *TestLogger {
	c := &TestLogger{state: tl.state, name: tl.name}
	c.fields = make([]Field, len(tl.fields), len(tl.fields)+4)
	copy(c.fields, tl.fields)

	return c
}

func (tl *TestLogger) With(fields ...Field) Logger {
	c := tl.clone()
	c.fields = append(c.fields, fields...)

	return c
}

func (tl *TestLogger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return tl
	}

	c := tl.clone()
	c.fields = append(c.fields, ContextFields(ctx)...)

	return c
}

func (tl *TestLogger) Named(name string) Logger {
	c := tl.clone()
	if c.name == "" {
		c.name = name
	} else {
		c.name = c.name + "." + name
	}

	return c
}

func (tl *TestLogger) Sugar() SugarLogger { return &testSugarLogger{tl: tl} }
func (tl *TestLogger) Sync() error        { return nil }

// GetLogs returns a copy of every captured entry, including those recorded by
// any With/Named descendant.
func (tl *TestLogger) GetLogs() []LogEntry {
	tl.state.mu.RLock()
	defer tl.state.mu.RUnlock()

	out := make([]LogEntry, len(tl.state.logs))
	copy(out, tl.state.logs)

	return out
}

// GetLogsByLevel returns the captured entries at one level.
func (tl *TestLogger) GetLogsByLevel(level string) []LogEntry {
	var out []LogEntry

	for _, e := range tl.GetLogs() {
		if e.Level == level {
			out = append(out, e)
		}
	}

	return out
}

// Clear discards every captured entry.
func (tl *TestLogger) Clear() {
	tl.state.mu.Lock()
	defer tl.state.mu.Unlock()

	tl.state.logs = tl.state.logs[:0]
}

// AssertHasLog reports whether an entry with this level and message exists.
func (tl *TestLogger) AssertHasLog(level, message string) bool {
	for _, e := range tl.GetLogs() {
		if e.Level == level && e.Message == message {
			return true
		}
	}

	return false
}

// CountLogs returns how many entries were captured at one level.
func (tl *TestLogger) CountLogs(level string) int {
	n := 0

	for _, e := range tl.GetLogs() {
		if e.Level == level {
			n++
		}
	}

	return n
}

type testSugarLogger struct {
	tl   *TestLogger
	args []Field
}

func (s *testSugarLogger) merge(kv []any) []Field {
	f := keysAndValuesToFields(kv...)
	if len(s.args) == 0 {
		return f
	}

	out := make([]Field, 0, len(s.args)+len(f))
	out = append(out, s.args...)

	return append(out, f...)
}

func (s *testSugarLogger) Debugw(msg string, kv ...any) { s.tl.add("DEBUG", msg, s.merge(kv)) }
func (s *testSugarLogger) Infow(msg string, kv ...any)  { s.tl.add("INFO", msg, s.merge(kv)) }
func (s *testSugarLogger) Warnw(msg string, kv ...any)  { s.tl.add("WARN", msg, s.merge(kv)) }
func (s *testSugarLogger) Errorw(msg string, kv ...any) { s.tl.add("ERROR", msg, s.merge(kv)) }
func (s *testSugarLogger) Fatalw(msg string, kv ...any) { s.tl.add("FATAL", msg, s.merge(kv)) }

func (s *testSugarLogger) With(args ...any) SugarLogger {
	return &testSugarLogger{tl: s.tl, args: s.merge(args)}
}
