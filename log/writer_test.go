package log

import (
	"bytes"
	"os"
	"path/filepath"
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

	for range goroutines {
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

// Close must actually release the descriptor, not just return nil. os.Remove
// succeeds on an open file on POSIX, so a test that only removes the file
// proves nothing; writing through the closed handle does.
func TestSyncWriterCloseReleasesAnOwnedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.log")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	w := newOwnedSyncWriter(f)
	if _, err := w.Write([]byte("before\n")); err != nil {
		t.Fatalf("Write before Close: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := f.WriteString("after\n"); err == nil {
		t.Error("the file is still open after Close")
	}
}

// A writer this package did not open must survive Close untouched.
func TestSyncWriterCloseSparesAnUnownedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unowned.log")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := newSyncWriter(f).Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := f.WriteString("still open\n"); err != nil {
		t.Errorf("Close released a file it does not own: %v", err)
	}
}

func TestSyncWriterSyncOnNonSyncer(t *testing.T) {
	// A bytes.Buffer has no Sync method; Sync must be a no-op, not a panic.
	w := newSyncWriter(&bytes.Buffer{})
	if err := w.Sync(); err != nil {
		t.Errorf("Sync() on a non-syncer returned %v, want nil", err)
	}
}

// os.Stdout and os.Stderr implement Sync() error themselves, and calling it
// fails with a platform-specific error the instant they are not a real
// terminal: piped, redirected to a file, or captured by `go test`, which is
// exactly how test binaries are usually run. syncWriter never buffers, so
// there is nothing to flush for either of them; Sync must not surface that
// OS-level error.
func TestSyncWriterIgnoresStdoutAndStderrSyncErrors(t *testing.T) {
	if err := newSyncWriter(os.Stdout).Sync(); err != nil {
		t.Errorf("Sync() on os.Stdout returned %v, want nil", err)
	}

	if err := newSyncWriter(os.Stderr).Sync(); err != nil {
		t.Errorf("Sync() on os.Stderr returned %v, want nil", err)
	}
}
