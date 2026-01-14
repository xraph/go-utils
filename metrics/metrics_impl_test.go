package metrics

import (
	"context"
	"fmt"
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
	counter := NewCounter("test_counter")

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
	counter := NewCounter("concurrent_counter")
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
	counter := NewCounter("exemplar_counter")

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
	counter := NewCounter("timestamp_counter")

	before := time.Now()

	time.Sleep(10 * time.Millisecond)
	counter.Inc()

	after := time.Now()

	ts := counter.Timestamp()
	assert.True(t, ts.After(before))
	assert.True(t, ts.Before(after) || ts.Equal(after))
}

func TestCounter_Describe(t *testing.T) {
	counter := NewCounter("described_counter",
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
	counter := NewCounter("reset_counter")
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
	gauge := NewGauge("test_gauge")

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
	gauge := NewGauge("negative_gauge")

	gauge.Set(-10.5)
	assert.Equal(t, -10.5, gauge.Value())

	gauge.Add(-5)
	assert.Equal(t, -15.5, gauge.Value())
}

func TestGauge_SetToCurrentTime(t *testing.T) {
	gauge := NewGauge("time_gauge")

	before := time.Now().Unix()

	gauge.SetToCurrentTime()

	after := time.Now().Unix()

	value := gauge.Value()
	assert.True(t, value >= float64(before))
	assert.True(t, value <= float64(after))
}

func TestGauge_ConcurrentModifications(t *testing.T) {
	gauge := NewGauge("concurrent_gauge")
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
	histogram := NewHistogram("test_histogram",
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
	histogram := NewHistogram("minmax_histogram")

	histogram.Observe(10)
	histogram.Observe(5)
	histogram.Observe(20)
	histogram.Observe(3)

	assert.Equal(t, 3.0, histogram.Min())
	assert.Equal(t, 20.0, histogram.Max())
}

func TestHistogram_Quantiles(t *testing.T) {
	histogram := NewHistogram("quantile_histogram",
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

func TestHistogram_Percentile(t *testing.T) {
	histogram := NewHistogram("percentile_histogram",
		WithBuckets(0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100),
	)

	// Add values: 0-99
	for i := range 100 {
		histogram.Observe(float64(i))
	}

	// P50 should be around 50
	p50 := histogram.Percentile(0.5)
	assert.InDelta(t, 50.0, p50, 10.0)

	// P95 should be around 95
	p95 := histogram.Percentile(0.95)
	assert.InDelta(t, 95.0, p95, 10.0)

	// Percentile and Quantile should return the same value
	assert.Equal(t, histogram.Percentile(0.5), histogram.Quantile(0.5))
	assert.Equal(t, histogram.Percentile(0.95), histogram.Quantile(0.95))
	assert.Equal(t, histogram.Percentile(0.99), histogram.Quantile(0.99))
}

func TestHistogram_Exemplars(t *testing.T) {
	histogram := NewHistogram("exemplar_histogram")

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
	histogram := NewHistogram("concurrent_histogram")
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
	histogram := NewHistogram("empty_histogram")

	assert.Equal(t, uint64(0), histogram.Count())
	assert.Equal(t, 0.0, histogram.Sum())
	assert.Equal(t, 0.0, histogram.Mean())
	assert.Equal(t, 0.0, histogram.Min())
	assert.Equal(t, 0.0, histogram.Max())
	assert.Equal(t, 0.0, histogram.Quantile(0.5))
}

func TestHistogram_Reset(t *testing.T) {
	histogram := NewHistogram("reset_histogram")

	for i := range 100 {
		histogram.Observe(float64(i))
	}

	assert.Equal(t, uint64(100), histogram.Count())

	err := histogram.Reset()
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), histogram.Count())
	assert.Equal(t, 0.0, histogram.Sum())
}

func TestHistogram_Buckets(t *testing.T) {
	// Create histogram with custom buckets
	histogram := NewHistogram("bucket_histogram",
		WithBuckets(10, 20, 30, 40, 50),
	)

	// Observe values in different buckets
	histogram.Observe(5)   // bucket 10
	histogram.Observe(15)  // bucket 20
	histogram.Observe(15)  // bucket 20
	histogram.Observe(25)  // bucket 30
	histogram.Observe(35)  // bucket 40
	histogram.Observe(45)  // bucket 50
	histogram.Observe(100) // bucket +Inf (last bucket)

	buckets := histogram.Buckets()
	assert.NotNil(t, buckets)
	assert.Equal(t, 5, len(buckets), "Should have 5 buckets")

	// Verify bucket boundaries are present
	assert.Contains(t, buckets, 10.0)
	assert.Contains(t, buckets, 20.0)
	assert.Contains(t, buckets, 30.0)
	assert.Contains(t, buckets, 40.0)
	assert.Contains(t, buckets, 50.0)

	// Verify bucket counts
	assert.Equal(t, uint64(1), buckets[10.0], "Bucket 10 should have 1 observation")
	assert.Equal(t, uint64(2), buckets[20.0], "Bucket 20 should have 2 observations")
	assert.Equal(t, uint64(1), buckets[30.0], "Bucket 30 should have 1 observation")
	assert.Equal(t, uint64(1), buckets[40.0], "Bucket 40 should have 1 observation")
	assert.Equal(t, uint64(1), buckets[50.0], "Bucket 50 should have 1 observation")
}

func TestHistogram_Buckets_DefaultBuckets(t *testing.T) {
	histogram := NewHistogram("default_bucket_histogram")

	// Add some observations
	histogram.Observe(1)
	histogram.Observe(50)
	histogram.Observe(500)
	histogram.Observe(5000)

	buckets := histogram.Buckets()
	assert.NotNil(t, buckets)
	assert.Greater(t, len(buckets), 0, "Should have buckets")

	// Verify it returns the default bucket structure
	assert.Contains(t, buckets, 10.0)
	assert.Contains(t, buckets, 100.0)
	assert.Contains(t, buckets, 1000.0)
}

func TestHistogram_Buckets_Empty(t *testing.T) {
	histogram := NewHistogram("empty_bucket_histogram")

	buckets := histogram.Buckets()
	assert.NotNil(t, buckets)

	// All buckets should be zero
	for _, count := range buckets {
		assert.Equal(t, uint64(0), count)
	}
}

// =============================================================================
// SUMMARY TESTS
// =============================================================================

func TestSummary_BasicObservations(t *testing.T) {
	summary := NewSummary("test_summary")

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
	summary := NewSummary("quantile_summary")

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
	summary := NewSummary("minmax_summary")

	summary.Observe(100)
	summary.Observe(50)
	summary.Observe(200)
	summary.Observe(25)

	assert.Equal(t, 25.0, summary.Min())
	assert.Equal(t, 200.0, summary.Max())
}

func TestSummary_StdDev(t *testing.T) {
	summary := NewSummary("stddev_summary")

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
	summary := NewSummary("concurrent_summary")
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
	timer := NewTimer("test_timer")

	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(300 * time.Millisecond)

	assert.Equal(t, uint64(3), timer.Count())

	sum := timer.Sum()
	assert.InDelta(t, 600*time.Millisecond, sum, float64(10*time.Millisecond))

	mean := timer.Mean()
	assert.InDelta(t, 200*time.Millisecond, mean, float64(10*time.Millisecond))
}

func TestTimer_Value(t *testing.T) {
	timer := NewTimer("test_timer")

	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(300 * time.Millisecond)

	// Value() should return the same as Sum()
	value := timer.Value()
	sum := timer.Sum()

	assert.Equal(t, sum, value, "Value() should equal Sum()")
	assert.InDelta(t, 600*time.Millisecond, value, float64(10*time.Millisecond))
}

func TestTimer_TimeFunction(t *testing.T) {
	timer := NewTimer("defer_timer")

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
	timer := NewTimer("minmax_timer")

	timer.Record(50 * time.Millisecond)
	timer.Record(100 * time.Millisecond)
	timer.Record(150 * time.Millisecond)

	minVal := timer.Min()
	assert.InDelta(t, 50*time.Millisecond, minVal, float64(5*time.Millisecond))

	maxVal := timer.Max()
	assert.InDelta(t, 150*time.Millisecond, maxVal, float64(5*time.Millisecond))
}

func TestTimer_Percentiles(t *testing.T) {
	timer := NewTimer("percentile_timer",
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
	timer := NewTimer("exemplar_timer")

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
	counter := NewCounter("large_counter")

	largeValue := math.MaxFloat64 / 2
	counter.Add(largeValue)

	assert.False(t, math.IsInf(counter.Value(), 0))
	assert.False(t, math.IsNaN(counter.Value()))
}

func TestEdgeCase_VerySmallValues(t *testing.T) {
	gauge := NewGauge("small_gauge")

	smallValue := math.SmallestNonzeroFloat64
	gauge.Set(smallValue)

	assert.Equal(t, smallValue, gauge.Value())
}

func TestEdgeCase_NegativeCounterAdd(t *testing.T) {
	counter := NewCounter("negative_test_counter")

	counter.Add(10)
	counter.Add(-5) // Should be ignored

	assert.Equal(t, 10.0, counter.Value())
}

func TestEdgeCase_EmptyBuckets(t *testing.T) {
	// Should use default buckets
	histogram := NewHistogram("default_bucket_histogram")

	histogram.Observe(50)
	assert.Equal(t, uint64(1), histogram.Count())
}

// =============================================================================
// LABEL TESTS
// =============================================================================

func TestLabels_WithLabels(t *testing.T) {
	counter := NewCounter("labeled_counter",
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
	counter := NewCounter("const_labeled_counter",
		WithConstLabels(map[string]string{
			"version": "1.0.0",
			"env":     "production",
		}),
	)

	metadata := counter.Describe()
	assert.Equal(t, "1.0.0", metadata.ConstLabels["version"])
	assert.Equal(t, "production", metadata.ConstLabels["env"])
}

// =============================================================================
// DEFAULT TAGS TESTS
// =============================================================================

func TestMetricsCollector_DefaultTags_Counter(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			Namespace: "myapp",
			DefaultTags: map[string]string{
				"env":     "production",
				"service": "api",
				"version": "1.0.0",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	counter := collector.Counter("requests")

	metadata := counter.Describe()
	assert.Equal(t, "myapp", metadata.Namespace)
	assert.Equal(t, "production", metadata.ConstLabels["env"])
	assert.Equal(t, "api", metadata.ConstLabels["service"])
	assert.Equal(t, "1.0.0", metadata.ConstLabels["version"])
	assert.Equal(t, "myapp_requests", metadata.Name)
}

func TestMetricsCollector_DefaultTags_Gauge(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"region": "us-east-1",
				"env":    "staging",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	gauge := collector.Gauge("memory_usage")

	metadata := gauge.Describe()
	assert.Equal(t, "us-east-1", metadata.ConstLabels["region"])
	assert.Equal(t, "staging", metadata.ConstLabels["env"])
}

func TestMetricsCollector_DefaultTags_Histogram(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"datacenter": "dc1",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	histogram := collector.Histogram("request_duration")

	metadata := histogram.Describe()
	assert.Equal(t, "dc1", metadata.ConstLabels["datacenter"])
}

func TestMetricsCollector_DefaultTags_Summary(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"team": "platform",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	summary := collector.Summary("response_size")

	metadata := summary.Describe()
	assert.Equal(t, "platform", metadata.ConstLabels["team"])
}

func TestMetricsCollector_DefaultTags_Timer(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"app": "backend",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	timer := collector.Timer("processing_time")

	metadata := timer.Describe()
	assert.Equal(t, "backend", metadata.ConstLabels["app"])
}

func TestMetricsCollector_DefaultTags_Precedence(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			Namespace: "default_ns",
			DefaultTags: map[string]string{
				"env":     "production",
				"service": "api",
				"version": "1.0.0",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))

	// Metric-specific options should override defaults
	counter := collector.Counter("requests",
		WithNamespace("custom_ns"),
		WithConstLabels(map[string]string{
			"env":      "staging", // Override default
			"instance": "i-123",   // Add new label
		}),
	)

	metadata := counter.Describe()
	assert.Equal(t, "custom_ns", metadata.Namespace, "Metric-specific namespace should override default")
	assert.Equal(t, "staging", metadata.ConstLabels["env"], "Metric-specific env should override default")
	assert.Equal(t, "i-123", metadata.ConstLabels["instance"], "Metric-specific labels should be present")

	// Default tags that weren't overridden should still be present
	// Note: WithConstLabels replaces all const labels, so only the explicitly set ones remain
	assert.NotContains(t, metadata.ConstLabels, "service", "Non-overridden defaults are replaced when using WithConstLabels")
}

func TestMetricsCollector_DefaultTags_NoConfig(t *testing.T) {
	// Should work fine with nil config
	collector := NewMetricsCollector("test_collector")
	counter := collector.Counter("requests")

	metadata := counter.Describe()
	assert.Empty(t, metadata.ConstLabels)
	assert.Empty(t, metadata.Namespace)
}

func TestMetricsCollector_DefaultTags_EmptyDefaultTags(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))
	counter := collector.Counter("requests")

	metadata := counter.Describe()
	assert.Empty(t, metadata.ConstLabels)
}

func TestMetricsCollector_DefaultTags_AllMetricTypes(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			Namespace: "testapp",
			DefaultTags: map[string]string{
				"env":     "test",
				"service": "metrics",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))

	// Create all metric types
	counter := collector.Counter("counter_metric")
	gauge := collector.Gauge("gauge_metric")
	histogram := collector.Histogram("histogram_metric")
	summary := collector.Summary("summary_metric")
	timer := collector.Timer("timer_metric")

	// Verify all have default tags
	metrics := []struct {
		name     string
		metadata MetricMetadata
	}{
		{"counter", counter.Describe()},
		{"gauge", gauge.Describe()},
		{"histogram", histogram.Describe()},
		{"summary", summary.Describe()},
		{"timer", timer.Describe()},
	}

	for _, m := range metrics {
		assert.Equal(t, "testapp", m.metadata.Namespace, "Namespace should be set for %s", m.name)
		assert.Equal(t, "test", m.metadata.ConstLabels["env"], "env tag should be set for %s", m.name)
		assert.Equal(t, "metrics", m.metadata.ConstLabels["service"], "service tag should be set for %s", m.name)
	}
}

func TestMetricsCollector_DefaultTags_Concurrent(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"env": "test",
			},
		},
	}

	collector := NewMetricsCollector("test_collector", WithConfig(config))

	// Create metrics concurrently
	var wg sync.WaitGroup

	numGoroutines := 10

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			counter := collector.Counter(fmt.Sprintf("concurrent_counter_%d", id))
			metadata := counter.Describe()
			assert.Equal(t, "test", metadata.ConstLabels["env"])
		}(i)
	}

	wg.Wait()
}

