package metrics

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/beorn7/perks/quantile"
)

// =============================================================================
// METRIC CORE - Shared base for all metrics
// =============================================================================

// metricCore contains shared fields for all metric implementations.
type metricCore struct {
	mu          sync.RWMutex
	name        string
	metricType  MetricType
	description string
	unit        string
	namespace   string
	subsystem   string
	constLabels map[string]string
	labels      map[string]string
	timestamp   atomic.Value // stores time.Time
}

// newMetricCore creates a new metric core with options applied.
func newMetricCore(name string, metricType MetricType, opts ...MetricOption) *metricCore {
	options := &MetricOptions{}
	for _, opt := range opts {
		opt(options)
	}

	mc := &metricCore{
		name:        name,
		metricType:  metricType,
		description: options.Description,
		unit:        options.Unit,
		namespace:   options.Namespace,
		subsystem:   options.Subsystem,
		constLabels: options.ConstLabels,
		labels:      options.Labels,
	}

	mc.timestamp.Store(time.Now())

	return mc
}

// Describe returns metadata about the metric.
func (mc *metricCore) describe() MetricMetadata {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return MetricMetadata{
		Name:        mc.fullName(),
		Type:        mc.metricType,
		Description: mc.description,
		Unit:        mc.unit,
		Namespace:   mc.namespace,
		Subsystem:   mc.subsystem,
		ConstLabels: mc.constLabels,
	}
}

// fullName returns the fully qualified metric name.
func (mc *metricCore) fullName() string {
	parts := make([]string, 0, 3)
	if mc.namespace != "" {
		parts = append(parts, mc.namespace)
	}

	if mc.subsystem != "" {
		parts = append(parts, mc.subsystem)
	}

	parts = append(parts, mc.name)

	name := ""

	var nameSb84 strings.Builder

	for i, part := range parts {
		if i > 0 {
			nameSb84.WriteString("_")
		}

		nameSb84.WriteString(part)
	}

	name += nameSb84.String()

	return name
}

// updateTimestamp updates the timestamp to current time.
func (mc *metricCore) updateTimestamp() {
	mc.timestamp.Store(time.Now())
}

// getTimestamp returns the current timestamp.
func (mc *metricCore) getTimestamp() time.Time {
	return mc.timestamp.Load().(time.Time)
}

// =============================================================================
// EXEMPLAR STORE - Lock-free ring buffer for exemplars
// =============================================================================

const exemplarBufferSize = 10

// exemplarStore is a thread-safe ring buffer for storing exemplars.
type exemplarStore struct {
	exemplars [exemplarBufferSize]atomic.Value // each stores *Exemplar
	index     atomic.Uint32
}

// newExemplarStore creates a new exemplar store.
func newExemplarStore() *exemplarStore {
	return &exemplarStore{}
}

// Add adds an exemplar to the store.
func (es *exemplarStore) Add(exemplar Exemplar) {
	idx := es.index.Add(1) % exemplarBufferSize
	es.exemplars[idx].Store(&exemplar)
}

// GetAll returns all stored exemplars.
func (es *exemplarStore) GetAll() []Exemplar {
	result := make([]Exemplar, 0, exemplarBufferSize)

	for i := range exemplarBufferSize {
		if val := es.exemplars[i].Load(); val != nil {
			if ex, ok := val.(*Exemplar); ok && ex != nil {
				result = append(result, *ex)
			}
		}
	}

	return result
}

// =============================================================================
// COUNTER IMPLEMENTATION
// =============================================================================

// counterImpl is a thread-safe counter implementation.
type counterImpl struct {
	*metricCore

	value     atomic.Uint64 // stores float64 bits
	exemplars *exemplarStore
}

// newCounter creates a new counter.
func newCounter(name string, opts ...MetricOption) *counterImpl {
	return &counterImpl{
		metricCore: newMetricCore(name, MetricTypeCounter, opts...),
		exemplars:  newExemplarStore(),
	}
}

func (c *counterImpl) Inc() {
	c.Add(1)
}

