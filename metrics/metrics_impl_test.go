package metrics

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// COUNTER TESTS
// =============================================================================

func TestCounter_BasicOperations(t *testing.T) {
	counter := newCounter("test_counter")

	assert.Equal(t, 0.0, counter.Value())

	counter.Inc()
	assert.Equal(t, 1.0, counter.Value())

	counter.Add(5.5)
	assert.Equal(t, 6.5, counter.Value())

	// Counters can't decrease
	counter.Add(-1)
	assert.Equal(t, 6.5, counter.Value())
}

func TestCounter_ConcurrentIncrements(t *testing.T) {
	counter := newCounter("concurrent_counter")
	numGoroutines := 100
	incrementsPerGoroutine := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			for range incrementsPerGoroutine {
				counter.Inc()
			}
		}()
	}

	wg.Wait()

	expected := float64(numGoroutines * incrementsPerGoroutine)
	assert.Equal(t, expected, counter.Value())
}

func TestCounter_Exemplars(t *testing.T) {
	counter := newCounter("exemplar_counter")

	exemplar := Exemplar{
		Value:     10.0,
		Timestamp: time.Now(),
		TraceID:   "trace123",
		SpanID:    "span456",
	}

	counter.AddWithExemplar(10, exemplar)

	exemplars := counter.Exemplars()
	assert.NotEmpty(t, exemplars)
	assert.Equal(t, "trace123", exemplars[0].TraceID)
	assert.Equal(t, "span456", exemplars[0].SpanID)
}

func TestCounter_Timestamp(t *testing.T) {
	counter := newCounter("timestamp_counter")

	before := time.Now()

	time.Sleep(10 * time.Millisecond)
	counter.Inc()

	after := time.Now()

	ts := counter.Timestamp()
	assert.True(t, ts.After(before))
	assert.True(t, ts.Before(after) || ts.Equal(after))
}

func TestCounter_Describe(t *testing.T) {
	counter := newCounter("described_counter",
		WithDescription("Test counter"),
		WithUnit("requests"),
		WithNamespace("myapp"),
	)

	metadata := counter.Describe()
	assert.Equal(t, MetricTypeCounter, metadata.Type)
	assert.Equal(t, "Test counter", metadata.Description)
	assert.Equal(t, "requests", metadata.Unit)
	assert.Equal(t, "myapp", metadata.Namespace)
	assert.Contains(t, metadata.Name, "described_counter")
}

func TestCounter_Reset(t *testing.T) {
	counter := newCounter("reset_counter")
	counter.Add(100)
	assert.Equal(t, 100.0, counter.Value())

	err := counter.Reset()
	assert.NoError(t, err)
	assert.Equal(t, 0.0, counter.Value())
}

// =============================================================================
// GAUGE TESTS
// =============================================================================

func TestGauge_BasicOperations(t *testing.T) {
	gauge := newGauge("test_gauge")

	gauge.Set(42.5)
	assert.Equal(t, 42.5, gauge.Value())

	gauge.Inc()
	assert.Equal(t, 43.5, gauge.Value())

	gauge.Dec()
	assert.Equal(t, 42.5, gauge.Value())

	gauge.Add(10)
	assert.Equal(t, 52.5, gauge.Value())

	gauge.Sub(2.5)
	assert.Equal(t, 50.0, gauge.Value())
}

func TestGauge_NegativeValues(t *testing.T) {
	gauge := newGauge("negative_gauge")

	gauge.Set(-10.5)
	assert.Equal(t, -10.5, gauge.Value())

	gauge.Add(-5)
	assert.Equal(t, -15.5, gauge.Value())
}

func TestGauge_SetToCurrentTime(t *testing.T) {
	gauge := newGauge("time_gauge")

	before := time.Now().Unix()

	gauge.SetToCurrentTime()

	after := time.Now().Unix()

	value := gauge.Value()
	assert.True(t, value >= float64(before))
	assert.True(t, value <= float64(after))
}

func TestGauge_ConcurrentModifications(t *testing.T) {
	gauge := newGauge("concurrent_gauge")
	numGoroutines := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Half increment, half decrement
	for range numGoroutines {
		go func() {
			defer wg.Done()

			gauge.Inc()
		}()
		go func() {
			defer wg.Done()

			gauge.Dec()
		}()
	}

	wg.Wait()

	// Should be back to 0
	assert.Equal(t, 0.0, gauge.Value())
}