func TestMetricsCollector_MergeDefaultOptions(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			Namespace: "myapp",
			DefaultTags: map[string]string{
				"env": "prod",
			},
		},
	}

	mc := NewMetricsCollector("test", WithConfig(config)).(*metricsCollector)

	// Test with no options
	merged := mc.mergeDefaultOptions(nil)
	assert.Len(t, merged, 2, "Should have namespace and default tags options")

	// Test with existing options
	opts := []MetricOption{
		WithDescription("test metric"),
		WithUnit("requests"),
	}

	merged = mc.mergeDefaultOptions(opts)
	assert.Len(t, merged, 4, "Should have namespace, default tags, and 2 custom options")

	// Test with nil config
	mcNilConfig := NewMetricsCollector("test").(*metricsCollector)
	merged = mcNilConfig.mergeDefaultOptions(opts)
	assert.Equal(t, opts, merged, "Should return original opts when config is nil")
}

// =============================================================================
// LABEL CARDINALITY TESTS
// =============================================================================

func TestMetricsCollector_Cardinality_Basic(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 5, // Very low limit for testing
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create metrics with different names and label combinations
	_ = collector.Counter("requests_v1", WithLabels(map[string]string{"endpoint": "/api/v1"}))
	_ = collector.Counter("requests_v2", WithLabels(map[string]string{"endpoint": "/api/v2"}))
	_ = collector.Counter("requests_v3", WithLabels(map[string]string{"endpoint": "/api/v3"}))

	stats := collector.Stats()
	assert.Equal(t, 3, stats.LabelCardinality, "Should track 3 label combinations")
	assert.Equal(t, 5, stats.MaxLabelCardinality, "Should have max of 5")
}

