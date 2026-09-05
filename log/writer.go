package log

import (
	"io"
	"sync"
)

// syncWriter serialises writes so that one encoded log line reaches the
// underlying writer as exactly one Write call under one lock. Without this,
// two goroutines logging at the same time can interleave inside a line.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSyncWriter(w io.Writer) *syncWriter {
	return &syncWriter{w: w}
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.w.Write(p)
}

// Sync flushes the underlying writer when it supports it. Files do; buffers and
// pipes do not, and that is not an error.
func (s *syncWriter) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sy, ok := s.w.(interface{ Sync() error }); ok {
		return sy.Sync()
	}
	if fl, ok := s.w.(interface{ Flush() error }); ok {
		return fl.Flush()
	}

	return nil
}