func (c *counterImpl) Add(delta float64) {
	if delta < 0 {
		return // Counters can't decrease
	}

	for {
		oldBits := c.value.Load()
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal + delta
		newBits := math.Float64bits(newVal)

		if c.value.CompareAndSwap(oldBits, newBits) {
			c.updateTimestamp()

			break
		}
	}
}

func (c *counterImpl) AddWithExemplar(delta float64, exemplar Exemplar) {
	c.Add(delta)
	c.exemplars.Add(exemplar)
}

func (c *counterImpl) Value() float64 {
	return math.Float64frombits(c.value.Load())
}

func (c *counterImpl) Timestamp() time.Time {
	return c.getTimestamp()
}

func (c *counterImpl) Exemplars() []Exemplar {
	return c.exemplars.GetAll()
}

func (c *counterImpl) Describe() MetricMetadata {
	return c.describe()
}

func (c *counterImpl) WithLabels(labels map[string]string) Counter {
	// Create a new counter with merged labels
	newCounter := newCounter(c.name, WithLabels(labels))
	newCounter.description = c.description
	newCounter.unit = c.unit
	newCounter.namespace = c.namespace
	newCounter.subsystem = c.subsystem

	return newCounter
}

func (c *counterImpl) Reset() error {
	c.value.Store(0)
	c.updateTimestamp()

	return nil
}

// =============================================================================
// GAUGE IMPLEMENTATION
// =============================================================================

// gaugeImpl is a thread-safe gauge implementation.
type gaugeImpl struct {
	*metricCore

	value atomic.Uint64 // stores float64 bits
}

// newGauge creates a new gauge.
func newGauge(name string, opts ...MetricOption) *gaugeImpl {
	return &gaugeImpl{
		metricCore: newMetricCore(name, MetricTypeGauge, opts...),
	}
}

func (g *gaugeImpl) Set(value float64) {
	g.value.Store(math.Float64bits(value))
	g.updateTimestamp()
}

func (g *gaugeImpl) Inc() {
	g.Add(1)
}

func (g *gaugeImpl) Dec() {
	g.Sub(1)
}

func (g *gaugeImpl) Add(delta float64) {
	for {
		oldBits := g.value.Load()
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal + delta
		newBits := math.Float64bits(newVal)

		if g.value.CompareAndSwap(oldBits, newBits) {
			g.updateTimestamp()

			break
		}
	}
}

func (g *gaugeImpl) Sub(delta float64) {
	g.Add(-delta)
}

func (g *gaugeImpl) SetToCurrentTime() {
	now := time.Now()
	g.Set(float64(now.Unix()))
}

func (g *gaugeImpl) Value() float64 {
	return math.Float64frombits(g.value.Load())
}

func (g *gaugeImpl) Timestamp() time.Time {
	return g.getTimestamp()
}

func (g *gaugeImpl) Describe() MetricMetadata {
	return g.describe()
}

func (g *gaugeImpl) WithLabels(labels map[string]string) Gauge {
	newGauge := newGauge(g.name, WithLabels(labels))
	newGauge.description = g.description
	newGauge.unit = g.unit
	newGauge.namespace = g.namespace
	newGauge.subsystem = g.subsystem

	return newGauge
}

func (g *gaugeImpl) Reset() error {
	g.value.Store(0)
	g.updateTimestamp()

	return nil
}

// =============================================================================
// HISTOGRAM IMPLEMENTATION
// =============================================================================

// histogramImpl is a thread-safe histogram implementation.
type histogramImpl struct {
	*metricCore

	mu        sync.RWMutex
	buckets   []float64       // Sorted bucket boundaries
	counts    []atomic.Uint64 // Bucket counts
	sum       atomic.Uint64   // Sum of observations (float64 bits)
	count     atomic.Uint64   // Total count
	min       atomic.Uint64   // Minimum value (float64 bits)
	max       atomic.Uint64   // Maximum value (float64 bits)
	exemplars *exemplarStore
}

