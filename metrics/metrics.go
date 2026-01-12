package metrics

import (
	"time"

	"github.com/xraph/go-utils/di"
)

// ExportFormat represents the format for metrics export.
type ExportFormat string

const (
	ExportFormatPrometheus ExportFormat = "prometheus"
	ExportFormatJSON       ExportFormat = "json"
	ExportFormatInflux     ExportFormat = "influx"
	ExportFormatStatsD     ExportFormat = "statsd"
)

// MetricType represents the type of metric.
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeTimer     MetricType = "timer"
)

// =============================================================================
// DEFAULT BUCKET CONFIGURATIONS (OTEL-aligned)
// =============================================================================

var (
	// DefaultHistogramBuckets are OpenTelemetry-recommended bucket boundaries
	// for general-purpose histogram metrics. Values are in base units.
	// Suitable for latency, size, and other distribution metrics.
	DefaultHistogramBuckets = []float64{
		0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000,
		2500, 5000, 7500, 10000,
	}

	// DefaultDurationBuckets are optimized for timer metrics measuring durations.
	// Values are in milliseconds, covering microsecond to multi-second ranges.
	// Suitable for API latency, database query time, and processing duration.
	DefaultDurationBuckets = []float64{
		1, 2, 5, 10, 25, 50, 100, 250, 500,
		1000, 2500, 5000, 10000,
	}

	// DefaultPercentiles are commonly tracked percentile values.
	// P50 (median), P90, P95, P99, and P99.9 cover most monitoring needs.
	DefaultPercentiles = []float64{0.5, 0.9, 0.95, 0.99, 0.999}
)

// MetricsStorageConfig contains storage configuration.
type MetricsStorageConfig[T any] struct {
	Type   string `json:"type"   yaml:"type"`
	Config T      `json:"config" yaml:"config"`
}

// MetricsExporterConfig contains configuration for exporters.
type MetricsExporterConfig[T any] struct {
	Enabled  bool          `json:"enabled"  yaml:"enabled"`
	Interval time.Duration `json:"interval" yaml:"interval"`
	Config   T             `json:"config"   yaml:"config"`
}

// MetricsFeatures configures which metric collection features are enabled.
type MetricsFeatures struct {
	SystemMetrics  bool `json:"system_metrics"  yaml:"system_metrics"`
	RuntimeMetrics bool `json:"runtime_metrics" yaml:"runtime_metrics"`
	HTTPMetrics    bool `json:"http_metrics"    yaml:"http_metrics"`
}

// MetricsCollection configures how metrics are collected.
type MetricsCollection struct {
	Interval    time.Duration     `json:"interval"     yaml:"interval"`
	Namespace   string            `json:"namespace"    yaml:"namespace"`
	Path        string            `json:"path"         yaml:"path"`
	DefaultTags map[string]string `json:"default_tags" yaml:"default_tags"`
}

// MetricsLimits configures resource limits for metrics collection.
type MetricsLimits struct {
	MaxMetrics int `json:"max_metrics" yaml:"max_metrics"`
	BufferSize int `json:"buffer_size" yaml:"buffer_size"`
}

// MetricsConfig configures metrics collection.
type MetricsConfig struct {
	Enabled    bool                                             `json:"enabled"    yaml:"enabled"`
	Features   MetricsFeatures                                  `json:"features"   yaml:"features"`
	Collection MetricsCollection                                `json:"collection" yaml:"collection"`
	Limits     MetricsLimits                                    `json:"limits"     yaml:"limits"`
	Storage    *MetricsStorageConfig[map[string]any]            `json:"storage"    yaml:"storage"`
	Exporters  map[string]MetricsExporterConfig[map[string]any] `json:"exporters"  yaml:"exporters"`
}

// MetricOption is a functional option for configuring metrics.
type MetricOption func(*MetricOptions)

