package metrics

import (
	"context"
	"sync"
	"time"
)

// =============================================================================
// MOCK METRICS
// =============================================================================

// MockMetrics is a mock implementation of the Metrics interface.
// Thread-safe with call tracking for verification in tests.
type MockMetrics struct {
	mu sync.RWMutex

	// Service interface
	NameFunc   func() string
	StartFunc  func(ctx context.Context) error
	StopFunc   func(ctx context.Context) error
	HealthFunc func(ctx context.Context) error

	// MetricFactory interface
	CounterFunc   func(name string, opts ...MetricOption) Counter
	GaugeFunc     func(name string, opts ...MetricOption) Gauge
	HistogramFunc func(name string, opts ...MetricOption) Histogram
	TimerFunc     func(name string, opts ...MetricOption) Timer

	// MetricExporter interface
	ExportFunc       func(format ExportFormat) ([]byte, error)
	ExportToFileFunc func(format ExportFormat, filename string) error

	// CollectorRegistry interface
	RegisterCollectorFunc   func(collector CustomCollector) error
	UnregisterCollectorFunc func(name string) error
	ListCollectorsFunc      func() []CustomCollector

	// MetricRepository interface
	ListMetricsFunc       func() map[string]any
	ListMetricsByTypeFunc func(metricType MetricType) map[string]any
	ListMetricsByTagFunc  func(tagKey, tagValue string) map[string]any
	StatsFunc             func() CollectorStats

	// MetricManager interface
	ResetFunc       func() error
	ResetMetricFunc func(name string) error
	ReloadFunc      func(config *MetricsConfig) error

	// Call tracking
	NameCalls         int
	StartCalls        int
	StopCalls         int
	HealthCalls       int
	CounterCalls      int
	GaugeCalls        int
	HistogramCalls    int
	TimerCalls        int
	ExportCalls       int
	ExportToFileCalls int
	ResetCalls        int
	ReloadCalls       int
}

// NewMockMetrics creates a new mock metrics with sensible defaults.
func NewMockMetrics() *MockMetrics {
	m := &MockMetrics{}

	// Default implementations
	m.NameFunc = func() string { return "mock-metrics" }
	m.StartFunc = func(ctx context.Context) error { return nil }
	m.StopFunc = func(ctx context.Context) error { return nil }
	m.HealthFunc = func(ctx context.Context) error { return nil }

	m.CounterFunc = func(name string, opts ...MetricOption) Counter {
		return NewMockCounter()
	}
	m.GaugeFunc = func(name string, opts ...MetricOption) Gauge {
		return NewMockGauge()
	}
	m.HistogramFunc = func(name string, opts ...MetricOption) Histogram {
		return NewMockHistogram()
	}
	m.TimerFunc = func(name string, opts ...MetricOption) Timer {
		return NewMockTimer()
	}

	m.ExportFunc = func(format ExportFormat) ([]byte, error) {
		return []byte("{}"), nil
	}
	m.ExportToFileFunc = func(format ExportFormat, filename string) error {
		return nil
	}

	m.RegisterCollectorFunc = func(collector CustomCollector) error {
		return nil
	}
	m.UnregisterCollectorFunc = func(name string) error {
		return nil
	}
	m.ListCollectorsFunc = func() []CustomCollector {
		return []CustomCollector{}
	}

	m.ListMetricsFunc = func() map[string]any {
		return make(map[string]any)
	}
	m.ListMetricsByTypeFunc = func(metricType MetricType) map[string]any {
		return make(map[string]any)
	}
	m.ListMetricsByTagFunc = func(tagKey, tagValue string) map[string]any {
		return make(map[string]any)
	}
	m.StatsFunc = func() CollectorStats {
		return CollectorStats{
			Name:    "mock-metrics",
			Started: true,
		}
	}

	m.ResetFunc = func() error { return nil }
	m.ResetMetricFunc = func(name string) error { return nil }
	m.ReloadFunc = func(config *MetricsConfig) error { return nil }

	return m
}

// Service interface implementation

func (m *MockMetrics) Name() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NameCalls++

	return m.NameFunc()
}

func (m *MockMetrics) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCalls++

	return m.StartFunc(ctx)
}

func (m *MockMetrics) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalls++

	return m.StopFunc(ctx)
}

func (m *MockMetrics) Health(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HealthCalls++

	return m.HealthFunc(ctx)
}

// MetricFactory interface implementation

func (m *MockMetrics) Counter(name string, opts ...MetricOption) Counter {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CounterCalls++

	return m.CounterFunc(name, opts...)
}

func (m *MockMetrics) Gauge(name string, opts ...MetricOption) Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GaugeCalls++

	return m.GaugeFunc(name, opts...)
}

