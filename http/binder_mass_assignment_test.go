package http

import (
	"context"
	"encoding/xml"
	gohttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMassAssignCtx builds a POST context with the given route params, headers
// and body.
func newMassAssignCtx(t *testing.T, target, contentType, body string, params, headers map[string]string) *Ctx {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), gohttp.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	rp := AcquireRouteParams()

	t.Cleanup(func() { ReleaseRouteParams(rp) })

	for k, v := range params {
		rp.Set(k, v)
	}

	req = req.WithContext(context.WithValue(req.Context(), RouteParamsKey, rp))

	c, ok := NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
	require.True(t, ok)
	t.Cleanup(c.Cleanup)

	return c
}

type transferRequest struct {
	TenantID string `path:"tenantId"`
	UserID   string `header:"X-User-Id"`
	DryRun   bool   `query:"dryRun"`
	Amount   int    `json:"amount"`
}

// The verified exploit: encoding/json matches untagged field names
// case-insensitively, so a body naming TenantID, UserID and DryRun used to
// replace the values bound from the path, header and query string after a
// tenant check had already approved the path value.
func TestBindRequest_BodyCannotOverwriteParams(t *testing.T) {
	c := newMassAssignCtx(t,
		"/tenants/attacker-tenant/transfers?dryRun=true",
		"application/json",
		`{"TenantID":"victim-tenant","UserID":"admin","DryRun":false,"amount":5}`,
		map[string]string{"tenantId": "attacker-tenant"},
		map[string]string{"X-User-Id": "attacker"},
	)

	var req transferRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "attacker-tenant", req.TenantID, "path value must win over the body")
	assert.Equal(t, "attacker", req.UserID, "header value must win over the body")
	assert.True(t, req.DryRun, "query value must win over the body")
	assert.Equal(t, 5, req.Amount, "genuine body fields must still bind")
}

// Lower-case and mixed-case keys reach the same fields through encoding/json's
// case-insensitive matching.
func TestBindRequest_BodyCannotOverwriteParamsAnyCase(t *testing.T) {
	c := newMassAssignCtx(t,
		"/tenants/attacker-tenant/transfers?dryRun=true",
		"application/json",
		`{"tenantid":"victim-tenant","USERID":"admin","dryrun":false,"amount":5}`,
		map[string]string{"tenantId": "attacker-tenant"},
		map[string]string{"X-User-Id": "attacker"},
	)

	var req transferRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "attacker-tenant", req.TenantID)
	assert.Equal(t, "attacker", req.UserID)
	assert.True(t, req.DryRun)
}

type optionalParamRequest struct {
	Role    string  `optional:"true"  query:"role"`
	Actor   *string `header:"X-Actor"`
	Limit   int     `default:"10"     query:"limit"`
	Payload string  `json:"payload"`
}

// When the parameter is absent the body still must not supply it. Otherwise
// an attacker only has to omit the query parameter to choose its value.
func TestBindRequest_BodyCannotFillAbsentParams(t *testing.T) {
	c := newMassAssignCtx(t, "/x", "application/json",
		`{"Role":"admin","Actor":"root","Limit":100000,"payload":"ok"}`, nil, nil)

	var req optionalParamRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Empty(t, req.Role)
	assert.Nil(t, req.Actor)
	assert.Equal(t, 10, req.Limit, "the default applies, not the body")
	assert.Equal(t, "ok", req.Payload)
}

// A value the caller put in a parameter field before binding survives when the
// parameter is absent, and the body cannot replace it.
func TestBindRequest_PrepopulatedParamSurvivesBody(t *testing.T) {
	c := newMassAssignCtx(t, "/x", "application/json", `{"Role":"admin","payload":"ok"}`, nil, nil)

	req := optionalParamRequest{Role: "viewer"}
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "viewer", req.Role)
	assert.Equal(t, "ok", req.Payload)
}

type dualTaggedRequest struct {
	TenantID string `json:"tenantId" path:"tenantId"`
	Scope    string `json:"scope"    optional:"true" query:"scope"`
	Name     string `json:"name"`
}

// A field with both a json tag and a parameter tag binds from the parameter
// only. The json tag keeps naming it in responses and errors.
func TestBindRequest_ParamWinsOverJSONTag(t *testing.T) {
	c := newMassAssignCtx(t, "/tenants/t1", "application/json",
		`{"tenantId":"victim","scope":"admin","name":"rex"}`,
		map[string]string{"tenantId": "t1"}, nil)

	var req dualTaggedRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "t1", req.TenantID)
	assert.Empty(t, req.Scope, "an absent query parameter must not be filled from the body")
	assert.Equal(t, "rex", req.Name)
}

