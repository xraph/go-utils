package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Example test demonstrating MockMetrics usage.
func TestMockMetrics_Example(t *testing.T) {
	mock := NewMockMetrics()

	// Test service lifecycle
	ctx := context.Background()

	if err := mock.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if mock.StartCalls != 1 {
		t.Errorf("expected 1 Start call, got %d", mock.StartCalls)
	}

	// Test metric creation
	counter := mock.Counter("test.counter")
	counter.Inc()
	counter.Add(5)

	if counter.Value() != 6 {
		t.Errorf("expected counter value 6, got %f", counter.Value())
	}

	// Test gauge
	gauge := mock.Gauge("test.gauge")
	gauge.Set(10)
	gauge.Inc()
	gauge.Dec()

	if gauge.Value() != 10 {
		t.Errorf("expected gauge value 10, got %f", gauge.Value())
	}

	// Test histogram
	histogram := mock.Histogram("test.histogram")
	histogram.Observe(1.5)
	histogram.Observe(2.5)
	histogram.Observe(3.5)

	// Test timer
	timer := mock.Timer("test.timer")
	done := timer.Time()

	time.Sleep(10 * time.Millisecond)
	done()

	if timer.Count() != 1 {
		t.Errorf("expected timer count 1, got %d", timer.Count())
	}

	// Test export
	data, err := mock.Export(ExportFormatJSON)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected export data, got empty")
	}

	// Test stats
	stats := mock.Stats()
	if stats.Name != "mock-metrics" {
		t.Errorf("expected name 'mock-metrics', got %s", stats.Name)
	}

	// Test health check
	if err := mock.Health(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Test stop
	if err := mock.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if mock.StopCalls != 1 {
		t.Errorf("expected 1 Stop call, got %d", mock.StopCalls)
	}
}

// Example test demonstrating MockHealthManager usage.
func TestMockHealthManager_Example(t *testing.T) {
	mock := NewMockHealthManager()
	ctx := context.Background()

	// Configure metadata
	mock.SetEnvironment("production")
	mock.SetVersion("2.0.0")
	mock.SetHostname("api-server-01")

	if mock.Environment() != "production" {
		t.Errorf("expected environment 'production', got %s", mock.Environment())
	}

	if mock.Version() != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", mock.Version())
	}

	// Register health checks
	dbCheck := NewMockHealthCheck("database").WithCritical(true)
	if err := mock.Register(dbCheck); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	cacheCheck := NewMockHealthCheck("cache").WithCritical(false)
	if err := mock.Register(cacheCheck); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify registration
	checks := mock.ListChecks()
	if len(checks) != 2 {
		t.Errorf("expected 2 registered checks, got %d", len(checks))
	}

	// Test health check
	report := mock.Check(ctx)
	if report == nil {
		t.Fatal("expected health report, got nil")
	}

	if report.Overall != HealthStatusHealthy {
		t.Errorf("expected healthy status, got %s", report.Overall)
	}

	if report.Environment != "production" {
		t.Errorf("expected environment 'production', got %s", report.Environment)
	}

	// Test individual check
	result := mock.CheckOne(ctx, "database")
	if result == nil {
		t.Fatal("expected health result, got nil")
	}

	if result.Name != "database" {
		t.Errorf("expected check name 'database', got %s", result.Name)
	}

	// Test status
	status := mock.Status()
	if status != HealthStatusHealthy {
		t.Errorf("expected healthy status, got %s", status)
	}

	// Test stats
	stats := mock.Stats()
	if stats.RegisteredChecks != 2 {
		t.Errorf("expected 2 registered checks in stats, got %d", stats.RegisteredChecks)
	}

	// Test subscription
	callback := func(result *HealthResult) {
		// Callback would be invoked on health status changes
	}

	if err := mock.Subscribe(callback); err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if mock.SubscribeCalls != 1 {
		t.Errorf("expected 1 Subscribe call, got %d", mock.SubscribeCalls)
	}

	// Test unregister
	if err := mock.Unregister("cache"); err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	checks = mock.ListChecks()
	if len(checks) != 1 {
		t.Errorf("expected 1 registered check after unregister, got %d", len(checks))
	}
}

// Example: Custom behavior injection for specific test scenarios.
func TestMockMetrics_CustomBehavior(t *testing.T) {
	mock := NewMockMetrics()

	// Inject custom counter behavior
	callCount := 0
	mock.CounterFunc = func(name string, opts ...MetricOption) Counter {
		callCount++
		counter := NewMockCounter()
		// Pre-initialize with a value for testing
		counter.Add(100)

		return counter
	}

	counter := mock.Counter("custom.counter")
	if counter.Value() != 100 {
		t.Errorf("expected pre-initialized value 100, got %f", counter.Value())
	}

	if callCount != 1 {
		t.Errorf("expected 1 CounterFunc call, got %d", callCount)
	}
}

// Example: Error injection for failure scenarios.
func TestMockHealthManager_ErrorInjection(t *testing.T) {
	mock := NewMockHealthManager()
	ctx := context.Background()

	// Inject error for testing failure scenarios
	mock.StartFunc = func(ctx context.Context) error {
		return context.DeadlineExceeded
	}

	err := mock.Start(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}

	// Inject unhealthy status
	mock.CheckFunc = func(ctx context.Context) *HealthReport {
		report := NewHealthReport()
		report.Overall = HealthStatusUnhealthy
		report.AddResult(NewHealthResult(
			"database",
			HealthStatusUnhealthy,
			"Connection timeout",
		).With(WithCritical(true)))

		return report
	}

	report := mock.Check(ctx)
	if !report.IsUnhealthy() {
		t.Errorf("expected unhealthy report, got %s", report.Overall)
	}

	// Verify critical failures
	analyzer := NewHealthReportAnalyzer(report)
	if analyzer.FailedCriticalCount() != 1 {
		t.Errorf("expected 1 failed critical check, got %d", analyzer.FailedCriticalCount())
	}
}