// newHistogram creates a new histogram.
func newHistogram(name string, opts ...MetricOption) *histogramImpl {
	options := &MetricOptions{}
	for _, opt := range opts {
		opt(options)
	}

	buckets := options.Buckets
	if len(buckets) == 0 {
		buckets = DefaultHistogramBuckets
	}

	// Ensure buckets are sorted
	sortedBuckets := make([]float64, len(buckets))
	copy(sortedBuckets, buckets)
	sort.Float64s(sortedBuckets)

	counts := make([]atomic.Uint64, len(sortedBuckets)+1) // +1 for +Inf bucket

	h := &histogramImpl{
		metricCore: newMetricCore(name, MetricTypeHistogram, opts...),
		buckets:    sortedBuckets,
		counts:     counts,
		exemplars:  newExemplarStore(),
	}

	// Initialize min to max float64, max to 0
	h.min.Store(math.Float64bits(math.MaxFloat64))
	h.max.Store(0)

	return h
}

func (h *histogramImpl) Observe(value float64) {
	h.ObserveWithExemplar(value, Exemplar{})
}

func (h *histogramImpl) ObserveWithExemplar(value float64, exemplar Exemplar) {
	// Update count
	h.count.Add(1)

	// Update sum
	for {
		oldBits := h.sum.Load()
		oldSum := math.Float64frombits(oldBits)
		newSum := oldSum + value
		newBits := math.Float64bits(newSum)

		if h.sum.CompareAndSwap(oldBits, newBits) {
			break
		}
	}

	// Update min
	valueBits := math.Float64bits(value)

	for {
		oldMinBits := h.min.Load()

		oldMin := math.Float64frombits(oldMinBits)
		if value >= oldMin {
			break
		}

		if h.min.CompareAndSwap(oldMinBits, valueBits) {
			break
		}
	}

	// Update max
	for {
		oldMaxBits := h.max.Load()

		oldMax := math.Float64frombits(oldMaxBits)
		if value <= oldMax {
			break
		}

		if h.max.CompareAndSwap(oldMaxBits, valueBits) {
			break
		}
	}

	// Find bucket using binary search
	idx := sort.SearchFloat64s(h.buckets, value)
	h.counts[idx].Add(1)

	// Store exemplar if provided
	if exemplar.TraceID != "" || exemplar.SpanID != "" {
		exemplar.Value = value
		exemplar.Timestamp = time.Now()
		h.exemplars.Add(exemplar)
	}

	h.updateTimestamp()
}

func (h *histogramImpl) Count() uint64 {
	return h.count.Load()
}

func (h *histogramImpl) Sum() float64 {
	return math.Float64frombits(h.sum.Load())
}

func (h *histogramImpl) Mean() float64 {
	count := h.Count()
	if count == 0 {
		return 0
	}

	return h.Sum() / float64(count)
}

func (h *histogramImpl) StdDev() float64 {
	// For histogram, we can only estimate stddev from bucket data
	// This is an approximation
	count := h.Count()
	if count < 2 {
		return 0
	}

	mean := h.Mean()
	variance := 0.0

	// Estimate variance from bucket midpoints
	h.mu.RLock()
	defer h.mu.RUnlock()

	for i, boundary := range h.buckets {
		bucketCount := float64(h.counts[i].Load())
		if bucketCount == 0 {
			continue
		}

		// Use bucket midpoint
		var midpoint float64
		if i == 0 {
			midpoint = boundary / 2
		} else {
			midpoint = (h.buckets[i-1] + boundary) / 2
		}

		diff := midpoint - mean
		variance += bucketCount * diff * diff
	}

	variance /= float64(count)

	return math.Sqrt(variance)
}

func (h *histogramImpl) Min() float64 {
	minBits := h.min.Load()

	minVal := math.Float64frombits(minBits)
	if minVal == math.MaxFloat64 {
		return 0 // No observations yet
	}

	return minVal
}

func (h *histogramImpl) Max() float64 {
	return math.Float64frombits(h.max.Load())
}

func (h *histogramImpl) Quantile(q float64) float64 {
	if q < 0 || q > 1 {
		return 0
	}

	count := h.Count()
	if count == 0 {
		return 0
	}

	// Find the bucket containing the quantile
	targetRank := uint64(float64(count) * q)
	cumulative := uint64(0)

	h.mu.RLock()
	defer h.mu.RUnlock()

	for i := range h.buckets {
		cumulative += h.counts[i].Load()
		if cumulative >= targetRank {
			return h.buckets[i]
		}
	}

	// If we get here, return the last bucket
	if len(h.buckets) > 0 {
		return h.buckets[len(h.buckets)-1]
	}

	return 0
}