type referenceParamRequest struct {
	Tags  []string `optional:"true"  query:"tag"`
	Owner *string  `header:"X-Owner"`
	Note  string   `json:"note"`
}

// Slices and pointers bound from parameters must not be decoded into in place.
// encoding/json reuses an existing slice's backing array and writes through an
// existing pointer, so restoring the field header alone would not be enough.
func TestBindRequest_BodyCannotWriteThroughParamReferences(t *testing.T) {
	c := newMassAssignCtx(t, "/x?tag=a&tag=b", "application/json",
		`{"Tags":["evil"],"Owner":"evil","note":"n"}`,
		nil, map[string]string{"X-Owner": "rex"})

	var req referenceParamRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, []string{"a", "b"}, req.Tags)
	require.NotNil(t, req.Owner)
	assert.Equal(t, "rex", *req.Owner)
	assert.Equal(t, "n", req.Note)
}

type TenantScope struct {
	TenantID string `path:"tenantId"`
}

type ActorScope struct {
	UserID string `header:"X-User-Id"`
}

type embeddedParamRequest struct {
	TenantScope
	*ActorScope

	Amount int `json:"amount"`
}

// Parameters promoted from embedded structs, by value and by pointer, are
// protected the same way.
func TestBindRequest_BodyCannotOverwriteEmbeddedParams(t *testing.T) {
	c := newMassAssignCtx(t, "/tenants/t1", "application/json",
		`{"TenantID":"victim","UserID":"admin","amount":7}`,
		map[string]string{"tenantId": "t1"},
		map[string]string{"X-User-Id": "rex"})

	var req embeddedParamRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "t1", req.TenantID)
	require.NotNil(t, req.ActorScope)
	assert.Equal(t, "rex", req.UserID)
	assert.Equal(t, 7, req.Amount)
}

type hiddenScope struct {
	TenantID string `path:"tenantId"`
}

type unexportedEmbedRequest struct {
	hiddenScope

	Amount int `json:"amount"`
}

// encoding/json fills the exported fields of an unexported embedded struct.
// The binder does not bind parameters there, but the body must not be able to
// set them either.
func TestBindRequest_BodyCannotSetParamsInUnexportedEmbed(t *testing.T) {
	c := newMassAssignCtx(t, "/tenants/t1", "application/json",
		`{"TenantID":"victim","amount":7}`,
		map[string]string{"tenantId": "t1"}, nil)

	var req unexportedEmbedRequest
	require.NoError(t, c.BindRequest(&req))

	assert.NotEqual(t, "victim", req.TenantID)
	assert.Equal(t, 7, req.Amount)
}

type ExportedActor struct {
	UserID string `header:"X-User-Id"`
}

type pointerEmbeddedRequest struct {
	*ExportedActor

	Amount int `json:"amount"`
}

// A caller-supplied embedded pointer must not be written through by the
// decoder either.
func TestBindRequest_BodyCannotWriteThroughEmbeddedPointer(t *testing.T) {
	c := newMassAssignCtx(t, "/x", "application/json",
		`{"UserID":"admin","amount":7}`,
		nil, map[string]string{"X-User-Id": "rex"})

	actor := &ExportedActor{}
	req := pointerEmbeddedRequest{ExportedActor: actor}
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "rex", req.UserID)
	assert.Equal(t, "rex", actor.UserID, "the caller's embedded struct must not see the body value")
	assert.Equal(t, 7, req.Amount)
}

type xmlTransferRequest struct {
	XMLName  xml.Name `xml:"transfer"`
	TenantID string   `path:"tenantId"`
	Amount   int      `json:"amount"   xml:"amount"`
}

// XML decoding fills untagged fields by name too, so it goes through the same
// isolation.
func TestBindRequest_XMLBodyCannotOverwriteParams(t *testing.T) {
	c := newMassAssignCtx(t, "/tenants/t1", "application/xml",
		`<transfer><TenantID>victim</TenantID><amount>3</amount></transfer>`,
		map[string]string{"tenantId": "t1"}, nil)

	var req xmlTransferRequest
	require.NoError(t, c.BindRequest(&req))

	assert.Equal(t, "t1", req.TenantID)
	assert.Equal(t, 3, req.Amount)
}

// Validation still sees the parameter value rather than the body's.
func TestBindRequest_ValidationSeesParamValue(t *testing.T) {
	type req struct {
		Code string `pattern:"^[a-z]+$" query:"code"`
		Note string `json:"note"`
	}

	c := newMassAssignCtx(t, "/x?code=ABC", "application/json", `{"Code":"abc","note":"n"}`, nil, nil)

	var r req
	require.Error(t, c.BindRequest(&r), "the query value fails the pattern and the body must not rescue it")
	assert.Equal(t, "ABC", r.Code)
}
