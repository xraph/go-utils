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

// sinkEntry is one attachment. The cell holds pointers to entries so detach
// can find its own attachment by identity, whatever the sink's dynamic type.
// A func-typed or slice-holding Sink is not comparable, and the same sink
// attached twice is two attachments.
type sinkEntry struct {
	s Sink
}

// sinkCell is the set of sinks shared by every logger that came from one New
// call, the same way the level cell is shared. Readers load the pointer and
// never lock. Writers copy the slice under mu and store a new one.
type sinkCell struct {
	mu sync.Mutex
	p  atomic.Pointer[[]*sinkEntry]
}

func (c *sinkCell) add(e *sinkEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var next []*sinkEntry

	if cur := c.p.Load(); cur != nil {
		next = make([]*sinkEntry, len(*cur), len(*cur)+1)
		copy(next, *cur)
	}

	next = append(next, e)
	c.p.Store(&next)
}

// remove drops e from the cell. It is a no-op when e is already gone.
func (c *sinkCell) remove(e *sinkEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cur := c.p.Load()
	if cur == nil {
		return
	}

	for i, have := range *cur {
		if have != e {
			continue
		}

		if len(*cur) == 1 {
			c.p.Store(nil)

			return
		}

		next := make([]*sinkEntry, 0, len(*cur)-1)
		next = append(next, (*cur)[:i]...)
		next = append(next, (*cur)[i+1:]...)
		c.p.Store(&next)

		return
	}
}

// AttachSink adds s to l. The sink set is shared by every logger built from
// the same New call, so s receives every line written by any logger that
// shares l's root: l itself, loggers derived from it with With or Named
// (including ones returned before this call), and its parents and siblings.
// Attaching to a child therefore also delivers lines written through the root.
//
// detach removes this attachment only, and a second call to detach does
// nothing. Attaching the same sink twice makes two attachments, each removed by
// its own detach. Any Sink works, comparable or not.
func AttachSink(l Logger, s Sink) (detach func(), err error) {
	lg, ok := l.(*logger)
	if !ok || lg.sinks == nil {
		return func() {}, ErrSinksUnsupported
	}

	cell := lg.sinks
	e := &sinkEntry{s: s}
	cell.add(e)

	var once sync.Once

	return func() { once.Do(func() { cell.remove(e) }) }, nil
}

// emit hands one line to every attached sink. A sink that panics loses its own
// line and nothing else.
func emit(sinks []*sinkEntry, rec Record) {
	for _, e := range sinks {
		emitSafe(e.s, rec)
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
