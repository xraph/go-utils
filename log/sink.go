package log

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Record is one written log line, handed to every attached Sink.
type Record struct {
	Time    time.Time
	Level   LogLevel // "debug", "info", "warn", "error", "fatal"
	Message string
	Logger  string  // the Named path, empty for the root
	Caller  string  // "dir/file.go:123", empty when caller capture is off
	Fields  []Field // logger fields, then call fields; valid only during Emit
}

// Sink receives every line a logger writes, after the level check.
// Emit must not keep Record.Fields after it returns; copy what you need.
type Sink interface {
	Emit(rec Record)
}

// Flusher is implemented by a Sink that buffers. Fatal flushes every
// attached Flusher before the process exits.
type Flusher interface {
	Flush()
}

// ErrSinksUnsupported is returned by AttachSink for a Logger that cannot
// carry sinks: the noop logger, the capture logger, or a custom implementation.
var ErrSinksUnsupported = errors.New("log: logger does not support sinks")

// sinkCell is the set of sinks shared by a logger and every logger derived
// from it, the same way the level cell is shared. Readers load the pointer and
// never lock. Writers copy the slice under mu and store a new one.
type sinkCell struct {
	mu sync.Mutex
	p  atomic.Pointer[[]Sink]
}

func (c *sinkCell) add(s Sink) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var next []Sink

	if cur := c.p.Load(); cur != nil {
		next = make([]Sink, len(*cur), len(*cur)+1)
		copy(next, *cur)
	}

	next = append(next, s)
	c.p.Store(&next)
}

// remove drops the first sink equal to s. Detaching the same sink twice from
// one AttachSink call is guarded by the caller's sync.Once.
func (c *sinkCell) remove(s Sink) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cur := c.p.Load()
	if cur == nil {
		return
	}

	for i, have := range *cur {
		if have != s {
			continue
		}

		if len(*cur) == 1 {
			c.p.Store(nil)

			return
		}

		next := make([]Sink, 0, len(*cur)-1)
		next = append(next, (*cur)[:i]...)
		next = append(next, (*cur)[i+1:]...)
		c.p.Store(&next)

		return
	}
}

// AttachSink adds s to l and to every logger derived from l, including
// loggers With and Named returned before the call. detach removes it, and a
// second call to detach does nothing.
//
// The sink must be comparable (a pointer, usually), because detach finds it by
// identity.
func AttachSink(l Logger, s Sink) (detach func(), err error) {
	lg, ok := l.(*logger)
	if !ok || lg.sinks == nil {
		return func() {}, ErrSinksUnsupported
	}

	cell := lg.sinks
	cell.add(s)

	var once sync.Once

	return func() { once.Do(func() { cell.remove(s) }) }, nil
}

// emit hands one line to every attached sink. A sink that panics loses its own
// line and nothing else.
func emit(sinks []Sink, rec Record) {
	for _, s := range sinks {
		emitSafe(s, rec)
	}
}

func emitSafe(s Sink, rec Record) {
	defer func() { _ = recover() }()

	s.Emit(rec)
}

// logLevel maps the internal level onto the exported LogLevel.
func (l level) logLevel() LogLevel {
	switch l {
	case debugLevel:
		return LevelDebug
	case warnLevel:
		return LevelWarn
	case errorLevel:
		return LevelError
	case fatalLevel:
		return LevelFatal
	default:
		return LevelInfo
	}
}