func TestMetricsCollector_Cardinality_LimitExceeded(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 3, // Very low limit for testing
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create metrics up to the limit with different names
	c1 := collector.Counter("requests_1", WithLabels(map[string]string{"endpoint": "/api/v1"}))
	c2 := collector.Counter("requests_2", WithLabels(map[string]string{"endpoint": "/api/v2"}))
	c3 := collector.Counter("requests_3", WithLabels(map[string]string{"endpoint": "/api/v3"}))

	// These should all be different instances (different names)
	assert.NotEqual(t, c1, c2)
	assert.NotEqual(t, c2, c3)

	// Try to create beyond the limit - should return basic counter without labels
	c4 := collector.Counter("requests_4", WithLabels(map[string]string{"endpoint": "/api/v4"}))

	// Should still get a counter back (not nil), but it might be a fallback
	assert.NotNil(t, c4)

	stats := collector.Stats()
	assert.Equal(t, 3, stats.LabelCardinality, "Should not exceed max cardinality")
}

func TestMetricsCollector_Cardinality_AllMetricTypes(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 10,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create different metric types with labels
	_ = collector.Counter("counter", WithLabels(map[string]string{"type": "a"}))
	_ = collector.Gauge("gauge", WithLabels(map[string]string{"type": "b"}))
	_ = collector.Histogram("histogram", WithLabels(map[string]string{"type": "c"}))
	_ = collector.Summary("summary", WithLabels(map[string]string{"type": "d"}))
	_ = collector.Timer("timer", WithLabels(map[string]string{"type": "e"}))

	stats := collector.Stats()
	assert.Equal(t, 5, stats.LabelCardinality, "Should track all metric type combinations")
}