func (h *histogramImpl) Exemplars() []Exemplar {
	return h.exemplars.GetAll()
}

func (h *histogramImpl) Describe() MetricMetadata {
	return h.describe()
}

func (h *histogramImpl) WithLabels(labels map[string]string) Histogram {
	newHist := newHistogram(h.name, WithLabels(labels), WithBuckets(h.buckets...))
	newHist.description = h.description
	newHist.unit = h.unit
	newHist.namespace = h.namespace
	newHist.subsystem = h.subsystem

	return newHist
}

func (h *histogramImpl) Reset() error {
	h.count.Store(0)
	h.sum.Store(0)
	h.min.Store(math.Float64bits(math.MaxFloat64))
	h.max.Store(0)

	for i := range h.counts {
		h.counts[i].Store(0)
	}

	h.updateTimestamp()

	return nil
}

// =============================================================================
// SUMMARY IMPLEMENTATION
// =============================================================================

// summaryImpl is a summary metric implementation.
type summaryImpl struct {
	*metricCore

	mu         sync.Mutex
	objectives map[float64]float64 // Quantile -> error margin
	stream     *quantile.Stream
	count      atomic.Uint64
	sum        atomic.Uint64
	values     []float64 // For accurate calculations
	maxAge     time.Duration
	ageBuckets uint32
	bufCap     uint32
}

// newSummary creates a new summary.
func newSummary(name string, opts ...MetricOption) *summaryImpl {
	options := &MetricOptions{}
	for _, opt := range opts {
		opt(options)
	}

	objectives := make(map[float64]float64)

	if len(options.Percentiles) > 0 {
		for _, p := range options.Percentiles {
			objectives[p] = 0.01 // 1% error margin
		}
	} else {
		// Default percentiles
		for _, p := range DefaultPercentiles {
			objectives[p] = 0.01
		}
	}

	s := &summaryImpl{
		metricCore: newMetricCore(name, MetricTypeSummary, opts...),
		objectives: objectives,
		stream:     quantile.NewTargeted(objectives),
		values:     make([]float64, 0, 1000),
		maxAge:     options.MaxAge,
		ageBuckets: options.AgeBuckets,
		bufCap:     options.BufCap,
	}

	return s
}

func (s *summaryImpl) Observe(value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count.Add(1)

	// Update sum
	for {
		oldBits := s.sum.Load()
		oldSum := math.Float64frombits(oldBits)
		newSum := oldSum + value
		newBits := math.Float64bits(newSum)

		if s.sum.CompareAndSwap(oldBits, newBits) {
			break
		}
	}

	// Add to stream for quantile calculation
	s.stream.Insert(value)

	// Store value for accurate calculations
	s.values = append(s.values, value)

	// Limit buffer size
	if s.bufCap > 0 && len(s.values) > int(s.bufCap) {
		s.values = s.values[len(s.values)-int(s.bufCap):]
	}

	s.updateTimestamp()
}

func (s *summaryImpl) Count() uint64 {
	return s.count.Load()
}

func (s *summaryImpl) Sum() float64 {
	return math.Float64frombits(s.sum.Load())
}

func (s *summaryImpl) Mean() float64 {
	count := s.Count()
	if count == 0 {
		return 0
	}

	return s.Sum() / float64(count)
}

func (s *summaryImpl) Quantile(q float64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.values) == 0 {
		return 0
	}

	return s.stream.Query(q)
}

func (s *summaryImpl) Min() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.values) == 0 {
		return 0
	}

	minVal := s.values[0]
	for _, v := range s.values {
		if v < minVal {
			minVal = v
		}
	}

	return minVal
}

func (s *summaryImpl) Max() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.values) == 0 {
		return 0
	}

	maxVal := s.values[0]
	for _, v := range s.values {
		if v > maxVal {
			maxVal = v
		}
	}

	return maxVal
}