// MetricOptions contains options for metric creation.
type MetricOptions struct {
	// Dynamic labels attached to metrics
	Labels map[string]string

	// Metadata (OTEL-aligned)
	Description string // Human-readable description of the metric
	Unit        string // Unit of measurement (e.g., "ms", "bytes", "requests")

	// Hierarchical naming
	Namespace string // Metric namespace prefix
	Subsystem string // Metric subsystem

	// Constant labels (set once, immutable)
	ConstLabels map[string]string

	// Histogram-specific configuration
	Buckets     []float64     // Explicit bucket boundaries for histogram
	Percentiles []float64     // Percentiles to track (0.0-1.0)
	MaxAge      time.Duration // Sliding window duration for time-based metrics
	AgeBuckets  uint32        // Number of time-based rotation buckets
	BufCap      uint32        // Buffer capacity for observations
}

// WithLabel adds a single label to the metric.
func WithLabel(key, value string) MetricOption {
	return func(opts *MetricOptions) {
		if opts.Labels == nil {
			opts.Labels = make(map[string]string)
		}

		opts.Labels[key] = value
	}
}

// WithLabels adds multiple labels to the metric.
func WithLabels(labels map[string]string) MetricOption {
	return func(opts *MetricOptions) {
		opts.Labels = labels
	}
}

// =============================================================================
// METADATA OPTIONS (OTEL-aligned)
// =============================================================================

// WithDescription sets a human-readable description for the metric.
// This description should explain what the metric measures.
func WithDescription(desc string) MetricOption {
	return func(opts *MetricOptions) {
		opts.Description = desc
	}
}

// WithUnit sets the unit of measurement for the metric.
// Common units: "ms", "s", "bytes", "requests", "percent", "1" (dimensionless).
// Follow OpenTelemetry semantic conventions for consistency.
func WithUnit(unit string) MetricOption {
	return func(opts *MetricOptions) {
		opts.Unit = unit
	}
}

// =============================================================================
// NAMING OPTIONS
// =============================================================================

// WithNamespace sets the metric namespace prefix.
// The namespace typically represents the application or service name.
func WithNamespace(ns string) MetricOption {
	return func(opts *MetricOptions) {
		opts.Namespace = ns
	}
}

// WithSubsystem sets the metric subsystem.
// The subsystem represents a component within the application.
func WithSubsystem(subsystem string) MetricOption {
	return func(opts *MetricOptions) {
		opts.Subsystem = subsystem
	}
}

// WithConstLabels sets constant, immutable labels for the metric.
// These labels cannot be changed after metric creation and are useful
// for static metadata like version, environment, or region.
func WithConstLabels(labels map[string]string) MetricOption {
	return func(opts *MetricOptions) {
		opts.ConstLabels = labels
	}
}

// =============================================================================
// HISTOGRAM OPTIONS
// =============================================================================

// WithBuckets sets explicit bucket boundaries for histogram metrics.
// Buckets define the ranges for distributing observations.
// Values should be in ascending order.
func WithBuckets(buckets ...float64) MetricOption {
	return func(opts *MetricOptions) {
		opts.Buckets = buckets
	}
}

// WithLinearBuckets generates linearly spaced bucket boundaries.
// Start is the first bucket boundary, width is the interval between buckets,
// and count is the number of buckets to generate.
// Example: WithLinearBuckets(0, 10, 5) generates [0, 10, 20, 30, 40].
func WithLinearBuckets(start, width float64, count int) MetricOption {
	return func(opts *MetricOptions) {
		if count <= 0 {
			return
		}

		buckets := make([]float64, count)
		for i := range count {
			buckets[i] = start + float64(i)*width
		}

		opts.Buckets = buckets
	}
}

// WithExponentialBuckets generates exponentially spaced bucket boundaries.
// Start is the first bucket boundary, factor is the multiplication factor,
// and count is the number of buckets to generate.
// Example: WithExponentialBuckets(1, 2, 5) generates [1, 2, 4, 8, 16].
func WithExponentialBuckets(start, factor float64, count int) MetricOption {
	return func(opts *MetricOptions) {
		if count <= 0 || start <= 0 || factor <= 1 {
			return
		}

		buckets := make([]float64, count)

		current := start
		for i := range count {
			buckets[i] = current
			current *= factor
		}

		opts.Buckets = buckets
	}
}

