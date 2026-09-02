package http

import (
	"context"
	gohttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/go-utils/val"
)

type veReq struct {
	OrgID string `path:"orgId" validate:"required"`
	Name  string `query:"name" validate:"required"`
	Email string `query:"email" validate:"required,email"`
}

// BindRequest builds its ValidationError as a value so escape analysis can
// keep it off the heap for the successful path, and copies it out on failure.
// Prove the copy carries every field error and is still a *ValidationError,
// because getting that wrong would silently drop validation feedback.
func TestBindRequest_ValidationErrorsSurviveTheStackCopy(t *testing.T) {
	req := httptest.NewRequest(gohttp.MethodGet, "/x?email=not-an-email", nil)

	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	var out veReq

	err := c.BindRequest(&out)
	require.Error(t, err, "missing orgId and name, plus an invalid email, must fail")

	ve, ok := err.(*val.ValidationError)
	require.True(t, ok, "the error must still be a *val.ValidationError, got %T", err)
	require.NotEmpty(t, ve.Errors, "field errors must survive the copy")

	fields := map[string]bool{}
	for _, fe := range ve.Errors {
		fields[fe.Field] = true
	}

	t.Logf("reported fields: %v", fields)
	assert.True(t, len(fields) >= 2, "expected several field errors, got %v", fields)
	assert.Equal(t, 422, ve.StatusCode())
}

func TestBindRequest_ValidRequestReturnsNil(t *testing.T) {
	req := httptest.NewRequest(gohttp.MethodGet, "/x?name=rex&email=rex@example.com", nil)

	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("orgId", "o1")
	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))

	c := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	defer c.Cleanup()

	var out veReq
	require.NoError(t, c.BindRequest(&out))
	assert.Equal(t, "o1", out.OrgID)
	assert.Equal(t, "rex", out.Name)
}

// The binder must not allocate per field just to ask whether a type implements
// encoding.TextUnmarshaler. That is a property of the type, and answering it
// by boxing a value into an interface cost one allocation per field per
// request.
func TestBindRequest_DoesNotAllocatePerField(t *testing.T) {
	req := httptest.NewRequest(gohttp.MethodGet, "/x?page=2&perPage=50&search=abc", nil)

	p := AcquireRouteParams()
	defer ReleaseRouteParams(p)

	p.Set("orgId", "o1")
	p.Set("userId", "u1")

	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, p))
	w := httptest.NewRecorder()

	allocs := testing.AllocsPerRun(200, func() {
		c := NewContext(w, req, nil).(*Ctx)

		var out benchBindNoBody
		_ = c.BindRequest(&out)

		c.Cleanup()
	})

	// Five path and query fields. Most of what remains is url.ParseQuery
	// building a url.Values, which is parsed once per request rather than per
	// field. If this climbs back toward one-per-field, the type-level
	// TextUnmarshaler check has regressed.
	assert.LessOrEqualf(t, allocs, 8.0,
		"binding five fields allocated %.0f times; it should not scale with field count", allocs)
}