func (s *summaryImpl) StdDev() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.values) < 2 {
		return 0
	}

	mean := s.Mean()
	variance := 0.0

	for _, v := range s.values {
		diff := v - mean
		variance += diff * diff
	}

	variance /= float64(len(s.values))

	return math.Sqrt(variance)
}

func (s *summaryImpl) Describe() MetricMetadata {
	return s.describe()
}

func (s *summaryImpl) WithLabels(labels map[string]string) Summary {
	newSummary := newSummary(s.name, WithLabels(labels))
	newSummary.description = s.description
	newSummary.unit = s.unit
	newSummary.namespace = s.namespace
	newSummary.subsystem = s.subsystem

	return newSummary
}

func (s *summaryImpl) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count.Store(0)
	s.sum.Store(0)
	s.stream.Reset()
	s.values = make([]float64, 0, 1000)
	s.updateTimestamp()

	return nil
}

// =============================================================================
// TIMER IMPLEMENTATION
// =============================================================================

// timerImpl is a timer metric implementation.
type timerImpl struct {
	*metricCore

	histogram *histogramImpl
	exemplars *exemplarStore
}

// newTimer creates a new timer.
func newTimer(name string, opts ...MetricOption) *timerImpl {
	options := &MetricOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use duration buckets if not specified
	if len(options.Buckets) == 0 {
		options.Buckets = DefaultDurationBuckets
	}

	t := &timerImpl{
		metricCore: newMetricCore(name, MetricTypeTimer, opts...),
		histogram:  newHistogram(name+"_duration", opts...),
		exemplars:  newExemplarStore(),
	}

	return t
}

func (t *timerImpl) Record(duration time.Duration) {
	// Convert to milliseconds
	ms := float64(duration.Nanoseconds()) / 1e6
	t.histogram.Observe(ms)
	t.updateTimestamp()
}

func (t *timerImpl) RecordWithExemplar(duration time.Duration, exemplar Exemplar) {
	ms := float64(duration.Nanoseconds()) / 1e6
	t.histogram.ObserveWithExemplar(ms, exemplar)
	t.exemplars.Add(exemplar)
	t.updateTimestamp()
}

func (t *timerImpl) Time() func() {
	start := time.Now()

	return func() {
		t.Record(time.Since(start))
	}
}

func (t *timerImpl) Count() uint64 {
	return t.histogram.Count()
}

func (t *timerImpl) Sum() time.Duration {
	ms := t.histogram.Sum()

	return time.Duration(ms * 1e6) // Convert ms to nanoseconds
}

func (t *timerImpl) Mean() time.Duration {
	ms := t.histogram.Mean()

	return time.Duration(ms * 1e6)
}

func (t *timerImpl) StdDev() time.Duration {
	ms := t.histogram.StdDev()

	return time.Duration(ms * 1e6)
}

func (t *timerImpl) Min() time.Duration {
	ms := t.histogram.Min()

	return time.Duration(ms * 1e6)
}

func (t *timerImpl) Max() time.Duration {
	ms := t.histogram.Max()

	return time.Duration(ms * 1e6)
}

func (t *timerImpl) Percentile(percentile float64) time.Duration {
	ms := t.histogram.Quantile(percentile)

	return time.Duration(ms * 1e6)
}

func (t *timerImpl) Quantile(q float64) time.Duration {
	return t.Percentile(q)
}

func (t *timerImpl) Exemplars() []Exemplar {
	return t.exemplars.GetAll()
}

func (t *timerImpl) Describe() MetricMetadata {
	return t.describe()
}

func (t *timerImpl) WithLabels(labels map[string]string) Timer {
	newTimer := newTimer(t.name, WithLabels(labels))
	newTimer.description = t.description
	newTimer.unit = t.unit
	newTimer.namespace = t.namespace
	newTimer.subsystem = t.subsystem

	return newTimer
}

func (t *timerImpl) Reset() error {
	if err := t.histogram.Reset(); err != nil {
		return err
	}

	t.updateTimestamp()

	return nil
}

// =============================================================================
// METRICS COLLECTOR - Factory and Registry
// =============================================================================

