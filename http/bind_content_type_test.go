package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contentTypeProbe is bound from whichever encoding the test sends.
type contentTypeProbe struct {
	Name string `json:"name" xml:"name"`
}

func bindWithContentType(t *testing.T, body, contentType string) (contentTypeProbe, error) {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/x", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)

	var bound contentTypeProbe

	err := NewContext(httptest.NewRecorder(), req, nil).(*Ctx).Bind(&bound)

	return bound, err
}

// ── Content-Type parameters ────────────

// A charset parameter is routine, and must not push the request into the
// unsupported-type branch.
func TestBind_JSONWithCharsetParameter(t *testing.T) {
	bound, err := bindWithContentType(t, `{"name":"ada"}`, "application/json; charset=utf-8")

	require.NoError(t, err)
	assert.Equal(t, "ada", bound.Name)
}

// The same header without the conventional space after the semicolon.
func TestBind_JSONWithUnspacedCharsetParameter(t *testing.T) {
	bound, err := bindWithContentType(t, `{"name":"ada"}`, "application/json;charset=UTF-8")

	require.NoError(t, err)
	assert.Equal(t, "ada", bound.Name)
}

func TestBind_XMLWithCharsetParameter(t *testing.T) {
	bound, err := bindWithContentType(t, `<probe><name>ada</name></probe>`, "application/xml;charset=UTF-8")

	require.NoError(t, err)
	assert.Equal(t, "ada", bound.Name)
}

func TestBind_MissingContentTypeStillDefaultsToJSON(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/x",
		strings.NewReader(`{"name":"ada"}`))

	var bound contentTypeProbe
	require.NoError(t, NewContext(httptest.NewRecorder(), req, nil).(*Ctx).Bind(&bound))

	assert.Equal(t, "ada", bound.Name)
}

// ── near misses ────────────

func TestBind_UnknownContentTypeIsRejected(t *testing.T) {
	_, err := bindWithContentType(t, "x", "application/octet-stream")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported content type")
}

// Matching the media type rather than a prefix keeps a longer, unrelated type
// from being treated as a form.
func TestBind_NearMissFormContentTypeIsRejected(t *testing.T) {
	_, err := bindWithContentType(t, "name=ada", "application/x-www-form-urlencoded-not-really")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported content type")
}

func TestBind_NearMissJSONContentTypeIsRejected(t *testing.T) {
	_, err := bindWithContentType(t, `{"name":"ada"}`, "application/json-lines")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported content type")
}