func TestMetricsCollector_Cardinality_SameLabels(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 10,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create same metric with same labels multiple times
	labels := map[string]string{"endpoint": "/api/v1"}
	_ = collector.Counter("requests", WithLabels(labels))
	_ = collector.Counter("requests", WithLabels(labels))
	_ = collector.Counter("requests", WithLabels(labels))

	stats := collector.Stats()
	// Should only count once since it's the same combination
	assert.LessOrEqual(t, stats.LabelCardinality, 1, "Should not double-count same combination")
}

func TestMetricsCollector_Cardinality_WithDefaultTags(t *testing.T) {
	config := &MetricsConfig{
		Collection: MetricsCollection{
			DefaultTags: map[string]string{
				"env":     "test",
				"service": "api",
			},
		},
		Limits: MetricsLimits{
			MaxMetrics: 10,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create metrics - default tags should be included in cardinality
	_ = collector.Counter("requests", WithLabels(map[string]string{"endpoint": "/api/v1"}))
	_ = collector.Counter("requests", WithLabels(map[string]string{"endpoint": "/api/v2"}))

	stats := collector.Stats()
	assert.Greater(t, stats.LabelCardinality, 0, "Should track cardinality with default tags")
}

func TestMetricsCollector_Cardinality_NoConfig(t *testing.T) {
	collector := NewMetricsCollector("test")

	// Should use default max cardinality
	_ = collector.Counter("requests", WithLabels(map[string]string{"endpoint": "/api/v1"}))

	stats := collector.Stats()
	assert.Equal(t, MaxLabelCardinality, stats.MaxLabelCardinality, "Should use default max")
	assert.Equal(t, 1, stats.LabelCardinality, "Should track 1 combination")
}

func TestMetricsCollector_Cardinality_Concurrent(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 100,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	var wg sync.WaitGroup

	numGoroutines := 10

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range 5 {
				_ = collector.Counter("requests",
					WithLabels(map[string]string{
						"worker": fmt.Sprintf("w%d", id),
						"task":   fmt.Sprintf("t%d", j),
					}))
			}
		}(i)
	}

	wg.Wait()

	stats := collector.Stats()
	// Should have up to 50 unique combinations (10 workers * 5 tasks)
	assert.LessOrEqual(t, stats.LabelCardinality, 50, "Cardinality should be tracked correctly under concurrency")
	assert.Greater(t, stats.LabelCardinality, 0, "Should have tracked some combinations")
}

