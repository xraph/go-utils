package collectors

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xraph/go-utils/metrics"
)

// =============================================================================
// CORE INTERFACES
// =============================================================================

// CustomMetricSource defines the interface for any datasource that can be collected.
// Implementations should return current metric values in a MetricSnapshot.
type CustomMetricSource interface {
	// Name returns the collector name
	Name() string

	// Collect returns the current metric values
	// Called periodically by the collector
	Collect(ctx context.Context) (*MetricSnapshot, error)
}

// =============================================================================
// METRIC SNAPSHOT
// =============================================================================

// MetricSnapshot represents collected metric values at a point in time.
// All map fields are optional - only provide the metric types you need.
type MetricSnapshot struct {
	// Counters: monotonically increasing values
	// The builder tracks deltas automatically
	Counters map[string]float64

	// Gauges: absolute values that can increase or decrease
	Gauges map[string]float64

	// Histograms: observations to be recorded
	// Each collection can contain multiple observations per metric
	Histograms map[string][]float64

	// Summaries: observations for accurate quantile calculations
	// Each collection can contain multiple observations per metric
	Summaries map[string][]float64

	// Timers: duration observations
	// Each collection can contain multiple observations per metric
	Timers map[string][]time.Duration

	// Labels: optional labels to apply to all metrics in this snapshot
	Labels map[string]string

	// Timestamp: when the snapshot was collected
	Timestamp time.Time
}

// Validate checks if the snapshot is valid.
func (s *MetricSnapshot) Validate() error {
	if s == nil {
		return ErrNilSnapshot
	}

	return nil
}

// =============================================================================
// CUSTOM COLLECTOR BUILDER (Pull-based)
// =============================================================================

// CustomCollectorBuilder automatically collects metrics from any datasource
// implementing CustomMetricSource. Supports periodic polling (pull model).
type CustomCollectorBuilder struct {
	source   CustomMetricSource
	interval time.Duration
	metrics  metrics.Metrics
	options  []metrics.MetricOption

	// Context is stored for goroutine lifecycle management (legitimate use case)
	ctx    context.Context //nolint:containedctx // Required for collection loop cancellation
	cancel context.CancelFunc
	wg     sync.WaitGroup // Tracks collection goroutine

	// Internal metric registry
	counters      map[string]metrics.Counter
	gauges        map[string]metrics.Gauge
	histograms    map[string]metrics.Histogram
	summaries     map[string]metrics.Summary
	timers        map[string]metrics.Timer
	counterValues map[string]float64 // Track previous counter values for delta calculation

	mu      sync.RWMutex
	started atomic.Bool
}

// NewCustomCollectorBuilder creates a new collector builder for the given datasource.
func NewCustomCollectorBuilder(source CustomMetricSource, opts ...metrics.MetricOption) *CustomCollectorBuilder {
	ctx, cancel := context.WithCancel(context.Background())

	return &CustomCollectorBuilder{
		source:        source,
		interval:      10 * time.Second, // default poll interval
		metrics:       metrics.NewMetricsCollector(source.Name()),
		options:       opts,
		ctx:           ctx,
		cancel:        cancel,
		counters:      make(map[string]metrics.Counter),
		gauges:        make(map[string]metrics.Gauge),
		histograms:    make(map[string]metrics.Histogram),
		summaries:     make(map[string]metrics.Summary),
		timers:        make(map[string]metrics.Timer),
		counterValues: make(map[string]float64),
	}
}

// WithInterval sets the collection interval for polling.
func (b *CustomCollectorBuilder) WithInterval(d time.Duration) *CustomCollectorBuilder {
	b.interval = d

	return b
}

// WithOptions adds metric options that will be applied to all created metrics.
func (b *CustomCollectorBuilder) WithOptions(opts ...metrics.MetricOption) *CustomCollectorBuilder {
	b.options = append(b.options, opts...)

	return b
}

// Start begins automatic metric collection in a background goroutine.
func (b *CustomCollectorBuilder) Start() error {
	if b.started.Swap(true) {
		return ErrAlreadyStarted
	}

	b.wg.Add(1)

	go b.collectLoop()

	return nil
}

