package log

import (
	"errors"
	"io"
	"testing"
	"time"
)

var errBench = errors.New("boom")

func benchFields() []Field {
	return []Field{
		String("method", "GET"),
		String("path", "/v1/users"),
		Int("status", 200),
		Duration("latency", 3*time.Millisecond),
		Error(errBench),
	}
}

func BenchmarkJSONEncoder(b *testing.B) {
	l := New(Config{Format: FormatJSON, Output: io.Discard, Level: LevelInfo})

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		l.Info("request completed", benchFields()...)
	}
}

func BenchmarkPrettyEncoder(b *testing.B) {
	l := New(Config{Format: FormatPretty, Output: io.Discard, Level: LevelInfo})

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		l.Info("request completed", benchFields()...)
	}
}

func BenchmarkDisabledDebug(b *testing.B) {
	l := New(Config{Format: FormatJSON, Output: io.Discard, Level: LevelInfo})

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		l.Debug("cache lookup", String("key", "user:1234"), Int("shard", 7))
	}
}

func BenchmarkNoop(b *testing.B) {
	l := NewNoopLogger()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		l.Info("cache lookup", String("key", "user:1234"), Int("shard", 7))
	}
}

func BenchmarkWith(b *testing.B) {
	l := New(Config{Format: FormatJSON, Output: io.Discard}).With(
		String("service", "api"), String("version", "1.2.3"), String("env", "prod"))

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_ = l.With(String("request_id", "abc123"))
	}
}
