package log_test

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/xraph/go-utils/log"
)

// TestBeautifulLogger tests BeautifulLogger functionality.
func TestBeautifulLogger(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		logger := log.NewBeautifulLogger("test-app")
		if logger == nil {
			t.Fatal("Expected non-nil logger")
		}
	})

	t.Run("WithFields", func(t *testing.T) {
		logger := log.NewBeautifulLogger("test")

		childLog := logger.With(
			log.String("service", "test-service"),
			log.Int("version", 1),
		)

		// Just verify no panics
		childLog.Info("message with fields")
	})

	t.Run("WithContext", func(t *testing.T) {
		logger := log.NewBeautifulLogger("test")

		ctx := context.Background()
		ctx = log.WithRequestID(ctx, "req-123")
		ctx = log.WithUserID(ctx, "user-456")

		contextLog := logger.WithContext(ctx)
		contextLog.Info("context message")
	})

	t.Run("Named", func(t *testing.T) {
		logger := log.NewBeautifulLogger("parent")

		namedLog := logger.Named("child")
		namedLog.Info("named logger message")
	})

	t.Run("FormattedLogging", func(t *testing.T) {
		logger := log.NewBeautifulLogger("test")

		logger.Infof("formatted message: %s, number: %d", "test", 42)
	})

	t.Run("ConcurrentLogging", func(t *testing.T) {
		logger := log.NewBeautifulLogger("test")

		// Test concurrent writes don't cause data races
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func(id int) {
				logger.Info("concurrent message", log.Int("id", id))
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})

	t.Run("CompactLogger", func(t *testing.T) {
		logger := log.NewBeautifulLoggerCompact("test")
		logger.Info("compact message")
	})

	t.Run("MinimalLogger", func(t *testing.T) {
		logger := log.NewBeautifulLoggerMinimal("test")
		logger.Info("minimal message")
	})

	t.Run("JSONLogger", func(t *testing.T) {
		logger := log.NewBeautifulLoggerJSON("test")
		logger.Info("json message")
	})
}

// BenchmarkBeautifulLogger benchmarks BeautifulLogger performance.
func BenchmarkBeautifulLogger(b *testing.B) {
	// Redirect output to discard
	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = old
		w.Close()
	}()

	logger := log.NewBeautifulLogger("bench")

	fields := []log.Field{
		log.String("operation", "benchmark"),
		log.Int("count", 100),
		log.Duration("elapsed", time.Millisecond),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", fields...)
	}
}

// BenchmarkBeautifulLoggerWith benchmarks With() method.
func BenchmarkBeautifulLoggerWith(b *testing.B) {
	// Discard output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	go io.Copy(io.Discard, r)
	defer func() {
		os.Stdout = old
		w.Close()
		r.Close()
	}()

	logger := log.NewBeautifulLogger("bench")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		childLog := logger.With(log.String("key", "value"))
		childLog.Info("test")
	}
}