// WithPercentiles sets the specific percentiles to track for histogram metrics.
// Percentiles should be between 0.0 and 1.0.
// Example: WithPercentiles(0.5, 0.95, 0.99) tracks 50th, 95th, and 99th percentiles.
func WithPercentiles(percentiles ...float64) MetricOption {
	return func(opts *MetricOptions) {
		opts.Percentiles = percentiles
	}
}

// WithMaxAge sets the sliding window duration for time-based histogram metrics.
// Observations older than maxAge will be excluded from statistics.
// This enables real-time percentile calculations over recent data.
func WithMaxAge(duration time.Duration) MetricOption {
	return func(opts *MetricOptions) {
		opts.MaxAge = duration
	}
}

// WithAgeBuckets sets the number of time-based rotation buckets.
// Used in conjunction with MaxAge to implement sliding window histograms.
// More buckets provide smoother rotation but use more memory.
func WithAgeBuckets(count uint32) MetricOption {
	return func(opts *MetricOptions) {
		opts.AgeBuckets = count
	}
}

// WithBufCap sets the buffer capacity for histogram observations.
// This controls how many observations can be buffered before statistics
// are computed. Higher values use more memory but may improve performance.
func WithBufCap(capacity uint32) MetricOption {
	return func(opts *MetricOptions) {
		opts.BufCap = capacity
	}
}

// =============================================================================
// COMPOSITE OPTIONS (Convenience functions with sensible defaults)
// =============================================================================

// WithDefaultHistogramBuckets applies OpenTelemetry-recommended histogram buckets.
// These buckets are suitable for general-purpose distribution metrics like
// request sizes, cache hit counts, or other non-duration metrics.
func WithDefaultHistogramBuckets() MetricOption {
	return func(opts *MetricOptions) {
		opts.Buckets = DefaultHistogramBuckets
	}
}

// WithDefaultTimerBuckets applies sensible bucket boundaries for timer metrics.
// These buckets are optimized for duration measurements (in milliseconds)
// and are suitable for API latency, database queries, and processing time.
func WithDefaultTimerBuckets() MetricOption {
	return func(opts *MetricOptions) {
		opts.Buckets = DefaultDurationBuckets
	}
}

// WithDefaultPercentiles applies commonly tracked percentile values.
// Includes P50 (median), P90, P95, P99, and P99.9.
func WithDefaultPercentiles() MetricOption {
	return func(opts *MetricOptions) {
		opts.Percentiles = DefaultPercentiles
	}
}

// WithSlidingWindow configures a time-based sliding window for histogram metrics.
// This enables real-time percentile calculations over recent observations.
// Duration specifies the time window, and buckets controls rotation granularity.
// Example: WithSlidingWindow(5*time.Minute, 5) creates a 5-minute window with 5 rotation buckets.
func WithSlidingWindow(duration time.Duration, buckets uint32) MetricOption {
	return func(opts *MetricOptions) {
		opts.MaxAge = duration
		opts.AgeBuckets = buckets
	}
}

// MetricFactory creates metrics with optional labels.
type MetricFactory interface {
	// Counter creates a counter metric.
	Counter(name string, opts ...MetricOption) Counter

	// Gauge creates a gauge metric.
	Gauge(name string, opts ...MetricOption) Gauge

	// Histogram creates a histogram metric.
	Histogram(name string, opts ...MetricOption) Histogram

	// Timer creates a timer metric.
	Timer(name string, opts ...MetricOption) Timer
}

// MetricExporter exports metrics in various formats.
type MetricExporter interface {
	// Export exports metrics in the specified format.
	Export(format ExportFormat) ([]byte, error)

	// ExportToFile exports metrics to a file.
	ExportToFile(format ExportFormat, filename string) error
}

// CollectorRegistry manages custom metric collectors.
type CollectorRegistry interface {
	// RegisterCollector registers a custom collector.
	RegisterCollector(collector CustomCollector) error

	// UnregisterCollector removes a collector by name.
	UnregisterCollector(name string) error

	// ListCollectors returns all registered collectors.
	ListCollectors() []CustomCollector
}

