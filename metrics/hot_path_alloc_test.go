package metrics

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inc, Dec and Observe run on every request in forge's HTTP server metrics,
// so none of them may allocate.
func TestHotPathDoesNotAllocate(t *testing.T) {
	counter := NewCounter("c")
	gauge := NewGauge("g")
	hist := NewHistogram("h")
	ex := Exemplar{TraceID: "0af7651916cd43dd8448eb211c80319c", SpanID: "b7ad6b7169203331"}

	for name, f := range map[string]func(){
		"Counter.Inc":                   counter.Inc,
		"Counter.AddWithExemplar":       func() { counter.AddWithExemplar(1, ex) },
		"Gauge.Inc":                     gauge.Inc,
		"Gauge.Dec":                     gauge.Dec,
		"Gauge.Set":                     func() { gauge.Set(3) },
		"Histogram.Observe":             func() { hist.Observe(0.2) },
		"Histogram.ObserveWithExemplar": func() { hist.ObserveWithExemplar(0.2, ex) },
	} {
		if allocs := testing.AllocsPerRun(100, f); allocs != 0 {
			t.Errorf("%s allocated %.0f times, want 0", name, allocs)
		}
	}
}

func TestTimestampTracksLastUpdate(t *testing.T) {
	gauge := NewGauge("ts")
	created := gauge.Timestamp()

	time.Sleep(2 * time.Millisecond)
	gauge.Inc()

	assert.True(t, gauge.Timestamp().After(created), "Inc did not move the timestamp")
}

// The store keeps the newest exemplarBufferSize exemplars, whole.
func TestExemplarStoreKeepsNewest(t *testing.T) {
	es := newExemplarStore()

	for i := range exemplarBufferSize + 3 {
		es.Add(Exemplar{Value: float64(i), TraceID: "t", SpanID: "s"})
	}

	got := es.GetAll()
	require.Len(t, got, exemplarBufferSize)

	seen := map[float64]bool{}
	for _, ex := range got {
		seen[ex.Value] = true

		assert.Equal(t, "t", ex.TraceID)
		assert.Equal(t, "s", ex.SpanID)
	}

	for i := 3; i < exemplarBufferSize+3; i++ {
		assert.True(t, seen[float64(i)], "exemplar %d missing", i)
	}
}

// Readers never see a torn exemplar while writers race on the same slots.
func TestExemplarStoreConcurrent(t *testing.T) {
	es := newExemplarStore()

	var wg sync.WaitGroup

	for w := range 8 {
		wg.Go(func() {
			id := string(rune('a' + w))
			for range 1000 {
				es.Add(Exemplar{TraceID: id, SpanID: id})
			}
		})
	}

	wg.Go(func() {
		for range 1000 {
			for _, ex := range es.GetAll() {
				if ex.TraceID != ex.SpanID {
					t.Errorf("torn exemplar: trace %q span %q", ex.TraceID, ex.SpanID)

					return
				}
			}
		}
	})

	wg.Wait()
}