func (m *MockMetrics) Histogram(name string, opts ...MetricOption) Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HistogramCalls++

	return m.HistogramFunc(name, opts...)
}

func (m *MockMetrics) Timer(name string, opts ...MetricOption) Timer {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TimerCalls++

	return m.TimerFunc(name, opts...)
}

// MetricExporter interface implementation

func (m *MockMetrics) Export(format ExportFormat) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExportCalls++

	return m.ExportFunc(format)
}

func (m *MockMetrics) ExportToFile(format ExportFormat, filename string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExportToFileCalls++

	return m.ExportToFileFunc(format, filename)
}

// CollectorRegistry interface implementation

func (m *MockMetrics) RegisterCollector(collector CustomCollector) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.RegisterCollectorFunc(collector)
}

func (m *MockMetrics) UnregisterCollector(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.UnregisterCollectorFunc(name)
}

func (m *MockMetrics) ListCollectors() []CustomCollector {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ListCollectorsFunc()
}

// MetricRepository interface implementation

func (m *MockMetrics) ListMetrics() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ListMetricsFunc()
}

func (m *MockMetrics) ListMetricsByType(metricType MetricType) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ListMetricsByTypeFunc(metricType)
}

func (m *MockMetrics) ListMetricsByTag(tagKey, tagValue string) map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ListMetricsByTagFunc(tagKey, tagValue)
}

func (m *MockMetrics) Stats() CollectorStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.StatsFunc()
}

// MetricManager interface implementation

func (m *MockMetrics) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ResetCalls++

	return m.ResetFunc()
}

func (m *MockMetrics) ResetMetric(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ResetMetricFunc(name)
}

func (m *MockMetrics) Reload(config *MetricsConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReloadCalls++

	return m.ReloadFunc(config)
}

// =============================================================================
// MOCK METRIC TYPES
// =============================================================================

// MockCounter is a mock implementation of Counter.
type MockCounter struct {
	mu    sync.RWMutex
	value float64
}

func NewMockCounter() *MockCounter {
	return &MockCounter{}
}

func (c *MockCounter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value++
}

func (c *MockCounter) Add(delta float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value += delta
}

func (c *MockCounter) Value() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.value
}

func (c *MockCounter) WithLabels(labels map[string]string) Counter {
	return NewMockCounter()
}

func (c *MockCounter) Reset() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.value = 0

	return nil
}

// MockGauge is a mock implementation of Gauge.
type MockGauge struct {
	mu    sync.RWMutex
	value float64
}

func NewMockGauge() *MockGauge {
	return &MockGauge{}
}

func (g *MockGauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value = value
}

func (g *MockGauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value++
}

func (g *MockGauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value--
}

func (g *MockGauge) Add(delta float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value += delta
}

func (g *MockGauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.value
}

func (g *MockGauge) WithLabels(labels map[string]string) Gauge {
	return NewMockGauge()
}

func (g *MockGauge) Reset() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.value = 0

	return nil
}

// MockHistogram is a mock implementation of Histogram.
type MockHistogram struct {
	mu     sync.RWMutex
	values []float64
}

func NewMockHistogram() *MockHistogram {
	return &MockHistogram{
		values: make([]float64, 0),
	}
}

func (h *MockHistogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.values = append(h.values, value)
}

func (h *MockHistogram) WithLabels(labels map[string]string) Histogram {
	return NewMockHistogram()
}

func (h *MockHistogram) Reset() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.values = make([]float64, 0)

	return nil
}

// MockTimer is a mock implementation of Timer.
type MockTimer struct {
	mu        sync.RWMutex
	durations []time.Duration
}

func NewMockTimer() *MockTimer {
	return &MockTimer{
		durations: make([]time.Duration, 0),
	}
}

func (t *MockTimer) Record(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.durations = append(t.durations, duration)
}

func (t *MockTimer) Time() func() {
	start := time.Now()

	return func() {
		t.Record(time.Since(start))
	}
}

func (t *MockTimer) Count() uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return uint64(len(t.durations))
}

func (t *MockTimer) Mean() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.durations) == 0 {
		return 0
	}

	var sum time.Duration
	for _, d := range t.durations {
		sum += d
	}

	return sum / time.Duration(len(t.durations))
}

func (t *MockTimer) Percentile(percentile float64) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.durations) == 0 {
		return 0
	}

	// Simple implementation - not production-quality percentile calculation
	return t.durations[len(t.durations)-1]
}

func (t *MockTimer) Min() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.durations) == 0 {
		return 0
	}

	minDuration := t.durations[0]
	for _, d := range t.durations {
		if d < minDuration {
			minDuration = d
		}
	}

	return minDuration
}

func (t *MockTimer) Max() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.durations) == 0 {
		return 0
	}

	maxDuration := t.durations[0]
	for _, d := range t.durations {
		if d > maxDuration {
			maxDuration = d
		}
	}

	return maxDuration
}

