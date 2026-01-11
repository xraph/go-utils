package testdata_test

import (
	"context"
	"testing"

	"github.com/xraph/go-utils/metrics"
)

// TestExternalPackageCanUseMocks verifies that mocks are accessible from external packages.
func TestExternalPackageCanUseMocks(t *testing.T) {
	// This test is in a different package (testdata_test) to ensure
	// the mocks are properly exported and usable by other packages.

	mock := metrics.NewMockMetrics()

	ctx := context.Background()

	if err := mock.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	counter := mock.Counter("test.counter")
	counter.Inc()

	if counter.Value() != 1 {
		t.Errorf("expected counter value 1, got %f", counter.Value())
	}

	if mock.CounterCalls != 1 {
		t.Errorf("expected 1 Counter call, got %d", mock.CounterCalls)
	}

	if err := mock.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

// TestExternalPackageCanUseHealthMocks verifies health manager mocks are accessible.
func TestExternalPackageCanUseHealthMocks(t *testing.T) {
	mock := metrics.NewMockHealthManager()

	ctx := context.Background()

	mock.SetEnvironment("test")
	mock.SetVersion("1.0.0")

	if mock.Environment() != "test" {
		t.Errorf("expected environment 'test', got %s", mock.Environment())
	}

	report := mock.Check(ctx)
	if report == nil {
		t.Fatal("expected health report, got nil")
	}

	if !report.IsHealthy() {
		t.Errorf("expected healthy status, got %s", report.Overall)
	}

	if mock.CheckCalls != 1 {
		t.Errorf("expected 1 Check call, got %d", mock.CheckCalls)
	}
}

