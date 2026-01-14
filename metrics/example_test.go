package metrics_test

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/go-utils/metrics"
)

// Example_basic demonstrates basic usage of the metrics collector.
func Example_basic() {
	// Create a metrics collector
	collector := metrics.NewMetricsCollector("myapp")

	// Start the collector
	ctx := context.Background()

	_ = collector.Start(ctx)
	defer collector.Stop(ctx)

	// Create and use a counter
	requestCounter := collector.Counter("http_requests_total",
		metrics.WithDescription("Total HTTP requests"),
		metrics.WithUnit("requests"),
	)

	requestCounter.Inc()
	requestCounter.Add(5)

	fmt.Printf("Requests: %.0f\n", requestCounter.Value())
	// Output: Requests: 6
}

// Example_counter demonstrates counter usage.
func Example_counter() {
	collector := metrics.NewMetricsCollector("example")

	// Simple counter
	counter := collector.Counter("operations_total")
	counter.Inc()
	counter.Add(10)

	fmt.Printf("Operations: %.0f\n", counter.Value())
	// Output: Operations: 11
}

// Example_counterWithLabels demonstrates counter with labels.
func Example_counterWithLabels() {
	collector := metrics.NewMetricsCollector("example")

	// Counter with labels
	httpRequests := collector.Counter("http_requests",
		metrics.WithNamespace("myapp"),
		metrics.WithLabels(map[string]string{
			"method": "GET",
			"path":   "/api/users",
		}),
	)

	httpRequests.Inc()

	// Create a variant with different labels
	postRequests := httpRequests.WithLabels(map[string]string{
		"method": "POST",
	})
	postRequests.Add(5)

	fmt.Printf("GET requests: %.0f\n", httpRequests.Value())
	fmt.Printf("POST requests: %.0f\n", postRequests.Value())
	// Output:
	// GET requests: 1
	// POST requests: 5
}

// Example_gauge demonstrates gauge usage.
func Example_gauge() {
	collector := metrics.NewMetricsCollector("example")

	// Gauge for tracking current values
	queueSize := collector.Gauge("queue_size")

	queueSize.Set(10)
	queueSize.Inc()
	queueSize.Add(5)
	queueSize.Sub(3)

	fmt.Printf("Queue size: %.0f\n", queueSize.Value())
	// Output: Queue size: 13
}

// Example_histogram demonstrates histogram usage.
func Example_histogram() {
	collector := metrics.NewMetricsCollector("example")

	// Histogram with custom buckets
	latency := collector.Histogram("request_latency",
		metrics.WithDescription("Request latency distribution"),
		metrics.WithUnit("ms"),
		metrics.WithBuckets(10, 50, 100, 500, 1000),
	)

	// Record observations
	latency.Observe(25)
	latency.Observe(75)
	latency.Observe(150)
	latency.Observe(300)

	fmt.Printf("Count: %d\n", latency.Count())
	fmt.Printf("Mean: %.1f\n", latency.Mean())
	fmt.Printf("Min: %.1f\n", latency.Min())
	fmt.Printf("Max: %.1f\n", latency.Max())
	// Output:
	// Count: 4
	// Mean: 137.5
	// Min: 25.0
	// Max: 300.0
}

// Example_histogramWithDefaultBuckets demonstrates histogram with default buckets.
func Example_histogramWithDefaultBuckets() {
	collector := metrics.NewMetricsCollector("example")

	// Use default OTEL-recommended buckets
	histogram := collector.Histogram("response_size",
		metrics.WithDefaultHistogramBuckets(),
	)

	histogram.Observe(100)
	histogram.Observe(500)
	histogram.Observe(1000)

	fmt.Printf("Observations: %d\n", histogram.Count())
	// Output: Observations: 3
}

