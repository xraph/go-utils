package log

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// Regression test for finding 7. The old ColoredWriteSyncer issued three
// separate writes per line (colour prefix, content, reset), so a concurrent
// goroutine could land in the middle of one line.
func TestSyncWriterDoesNotTearLines(t *testing.T) {
	var buf bytes.Buffer
	w := newSyncWriter(&buf)

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			// A long line makes interleaving easy to detect.
			w.Write([]byte("START" + strings.Repeat("x", 500) + "END\n"))
		}()
	}
	wg.Wait()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != goroutines {
		t.Fatalf("got %d lines, want %d", len(lines), goroutines)
	}
	for i, ln := range lines {
		if !strings.HasPrefix(ln, "START") || !strings.HasSuffix(ln, "END") {
			t.Errorf("line %d is torn: %.60q", i, ln)
		}
		if len(ln) != 5+500+3 {
			t.Errorf("line %d has length %d, want %d", i, len(ln), 5+500+3)
		}
	}
}

func TestSyncWriterSyncOnNonSyncer(t *testing.T) {
	// A bytes.Buffer has no Sync method; Sync must be a no-op, not a panic.
	w := newSyncWriter(&bytes.Buffer{})
	if err := w.Sync(); err != nil {
		t.Errorf("Sync() on a non-syncer returned %v, want nil", err)
	}
}
