package http

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FormBindRequest mirrors the shape real OAuth2 endpoints use: every field
// carries both a json and a form tag so one handler serves either encoding.
type FormBindRequest struct {
	GrantType    string   `form:"grant_type"      json:"grant_type"`
	Code         string   `form:"code"            json:"code,omitempty"`
	ClientSecret string   `form:"client_secret"   json:"client_secret,omitempty"`
	Scopes       []string `form:"scope,omitempty" json:"scope,omitempty"`
	Retries      int      `default:"3"            form:"retries,omitempty"       json:"retries,omitempty"`
}

func postForm(t *testing.T, target, body, contentType string) *Ctx {
	t.Helper()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)

	return NewContext(httptest.NewRecorder(), req, nil).(*Ctx)
}

func TestBindRequest_FormURLEncodedBody(t *testing.T) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"abc123"},
		"client_secret": {"s3cret"},
	}

	var req FormBindRequest
	require.NoError(t, postForm(t, "/token", form.Encode(), "application/x-www-form-urlencoded").BindRequest(&req))

	assert.Equal(t, "authorization_code", req.GrantType)
	assert.Equal(t, "abc123", req.Code)
	assert.Equal(t, "s3cret", req.ClientSecret)
	assert.Equal(t, 3, req.Retries, "default should apply when the field is absent")
}

func TestBindRequest_FormContentTypeWithCharset(t *testing.T) {
	var req FormBindRequest
	require.NoError(t, postForm(t, "/token", "grant_type=client_credentials",
		"application/x-www-form-urlencoded; charset=UTF-8").BindRequest(&req))

	assert.Equal(t, "client_credentials", req.GrantType)
}

// Repeated parameters fill a slice rather than collapsing to the first value.
func TestBindRequest_FormRepeatedParameterFillsSlice(t *testing.T) {
	body := "grant_type=x&scope=openid&scope=profile&scope=email"

	var req FormBindRequest
	require.NoError(t, postForm(t, "/token", body, "application/x-www-form-urlencoded").BindRequest(&req))

	assert.Equal(t, []string{"openid", "profile", "email"}, req.Scopes)
}

// A lone comma-separated value is the other common wire form for a list.
func TestBindRequest_FormCommaSeparatedValueFillsSlice(t *testing.T) {
	var req FormBindRequest
	require.NoError(t, postForm(t, "/token", "grant_type=x&scope=openid,profile",
		"application/x-www-form-urlencoded").BindRequest(&req))

	assert.Equal(t, []string{"openid", "profile"}, req.Scopes)
}

