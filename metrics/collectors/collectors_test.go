package collectors

import (
	"context"
	"errors"
	"maps"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/go-utils/metrics"
)

// =============================================================================
// MOCK METRIC SOURCE
// =============================================================================

type mockMetricSource struct {
	name      string
	data      *MetricSnapshot
	err       error
	callCount atomic.Int32
	mu        sync.RWMutex
}

func newMockMetricSource(name string) *mockMetricSource {
	return &mockMetricSource{
		name: name,
		data: &MetricSnapshot{
			Counters:   make(map[string]float64),
			Gauges:     make(map[string]float64),
			Histograms: make(map[string][]float64),
			Summaries:  make(map[string][]float64),
			Timers:     make(map[string][]time.Duration),
			Timestamp:  time.Now(),
		},
	}
}

func (m *mockMetricSource) Name() string {
	return m.name
}

func (m *mockMetricSource) Collect(ctx context.Context) (*MetricSnapshot, error) {
	m.callCount.Add(1)

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.err != nil {
		return nil, m.err
	}

	// Return a copy to avoid race conditions
	snapshot := &MetricSnapshot{
		Counters:   make(map[string]float64),
		Gauges:     make(map[string]float64),
		Histograms: make(map[string][]float64),
		Summaries:  make(map[string][]float64),
		Timers:     make(map[string][]time.Duration),
		Timestamp:  time.Now(),
	}

	maps.Copy(snapshot.Counters, m.data.Counters)

	maps.Copy(snapshot.Gauges, m.data.Gauges)

	for k, v := range m.data.Histograms {
		snapshot.Histograms[k] = append([]float64{}, v...)
	}

	for k, v := range m.data.Summaries {
		snapshot.Summaries[k] = append([]float64{}, v...)
	}

	for k, v := range m.data.Timers {
		snapshot.Timers[k] = append([]time.Duration{}, v...)
	}

	return snapshot, nil
}

func (m *mockMetricSource) SetData(data *MetricSnapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = data
}

func (m *mockMetricSource) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.err = err
}

func (m *mockMetricSource) GetCallCount() int32 {
	return m.callCount.Load()
}

// =============================================================================
// TESTS: MetricSnapshot
// =============================================================================

func TestMetricSnapshot_Validate(t *testing.T) {
	t.Run("Valid snapshot", func(t *testing.T) {
		snapshot := &MetricSnapshot{
			Counters: map[string]float64{"test": 1.0},
		}
		assert.NoError(t, snapshot.Validate())
	})

	t.Run("Nil snapshot", func(t *testing.T) {
		var snapshot *MetricSnapshot
		assert.ErrorIs(t, snapshot.Validate(), ErrNilSnapshot)
	})

	t.Run("Empty snapshot is valid", func(t *testing.T) {
		snapshot := &MetricSnapshot{}
		assert.NoError(t, snapshot.Validate())
	})
}

// =============================================================================
// TESTS: CustomCollectorBuilder - Basic Operations
// =============================================================================

func TestCustomCollectorBuilder_Creation(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	assert.NotNil(t, builder)
	assert.Equal(t, 10*time.Second, builder.interval)
	assert.NotNil(t, builder.metrics)
	assert.NotNil(t, builder.counters)
	assert.NotNil(t, builder.gauges)
	assert.NotNil(t, builder.histograms)
	assert.NotNil(t, builder.summaries)
	assert.NotNil(t, builder.timers)
}

func TestCustomCollectorBuilder_WithInterval(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source).
		WithInterval(5 * time.Second)

	assert.Equal(t, 5*time.Second, builder.interval)
}

func TestCustomCollectorBuilder_WithOptions(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source).
		WithOptions(
			metrics.WithNamespace("test"),
			metrics.WithSubsystem("collector"),
		)

	assert.Len(t, builder.options, 2)
}

func TestCustomCollectorBuilder_StartStop(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source).
		WithInterval(100 * time.Millisecond)

	// Start
	err := builder.Start()
	require.NoError(t, err)
	assert.True(t, builder.started.Load())

	// Try to start again
	err = builder.Start()
	assert.ErrorIs(t, err, ErrAlreadyStarted)

	// Stop
	err = builder.Stop()
	require.NoError(t, err)
	assert.False(t, builder.started.Load())

	// Try to stop again
	err = builder.Stop()
	assert.ErrorIs(t, err, ErrNotStarted)
}

func TestCustomCollectorBuilder_Metrics(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	m := builder.Metrics()
	assert.NotNil(t, m)
	assert.Equal(t, "test", m.Name())
}

// =============================================================================
// TESTS: CustomCollectorBuilder - Collection
// =============================================================================

func TestCustomCollectorBuilder_CollectOnce(t *testing.T) {
	source := newMockMetricSource("test")
	source.data.Counters["requests_total"] = 100
	source.data.Gauges["temperature"] = 25.5

	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	// Verify metrics were created and updated
	assert.Len(t, builder.counters, 1)
	assert.Len(t, builder.gauges, 1)

	counter := builder.counters["requests_total"]
	assert.NotNil(t, counter)
	assert.Equal(t, 100.0, counter.Value())

	gauge := builder.gauges["temperature"]
	assert.NotNil(t, gauge)
	assert.Equal(t, 25.5, gauge.Value())
}