// MetricRepository provides queries and introspection of metrics.
type MetricRepository interface {
	// ListMetrics returns all metrics.
	ListMetrics() map[string]any

	// ListMetricsByType returns metrics filtered by type.
	ListMetricsByType(metricType MetricType) map[string]any

	// ListMetricsByTag returns metrics filtered by tag.
	ListMetricsByTag(tagKey, tagValue string) map[string]any

	// Stats returns collector statistics.
	Stats() CollectorStats
}

// MetricManager handles metric lifecycle and configuration.
type MetricManager interface {
	// Reset resets all metrics.
	Reset() error

	// ResetMetric resets a specific metric.
	ResetMetric(name string) error

	// Reload reloads the metrics configuration at runtime.
	Reload(config *MetricsConfig) error
}

// Metrics is the composite interface providing full metrics functionality.
// Implementations should satisfy all constituent interfaces.
type Metrics interface {
	di.Service
	di.HealthChecker
	MetricFactory
	MetricExporter
	CollectorRegistry
	MetricRepository
	MetricManager
}

// Counter tracks monotonically increasing values.
type Counter interface {
	Inc()
	Add(delta float64)
	Value() float64
	WithLabels(labels map[string]string) Counter
	Reset() error
}

// Gauge tracks values that can go up or down.
type Gauge interface {
	Set(value float64)
	Inc()
	Dec()
	Add(delta float64)
	Value() float64
	WithLabels(labels map[string]string) Gauge
	Reset() error
}

// Histogram tracks distributions of values.
type Histogram interface {
	Observe(value float64)
	WithLabels(labels map[string]string) Histogram
	Reset() error
}

// HistogramStats provides statistical information about histogram observations.
type HistogramStats interface {
	Buckets() map[float64]uint64
	Count() uint64
	Sum() float64
	Mean() float64
	Percentile(percentile float64) float64
}

// HistogramWithStats combines observation and statistics capabilities.
type HistogramWithStats interface {
	Histogram
	HistogramStats
}

// TimedHistogram supports time-based observations.
type TimedHistogram interface {
	Histogram
	ObserveDuration(start time.Time)
}

// Timer represents a timer metric.
type Timer interface {
	Record(duration time.Duration)
	Time() func()
	Count() uint64
	Mean() time.Duration
	Percentile(percentile float64) time.Duration
	Min() time.Duration
	Max() time.Duration
	WithLabels(labels map[string]string) Timer
	Reset() error
}

// CustomCollector defines interface for custom metrics collectors.
type CustomCollector interface {
	Name() string
	Collect() map[string]any
	Reset() error
}

// ToggleableCollector is a collector that can be enabled or disabled.
type ToggleableCollector interface {
	CustomCollector
	IsEnabled() bool
}

// =============================================================================
// EXPORTER INTERFACE
// =============================================================================

// Exporter defines the interface for metrics export.
type Exporter interface {
	// Export exports metrics in the specific format
	Export(metrics map[string]any) ([]byte, error)

	// Format returns the export format identifier
	Format() string

	// Stats returns exporter statistics
	Stats() ExporterStats
}

// ExporterStats contains statistics about a metrics exporter.
type ExporterStats struct {
	ExportCount    int64     `json:"export_count"`
	LastExportTime time.Time `json:"last_export_time"`
	BytesExported  int64     `json:"bytes_exported"`
	ErrorCount     int64     `json:"error_count"`
	LastError      string    `json:"last_error,omitempty"`
}

// CollectorStats contains statistics about the metrics collector.
type CollectorStats struct {
	Name               string         `json:"name"`
	Started            bool           `json:"started"`
	StartTime          time.Time      `json:"start_time"`
	Uptime             time.Duration  `json:"uptime"`
	MetricsCreated     int64          `json:"metrics_created"`
	MetricsCollected   int64          `json:"metrics_collected"`
	CustomCollectors   int            `json:"custom_collectors"`
	ActiveMetrics      int            `json:"active_metrics"`
	LastCollectionTime time.Time      `json:"last_collection_time"`
	Errors             []string       `json:"errors"`
	ExporterStats      map[string]any `json:"exporter_stats"`
}