func TestMetricsCollector_Cardinality_RespectsExistingMetrics(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 5,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create some metrics
	labels1 := map[string]string{"endpoint": "/api/v1"}
	c1 := collector.Counter("requests", WithLabels(labels1))
	c1.Inc()

	// Request same metric again - should return existing
	c2 := collector.Counter("requests", WithLabels(labels1))
	assert.Equal(t, 1.0, c2.Value(), "Should return existing metric with same state")
}

func TestMetricsCollector_Cardinality_ConstLabels(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 10,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create metrics with const labels
	_ = collector.Counter("requests", WithConstLabels(map[string]string{"version": "v1"}))
	_ = collector.Counter("requests", WithConstLabels(map[string]string{"version": "v2"}))

	stats := collector.Stats()
	assert.Greater(t, stats.LabelCardinality, 0, "Should track const labels in cardinality")
}

func TestMetricsCollector_Cardinality_MixedLabels(t *testing.T) {
	config := &MetricsConfig{
		Limits: MetricsLimits{
			MaxMetrics: 10,
		},
	}

	collector := NewMetricsCollector("test", WithConfig(config))

	// Create metrics with mix of regular and const labels using different names
	_ = collector.Counter("requests_a",
		WithLabels(map[string]string{"endpoint": "/api/v1"}),
		WithConstLabels(map[string]string{"version": "1.0"}))

	_ = collector.Counter("requests_b",
		WithLabels(map[string]string{"endpoint": "/api/v2"}),
		WithConstLabels(map[string]string{"version": "1.0"}))

	stats := collector.Stats()
	assert.Equal(t, 2, stats.LabelCardinality, "Should track both combinations")
}
