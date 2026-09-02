package http

import (
	"bytes"
	"context"
	gohttp "net/http"
	"net/http/httptest"
	"testing"
)

// Binding benchmarks over a request shape closer to real usage than a single
// field: path, query, header and body in one struct.
//
// Allocations here used to scale with field count, because the binder asked
// whether each field's type implements encoding.TextUnmarshaler by boxing the
// value into an interface. It is a type-level question and is answered with
// reflect.Type.Implements now.
type benchBindReq struct {
	OrgID     string `path:"orgId"          validate:"required"`
	UserID    string `path:"userId"         validate:"required"`
	Page      int    `query:"page"`
	PerPage   int    `query:"perPage"`
	Search    string `query:"search"`
	RequestID string `header:"X-Request-Id"`
	Name      string `json:"name"           validate:"required"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	Note      string `json:"note"`
}

type benchNopCloser struct{ *bytes.Reader }

func (benchNopCloser) Close() error { return nil }

func benchBindRequest(tb testing.TB) *gohttp.Request {
	tb.Helper()

	req := httptest.NewRequestWithContext(
		tb.Context(), gohttp.MethodPost, "/orgs/o1/users/u1?page=2&perPage=50&search=abc", nil,
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "req-123")

	params := AcquireRouteParams()
	params.Set("orgId", "o1")
	params.Set("userId", "u1")

	return req.WithContext(context.WithValue(req.Context(), RouteParamsKey, params))
}

func BenchmarkBindRequest_Full(b *testing.B) {
	body := []byte(`{"name":"rex","email":"rex@example.com","age":30,"note":"hello"}`)
	reader := bytes.NewReader(body)

	req := benchBindRequest(b)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		reader.Reset(body)

		req.Body = benchNopCloser{reader}
		req.ContentLength = int64(len(body))

		c, ok := NewContext(w, req, nil).(*Ctx)
		if !ok {
			b.Fatal("unexpected Context implementation")
		}

		var out benchBindReq

		if err := c.BindRequest(&out); err != nil {
			b.Fatalf("bind: %v", err)
		}

		c.Cleanup()
	}
}

// benchBindNoBody isolates the path and query walk from JSON decoding.
type benchBindNoBody struct {
	OrgID   string `path:"orgId"    validate:"required"`
	UserID  string `path:"userId"   validate:"required"`
	Page    int    `query:"page"`
	PerPage int    `query:"perPage"`
	Search  string `query:"search"`
}

func BenchmarkBindRequest_NoBody(b *testing.B) {
	req := benchBindRequest(b)
	req.Method = gohttp.MethodGet

	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		c, ok := NewContext(w, req, nil).(*Ctx)
		if !ok {
			b.Fatal("unexpected Context implementation")
		}

		var out benchBindNoBody

		if err := c.BindRequest(&out); err != nil {
			b.Fatalf("bind: %v", err)
		}

		c.Cleanup()
	}
}