// =============================================================================
// HISTOGRAM TESTS
// =============================================================================

func TestHistogram_BasicObservations(t *testing.T) {
	histogram := newHistogram("test_histogram",
		WithBuckets(1, 5, 10, 50, 100),
	)

	histogram.Observe(3)
	histogram.Observe(7)
	histogram.Observe(25)
	histogram.Observe(75)

	assert.Equal(t, uint64(4), histogram.Count())
	assert.Equal(t, 110.0, histogram.Sum())
	assert.Equal(t, 27.5, histogram.Mean())
}

func TestHistogram_MinMax(t *testing.T) {
	histogram := newHistogram("minmax_histogram")

	histogram.Observe(10)
	histogram.Observe(5)
	histogram.Observe(20)
	histogram.Observe(3)

	assert.Equal(t, 3.0, histogram.Min())
	assert.Equal(t, 20.0, histogram.Max())
}

func TestHistogram_Quantiles(t *testing.T) {
	histogram := newHistogram("quantile_histogram",
		WithBuckets(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
	)

	// Add values: 0-99
	for i := range 100 {
		histogram.Observe(float64(i))
	}

	// P50 should be around 50
	p50 := histogram.Quantile(0.5)
	assert.InDelta(t, 50.0, p50, 10.0)

	// P95 should be around 95
	p95 := histogram.Quantile(0.95)
	assert.InDelta(t, 95.0, p95, 10.0)
}

func TestHistogram_Exemplars(t *testing.T) {
	histogram := newHistogram("exemplar_histogram")

	exemplar := Exemplar{
		TraceID: "trace789",
		SpanID:  "span012",
	}

	histogram.ObserveWithExemplar(42, exemplar)

	exemplars := histogram.Exemplars()
	require.NotEmpty(t, exemplars)
	assert.Equal(t, "trace789", exemplars[0].TraceID)
	assert.Equal(t, 42.0, exemplars[0].Value)
}

func TestHistogram_ConcurrentObservations(t *testing.T) {
	histogram := newHistogram("concurrent_histogram")
	numGoroutines := 100
	observationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(base int) {
			defer wg.Done()

			for j := range observationsPerGoroutine {
				histogram.Observe(float64(base + j))
			}
		}(i * observationsPerGoroutine)
	}

	wg.Wait()

	expected := uint64(numGoroutines * observationsPerGoroutine)
	assert.Equal(t, expected, histogram.Count())
}

func TestHistogram_EmptyState(t *testing.T) {
	histogram := newHistogram("empty_histogram")

	assert.Equal(t, uint64(0), histogram.Count())
	assert.Equal(t, 0.0, histogram.Sum())
	assert.Equal(t, 0.0, histogram.Mean())
	assert.Equal(t, 0.0, histogram.Min())
	assert.Equal(t, 0.0, histogram.Max())
	assert.Equal(t, 0.0, histogram.Quantile(0.5))
}

func TestHistogram_Reset(t *testing.T) {
	histogram := newHistogram("reset_histogram")

	for i := range 100 {
		histogram.Observe(float64(i))
	}

	assert.Equal(t, uint64(100), histogram.Count())

	err := histogram.Reset()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), histogram.Count())
	assert.Equal(t, 0.0, histogram.Sum())
}

// =============================================================================
// SUMMARY TESTS
// =============================================================================

func TestSummary_BasicObservations(t *testing.T) {
	summary := newSummary("test_summary")

	summary.Observe(10)
	summary.Observe(20)
	summary.Observe(30)
	summary.Observe(40)
	summary.Observe(50)

	assert.Equal(t, uint64(5), summary.Count())
	assert.Equal(t, 150.0, summary.Sum())
	assert.Equal(t, 30.0, summary.Mean())
}

func TestSummary_Quantiles(t *testing.T) {
	summary := newSummary("quantile_summary")

	// Add values 1-100
	for i := 1; i <= 100; i++ {
		summary.Observe(float64(i))
	}

	// Test quantiles
	p50 := summary.Quantile(0.5)
	assert.InDelta(t, 50.0, p50, 5.0) // Allow 5% error

	p90 := summary.Quantile(0.9)
	assert.InDelta(t, 90.0, p90, 5.0)

	p99 := summary.Quantile(0.99)
	assert.InDelta(t, 99.0, p99, 5.0)
}