func (t *MockTimer) WithLabels(labels map[string]string) Timer {
	return NewMockTimer()
}

func (t *MockTimer) Reset() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.durations = make([]time.Duration, 0)

	return nil
}

// =============================================================================
// MOCK HEALTH MANAGER
// =============================================================================

// MockHealthManager is a mock implementation of the HealthManager interface.
// Thread-safe with call tracking for verification in tests.
type MockHealthManager struct {
	mu sync.RWMutex

	// HealthService interface
	NameFunc   func() string
	StartFunc  func(ctx context.Context) error
	StopFunc   func(ctx context.Context) error
	HealthFunc func(ctx context.Context) error

	// HealthChecker interface
	CheckFunc    func(ctx context.Context) *HealthReport
	CheckOneFunc func(ctx context.Context, name string) *HealthResult
	StatusFunc   func() HealthStatus

	// HealthCheckRegistry interface
	RegisterFunc   func(check HealthCheck) error
	RegisterFnFunc func(name string, check HealthCheckFn) error
	UnregisterFunc func(name string) error
	ListChecksFunc func() map[string]HealthCheck

	// HealthReporter interface
	LastReportFunc func() *HealthReport
	StatsFunc      func() *HealthCheckerStats

	// HealthMetadata interface
	SetEnvironmentFunc func(name string)
	SetVersionFunc     func(version string)
	SetHostnameFunc    func(hostname string)
	EnvironmentFunc    func() string
	HostnameFunc       func() string
	VersionFunc        func() string
	StartTimeFunc      func() time.Time

	// HealthSubscriber interface
	SubscribeFunc func(callback HealthCallback) error

	// HealthConfigurable interface
	ReloadFunc func(config *HealthConfig) error

	// Call tracking
	NameCalls           int
	StartCalls          int
	StopCalls           int
	HealthCalls         int
	CheckCalls          int
	CheckOneCalls       int
	StatusCalls         int
	RegisterCalls       int
	RegisterFnCalls     int
	UnregisterCalls     int
	SetEnvironmentCalls int
	SetVersionCalls     int
	SetHostnameCalls    int
	SubscribeCalls      int
	ReloadCalls         int

	// State
	environment string
	version     string
	hostname    string
	startTime   time.Time
	status      HealthStatus
	lastReport  *HealthReport
	checks      map[string]HealthCheck
}

// NewMockHealthManager creates a new mock health manager with sensible defaults.
func NewMockHealthManager() *MockHealthManager {
	m := &MockHealthManager{
		startTime:   time.Now(),
		status:      HealthStatusHealthy,
		checks:      make(map[string]HealthCheck),
		environment: "test",
		version:     "1.0.0",
		hostname:    "localhost",
	}

	// Default implementations
	m.NameFunc = func() string { return "mock-health-manager" }
	m.StartFunc = func(ctx context.Context) error { return nil }
	m.StopFunc = func(ctx context.Context) error { return nil }
	m.HealthFunc = func(ctx context.Context) error { return nil }

	m.CheckFunc = func(ctx context.Context) *HealthReport {
		report := NewHealthReport()
		report.Overall = m.status
		report.Version = m.version
		report.Environment = m.environment
		report.Hostname = m.hostname
		report.Uptime = time.Since(m.startTime)
		m.lastReport = report

		return report
	}

	m.CheckOneFunc = func(ctx context.Context, name string) *HealthResult {
		return NewHealthResult(name, HealthStatusHealthy, "OK")
	}

	m.StatusFunc = func() HealthStatus {
		return m.status
	}

	m.RegisterFunc = func(check HealthCheck) error {
		m.checks[check.Name()] = check

		return nil
	}

	m.RegisterFnFunc = func(name string, check HealthCheckFn) error {
		return nil
	}

	m.UnregisterFunc = func(name string) error {
		delete(m.checks, name)

		return nil
	}

	m.ListChecksFunc = func() map[string]HealthCheck {
		return m.checks
	}

	m.LastReportFunc = func() *HealthReport {
		return m.lastReport
	}

	m.StatsFunc = func() *HealthCheckerStats {
		return &HealthCheckerStats{
			RegisteredChecks: len(m.checks),
			Started:          true,
			Uptime:           time.Since(m.startTime),
			OverallStatus:    m.status,
		}
	}

	m.SetEnvironmentFunc = func(name string) {
		m.environment = name
	}

	m.SetVersionFunc = func(version string) {
		m.version = version
	}

	m.SetHostnameFunc = func(hostname string) {
		m.hostname = hostname
	}

	m.EnvironmentFunc = func() string {
		return m.environment
	}

	m.HostnameFunc = func() string {
		return m.hostname
	}

	m.VersionFunc = func() string {
		return m.version
	}

	m.StartTimeFunc = func() time.Time {
		return m.startTime
	}

	m.SubscribeFunc = func(callback HealthCallback) error {
		return nil
	}

	m.ReloadFunc = func(config *HealthConfig) error {
		return nil
	}

	return m
}