func TestCustomCollectorBuilder_CounterDelta(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// First collection: counter = 100
	source.data.Counters["requests_total"] = 100
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	counter := builder.counters["requests_total"]
	assert.Equal(t, 100.0, counter.Value())

	// Second collection: counter = 150 (delta = 50)
	source.data.Counters["requests_total"] = 150
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 150.0, counter.Value())

	// Third collection: counter = 200 (delta = 50)
	source.data.Counters["requests_total"] = 200
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 200.0, counter.Value())
}

func TestCustomCollectorBuilder_CounterReset(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// First collection: counter = 100
	source.data.Counters["requests_total"] = 100
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	counter := builder.counters["requests_total"]
	assert.Equal(t, 100.0, counter.Value())

	// Counter reset detected: counter = 10 (less than previous)
	// Should treat 10 as the delta
	source.data.Counters["requests_total"] = 10
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 110.0, counter.Value())
}

func TestCustomCollectorBuilder_GaugeAbsolute(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Set gauge to 100
	source.data.Gauges["memory_used"] = 100
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	gauge := builder.gauges["memory_used"]
	assert.Equal(t, 100.0, gauge.Value())

	// Set gauge to 50 (can decrease)
	source.data.Gauges["memory_used"] = 50
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 50.0, gauge.Value())

	// Set gauge to 150 (can increase)
	source.data.Gauges["memory_used"] = 150
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, 150.0, gauge.Value())
}

func TestCustomCollectorBuilder_HistogramObservations(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Observe values
	source.data.Histograms["response_time"] = []float64{10, 20, 30}
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	histogram := builder.histograms["response_time"]
	assert.NotNil(t, histogram)
	assert.Equal(t, uint64(3), histogram.Count())
	assert.Equal(t, 60.0, histogram.Sum())

	// More observations
	source.data.Histograms["response_time"] = []float64{40, 50}
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Equal(t, uint64(5), histogram.Count())
	assert.Equal(t, 150.0, histogram.Sum())
}

func TestCustomCollectorBuilder_SummaryObservations(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Observe values
	source.data.Summaries["request_duration"] = []float64{1, 2, 3, 4, 5}
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	summary := builder.summaries["request_duration"]
	assert.NotNil(t, summary)
	assert.Equal(t, uint64(5), summary.Count())
	assert.Equal(t, 15.0, summary.Sum())
}

func TestCustomCollectorBuilder_TimerObservations(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Record durations
	source.data.Timers["processing_time"] = []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	timer := builder.timers["processing_time"]
	assert.NotNil(t, timer)
	assert.Equal(t, uint64(3), timer.Count())
}

func TestCustomCollectorBuilder_ErrorHandling(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Set error
	testErr := errors.New("collection failed")
	source.SetError(testErr)

	err := builder.CollectOnce(ctx)
	assert.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestCustomCollectorBuilder_PeriodicCollection(t *testing.T) {
	source := newMockMetricSource("test")
	source.data.Counters["requests_total"] = 0

	builder := NewCustomCollectorBuilder(source).
		WithInterval(50 * time.Millisecond)

	err := builder.Start()
	require.NoError(t, err)

	defer builder.Stop()

	// Wait for a few collections
	time.Sleep(150 * time.Millisecond)

	// Should have collected at least 2 times (initial + 2 intervals)
	callCount := source.GetCallCount()
	assert.GreaterOrEqual(t, callCount, int32(2))
}

// =============================================================================
// TESTS: PushableCollectorBuilder
// =============================================================================

func TestPushableCollectorBuilder_Creation(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source)

	assert.NotNil(t, builder)
	assert.NotNil(t, builder.pushChan)
	assert.Equal(t, 100, builder.bufferSize)
}

func TestPushableCollectorBuilder_WithBufferSize(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source).
		WithBufferSize(200)

	assert.Equal(t, 200, builder.bufferSize)
	assert.Equal(t, 200, cap(builder.pushChan))
}

func TestPushableCollectorBuilder_Push(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source).
		WithInterval(1 * time.Second) // Long interval to isolate push

	err := builder.Start()
	require.NoError(t, err)

	defer builder.Stop()

	// Push a snapshot
	snapshot := &MetricSnapshot{
		Counters: map[string]float64{"pushed_requests": 50},
		Gauges:   map[string]float64{"pushed_gauge": 75.5},
	}

	err = builder.Push(snapshot)
	require.NoError(t, err)

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	// Stop the collector before accessing internal state
	builder.Stop()

	// Verify metrics were updated
	counter := builder.counters["pushed_requests"]
	assert.NotNil(t, counter)
	assert.Equal(t, 50.0, counter.Value())

	gauge := builder.gauges["pushed_gauge"]
	assert.NotNil(t, gauge)
	assert.Equal(t, 75.5, gauge.Value())
}