// Stop halts metric collection and waits for the collection goroutine to exit.
func (b *CustomCollectorBuilder) Stop() error {
	if !b.started.Swap(false) {
		return ErrNotStarted
	}

	b.cancel()
	b.wg.Wait() // Wait for collection goroutine to exit

	return nil
}

// Metrics returns the underlying metrics collector for direct access.
func (b *CustomCollectorBuilder) Metrics() metrics.Metrics {
	return b.metrics
}

// CollectOnce performs a single collection without starting the automatic loop.
// Useful for testing or on-demand collection.
func (b *CustomCollectorBuilder) CollectOnce(ctx context.Context) error {
	snapshot, err := b.source.Collect(ctx)
	if err != nil {
		return err
	}

	if err := snapshot.Validate(); err != nil {
		return err
	}

	b.updateFromSnapshot(snapshot)

	return nil
}

// collectLoop periodically collects metrics from the source.
func (b *CustomCollectorBuilder) collectLoop() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	// Initial collection
	b.collect()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.collect()
		}
	}
}

// collect fetches metrics from the source and updates all metrics.
func (b *CustomCollectorBuilder) collect() {
	snapshot, err := b.source.Collect(b.ctx)
	if err != nil {
		// Log error but don't stop collecting
		// TODO: Add proper logging when logger is available
		return
	}

	if err := snapshot.Validate(); err != nil {
		return
	}

	b.updateFromSnapshot(snapshot)
}

// updateFromSnapshot applies the snapshot values to metrics.
func (b *CustomCollectorBuilder) updateFromSnapshot(snapshot *MetricSnapshot) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Update counters (track deltas)
	for name, value := range snapshot.Counters {
		counter := b.getOrCreateCounterLocked(name)

		// Get previous value and calculate delta
		oldValue := b.counterValues[name]
		if value >= oldValue {
			delta := value - oldValue
			if delta > 0 {
				counter.Add(delta)
			}
		} else {
			// Counter reset detected - treat current value as delta
			counter.Add(value)
		}

		// Update tracked value
		b.counterValues[name] = value
	}

	// Update gauges (absolute values)
	for name, value := range snapshot.Gauges {
		gauge := b.getOrCreateGaugeLocked(name)
		gauge.Set(value)
	}

	// Update histograms (observe all values)
	for name, values := range snapshot.Histograms {
		histogram := b.getOrCreateHistogramLocked(name)
		for _, v := range values {
			histogram.Observe(v)
		}
	}

	// Update summaries (observe all values)
	for name, values := range snapshot.Summaries {
		summary := b.getOrCreateSummaryLocked(name)
		for _, v := range values {
			summary.Observe(v)
		}
	}

	// Update timers (record all durations)
	for name, durations := range snapshot.Timers {
		timer := b.getOrCreateTimerLocked(name)
		for _, d := range durations {
			timer.Record(d)
		}
	}
}

// getOrCreateCounterLocked gets or creates a counter. Must be called with lock held.
func (b *CustomCollectorBuilder) getOrCreateCounterLocked(name string) metrics.Counter {
	if counter, exists := b.counters[name]; exists {
		return counter
	}

	counter := b.metrics.Counter(name, b.options...)
	b.counters[name] = counter

	return counter
}

// getOrCreateGaugeLocked gets or creates a gauge. Must be called with lock held.
func (b *CustomCollectorBuilder) getOrCreateGaugeLocked(name string) metrics.Gauge {
	if gauge, exists := b.gauges[name]; exists {
		return gauge
	}

	gauge := b.metrics.Gauge(name, b.options...)
	b.gauges[name] = gauge

	return gauge
}

// getOrCreateHistogramLocked gets or creates a histogram. Must be called with lock held.
func (b *CustomCollectorBuilder) getOrCreateHistogramLocked(name string) metrics.Histogram {
	if histogram, exists := b.histograms[name]; exists {
		return histogram
	}

	histogram := b.metrics.Histogram(name, b.options...)
	b.histograms[name] = histogram

	return histogram
}

// getOrCreateSummaryLocked gets or creates a summary. Must be called with lock held.
func (b *CustomCollectorBuilder) getOrCreateSummaryLocked(name string) metrics.Summary {
	if summary, exists := b.summaries[name]; exists {
		return summary
	}

	summary := b.metrics.Summary(name, b.options...)
	b.summaries[name] = summary

	return summary
}

