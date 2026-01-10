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

// MetricsStorageConfig contains storage configuration.
type MetricsStorageConfig struct {
	Type   string         `json:"type"   yaml:"type"`
	Config map[string]any `json:"config" yaml:"config"`
}

// MetricsExporterConfig contains configuration for exporters.
type MetricsExporterConfig struct {
	Enabled  bool           `json:"enabled"  yaml:"enabled"`
	Interval time.Duration  `json:"interval" yaml:"interval"`
	Config   map[string]any `json:"config"   yaml:"config"`
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
	Enabled    bool                             `json:"enabled"    yaml:"enabled"`
	Features   MetricsFeatures                  `json:"features"   yaml:"features"`
	Collection MetricsCollection                `json:"collection" yaml:"collection"`
	Limits     MetricsLimits                    `json:"limits"     yaml:"limits"`
	Storage    *MetricsStorageConfig            `json:"storage"    yaml:"storage"`
	Exporters  map[string]MetricsExporterConfig `json:"exporters"  yaml:"exporters"`
}

// MetricOption is a functional option for configuring metrics.
type MetricOption func(*MetricOptions)

// MetricOptions contains options for metric creation.
type MetricOptions struct {
	Labels map[string]string
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