func TestPushableCollectorBuilder_PushBeforeStart(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source)

	snapshot := &MetricSnapshot{
		Counters: map[string]float64{"test": 1},
	}

	err := builder.Push(snapshot)
	assert.ErrorIs(t, err, ErrNotStarted)
}

func TestPushableCollectorBuilder_PushBufferFull(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source).
		WithBufferSize(2) // Small buffer

	err := builder.Start()
	require.NoError(t, err)

	defer builder.Stop()

	snapshot := &MetricSnapshot{
		Counters: map[string]float64{"test": 1},
	}

	// Fill the buffer
	err = builder.Push(snapshot)
	assert.NoError(t, err)
	err = builder.Push(snapshot)
	assert.NoError(t, err)

	// Next push should fail
	err = builder.Push(snapshot)
	assert.ErrorIs(t, err, ErrPushBufferFull)
}

func TestPushableCollectorBuilder_HybridMode(t *testing.T) {
	source := newMockMetricSource("test")
	source.data.Counters["pull_counter"] = 0

	builder := NewPushableCollectorBuilder(source).
		WithInterval(100 * time.Millisecond)

	err := builder.Start()
	require.NoError(t, err)

	defer builder.Stop()

	// Push some metrics
	pushSnapshot := &MetricSnapshot{
		Counters: map[string]float64{"push_counter": 100},
	}
	err = builder.Push(pushSnapshot)
	require.NoError(t, err)

	// Update pull metrics
	source.SetData(&MetricSnapshot{
		Counters: map[string]float64{
			"pull_counter": 50,
		},
	})

	// Wait for at least one pull cycle
	time.Sleep(150 * time.Millisecond)

	// Stop the collector before accessing internal state
	builder.Stop()

	// Both push and pull metrics should exist
	assert.NotNil(t, builder.counters["push_counter"])
	assert.NotNil(t, builder.counters["pull_counter"])

	// Verify values
	pushCounter := builder.counters["push_counter"]
	assert.Equal(t, 100.0, pushCounter.Value())

	pullCounter := builder.counters["pull_counter"]
	assert.GreaterOrEqual(t, pullCounter.Value(), 50.0)
}

// =============================================================================
// TESTS: Concurrent Access
// =============================================================================

func TestCustomCollectorBuilder_ConcurrentCollections(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	var wg sync.WaitGroup

	// Concurrent collections
	numGoroutines := 10
	wg.Add(numGoroutines)

	for range numGoroutines {
		go func() {
			defer wg.Done()

			for range 10 {
				// Use SetData which has proper locking
				snapshot := &MetricSnapshot{
					Counters: map[string]float64{
						"concurrent_test": float64(time.Now().UnixNano()),
					},
				}
				source.SetData(snapshot)

				_ = builder.CollectOnce(ctx)
			}
		}()
	}

	wg.Wait()

	// Should not panic and should have created the metric
	assert.NotNil(t, builder.counters["concurrent_test"])
}

func TestPushableCollectorBuilder_ConcurrentPushes(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewPushableCollectorBuilder(source).
		WithBufferSize(1000)

	err := builder.Start()
	require.NoError(t, err)

	defer builder.Stop()

	var wg sync.WaitGroup

	numGoroutines := 10
	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for range 10 {
				snapshot := &MetricSnapshot{
					Counters: map[string]float64{"concurrent_push": float64(id)},
				}
				_ = builder.Push(snapshot)
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Stop the collector before accessing internal state
	builder.Stop()

	// Should not panic and should have created the metric
	assert.NotNil(t, builder.counters["concurrent_push"])
}

// =============================================================================
// TESTS: Lazy Metric Creation
// =============================================================================

func TestCustomCollectorBuilder_LazyMetricCreation(t *testing.T) {
	source := newMockMetricSource("test")
	builder := NewCustomCollectorBuilder(source)

	ctx := context.Background()

	// Initially no metrics
	assert.Empty(t, builder.counters)
	assert.Empty(t, builder.gauges)

	// First collection creates metrics
	source.data.Counters["new_counter"] = 100
	err := builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Len(t, builder.counters, 1)

	// Add new metric dynamically
	source.data.Gauges["new_gauge"] = 50
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Len(t, builder.counters, 1)
	assert.Len(t, builder.gauges, 1)

	// Add multiple new metrics
	source.data.Histograms["new_histogram"] = []float64{1, 2, 3}
	source.data.Summaries["new_summary"] = []float64{4, 5, 6}
	source.data.Timers["new_timer"] = []time.Duration{100 * time.Millisecond}
	err = builder.CollectOnce(ctx)
	require.NoError(t, err)

	assert.Len(t, builder.counters, 1)
	assert.Len(t, builder.gauges, 1)
	assert.Len(t, builder.histograms, 1)
	assert.Len(t, builder.summaries, 1)
	assert.Len(t, builder.timers, 1)
}

// =============================================================================
// TESTS: Errors
// =============================================================================

func TestCollectorError_Error(t *testing.T) {
	err := &CollectorError{Message: "test error"}
	assert.Equal(t, "test error", err.Error())
}