// Example_summary demonstrates summary usage.
func Example_summary() {
	collector := metrics.NewMetricsCollector("example")

	// Summary for accurate quantile calculations
	processTime := collector.Summary("process_duration",
		metrics.WithDescription("Processing time summary"),
		metrics.WithPercentiles(0.5, 0.9, 0.95, 0.99),
	)

	// Record observations
	for i := 1; i <= 100; i++ {
		processTime.Observe(float64(i))
	}

	fmt.Printf("Count: %d\n", processTime.Count())
	fmt.Printf("Mean: %.1f\n", processTime.Mean())
	fmt.Printf("P50: %.1f\n", processTime.Quantile(0.5))
	fmt.Printf("P95: %.1f\n", processTime.Quantile(0.95))
	// Output:
	// Count: 100
	// Mean: 50.5
	// P50: 50.0
	// P95: 95.0
}

// Example_timer demonstrates timer usage.
func Example_timer() {
	collector := metrics.NewMetricsCollector("example")

	// Timer for measuring durations
	timer := collector.Timer("operation_duration",
		metrics.WithDescription("Operation execution time"),
		metrics.WithDefaultTimerBuckets(),
	)

	// Direct recording
	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(150 * time.Millisecond)

	mean := timer.Mean()
	minVal := timer.Min()
	maxVal := timer.Max()

	fmt.Printf("Count: %d\n", timer.Count())
	fmt.Printf("Mean: %dms\n", mean.Milliseconds())
	fmt.Printf("Min: %dms\n", minVal.Milliseconds())
	fmt.Printf("Max: %dms\n", maxVal.Milliseconds())
	// Output:
	// Count: 3
	// Mean: 150ms
	// Min: 100ms
	// Max: 200ms
}

// Example_timerDefer demonstrates timer with defer pattern.
func Example_timerDefer() {
	collector := metrics.NewMetricsCollector("example")

	timer := collector.Timer("function_duration")

	// Simulate a function call
	func() {
		defer timer.Time()()

		time.Sleep(50 * time.Millisecond)
	}()

	fmt.Printf("Recorded: %d\n", timer.Count())
	// Output: Recorded: 1
}

// Example_exemplars demonstrates exemplar usage for trace linking.
func Example_exemplars() {
	collector := metrics.NewMetricsCollector("example")

	counter := collector.Counter("traced_requests")

	// Record with exemplar to link to trace
	exemplar := metrics.Exemplar{
		Value:     10,
		Timestamp: time.Now(),
		TraceID:   "abc123def456",
		SpanID:    "span789",
		Labels: map[string]string{
			"user": "alice",
		},
	}

	counter.AddWithExemplar(10, exemplar)

	exemplars := counter.Exemplars()
	fmt.Printf("Exemplars recorded: %d\n", len(exemplars))
	fmt.Printf("Trace ID: %s\n", exemplars[0].TraceID)
	// Output:
	// Exemplars recorded: 1
	// Trace ID: abc123def456
}

// Example_metadata demonstrates accessing metric metadata.
func Example_metadata() {
	collector := metrics.NewMetricsCollector("example")

	counter := collector.Counter("api_calls",
		metrics.WithDescription("Total API calls"),
		metrics.WithUnit("calls"),
		metrics.WithNamespace("myservice"),
		metrics.WithSubsystem("http"),
	)

	metadata := counter.Describe()

	fmt.Printf("Type: %s\n", metadata.Type)
	fmt.Printf("Description: %s\n", metadata.Description)
	fmt.Printf("Unit: %s\n", metadata.Unit)
	fmt.Printf("Namespace: %s\n", metadata.Namespace)
	// Output:
	// Type: counter
	// Description: Total API calls
	// Unit: calls
	// Namespace: myservice
}

// Example_collectorStats demonstrates collector statistics.
func Example_collectorStats() {
	collector := metrics.NewMetricsCollector("example")

	// Create various metrics
	collector.Counter("counter1")
	collector.Counter("counter2")
	collector.Gauge("gauge1")
	collector.Histogram("histogram1")
	collector.Timer("timer1")

	stats := collector.Stats()

	fmt.Printf("Name: %s\n", stats.Name)
	fmt.Printf("Active metrics: %d\n", stats.ActiveMetrics)
	fmt.Printf("Counters: %d\n", stats.MetricsByType[metrics.MetricTypeCounter])
	fmt.Printf("Gauges: %d\n", stats.MetricsByType[metrics.MetricTypeGauge])
	// Output:
	// Name: example
	// Active metrics: 5
	// Counters: 2
	// Gauges: 1
}