// getOrCreateTimerLocked gets or creates a timer. Must be called with lock held.
func (b *CustomCollectorBuilder) getOrCreateTimerLocked(name string) metrics.Timer {
	if timer, exists := b.timers[name]; exists {
		return timer
	}

	timer := b.metrics.Timer(name, b.options...)
	b.timers[name] = timer

	return timer
}

// =============================================================================
// PUSHABLE COLLECTOR BUILDER (Push-based)
// =============================================================================

// PushableCollectorBuilder extends CustomCollectorBuilder to support
// event-driven metric collection via Push() in addition to periodic polling.
type PushableCollectorBuilder struct {
	*CustomCollectorBuilder

	pushChan   chan *MetricSnapshot
	bufferSize int
}

// NewPushableCollectorBuilder creates a builder that supports both pull and push.
func NewPushableCollectorBuilder(source CustomMetricSource, opts ...metrics.MetricOption) *PushableCollectorBuilder {
	return &PushableCollectorBuilder{
		CustomCollectorBuilder: NewCustomCollectorBuilder(source, opts...),
		pushChan:               make(chan *MetricSnapshot, 100), // default buffer size
		bufferSize:             100,
	}
}

// WithInterval sets the collection interval for polling (overrides embedded method).
func (b *PushableCollectorBuilder) WithInterval(d time.Duration) *PushableCollectorBuilder {
	b.CustomCollectorBuilder.WithInterval(d)

	return b
}

// WithOptions adds metric options (overrides embedded method).
func (b *PushableCollectorBuilder) WithOptions(opts ...metrics.MetricOption) *PushableCollectorBuilder {
	b.CustomCollectorBuilder.WithOptions(opts...)

	return b
}

// WithBufferSize sets the push channel buffer size.
func (b *PushableCollectorBuilder) WithBufferSize(size int) *PushableCollectorBuilder {
	b.bufferSize = size
	// Recreate channel with new size
	b.pushChan = make(chan *MetricSnapshot, size)

	return b
}

// Push sends metrics for immediate collection (non-blocking).
// If the buffer is full, the push is dropped to prevent blocking.
func (b *PushableCollectorBuilder) Push(snapshot *MetricSnapshot) error {
	if !b.started.Load() {
		return ErrNotStarted
	}

	if err := snapshot.Validate(); err != nil {
		return err
	}

	select {
	case b.pushChan <- snapshot:
		return nil
	default:
		// Buffer full - drop the snapshot
		// TODO: Add metric for dropped pushes
		return ErrPushBufferFull
	}
}

// Start begins both periodic polling and push-based collection.
func (b *PushableCollectorBuilder) Start() error {
	if b.started.Swap(true) {
		return ErrAlreadyStarted
	}

	b.wg.Add(1)

	go b.collectLoopWithPush()

	return nil
}

// collectLoopWithPush handles both periodic pulls and pushed snapshots.
func (b *PushableCollectorBuilder) collectLoopWithPush() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()

	// Note: We don't do an initial collection here to avoid potential race
	// conditions. The first collection will happen on the first ticker or push.

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			// Pull-based collection
			b.collect()
		case snapshot := <-b.pushChan:
			// Push-based collection
			b.updateFromSnapshot(snapshot)
		}
	}
}

// =============================================================================
// ERRORS
// =============================================================================

var (
	// ErrNilSnapshot is returned when a nil snapshot is provided.
	ErrNilSnapshot = &CollectorError{Message: "snapshot is nil"}

	// ErrAlreadyStarted is returned when Start() is called on an already running collector.
	ErrAlreadyStarted = &CollectorError{Message: "collector already started"}

	// ErrNotStarted is returned when operations require the collector to be started.
	ErrNotStarted = &CollectorError{Message: "collector not started"}

	// ErrPushBufferFull is returned when the push buffer is full.
	ErrPushBufferFull = &CollectorError{Message: "push buffer full, snapshot dropped"}
)

// CollectorError represents a collector-related error.
type CollectorError struct {
	Message string
}

func (e *CollectorError) Error() string {
	return e.Message
}
