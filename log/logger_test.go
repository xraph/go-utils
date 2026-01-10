package log_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xraph/go-utils/log"
)

// BenchmarkLogger compares performance between different logger implementations.
func BenchmarkLogger(b *testing.B) {
	// Test data
	ctx := context.Background()
	ctx = log.WithRequestID(ctx, "test-request-123")
	ctx = log.WithUserID(ctx, "test-user-456")

	testFields := []log.Field{
		log.String("operation", "benchmark_test"),
		log.Int("iteration", 1000),
		log.Duration("elapsed", 100*time.Millisecond),
		log.Bool("success", true),
	}

	b.Run("NoopLogger", func(b *testing.B) {
		noopLog := log.NewNoopLogger()
		contextLog := noopLog.WithContext(ctx)

		b.ResetTimer()

		for range b.N {
			contextLog.Info("Benchmark test message", testFields...)
			contextLog.Error("Benchmark error message", append(testFields, log.Error(errors.New("test error")))...)
		}
	})

	b.Run("ProductionLogger", func(b *testing.B) {
		prodLog := log.NewProductionLogger()
		contextLog := prodLog.WithContext(ctx)

		b.ResetTimer()

		for range b.N {
			contextLog.Info("Benchmark test message", testFields...)
			contextLog.Error("Benchmark error message", append(testFields, log.Error(errors.New("test error")))...)
		}

		prodLog.Sync()
	})
}

// TestNoopLogger ensures noop logger implements interface correctly.
func TestNoopLogger(t *testing.T) {
	noopLog := log.NewNoopLogger()

	// Verify it implements the Logger interface
	var _ log.Logger = noopLog

	// Test all methods don't panic
	t.Run("BasicLogging", func(t *testing.T) {
		noopLog.Debug("debug message")
		noopLog.Info("info message")
		noopLog.Warn("warn message")
		noopLog.Error("error message")
		// Skip Fatal as it would terminate test

		noopLog.Debugf("debug %s", "formatted")
		noopLog.Infof("info %d", 42)
		noopLog.Warnf("warn %v", true)
		noopLog.Errorf("error %s", "test")
	})

	t.Run("WithMethods", func(t *testing.T) {
		ctx := context.Background()
		ctx = log.WithRequestID(ctx, "test-123")

		withFieldsLog := noopLog.With(log.String("key", "value"))
		withContextLog := noopLog.WithContext(ctx)
		namedLog := noopLog.Named("test")

		// Verify they return loggers (should be noop instances)
		var (
			_ log.Logger = withFieldsLog
			_ log.Logger = withContextLog
			_ log.Logger = namedLog
		)

		// Test chaining
		chainedLog := noopLog.With(log.String("k1", "v1")).
			WithContext(ctx).
			Named("chained").
			With(log.String("k2", "v2"))

		chainedLog.Info("This won't log anything")
	})

	t.Run("Sugar", func(t *testing.T) {
		sugar := noopLog.Sugar()

		var _ log.SugarLogger = sugar

		sugar.Infow("info with fields", "key1", "value1", "key2", 42)
		sugar.Errorw("error with fields", "error", "test error")

		chainedSugar := sugar.With("persistent", "field")
		chainedSugar.Debugw("debug message", "additional", "field")
	})

	t.Run("Sync", func(t *testing.T) {
		err := noopLog.Sync()
		if err != nil {
			t.Errorf("Sync should not return error, got: %v", err)
		}
	})
}

// TestLoggerInterface ensures all logger implementations satisfy the interface.
func TestLoggerInterface(t *testing.T) {
	testCases := []struct {
		name   string
		logger log.Logger
	}{
		{"NoopLogger", log.NewNoopLogger()},
		{"DevelopmentLogger", log.NewDevelopmentLogger()},
		{"ProductionLogger", log.NewProductionLogger()},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify interface compliance
			var _ log.Logger = tc.logger

			// Test that methods don't panic (except Fatal)
			tc.logger.Debug("test debug")
			tc.logger.Info("test info")
			tc.logger.Warn("test warn")
			tc.logger.Error("test error")

			tc.logger.Debugf("test debug %s", "formatted")
			tc.logger.Infof("test info %d", 42)
			tc.logger.Warnf("test warn %v", true)
			tc.logger.Errorf("test error %s", "formatted")

			// Test With methods
			withFields := tc.logger.With(log.String("test", "value"))

			var _ log.Logger = withFields

			ctx := context.Background()
			withContext := tc.logger.WithContext(ctx)

			var _ log.Logger = withContext

			named := tc.logger.Named("test")

			var _ log.Logger = named

			// Test Sugar
			sugar := tc.logger.Sugar()

			var _ log.SugarLogger = sugar

			// Test Sync
			err := tc.logger.Sync()
			// Only check error for non-noop loggers
			if tc.name != "NoopLogger" && err != nil {
				t.Logf("Sync returned error (may be expected): %v", err)
			}
		})
	}
}

