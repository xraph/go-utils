package http

import (
	"maps"
	"sync"
)

// routeParamsInline is the number of parameters stored without touching the
// heap. Routes with more than this are rare enough that the spill map is not
// worth optimizing.
const routeParamsInline = 8

// RouteParams carries path parameters from a router to a handler without
// allocating a map.
//
// A carrier is pooled and recycled when the handler returns. A handler that
// hands parameters to a goroutine outliving the request must call Clone
// first, exactly as it must not retain the request body.
type RouteParams struct {
	names  [routeParamsInline]string
	values [routeParamsInline]string
	n      int
	spill  map[string]string
}

// routeParamsKey is a distinct unexported type so it cannot collide with a
// string key, which is what the legacy "forge:params" contract used.
type routeParamsKey struct{}

// RouteParamsKey is the value routers pass to context.WithValue.
var RouteParamsKey = routeParamsKey{}

var routeParamsPool = sync.Pool{
	New: func() any { return &RouteParams{} },
}

// AcquireRouteParams takes a reset carrier from the pool.
func AcquireRouteParams() *RouteParams {
	return routeParamsPool.Get().(*RouteParams)
}

// ReleaseRouteParams resets a carrier and returns it to the pool.
//
// After this call the carrier must not be read. Anything that outlives the
// handler needs a Clone taken beforehand.
func ReleaseRouteParams(p *RouteParams) {
	if p == nil {
		return
	}

	p.Reset()
	routeParamsPool.Put(p)
}

// Set records a parameter. A repeated name overwrites the earlier value.
func (p *RouteParams) Set(name, value string) {
	if p == nil {
		return
	}

	for i := range p.n {
		if p.names[i] == name {
			p.values[i] = value

			return
		}
	}

	if p.spill != nil {
		if _, ok := p.spill[name]; ok {
			p.spill[name] = value

			return
		}
	}

	if p.n < routeParamsInline {
		p.names[p.n] = name
		p.values[p.n] = value
		p.n++

		return
	}

	if p.spill == nil {
		p.spill = make(map[string]string, 4)
	}

	p.spill[name] = value
}

// Get returns a parameter value. The bool reports whether it was present, so
// an empty value is distinguishable from a missing one.
func (p *RouteParams) Get(name string) (string, bool) {
	if p == nil {
		return "", false
	}

	for i := range p.n {
		if p.names[i] == name {
			return p.values[i], true
		}
	}

	if p.spill != nil {
		v, ok := p.spill[name]

		return v, ok
	}

	return "", false
}

// Len reports how many parameters are held.
func (p *RouteParams) Len() int {
	if p == nil {
		return 0
	}

	return p.n + len(p.spill)
}

// Clone copies the parameters into an owned map.
//
// This is the escape hatch for anything that outlives the handler call, and
// the only safe way to keep parameters past the point the carrier is pooled.
func (p *RouteParams) Clone() map[string]string {
	if p == nil || p.Len() == 0 {
		return map[string]string{}
	}

	out := make(map[string]string, p.Len())

	for i := range p.n {
		out[p.names[i]] = p.values[i]
	}

	maps.Copy(out, p.spill)

	return out
}

// Reset clears the carrier for reuse.
func (p *RouteParams) Reset() {
	if p == nil {
		return
	}

	for i := range p.n {
		p.names[i] = ""
		p.values[i] = ""
	}

	p.n = 0

	if p.spill != nil {
		clear(p.spill)
	}
}
