package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// QueryMultiRequest mirrors an OAuth2 authorization request: RFC 8707 defines
// `resource` as repeatable, so the field has to collect every occurrence
// rather than the first one.
type QueryMultiRequest struct {
	ClientID  string   `query:"client_id"`
	Resources []string `query:"resource,omitempty"`
	Scopes    []string `default:"openid,profile" query:"scope,omitempty"`
}

func getQuery(t *testing.T, target string) *Ctx {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)

	return NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
}

func TestBindRequest_RepeatedQueryParamFillsSlice(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t,
		"/authorize?client_id=abc&resource=https://a.example.com&resource=https://b.example.com").
		BindRequest(&req))

	assert.Equal(t, "abc", req.ClientID)
	assert.Equal(t, []string{"https://a.example.com", "https://b.example.com"}, req.Resources,
		"every occurrence of a repeated query parameter must reach the slice")
}

func TestBindRequest_SingleQueryParamStillFillsSlice(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t, "/authorize?client_id=abc&resource=https://a.example.com").
		BindRequest(&req))

	assert.Equal(t, []string{"https://a.example.com"}, req.Resources)
}

// A lone value keeps expanding on commas, the way scope=openid,profile always
// has. Only repeated parameters are taken verbatim.
func TestBindRequest_LoneCommaValueStillExpands(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t, "/authorize?client_id=abc&scope=openid,email").BindRequest(&req))

	assert.Equal(t, []string{"openid", "email"}, req.Scopes)
}

func TestBindRequest_RepeatedQueryParamIsTakenVerbatim(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t, "/authorize?client_id=abc&scope=openid&scope=a,b").BindRequest(&req))

	assert.Equal(t, []string{"openid", "a,b"}, req.Scopes,
		"a repeated parameter must not be split further, the same as the form path")
}

func TestBindRequest_AbsentRepeatedQueryParamLeavesSliceEmpty(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t, "/authorize?client_id=abc").BindRequest(&req))

	assert.Empty(t, req.Resources)
}

func TestBindRequest_AbsentQuerySliceTakesItsDefault(t *testing.T) {
	var req QueryMultiRequest
	require.NoError(t, getQuery(t, "/authorize?client_id=abc").BindRequest(&req))

	assert.Equal(t, []string{"openid", "profile"}, req.Scopes)
}

func TestBindRequest_RequiredQuerySliceReportsWhenAbsent(t *testing.T) {
	type required struct {
		Resources []string `query:"resource" required:"true"`
	}

	var req required
	err := getQuery(t, "/authorize").BindRequest(&req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resource")
}