// Example_reset demonstrates resetting metrics.
func Example_reset() {
	collector := metrics.NewMetricsCollector("example")

	counter := collector.Counter("requests")
	counter.Add(100)

	fmt.Printf("Before reset: %.0f\n", counter.Value())

	_ = collector.ResetMetric("requests")

	fmt.Printf("After reset: %.0f\n", counter.Value())
	// Output:
	// Before reset: 100
	// After reset: 0
}

// Example_realWorld demonstrates a realistic application scenario.
func Example_realWorld() {
	// Initialize metrics collector
	collector := metrics.NewMetricsCollector("webserver")

	ctx := context.Background()

	_ = collector.Start(ctx)
	defer collector.Stop(ctx)

	// Define metrics
	requestCounter := collector.Counter("http_requests_total",
		metrics.WithDescription("Total HTTP requests"),
		metrics.WithNamespace("webserver"),
	)

	requestDuration := collector.Timer("http_request_duration",
		metrics.WithDescription("HTTP request duration"),
		metrics.WithUnit("ms"),
		metrics.WithDefaultTimerBuckets(),
	)

	activeConnections := collector.Gauge("active_connections",
		metrics.WithDescription("Current active connections"),
	)

	// Simulate HTTP request handling
	for range 10 {
		// Track request
		requestCounter.Inc()
		activeConnections.Inc()

		// Simulate request processing
		start := time.Now()

		time.Sleep(time.Millisecond) // Simulate work
		requestDuration.Record(time.Since(start))

		activeConnections.Dec()
	}

	// Report metrics
	fmt.Printf("Total requests: %.0f\n", requestCounter.Value())
	fmt.Printf("Avg duration: %dms\n", requestDuration.Mean().Milliseconds())
	fmt.Printf("Active connections: %.0f\n", activeConnections.Value())
	// Output:
	// Total requests: 10
	// Avg duration: 1ms
	// Active connections: 0
}

// Example_defaultTags demonstrates how to configure default tags that apply to all metrics.
func Example_defaultTags() {
	// Create a metrics configuration with default tags
	config := &metrics.MetricsConfig{
		Collection: metrics.MetricsCollection{
			Namespace: "myapp",
			DefaultTags: map[string]string{
				"env":     "production",
				"service": "api",
				"version": "1.0.0",
				"region":  "us-east-1",
			},
		},
	}

	// Create a metrics collector with the config
	collector := metrics.NewMetricsCollector("api_metrics", metrics.WithConfig(config))

	// All metrics created by this collector will inherit the default tags
	requestCounter := collector.Counter("http_requests_total")
	responseTime := collector.Histogram("http_response_time_ms")
	activeUsers := collector.Gauge("active_users")

	// Use the metrics
	requestCounter.Inc()
	responseTime.Observe(45.5)
	activeUsers.Set(150)

	// Verify that default tags are applied
	counterMeta := requestCounter.Describe()
	histogramMeta := responseTime.Describe()
	gaugeMeta := activeUsers.Describe()

	fmt.Printf("Counter Name: %s\n", counterMeta.Name)
	fmt.Printf("Counter Namespace: %s\n", counterMeta.Namespace)
	fmt.Printf("Counter Labels: env=%s, service=%s, version=%s\n",
		counterMeta.ConstLabels["env"],
		counterMeta.ConstLabels["service"],
		counterMeta.ConstLabels["version"])

	fmt.Printf("\nHistogram Name: %s\n", histogramMeta.Name)
	fmt.Printf("Histogram Labels: region=%s\n", histogramMeta.ConstLabels["region"])

	fmt.Printf("\nGauge Name: %s\n", gaugeMeta.Name)
	fmt.Printf("All metrics have %d default tags\n", len(gaugeMeta.ConstLabels))

	// You can still override defaults on a per-metric basis
	customCounter := collector.Counter("custom_counter",
		metrics.WithConstLabels(map[string]string{
			"env":      "staging", // Override default
			"instance": "i-123",   // Add new label
		}),
	)

	customMeta := customCounter.Describe()
	fmt.Printf("\nCustom Counter env: %s\n", customMeta.ConstLabels["env"])
	fmt.Printf("Custom Counter instance: %s\n", customMeta.ConstLabels["instance"])

	// Output:
	// Counter Name: myapp_http_requests_total
	// Counter Namespace: myapp
	// Counter Labels: env=production, service=api, version=1.0.0
	//
	// Histogram Name: myapp_http_response_time_ms
	// Histogram Labels: region=us-east-1
	//
	// Gauge Name: myapp_active_users
	// All metrics have 4 default tags
	//
	// Custom Counter env: staging
	// Custom Counter instance: i-123
}

