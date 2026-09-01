package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteParams_GetAndLen(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("id", "42")
	p.Set("slug", "hello")

	assert.Equal(t, 2, p.Len())

	v, ok := p.Get("id")
	assert.True(t, ok)
	assert.Equal(t, "42", v)

	v, ok = p.Get("slug")
	assert.True(t, ok)
	assert.Equal(t, "hello", v)

	_, ok = p.Get("missing")
	assert.False(t, ok)
}

// More than the inline capacity must still work. It is rare enough not to
// optimize, but it must not silently drop parameters.
func TestRouteParams_SpillsBeyondInlineCapacity(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	for i, n := range names {
		p.Set(n, string(rune('0'+i)))
	}

	assert.Equal(t, len(names), p.Len())

	for i, n := range names {
		v, ok := p.Get(n)
		require.Truef(t, ok, "parameter %q was dropped past the inline capacity", n)
		assert.Equal(t, string(rune('0'+i)), v)
	}
}

// Overwriting must update in place in both storage regions. Getting this
// wrong grows the carrier on every repeated Set, which a router doing
// per-request Set calls would hit immediately.
func TestRouteParams_SpillOverwriteDoesNotDuplicate(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	for i := range 10 {
		p.Set(string(rune('a'+i)), "v")
	}

	require.Equal(t, 10, p.Len())

	p.Set("j", "changed") // lives in the spill map
	p.Set("a", "changed") // lives inline

	assert.Equal(t, 10, p.Len(), "overwriting must not grow the carrier")

	v, _ := p.Get("j")
	assert.Equal(t, "changed", v)

	v, _ = p.Get("a")
	assert.Equal(t, "changed", v)
}

func TestRouteParams_Clone(t *testing.T) {
	p := AcquireRouteParams()

	p.Set("id", "42")
	p.Set("slug", "hello")

	clone := p.Clone()

	// Releasing must not disturb the clone. This is the escape hatch for a
	// handler that hands parameters to a goroutine outliving the request.
	ReleaseRouteParams(p)

	assert.Equal(t, map[string]string{"id": "42", "slug": "hello"}, clone)
}

func TestRouteParams_ReleaseResets(t *testing.T) {
	first := AcquireRouteParams()
	first.Set("id", "42")
	ReleaseRouteParams(first)

	second := AcquireRouteParams()
	defer ReleaseRouteParams(second)

	assert.Equal(t, 0, second.Len(), "a pooled carrier must come back empty")

	_, ok := second.Get("id")
	assert.False(t, ok, "a recycled carrier must not leak the previous request's values")
}

func TestRouteParams_NilIsSafeToRead(t *testing.T) {
	var p *RouteParams

	assert.Equal(t, 0, p.Len())

	_, ok := p.Get("id")
	assert.False(t, ok)

	assert.Empty(t, p.Clone())
}

// The whole point of the carrier is to keep the map allocation off the hot
// path. If this regresses, the typed key has no reason to exist.
func TestRouteParams_AcquireSetGetDoesNotAllocate(t *testing.T) {
	// Warm the pool so the first Get does not count the pool's own New.
	ReleaseRouteParams(AcquireRouteParams())

	allocs := testing.AllocsPerRun(200, func() {
		p := AcquireRouteParams()
		p.Set("id", "42")
		p.Set("slug", "hello")

		if _, ok := p.Get("id"); !ok {
			t.Fatal("id missing")
		}

		ReleaseRouteParams(p)
	})

	assert.LessOrEqual(t, allocs, 0.0, "acquire, set, get and release must not allocate")
}
