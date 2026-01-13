package metrics

import (
	"sync"
	"testing"
	"time"
)

// =============================================================================
// COUNTER BENCHMARKS
// =============================================================================

func BenchmarkCounter_Inc(b *testing.B) {
	counter := NewCounter("bench_counter")

	for b.Loop() {
		counter.Inc()
	}
}

func BenchmarkCounter_Add(b *testing.B) {
	counter := NewCounter("bench_counter")

	for i := 0; b.Loop(); i++ {
		counter.Add(float64(i))
	}
}

func BenchmarkCounter_AddWithExemplar(b *testing.B) {
	counter := NewCounter("bench_counter")
	exemplar := Exemplar{
		TraceID: "trace123",
		SpanID:  "span456",
	}

	for b.Loop() {
		counter.AddWithExemplar(1, exemplar)
	}
}

func BenchmarkCounter_Value(b *testing.B) {
	counter := NewCounter("bench_counter")
	counter.Add(1000)

	for b.Loop() {
		_ = counter.Value()
	}
}

func BenchmarkCounter_Concurrent(b *testing.B) {
	counter := NewCounter("concurrent_counter")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			counter.Inc()
		}
	})
}

// =============================================================================
// GAUGE BENCHMARKS
// =============================================================================

func BenchmarkGauge_Set(b *testing.B) {
	gauge := NewGauge("bench_gauge")

	for i := 0; b.Loop(); i++ {
		gauge.Set(float64(i))
	}
}

func BenchmarkGauge_Inc(b *testing.B) {
	gauge := NewGauge("bench_gauge")

	for b.Loop() {
		gauge.Inc()
	}
}

func BenchmarkGauge_Add(b *testing.B) {
	gauge := NewGauge("bench_gauge")

	for i := 0; b.Loop(); i++ {
		gauge.Add(float64(i))
	}
}

func BenchmarkGauge_Value(b *testing.B) {
	gauge := NewGauge("bench_gauge")
	gauge.Set(100)

	for b.Loop() {
		_ = gauge.Value()
	}
}

func BenchmarkGauge_Concurrent(b *testing.B) {
	gauge := NewGauge("concurrent_gauge")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			gauge.Inc()
		}
	})
}

// =============================================================================
// HISTOGRAM BENCHMARKS
// =============================================================================

func BenchmarkHistogram_Observe(b *testing.B) {
	histogram := NewHistogram("bench_histogram",
		WithBuckets(1, 5, 10, 50, 100, 500, 1000),
	)

	for i := 0; b.Loop(); i++ {
		histogram.Observe(float64(i % 1000))
	}
}

func BenchmarkHistogram_ObserveWithExemplar(b *testing.B) {
	histogram := NewHistogram("bench_histogram")
	exemplar := Exemplar{
		TraceID: "trace789",
		SpanID:  "span012",
	}

	for i := 0; b.Loop(); i++ {
		histogram.ObserveWithExemplar(float64(i%1000), exemplar)
	}
}

func BenchmarkHistogram_Quantile(b *testing.B) {
	histogram := NewHistogram("bench_histogram")

	for i := range 1000 {
		histogram.Observe(float64(i))
	}

	for b.Loop() {
		_ = histogram.Quantile(0.95)
	}
}

func BenchmarkHistogram_Mean(b *testing.B) {
	histogram := NewHistogram("bench_histogram")

	for i := range 1000 {
		histogram.Observe(float64(i))
	}

	for b.Loop() {
		_ = histogram.Mean()
	}
}

func BenchmarkHistogram_Concurrent(b *testing.B) {
	histogram := NewHistogram("concurrent_histogram")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			histogram.Observe(100.0)
		}
	})
}

// =============================================================================
// SUMMARY BENCHMARKS
// =============================================================================

func BenchmarkSummary_Observe(b *testing.B) {
	summary := NewSummary("bench_summary")

	for i := 0; b.Loop(); i++ {
		summary.Observe(float64(i % 1000))
	}
}

func BenchmarkSummary_Quantile(b *testing.B) {
	summary := NewSummary("bench_summary")

	for i := range 1000 {
		summary.Observe(float64(i))
	}

	for b.Loop() {
		_ = summary.Quantile(0.95)
	}
}

func BenchmarkSummary_Mean(b *testing.B) {
	summary := NewSummary("bench_summary")

	for i := range 1000 {
		summary.Observe(float64(i))
	}

	for b.Loop() {
		_ = summary.Mean()
	}
}

func BenchmarkSummary_StdDev(b *testing.B) {
	summary := NewSummary("bench_summary")

	for i := range 1000 {
		summary.Observe(float64(i))
	}

	for b.Loop() {
		_ = summary.StdDev()
	}
}

func BenchmarkSummary_Concurrent(b *testing.B) {
	summary := NewSummary("concurrent_summary")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			summary.Observe(100.0)
		}
	})
}

// =============================================================================
// TIMER BENCHMARKS
// =============================================================================

func BenchmarkTimer_Record(b *testing.B) {
	timer := NewTimer("bench_timer")

	for b.Loop() {
		timer.Record(100 * time.Millisecond)
	}
}

func BenchmarkTimer_RecordWithExemplar(b *testing.B) {
	timer := NewTimer("bench_timer")
	exemplar := Exemplar{
		TraceID: "timer_trace",
		SpanID:  "timer_span",
	}

	for b.Loop() {
		timer.RecordWithExemplar(100*time.Millisecond, exemplar)
	}
}