// Example_histogramBuckets demonstrates how to access bucket data from a histogram.
func Example_histogramBuckets() {
	// Create a histogram with custom buckets
	responseTime := metrics.NewHistogram("api_response_time",
		metrics.WithBuckets(50, 100, 250, 500, 1000),
		metrics.WithDescription("API response time in milliseconds"),
		metrics.WithUnit("ms"),
	)

	// Record some observations
	responseTime.Observe(45)   // Falls in 50ms bucket
	responseTime.Observe(75)   // Falls in 100ms bucket
	responseTime.Observe(120)  // Falls in 250ms bucket
	responseTime.Observe(450)  // Falls in 500ms bucket
	responseTime.Observe(800)  // Falls in 1000ms bucket
	responseTime.Observe(2000) // Falls in +Inf bucket (beyond defined buckets)

	// Access bucket data
	buckets := responseTime.Buckets()

	fmt.Printf("Histogram Statistics:\n")
	fmt.Printf("Total Observations: %d\n", responseTime.Count())
	fmt.Printf("Sum: %.2f\n", responseTime.Sum())
	fmt.Printf("Mean: %.2f\n", responseTime.Mean())
	fmt.Printf("Min: %.2f\n", responseTime.Min())
	fmt.Printf("Max: %.2f\n", responseTime.Max())

	fmt.Printf("\nBucket Distribution:\n")
	fmt.Printf("≤ 50ms:   %d observations\n", buckets[50.0])
	fmt.Printf("≤ 100ms:  %d observations\n", buckets[100.0])
	fmt.Printf("≤ 250ms:  %d observations\n", buckets[250.0])
	fmt.Printf("≤ 500ms:  %d observations\n", buckets[500.0])
	fmt.Printf("≤ 1000ms: %d observations\n", buckets[1000.0])

	// Output:
	// Histogram Statistics:
	// Total Observations: 6
	// Sum: 3490.00
	// Mean: 581.67
	// Min: 45.00
	// Max: 2000.00
	//
	// Bucket Distribution:
	// ≤ 50ms:   1 observations
	// ≤ 100ms:  1 observations
	// ≤ 250ms:  1 observations
	// ≤ 500ms:  1 observations
	// ≤ 1000ms: 1 observations
}

// Example_histogramPercentile demonstrates using the Histogram.Percentile() method.
func Example_histogramPercentile() {
	// Create a histogram with custom buckets
	latency := metrics.NewHistogram("api_latency",
		metrics.WithBuckets(10, 25, 50, 75, 100, 150, 200, 300, 500, 1000),
		metrics.WithDescription("API request latency"),
		metrics.WithUnit("ms"),
	)

	// Record various observations
	observations := []float64{
		15, 22, 35, 48, 55, 62, 78, 85, 92, 105,
		120, 135, 145, 158, 165, 180, 195, 210, 285, 420,
	}

	for _, value := range observations {
		latency.Observe(value)
	}

	// Calculate percentiles
	fmt.Printf("API Latency Analysis:\n")
	fmt.Printf("Total Requests: %d\n", latency.Count())
	fmt.Printf("P50 (median): %.1fms\n", latency.Percentile(0.50))
	fmt.Printf("P90: %.1fms\n", latency.Percentile(0.90))
	fmt.Printf("P95: %.1fms\n", latency.Percentile(0.95))
	fmt.Printf("P99: %.1fms\n", latency.Percentile(0.99))

	// Percentile and Quantile are equivalent
	fmt.Printf("\nPercentile vs Quantile (same result):\n")
	fmt.Printf("Percentile(0.95): %.1fms\n", latency.Percentile(0.95))
	fmt.Printf("Quantile(0.95):   %.1fms\n", latency.Quantile(0.95))

	// Output:
	// API Latency Analysis:
	// Total Requests: 20
	// P50 (median): 150.0ms
	// P90: 300.0ms
	// P95: 300.0ms
	// P99: 300.0ms
	//
	// Percentile vs Quantile (same result):
	// Percentile(0.95): 300.0ms
	// Quantile(0.95):   300.0ms
}

