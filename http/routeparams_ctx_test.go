package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCtx_PrefersTheTypedCarrier(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("id", "42")

	req := httptest.NewRequest(nethttp.MethodGet, "/users/42", nil)
	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	assert.Equal(t, "42", c.Param("id"))

	n, err := c.ParamInt("id")
	require.NoError(t, err)
	assert.Equal(t, 42, n)
}

// Version skew in one direction: an old router still writing the map must
// keep working against a new go-utils.
func TestCtx_FallsBackToTheLegacyMapKey(t *testing.T) {
	req := httptest.NewRequest(nethttp.MethodGet, "/users/42", nil)
	req = req.WithContext(context.WithValue(req.Context(), "forge:params", map[string]string{"id": "42"})) //nolint:staticcheck // exercising the legacy contract

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	assert.Equal(t, "42", c.Param("id"))
}

// When both are present the typed carrier wins, so a router mid-migration
// that writes both does not serve stale values.
func TestCtx_TypedCarrierBeatsTheLegacyMap(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("id", "typed")

	req := httptest.NewRequest(nethttp.MethodGet, "/users/42", nil)
	ctx := context.WithValue(req.Context(), "forge:params", map[string]string{"id": "legacy"}) //nolint:staticcheck // exercising the legacy contract
	ctx = context.WithValue(ctx, RouteParamsKey, p)
	req = req.WithContext(ctx)

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	assert.Equal(t, "typed", c.Param("id"))
}

func TestCtx_ParamsMaterializesFromTheCarrier(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("id", "42")
	p.Set("slug", "hello")

	req := httptest.NewRequest(nethttp.MethodGet, "/x", nil)
	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	assert.Equal(t, map[string]string{"id": "42", "slug": "hello"}, c.Params())
}

func TestCtx_NoParamsAtAll(t *testing.T) {
	req := httptest.NewRequest(nethttp.MethodGet, "/x", nil)

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	assert.Equal(t, "", c.Param("id"))
	assert.Empty(t, c.Params())
}

// A pooled Ctx must not carry the previous request's carrier into the next
// one. Cleanup clears the field; this pins that.
func TestCtx_CleanupClearsTheCarrier(t *testing.T) {
	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("id", "42")

	req := httptest.NewRequest(nethttp.MethodGet, "/x", nil)
	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))

	first := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	require.Equal(t, "42", first.Param("id"))
	first.Cleanup()

	plain := httptest.NewRequest(nethttp.MethodGet, "/y", nil)

	second := NewContext(httptest.NewRecorder(), plain, nil).(*Ctx)
	defer second.Cleanup()

	assert.Equal(t, "", second.Param("id"), "a recycled Ctx must not inherit the previous carrier")
}