// metricsCollector is the main metrics collector implementation.
type metricsCollector struct {
	mu               sync.RWMutex
	name             string
	counters         map[string]*counterImpl
	gauges           map[string]*gaugeImpl
	histograms       map[string]*histogramImpl
	summaries        map[string]*summaryImpl
	timers           map[string]*timerImpl
	customCollectors map[string]CustomCollector
	startTime        time.Time
	started          atomic.Bool
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector(name string) Metrics {
	return &metricsCollector{
		name:             name,
		counters:         make(map[string]*counterImpl),
		gauges:           make(map[string]*gaugeImpl),
		histograms:       make(map[string]*histogramImpl),
		summaries:        make(map[string]*summaryImpl),
		timers:           make(map[string]*timerImpl),
		customCollectors: make(map[string]CustomCollector),
		startTime:        time.Now(),
	}
}

// Service interface implementation

func (mc *metricsCollector) Name() string {
	return mc.name
}

func (mc *metricsCollector) Start(ctx context.Context) error {
	mc.started.Store(true)

	return nil
}

func (mc *metricsCollector) Stop(ctx context.Context) error {
	mc.started.Store(false)

	return nil
}

func (mc *metricsCollector) Health(ctx context.Context) error {
	if !mc.started.Load() {
		return ErrNotStarted
	}

	return nil
}

// MetricFactory interface implementation

func (mc *metricsCollector) Counter(name string, opts ...MetricOption) Counter {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if counter, exists := mc.counters[name]; exists {
		return counter
	}

	counter := newCounter(name, opts...)
	mc.counters[name] = counter

	return counter
}

func (mc *metricsCollector) Gauge(name string, opts ...MetricOption) Gauge {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if gauge, exists := mc.gauges[name]; exists {
		return gauge
	}

	gauge := newGauge(name, opts...)
	mc.gauges[name] = gauge

	return gauge
}

func (mc *metricsCollector) Histogram(name string, opts ...MetricOption) Histogram {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if histogram, exists := mc.histograms[name]; exists {
		return histogram
	}

	histogram := newHistogram(name, opts...)
	mc.histograms[name] = histogram

	return histogram
}

func (mc *metricsCollector) Summary(name string, opts ...MetricOption) Summary {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if summary, exists := mc.summaries[name]; exists {
		return summary
	}

	summary := newSummary(name, opts...)
	mc.summaries[name] = summary

	return summary
}

func (mc *metricsCollector) Timer(name string, opts ...MetricOption) Timer {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if timer, exists := mc.timers[name]; exists {
		return timer
	}

	timer := newTimer(name, opts...)
	mc.timers[name] = timer

	return timer
}

// MetricExporter interface implementation

func (mc *metricsCollector) Export(format ExportFormat) ([]byte, error) {
	// Placeholder - would implement Prometheus, JSON, etc. export
	return []byte("{}"), nil
}

func (mc *metricsCollector) ExportToFile(format ExportFormat, filename string) error {
	// Placeholder - would write to file
	return nil
}

// CollectorRegistry interface implementation

func (mc *metricsCollector) RegisterCollector(collector CustomCollector) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	name := collector.Name()
	if _, exists := mc.customCollectors[name]; exists {
		return ErrCollectorAlreadyRegistered
	}

	mc.customCollectors[name] = collector

	return nil
}

func (mc *metricsCollector) UnregisterCollector(name string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.customCollectors[name]; !exists {
		return ErrCollectorNotFound
	}

	delete(mc.customCollectors, name)

	return nil
}

func (mc *metricsCollector) ListCollectors() []CustomCollector {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	collectors := make([]CustomCollector, 0, len(mc.customCollectors))
	for _, collector := range mc.customCollectors {
		collectors = append(collectors, collector)
	}

	return collectors
}

// MetricRepository interface implementation

func (mc *metricsCollector) ListMetrics() map[string]any {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := make(map[string]any)

	for name, counter := range mc.counters {
		metrics[name] = counter
	}

	for name, gauge := range mc.gauges {
		metrics[name] = gauge
	}

	for name, histogram := range mc.histograms {
		metrics[name] = histogram
	}

	for name, summary := range mc.summaries {
		metrics[name] = summary
	}

	for name, timer := range mc.timers {
		metrics[name] = timer
	}

	return metrics
}