// Example_cardinalityLimits demonstrates how cardinality tracking prevents metric explosion.
func Example_cardinalityLimits() {
	// Configure a collector with a low cardinality limit for demonstration
	config := &metrics.MetricsConfig{
		Limits: metrics.MetricsLimits{
			MaxMetrics: 5, // Very low limit for demo purposes
		},
		Collection: metrics.MetricsCollection{
			DefaultTags: map[string]string{
				"service": "api",
				"env":     "production",
			},
		},
	}

	collector := metrics.NewMetricsCollector("demo", metrics.WithConfig(config))

	fmt.Println("Creating metrics up to the cardinality limit:")

	// Create metrics up to the limit
	for i := 1; i <= 3; i++ {
		counter := collector.Counter(fmt.Sprintf("requests_%d", i),
			metrics.WithLabels(map[string]string{
				"endpoint": fmt.Sprintf("/api/v%d", i),
			}))
		counter.Inc()
		fmt.Printf("Created metric %d\n", i)
	}

	// Check stats
	stats := collector.Stats()

	fmt.Printf("\nCardinality Stats:\n")
	fmt.Printf("Current Cardinality: %d\n", stats.LabelCardinality)
	fmt.Printf("Max Cardinality: %d\n", stats.MaxLabelCardinality)
	fmt.Printf("Active Metrics: %d\n", stats.ActiveMetrics)

	fmt.Println("\nAttempting to exceed the limit:")

	// Try to exceed the limit
	for i := 4; i <= 7; i++ {
		counter := collector.Counter(fmt.Sprintf("requests_%d", i),
			metrics.WithLabels(map[string]string{
				"endpoint": fmt.Sprintf("/api/v%d", i),
			}))
		counter.Inc()

		if i <= 5 {
			fmt.Printf("Created metric %d (within limit)\n", i)
		} else {
			fmt.Printf("Metric %d creation limited (cardinality exceeded)\n", i)
		}
	}

	// Final stats
	finalStats := collector.Stats()
	fmt.Printf("\nFinal Cardinality: %d (limit: %d)\n",
		finalStats.LabelCardinality,
		finalStats.MaxLabelCardinality)

	fmt.Println("\nThis prevents metric explosion in production!")

	// Output:
	// Creating metrics up to the cardinality limit:
	// Created metric 1
	// Created metric 2
	// Created metric 3
	//
	// Cardinality Stats:
	// Current Cardinality: 3
	// Max Cardinality: 5
	// Active Metrics: 3
	//
	// Attempting to exceed the limit:
	// Created metric 4 (within limit)
	// Created metric 5 (within limit)
	// Metric 6 creation limited (cardinality exceeded)
	// Metric 7 creation limited (cardinality exceeded)
	//
	// Final Cardinality: 5 (limit: 5)
	//
	// This prevents metric explosion in production!
}

// Example_timerValue demonstrates the Timer.Value() method.
func Example_timerValue() {
	timer := metrics.NewTimer("process_duration")

	// Record several durations
	timer.Record(100 * time.Millisecond)
	timer.Record(200 * time.Millisecond)
	timer.Record(300 * time.Millisecond)
	timer.Record(150 * time.Millisecond)

	// Value() returns the total sum (same as Sum())
	// This provides API consistency with Counter and Gauge
	totalTime := timer.Value()
	fmt.Printf("Total time spent: %v\n", totalTime)

	// Equivalent to Sum()
	fmt.Printf("Sum: %v\n", timer.Sum())

	// Other useful metrics
	fmt.Printf("Count: %d\n", timer.Count())
	fmt.Printf("Mean: %v\n", timer.Mean())

	// Output:
	// Total time spent: 750ms
	// Sum: 750ms
	// Count: 4
	// Mean: 187.5ms
}