// HealthService interface implementation

func (m *MockHealthManager) Name() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.NameCalls++

	return m.NameFunc()
}

func (m *MockHealthManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StartCalls++

	return m.StartFunc(ctx)
}

func (m *MockHealthManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StopCalls++

	return m.StopFunc(ctx)
}

func (m *MockHealthManager) Health(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HealthCalls++

	return m.HealthFunc(ctx)
}

// HealthChecker interface implementation

func (m *MockHealthManager) Check(ctx context.Context) *HealthReport {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CheckCalls++

	return m.CheckFunc(ctx)
}

func (m *MockHealthManager) CheckOne(ctx context.Context, name string) *HealthResult {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CheckOneCalls++

	return m.CheckOneFunc(ctx, name)
}

func (m *MockHealthManager) Status() HealthStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StatusCalls++

	return m.StatusFunc()
}

// HealthCheckRegistry interface implementation

func (m *MockHealthManager) Register(check HealthCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterCalls++

	return m.RegisterFunc(check)
}

func (m *MockHealthManager) RegisterFn(name string, check HealthCheckFn) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RegisterFnCalls++

	return m.RegisterFnFunc(name, check)
}

func (m *MockHealthManager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UnregisterCalls++

	return m.UnregisterFunc(name)
}

func (m *MockHealthManager) ListChecks() map[string]HealthCheck {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ListChecksFunc()
}

// HealthReporter interface implementation

func (m *MockHealthManager) LastReport() *HealthReport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.LastReportFunc()
}

func (m *MockHealthManager) Stats() *HealthCheckerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.StatsFunc()
}

// HealthMetadata interface implementation

func (m *MockHealthManager) SetEnvironment(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SetEnvironmentCalls++
	m.SetEnvironmentFunc(name)
}

func (m *MockHealthManager) SetVersion(version string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SetVersionCalls++
	m.SetVersionFunc(version)
}

func (m *MockHealthManager) SetHostname(hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SetHostnameCalls++
	m.SetHostnameFunc(hostname)
}

func (m *MockHealthManager) Environment() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.EnvironmentFunc()
}

func (m *MockHealthManager) Hostname() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.HostnameFunc()
}

func (m *MockHealthManager) Version() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.VersionFunc()
}

func (m *MockHealthManager) StartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.StartTimeFunc()
}

// HealthSubscriber interface implementation

func (m *MockHealthManager) Subscribe(callback HealthCallback) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SubscribeCalls++

	return m.SubscribeFunc(callback)
}

// HealthConfigurable interface implementation

func (m *MockHealthManager) Reload(config *HealthConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReloadCalls++

	return m.ReloadFunc(config)
}

// =============================================================================
// MOCK HEALTH CHECK
// =============================================================================

// MockHealthCheck is a mock implementation of HealthCheck.
type MockHealthCheck struct {
	name         string
	timeout      time.Duration
	critical     bool
	dependencies []string
	CheckFunc    func(ctx context.Context) *HealthResult
}

// NewMockHealthCheck creates a new mock health check.
func NewMockHealthCheck(name string) *MockHealthCheck {
	return &MockHealthCheck{
		name:         name,
		timeout:      5 * time.Second,
		critical:     false,
		dependencies: []string{},
		CheckFunc: func(ctx context.Context) *HealthResult {
			return NewHealthResult(name, HealthStatusHealthy, "OK")
		},
	}
}

func (m *MockHealthCheck) Name() string {
	return m.name
}

func (m *MockHealthCheck) Check(ctx context.Context) *HealthResult {
	return m.CheckFunc(ctx)
}

func (m *MockHealthCheck) Timeout() time.Duration {
	return m.timeout
}

func (m *MockHealthCheck) Critical() bool {
	return m.critical
}

func (m *MockHealthCheck) Dependencies() []string {
	return m.dependencies
}

// WithTimeout sets the timeout for the mock health check.
func (m *MockHealthCheck) WithTimeout(timeout time.Duration) *MockHealthCheck {
	m.timeout = timeout

	return m
}

// WithCritical marks the health check as critical.
func (m *MockHealthCheck) WithCritical(critical bool) *MockHealthCheck {
	m.critical = critical

	return m
}

// WithDependencies sets the dependencies for the health check.
func (m *MockHealthCheck) WithDependencies(deps ...string) *MockHealthCheck {
	m.dependencies = deps

	return m
}
