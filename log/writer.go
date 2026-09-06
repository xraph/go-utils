package log

import (
	"io"
	"os"
	"sync"
)

// syncWriter serialises writes so that one encoded log line reaches the
// underlying writer as exactly one Write call under one lock. Without this,
// two goroutines logging at the same time can interleave inside a line.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer

	// owned is true only when this package opened w and is therefore
	// responsible for closing it. A writer handed in by the caller, and
	// os.Stdout/os.Stderr, are never ours to close.
	owned bool
}

func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w}
}

// newOwnedSyncWriter wraps a writer this package opened and must close.
func newOwnedSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w, owned: true}
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.w.Write(p)
}

// Close releases the underlying writer, but only when this package opened it.
// Closing a caller's writer, or os.Stdout, would be a surprise; closing a file
// this package opened is the only way to release it, which on Windows is the
// difference between a log file that can be rotated or deleted and one that
// cannot.
func (s *syncWriter) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.owned {
		return nil
	}

	c, ok := s.w.(io.Closer)
	if !ok {
		return nil
	}

	return c.Close()
}

// Sync flushes the underlying writer when it supports it. Files do; buffers and
// pipes do not, and that is not an error.
func (s *syncWriter) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// syncWriter never buffers; every Write already reached the underlying
	// writer's own Write, so there is nothing for stdout/stderr to flush.
	// fsync-ing them fails with a platform-specific, meaningless error the
	// moment they are not a real terminal, which is the common case in
	// production (piped through a container log driver, redirected to a
	// file, captured by CI) and in `go test` output capture. Skip the call
	// for those two rather than surface an error that does not reflect any
	// lost data.
	if s.w == os.Stdout || s.w == os.Stderr {
		return nil
	}

	if sy, ok := s.w.(interface{ Sync() error }); ok {
		return sy.Sync()
	}

	if fl, ok := s.w.(interface{ Flush() error }); ok {
		return fl.Flush()
	}

	return nil
}