func (mc *metricsCollector) ListMetricsByType(metricType MetricType) map[string]any {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := make(map[string]any)

	switch metricType {
	case MetricTypeCounter:
		for name, counter := range mc.counters {
			metrics[name] = counter
		}
	case MetricTypeGauge:
		for name, gauge := range mc.gauges {
			metrics[name] = gauge
		}
	case MetricTypeHistogram:
		for name, histogram := range mc.histograms {
			metrics[name] = histogram
		}
	case MetricTypeSummary:
		for name, summary := range mc.summaries {
			metrics[name] = summary
		}
	case MetricTypeTimer:
		for name, timer := range mc.timers {
			metrics[name] = timer
		}
	}

	return metrics
}

func (mc *metricsCollector) ListMetricsByTag(tagKey, tagValue string) map[string]any {
	// Placeholder - would filter by labels/tags
	return mc.ListMetrics()
}

func (mc *metricsCollector) Stats() CollectorStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metricsByType := make(map[MetricType]int)
	metricsByType[MetricTypeCounter] = len(mc.counters)
	metricsByType[MetricTypeGauge] = len(mc.gauges)
	metricsByType[MetricTypeHistogram] = len(mc.histograms)
	metricsByType[MetricTypeSummary] = len(mc.summaries)
	metricsByType[MetricTypeTimer] = len(mc.timers)

	totalMetrics := len(mc.counters) + len(mc.gauges) + len(mc.histograms) +
		len(mc.summaries) + len(mc.timers)

	return CollectorStats{
		Name:                   mc.name,
		Started:                mc.started.Load(),
		StartTime:              mc.startTime,
		Uptime:                 time.Since(mc.startTime),
		MetricsCreated:         int64(totalMetrics),
		ActiveMetrics:          totalMetrics,
		MetricsByType:          metricsByType,
		CustomCollectors:       len(mc.customCollectors),
		ActiveCustomCollectors: len(mc.customCollectors),
		HealthStatus:           "healthy",
		Degraded:               false,
	}
}

// MetricManager interface implementation

func (mc *metricsCollector) Reset() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for _, counter := range mc.counters {
		if err := counter.Reset(); err != nil {
			return err
		}
	}

	for _, gauge := range mc.gauges {
		if err := gauge.Reset(); err != nil {
			return err
		}
	}

	for _, histogram := range mc.histograms {
		if err := histogram.Reset(); err != nil {
			return err
		}
	}

	for _, summary := range mc.summaries {
		if err := summary.Reset(); err != nil {
			return err
		}
	}

	for _, timer := range mc.timers {
		if err := timer.Reset(); err != nil {
			return err
		}
	}

	return nil
}

func (mc *metricsCollector) ResetMetric(name string) error {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if counter, exists := mc.counters[name]; exists {
		return counter.Reset()
	}

	if gauge, exists := mc.gauges[name]; exists {
		return gauge.Reset()
	}

	if histogram, exists := mc.histograms[name]; exists {
		return histogram.Reset()
	}

	if summary, exists := mc.summaries[name]; exists {
		return summary.Reset()
	}

	if timer, exists := mc.timers[name]; exists {
		return timer.Reset()
	}

	return ErrMetricNotFound
}

func (mc *metricsCollector) Reload(config *MetricsConfig) error {
	// Placeholder - would reload configuration
	return nil
}

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrNotStarted                 = &MetricError{Message: "metrics collector not started"}
	ErrCollectorAlreadyRegistered = &MetricError{Message: "collector already registered"}
	ErrCollectorNotFound          = &MetricError{Message: "collector not found"}
	ErrMetricNotFound             = &MetricError{Message: "metric not found"}
)

// MetricError represents a metrics-related error.
type MetricError struct {
	Message string
}

func (e *MetricError) Error() string {
	return e.Message
}

// Float64bits and Float64frombits helpers for older Go versions compatibility.
// Note: math.Float64bits and math.Float64frombits are used for atomic float operations.