// The whole point of the fix: a missing required form field must be reported
// as missing, not silently left zero.
func TestBindRequest_FormMissingRequiredField(t *testing.T) {
	var req FormBindRequest

	err := postForm(t, "/token", "code=abc", "application/x-www-form-urlencoded").BindRequest(&req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grant_type")
}

// Form fields bind from the body alone. Reading the merged query-plus-body set
// would let a URL supply a credential the body never carried.
func TestBindRequest_FormIgnoresQueryStringOnPost(t *testing.T) {
	var req FormBindRequest

	err := postForm(t, "/token?grant_type=authorization_code&client_secret=leaked", "",
		"application/x-www-form-urlencoded").BindRequest(&req)

	require.Error(t, err, "query string must not satisfy a required form field")
	assert.Empty(t, req.ClientSecret)
}

// A body value must not be overridden by a same-named query parameter either.
func TestBindRequest_FormBodyWinsOverQueryString(t *testing.T) {
	var req FormBindRequest

	require.NoError(t, postForm(t, "/token?client_secret=leaked",
		"grant_type=x&client_secret=real", "application/x-www-form-urlencoded").BindRequest(&req))

	assert.Equal(t, "real", req.ClientSecret)
}

// Bodyless methods have no PostForm to read, so they fall back to the merged
// set and form: still resolves.
func TestBindRequest_FormFallsBackToQueryOnGet(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"/token?grant_type=refresh_token", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var bound FormBindRequest
	require.NoError(t, NewContext(httptest.NewRecorder(), req, nil).(*Ctx).BindRequest(&bound))

	assert.Equal(t, "refresh_token", bound.GrantType)
}

// Adding form tags must not break the JSON path for the same struct.
func TestBindRequest_JSONBodyStillBindsWhenFormTagsPresent(t *testing.T) {
	var bound FormBindRequest

	ctx := postForm(t, "/token", `{"grant_type":"authorization_code","code":"abc123"}`, "application/json")
	require.NoError(t, ctx.BindRequest(&bound))

	assert.Equal(t, "authorization_code", bound.GrantType)
	assert.Equal(t, "abc123", bound.Code)
}

// Non-file multipart fields land in PostForm too, so form: resolves there.
func TestBindRequest_MultipartFormBody(t *testing.T) {
	var body strings.Builder

	w := multipart.NewWriter(&body)
	require.NoError(t, w.WriteField("grant_type", "password"))
	require.NoError(t, w.WriteField("code", "abc123"))
	require.NoError(t, w.Close())

	var bound FormBindRequest
	require.NoError(t, postForm(t, "/token", body.String(), w.FormDataContentType()).BindRequest(&bound))

	assert.Equal(t, "password", bound.GrantType)
	assert.Equal(t, "abc123", bound.Code)
}

// Embedded structs are flattened, and their form fields must bind like any
// other.
func TestBindRequest_FormFieldsInEmbeddedStruct(t *testing.T) {
	type ClientAuth struct {
		ClientID string `form:"client_id" json:"client_id"`
	}

	type EmbeddedFormRequest struct {
		ClientAuth

		GrantType string `form:"grant_type" json:"grant_type"`
	}

	var bound EmbeddedFormRequest
	require.NoError(t, postForm(t, "/token", "grant_type=x&client_id=web-app",
		"application/x-www-form-urlencoded").BindRequest(&bound))

	assert.Equal(t, "web-app", bound.ClientID)
	assert.Equal(t, "x", bound.GrantType)
}

// ── form bodies a struct cannot receive ────────────

// JSONOnlyRequest declares body fields but tags none of them for form binding,
// so a form-encoded body has no route into it.
type JSONOnlyRequest struct {
	Name string `json:"name"`
	Bio  string `json:"bio,omitempty"`
}

// AllOptionalJSONRequest is the dangerous shape: no required field catches the
// empty bind, so the handler used to run on a wholly zero request.
type AllOptionalJSONRequest struct {
	Name string `json:"name,omitempty"`
	Bio  string `json:"bio,omitempty"`
}

func TestBindRequest_RejectsFormBodyForJSONOnlyStruct(t *testing.T) {
	var bound JSONOnlyRequest

	err := postForm(t, "/users", "name=ada&bio=engineer", "application/x-www-form-urlencoded").BindRequest(&bound)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "application/x-www-form-urlencoded")
	assert.Contains(t, err.Error(), "JSONOnlyRequest")
	assert.Empty(t, bound.Name, "nothing should have been bound")
}

// Without a required field there is no validation error to stand in for the
// real problem, so the binder itself has to report it.
func TestBindRequest_RejectsFormBodyForAllOptionalStruct(t *testing.T) {
	var bound AllOptionalJSONRequest

	err := postForm(t, "/users", "name=ada&bio=engineer", "application/x-www-form-urlencoded").BindRequest(&bound)

	require.Error(t, err)
	assert.Empty(t, bound.Name)
}

// Bind has no mechanism for pouring a form body into arbitrary fields, and says
// so rather than returning an untouched value with a nil error.
func TestBind_RejectsFormBody(t *testing.T) {
	var bound JSONOnlyRequest

	err := postForm(t, "/users", "name=ada", "application/x-www-form-urlencoded").Bind(&bound)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "FormValue")
	assert.Empty(t, bound.Name)
}

// A struct that declares no body fields at all has nothing to bind from any
// body, so a form Content-Type must stay harmless.
func TestBindRequest_FormBodyIgnoredWhenStructDeclaresNoBodyFields(t *testing.T) {
	type QueryOnlyRequest struct {
		Page string `query:"page"`
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/search?page=2", strings.NewReader("ignored=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var bound QueryOnlyRequest
	require.NoError(t, NewContext(httptest.NewRecorder(), req, nil).(*Ctx).BindRequest(&bound))

	assert.Equal(t, "2", bound.Page)
}

// A struct tagged only for forms has no json fields, and must still bind.
func TestBindRequest_FormOnlyStructNeedsNoJSONTags(t *testing.T) {
	type FormOnlyRequest struct {
		GrantType string `form:"grant_type"`
	}

	var bound FormOnlyRequest
	require.NoError(t, postForm(t, "/token", "grant_type=password",
		"application/x-www-form-urlencoded").BindRequest(&bound))

	assert.Equal(t, "password", bound.GrantType)
}