// TestContextFields tests context field extraction.
func TestContextFields(t *testing.T) {
	ctx := context.Background()
	ctx = log.WithRequestID(ctx, "req-123")
	ctx = log.WithTraceID(ctx, "trace-456")
	ctx = log.WithUserID(ctx, "user-789")

	// Test field extraction
	requestID := log.RequestIDFromContext(ctx)
	if requestID != "req-123" {
		t.Errorf("Expected request ID 'req-123', got '%s'", requestID)
	}

	traceID := log.TraceIDFromContext(ctx)
	if traceID != "trace-456" {
		t.Errorf("Expected trace ID 'trace-456', got '%s'", traceID)
	}

	userID := log.UserIDFromContext(ctx)
	if userID != "user-789" {
		t.Errorf("Expected user ID 'user-789', got '%s'", userID)
	}

	// Test with empty context
	emptyRequestID := log.RequestIDFromContext(context.Background())
	if emptyRequestID != "" {
		t.Errorf("Expected empty request ID from empty context, got '%s'", emptyRequestID)
	}
}

// TestPerformanceMonitor tests performance monitoring with noop log.
func TestPerformanceMonitor(t *testing.T) {
	noopLog := log.NewNoopLogger()

	t.Run("BasicMonitoring", func(t *testing.T) {
		pm := log.NewPerformanceMonitor(noopLog, "test_operation")
		pm.WithField(log.String("test", "value"))

		time.Sleep(10 * time.Millisecond)

		// Should not panic
		pm.Finish()
	})

	t.Run("ErrorMonitoring", func(t *testing.T) {
		pm := log.NewPerformanceMonitor(noopLog, "test_operation_with_error")

		time.Sleep(5 * time.Millisecond)

		// Should not panic
		pm.FinishWithError(errors.New("test error"))
	})
}

// TestStructuredLogging tests structured logging with noop log.
func TestStructuredLogging(t *testing.T) {
	noopLog := log.NewNoopLogger()

	t.Run("BasicStructured", func(t *testing.T) {
		structured := log.NewStructuredLog(noopLog)
		structured.WithField(log.String("key1", "value1")).
			WithFields(log.String("key2", "value2"), log.Int("key3", 42)).
			Info("Test message")
	})

	t.Run("WithGroups", func(t *testing.T) {
		structured := log.NewStructuredLog(noopLog)

		httpGroup := log.HTTPRequestGroup("GET", "/api/test", "TestAgent/1.0", 200)
		structured.WithGroup(httpGroup).Info("HTTP request")

		dbGroup := log.DatabaseQueryGroup("SELECT * FROM test", "test_table", 100, 50*time.Millisecond)
		structured.WithGroup(dbGroup).Info("Database query")

		serviceGroup := log.ServiceInfoGroup("test-service", "1.0.0", "test")
		structured.WithGroup(serviceGroup).Info("Service info")
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := context.Background()
		ctx = log.WithRequestID(ctx, "test-req-123")

		structured := log.NewStructuredLog(noopLog)
		structured.WithContext(ctx).Info("Context message")
	})
}

// BenchmarkFieldCreation compares field creation performance.
func BenchmarkFieldCreation(b *testing.B) {
	b.Run("BasicFields", func(b *testing.B) {
		for i := range b.N {
			fields := []log.Field{
				log.String("operation", "benchmark"),
				log.Int("iteration", i),
				log.Bool("success", true),
				log.Duration("elapsed", time.Millisecond),
			}
			_ = fields
		}
	})

	b.Run("LazyFields", func(b *testing.B) {
		for i := range b.N {
			fields := []log.Field{
				log.Lazy("timestamp", func() any {
					return time.Now().Unix()
				}),
				log.Lazy("random", func() any {
					return i * 42
				}),
			}
			_ = fields
		}
	})

	b.Run("ConditionalFields", func(b *testing.B) {
		for i := range b.N {
			fields := []log.Field{
				log.Conditional(i%2 == 0, "even", true),
				log.Conditional(i%3 == 0, "divisible_by_three", true),
				log.Nullable("value", func() any {
					if i > 100 {
						return i
					}

					return nil
				}()),
			}
			_ = fields
		}
	})
}