func BenchmarkTimer_Time(b *testing.B) {
	timer := NewTimer("bench_timer")

	for b.Loop() {
		done := timer.Time()
		done()
	}
}

func BenchmarkTimer_Percentile(b *testing.B) {
	timer := NewTimer("bench_timer")

	for i := range 1000 {
		timer.Record(time.Duration(i) * time.Millisecond)
	}

	for b.Loop() {
		_ = timer.Percentile(0.95)
	}
}

func BenchmarkTimer_Concurrent(b *testing.B) {
	timer := NewTimer("concurrent_timer")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			timer.Record(100 * time.Millisecond)
		}
	})
}

// =============================================================================
// METRICS COLLECTOR BENCHMARKS
// =============================================================================

func BenchmarkMetricsCollector_Counter(b *testing.B) {
	collector := NewMetricsCollector("bench_collector")

	for b.Loop() {
		counter := collector.Counter("requests_total")
		counter.Inc()
	}
}

func BenchmarkMetricsCollector_CreateMultipleMetrics(b *testing.B) {
	collector := NewMetricsCollector("bench_collector")

	for b.Loop() {
		collector.Counter("counter")
		collector.Gauge("gauge")
		collector.Histogram("histogram")
		collector.Timer("timer")
	}
}

func BenchmarkMetricsCollector_ListMetrics(b *testing.B) {
	collector := NewMetricsCollector("bench_collector")

	for i := range 100 {
		collector.Counter("counter_" + string(rune(i)))
		collector.Gauge("gauge_" + string(rune(i)))
	}

	for b.Loop() {
		_ = collector.ListMetrics()
	}
}

func BenchmarkMetricsCollector_Stats(b *testing.B) {
	collector := NewMetricsCollector("bench_collector")

	for i := range 50 {
		collector.Counter("counter_" + string(rune(i)))
		collector.Gauge("gauge_" + string(rune(i)))
	}

	for b.Loop() {
		_ = collector.Stats()
	}
}

// =============================================================================
// MEMORY ALLOCATION BENCHMARKS
// =============================================================================

func BenchmarkCounter_Allocations(b *testing.B) {
	counter := NewCounter("alloc_counter")

	b.ReportAllocs()

	for b.Loop() {
		counter.Inc()
	}
}

func BenchmarkHistogram_Allocations(b *testing.B) {
	histogram := NewHistogram("alloc_histogram")

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		histogram.Observe(float64(i))
	}
}

func BenchmarkTimer_Allocations(b *testing.B) {
	timer := NewTimer("alloc_timer")

	b.ReportAllocs()

	for b.Loop() {
		timer.Record(100 * time.Millisecond)
	}
}

// =============================================================================
// REAL-WORLD SCENARIO BENCHMARKS
// =============================================================================

func BenchmarkScenario_HTTPRequestTracking(b *testing.B) {
	collector := NewMetricsCollector("http_tracker")

	reqCounter := collector.Counter("http_requests_total")
	latencyTimer := collector.Timer("http_request_duration")
	activeConns := collector.Gauge("active_connections")

	for i := 0; b.Loop(); i++ {
		reqCounter.Inc()
		activeConns.Inc()
		latencyTimer.Record(time.Duration(i%100) * time.Millisecond)
		activeConns.Dec()
	}
}

func BenchmarkScenario_ConcurrentRequests(b *testing.B) {
	collector := NewMetricsCollector("concurrent_tracker")

	reqCounter := collector.Counter("requests_total")
	latencyHistogram := collector.Histogram("request_duration")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reqCounter.Inc()
			latencyHistogram.Observe(float64(100))
		}
	})
}

func BenchmarkScenario_MixedOperations(b *testing.B) {
	collector := NewMetricsCollector("mixed_ops")

	counter := collector.Counter("operations_total")
	gauge := collector.Gauge("queue_size")
	histogram := collector.Histogram("operation_duration")
	timer := collector.Timer("processing_time")

	for i := 0; b.Loop(); i++ {
		switch i % 4 {
		case 0:
			counter.Inc()
		case 1:
			gauge.Set(float64(i % 100))
		case 2:
			histogram.Observe(float64(i % 1000))
		case 3:
			timer.Record(time.Duration(i%100) * time.Millisecond)
		}
	}
}

// =============================================================================
// CONTENTION BENCHMARKS
// =============================================================================

func BenchmarkContention_Counter_LowContention(b *testing.B) {
	counter := NewCounter("low_contention_counter")
	numWorkers := 2

	b.ResetTimer()
	benchmarkConcurrentCounter(b, counter, numWorkers)
}

func BenchmarkContention_Counter_HighContention(b *testing.B) {
	counter := NewCounter("high_contention_counter")
	numWorkers := 16

	b.ResetTimer()
	benchmarkConcurrentCounter(b, counter, numWorkers)
}

func benchmarkConcurrentCounter(b *testing.B, counter *counterImpl, numWorkers int) {
	var wg sync.WaitGroup

	opsPerWorker := b.N / numWorkers

	for range numWorkers {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range opsPerWorker {
				counter.Inc()
			}
		}()
	}

	wg.Wait()
}