func TestSummary_MinMax(t *testing.T) {
	summary := newSummary("minmax_summary")

	summary.Observe(100)
	summary.Observe(50)
	summary.Observe(200)
	summary.Observe(25)

	assert.Equal(t, 25.0, summary.Min())
	assert.Equal(t, 200.0, summary.Max())
}

func TestSummary_StdDev(t *testing.T) {
	summary := newSummary("stddev_summary")

	// Add values with known stddev
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	for _, v := range values {
		summary.Observe(v)
	}

	stddev := summary.StdDev()
	// Expected stddev is ~2.0
	assert.InDelta(t, 2.0, stddev, 0.5)
}

func TestSummary_ConcurrentObservations(t *testing.T) {
	summary := newSummary("concurrent_summary")
	numGoroutines := 50
	observationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			for j := range observationsPerGoroutine {
				summary.Observe(float64(j))
			}
		}()
	}

	wg.Wait()

	expected := uint64(numGoroutines * observationsPerGoroutine)
	assert.Equal(t, expected, summary.Count())
}

// =============================================================================
// TIMER TESTS
// =============================================================================

func TestTimer_BasicRecording(t *testing.T) {
	timer := newTimer("test_timer")

	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(300 * time.Millisecond)

	assert.Equal(t, uint64(3), timer.Count())

	sum := timer.Sum()
	assert.InDelta(t, 600*time.Millisecond, sum, float64(10*time.Millisecond))

	mean := timer.Mean()
	assert.InDelta(t, 200*time.Millisecond, mean, float64(10*time.Millisecond))
}

func TestTimer_TimeFunction(t *testing.T) {
	timer := newTimer("defer_timer")

	func() {
		defer timer.Time()()

		time.Sleep(50 * time.Millisecond)
	}()

	assert.Equal(t, uint64(1), timer.Count())

	duration := timer.Mean()
	assert.True(t, duration >= 50*time.Millisecond)
	assert.True(t, duration < 100*time.Millisecond)
}

func TestTimer_MinMax(t *testing.T) {
	timer := newTimer("minmax_timer")

	timer.Record(50 * time.Millisecond)
	timer.Record(100 * time.Millisecond)
	timer.Record(150 * time.Millisecond)

	minVal := timer.Min()
	assert.InDelta(t, 50*time.Millisecond, minVal, float64(5*time.Millisecond))

	maxVal := timer.Max()
	assert.InDelta(t, 150*time.Millisecond, maxVal, float64(5*time.Millisecond))
}

