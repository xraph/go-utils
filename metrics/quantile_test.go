package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHistogram_PercentileSingleObservation(t *testing.T) {
	h := NewHistogram("single")
	h.Observe(30)

	for _, q := range []float64{0, 0.5, 0.95, 0.99, 1} {
		assert.InDelta(t, 30.0, h.Percentile(q), 1e-9, "q=%v", q)
	}
}

func TestHistogram_PercentileInterpolatesInsideBucket(t *testing.T) {
	h := NewHistogram("interp", WithBuckets(0, 100))
	for i := 1; i <= 100; i++ {
		h.Observe(float64(i))
	}

	// Every observation lands in (0, 100], so the old answer was 100 for
	// every quantile. Interpolating across the bucket lands within one
	// observation of the true quantile.
	assert.InDelta(t, 50.0, h.Percentile(0.5), 1)
	assert.InDelta(t, 95.0, h.Percentile(0.95), 1)
	assert.InDelta(t, 99.0, h.Percentile(0.99), 1)
}

func TestHistogram_PercentileStaysInsideObservedRange(t *testing.T) {
	h := NewHistogram("range", WithBuckets(10, 1000))
	h.Observe(400)
	h.Observe(410)

	assert.InDelta(t, 400.0, h.Percentile(0), 1e-9)
	assert.InDelta(t, 410.0, h.Percentile(1), 1e-9)

	p50 := h.Percentile(0.5)
	assert.GreaterOrEqual(t, p50, 400.0)
	assert.LessOrEqual(t, p50, 410.0)
}

func TestHistogram_PercentileAboveLastBucket(t *testing.T) {
	h := NewHistogram("overflow", WithBuckets(10, 20))
	h.Observe(5)
	h.Observe(500)

	// The second observation sits in the +Inf bucket. Its upper edge is the
	// observed maximum, not the last finite bucket.
	assert.InDelta(t, 500.0, h.Percentile(1), 1e-9)
	assert.Greater(t, h.Percentile(0.99), 20.0)
}

func TestTimer_PercentileSingleObservation(t *testing.T) {
	timer := NewTimer("single")
	timer.Record(30 * time.Millisecond)

	for _, q := range []float64{0.5, 0.95, 0.99} {
		assert.Equal(t, 30*time.Millisecond, timer.Percentile(q), "q=%v", q)
	}
}

func TestTimer_DefaultsToDurationBuckets(t *testing.T) {
	timer := NewTimer("buckets")
	assert.Equal(t, DefaultDurationBuckets, timer.histogram.buckets)

	custom := NewTimer("custom", WithBuckets(1, 2, 3))
	assert.Equal(t, []float64{1, 2, 3}, custom.histogram.buckets)
}