func TestTimer_Percentiles(t *testing.T) {
	timer := newTimer("percentile_timer",
		WithBuckets(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
	)

	// Record 100 timings: 0-99ms
	for i := range 100 {
		timer.Record(time.Duration(i) * time.Millisecond)
	}

	p50 := timer.Percentile(0.5)
	assert.InDelta(t, 50*time.Millisecond, p50, float64(10*time.Millisecond))

	p95 := timer.Percentile(0.95)
	assert.InDelta(t, 95*time.Millisecond, p95, float64(10*time.Millisecond))
}

func TestTimer_Exemplars(t *testing.T) {
	timer := newTimer("exemplar_timer")

	exemplar := Exemplar{
		TraceID: "timer_trace",
		SpanID:  "timer_span",
	}

	timer.RecordWithExemplar(100*time.Millisecond, exemplar)

	exemplars := timer.Exemplars()
	require.NotEmpty(t, exemplars)
	assert.Equal(t, "timer_trace", exemplars[0].TraceID)
}

// =============================================================================
// METRICS COLLECTOR TESTS
// =============================================================================

func TestMetricsCollector_Creation(t *testing.T) {
	collector := NewMetricsCollector("test_collector")
	assert.NotNil(t, collector)
	assert.Equal(t, "test_collector", collector.Name())
}

func TestMetricsCollector_StartStop(t *testing.T) {
	collector := NewMetricsCollector("lifecycle_collector")

	ctx := context.Background()

	err := collector.Start(ctx)
	assert.NoError(t, err)

	err = collector.Health(ctx)
	assert.NoError(t, err)

	err = collector.Stop(ctx)
	assert.NoError(t, err)
}

func TestMetricsCollector_CreateCounter(t *testing.T) {
	collector := NewMetricsCollector("counter_collector")

	counter1 := collector.Counter("requests_total")
	assert.NotNil(t, counter1)

	counter1.Inc()

	// Should return same instance
	counter2 := collector.Counter("requests_total")
	assert.Equal(t, 1.0, counter2.Value())
}

func TestMetricsCollector_CreateGauge(t *testing.T) {
	collector := NewMetricsCollector("gauge_collector")

	gauge := collector.Gauge("temperature")
	assert.NotNil(t, gauge)

	gauge.Set(23.5)
	assert.Equal(t, 23.5, gauge.Value())
}

func TestMetricsCollector_CreateHistogram(t *testing.T) {
	collector := NewMetricsCollector("histogram_collector")

	histogram := collector.Histogram("request_duration")
	assert.NotNil(t, histogram)

	histogram.Observe(100)
	assert.Equal(t, uint64(1), histogram.Count())
}

func TestMetricsCollector_CreateSummary(t *testing.T) {
	collector := NewMetricsCollector("summary_collector")

	summary := collector.Summary("response_size")
	assert.NotNil(t, summary)

	summary.Observe(1024)
	assert.Equal(t, uint64(1), summary.Count())
}

func TestMetricsCollector_CreateTimer(t *testing.T) {
	collector := NewMetricsCollector("timer_collector")

	timer := collector.Timer("processing_time")
	assert.NotNil(t, timer)

	timer.Record(50 * time.Millisecond)
	assert.Equal(t, uint64(1), timer.Count())
}

func TestMetricsCollector_ListMetrics(t *testing.T) {
	collector := NewMetricsCollector("list_collector")

	collector.Counter("counter1")
	collector.Gauge("gauge1")
	collector.Histogram("histogram1")
	collector.Summary("summary1")
	collector.Timer("timer1")

	metrics := collector.ListMetrics()
	assert.Len(t, metrics, 5)
}

func TestMetricsCollector_ListMetricsByType(t *testing.T) {
	collector := NewMetricsCollector("type_collector")

	collector.Counter("counter1")
	collector.Counter("counter2")
	collector.Gauge("gauge1")

	counters := collector.ListMetricsByType(MetricTypeCounter)
	assert.Len(t, counters, 2)

	gauges := collector.ListMetricsByType(MetricTypeGauge)
	assert.Len(t, gauges, 1)
}

func TestMetricsCollector_Stats(t *testing.T) {
	collector := NewMetricsCollector("stats_collector")

	collector.Counter("c1")
	collector.Gauge("g1")
	collector.Histogram("h1")

	stats := collector.Stats()
	assert.Equal(t, "stats_collector", stats.Name)
	assert.Equal(t, 3, stats.ActiveMetrics)
	assert.Equal(t, 1, stats.MetricsByType[MetricTypeCounter])
	assert.Equal(t, 1, stats.MetricsByType[MetricTypeGauge])
	assert.Equal(t, 1, stats.MetricsByType[MetricTypeHistogram])
}

func TestMetricsCollector_Reset(t *testing.T) {
	collector := NewMetricsCollector("reset_collector")

	counter := collector.Counter("counter")
	counter.Add(100)

	gauge := collector.Gauge("gauge")
	gauge.Set(50)

	err := collector.Reset()
	assert.NoError(t, err)

	assert.Equal(t, 0.0, counter.Value())
	assert.Equal(t, 0.0, gauge.Value())
}

func TestMetricsCollector_ResetMetric(t *testing.T) {
	collector := NewMetricsCollector("reset_metric_collector")

	counter := collector.Counter("counter")
	counter.Add(100)

	err := collector.ResetMetric("counter")
	assert.NoError(t, err)
	assert.Equal(t, 0.0, counter.Value())

	err = collector.ResetMetric("nonexistent")
	assert.Error(t, err)
}

// =============================================================================
// INTEGRATION TESTS
// =============================================================================

func TestIntegration_FullWorkflow(t *testing.T) {
	collector := NewMetricsCollector("integration_test")

	ctx := context.Background()
	require.NoError(t, collector.Start(ctx))

	// Create various metrics
	reqCounter := collector.Counter("http_requests_total",
		WithDescription("Total HTTP requests"),
		WithUnit("requests"),
		WithNamespace("myapp"),
	)

	latencyTimer := collector.Timer("http_request_duration",
		WithDescription("HTTP request latency"),
		WithUnit("ms"),
		WithDefaultTimerBuckets(),
	)

	activeConns := collector.Gauge("active_connections",
		WithDescription("Active connections"),
	)

	// Simulate application workload
	for i := range 100 {
		reqCounter.Inc()
		activeConns.Set(float64(i % 10))

		duration := time.Duration(10+i) * time.Millisecond
		latencyTimer.Record(duration)
	}

	// Verify metrics
	assert.Equal(t, 100.0, reqCounter.Value())
	assert.Equal(t, uint64(100), latencyTimer.Count())

	// Check metadata
	metadata := reqCounter.Describe()
	assert.Equal(t, "Total HTTP requests", metadata.Description)
	assert.Equal(t, "requests", metadata.Unit)
	assert.Contains(t, metadata.Name, "http_requests_total")

	// Check stats
	stats := collector.Stats()
	assert.True(t, stats.Started)
	assert.Equal(t, 3, stats.ActiveMetrics)

	require.NoError(t, collector.Stop(ctx))
}

func TestIntegration_HighConcurrency(t *testing.T) {
	collector := NewMetricsCollector("concurrency_test")

	counter := collector.Counter("concurrent_requests")
	histogram := collector.Histogram("concurrent_latency")
	gauge := collector.Gauge("concurrent_active")

	numWorkers := 100
	operationsPerWorker := 1000

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := range numWorkers {
		go func(id int) {
			defer wg.Done()

			for j := range operationsPerWorker {
				counter.Inc()
				histogram.Observe(float64(j))
				gauge.Set(float64(id))
			}
		}(i)
	}

	wg.Wait()

	expectedCount := float64(numWorkers * operationsPerWorker)
	assert.Equal(t, expectedCount, counter.Value())
	assert.Equal(t, uint64(numWorkers*operationsPerWorker), histogram.Count())
}

// =============================================================================
// EDGE CASES AND ERROR CONDITIONS
// =============================================================================

func TestEdgeCase_LargeValues(t *testing.T) {
	counter := newCounter("large_counter")

	largeValue := math.MaxFloat64 / 2
	counter.Add(largeValue)

	assert.False(t, math.IsInf(counter.Value(), 0))
	assert.False(t, math.IsNaN(counter.Value()))
}

func TestEdgeCase_VerySmallValues(t *testing.T) {
	gauge := newGauge("small_gauge")

	smallValue := math.SmallestNonzeroFloat64
	gauge.Set(smallValue)

	assert.Equal(t, smallValue, gauge.Value())
}

func TestEdgeCase_NegativeCounterAdd(t *testing.T) {
	counter := newCounter("negative_test_counter")

	counter.Add(10)
	counter.Add(-5) // Should be ignored

	assert.Equal(t, 10.0, counter.Value())
}

func TestEdgeCase_EmptyBuckets(t *testing.T) {
	// Should use default buckets
	histogram := newHistogram("default_bucket_histogram")

	histogram.Observe(50)
	assert.Equal(t, uint64(1), histogram.Count())
}

// =============================================================================
// LABEL TESTS
// =============================================================================

func TestLabels_WithLabels(t *testing.T) {
	counter := newCounter("labeled_counter",
		WithLabels(map[string]string{
			"method": "GET",
			"path":   "/api/users",
		}),
	)

	counter.Inc()
	assert.Equal(t, 1.0, counter.Value())

	// Create variant with different labels
	postCounter := counter.WithLabels(map[string]string{
		"method": "POST",
	})

	postCounter.Add(5)

	// Original should be unchanged
	assert.Equal(t, 1.0, counter.Value())
	// New variant should have its own value
	assert.Equal(t, 5.0, postCounter.Value())
}

func TestLabels_ConstLabels(t *testing.T) {
	counter := newCounter("const_labeled_counter",
		WithConstLabels(map[string]string{
			"version": "1.0.0",
			"env":     "production",
		}),
	)

	metadata := counter.Describe()
	assert.Equal(t, "1.0.0", metadata.ConstLabels["version"])
	assert.Equal(t, "production", metadata.ConstLabels["env"])
}
