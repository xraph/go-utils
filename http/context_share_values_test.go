package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestCtx() Context {
	return NewContext(
		httptest.NewRecorder(),
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil),
		nil,
	)
}

// A Ctx must never go back to the pool holding a values map it borrowed from
// another Ctx. If it does, the next request to draw it out clears and overwrites
// values belonging to a request that is still in flight — and that map carries
// request identity and tenant scope, so the failure is cross-request data
// corruption, not a stale read.
func TestCleanupRestoresBorrowedValuesMapBeforePooling(t *testing.T) {
	lender := newTestCtx()
	lender.Set("tenant", "acme")

	borrower := newTestCtx()
	borrower.(*Ctx).ShareValues(borrower.(*Ctx).valuesOf(lender))
	borrower.(forgeCleaner).Cleanup()

	// A brand new, unrelated request — likely the pooled borrower.
	other := newTestCtx()

	if got := other.Get("tenant"); got != nil {
		t.Errorf("new request observed a previous request's value: tenant=%v", got)
	}

	other.Set("tenant", "evilcorp")

	if got := lender.Get("tenant"); got != "acme" {
		t.Errorf("in-flight request's values were overwritten by a later request: tenant=%v, want acme", got)
	}
}

// ShareValues records the borrow so callers cannot forget to.
func TestShareValuesMarksBorrowAndReleaseRestoresIt(t *testing.T) {
	lender := newTestCtx().(*Ctx)
	borrower := newTestCtx().(*Ctx)

	if !borrower.OwnsValues() {
		t.Fatal("a fresh Ctx should own its values map")
	}

	borrower.ShareValues(lender.Values())

	if borrower.OwnsValues() {
		t.Error("ShareValues did not record the borrow")
	}

	borrower.ReleaseSharedValues()

	if !borrower.OwnsValues() {
		t.Error("ReleaseSharedValues did not restore ownership")
	}

	// Writes must no longer reach the lender.
	borrower.Set("k", "borrower")

	if got := lender.Get("k"); got != nil {
		t.Errorf("write leaked to the lender after release: k=%v", got)
	}
}

// ReleaseSharedValues is idempotent and safe on an owning Ctx.
func TestReleaseSharedValuesIsIdempotent(t *testing.T) {
	c := newTestCtx().(*Ctx)

	c.Set("k", "v")
	c.ReleaseSharedValues()

	if got := c.Get("k"); got != "v" {
		t.Errorf("release on an owning Ctx discarded its values: k=%v", got)
	}

	c.ReleaseSharedValues()

	if !c.OwnsValues() {
		t.Error("Ctx should still own its values map")
	}
}

// NewContext must not clear a borrowed map even if Cleanup was skipped.
func TestNewContextReplacesRatherThanClearsBorrowedMap(t *testing.T) {
	lender := newTestCtx().(*Ctx)
	lender.Set("tenant", "acme")

	// Simulate a Ctx that borrowed and reached the pool without Cleanup.
	stray := &Ctx{params: make(map[string]string, 8)}
	stray.ShareValues(lender.Values())
	ctxPool.Put(stray)

	// Draw contexts until the stray is recycled, then confirm the lender survived.
	for range 100 {
		_ = newTestCtx()

		if lender.Get("tenant") != "acme" {
			t.Fatal("NewContext cleared a map borrowed by a pooled Ctx")
		}
	}
}

// forgeCleaner mirrors the ContextWithClean interface consumers assert on.
type forgeCleaner interface{ Cleanup() }

// valuesOf is a test helper: reach the lender's map the way a middleware bridge
// does.
func (c *Ctx) valuesOf(other Context) map[string]any {
	if v, ok := other.(interface{ Values() map[string]any }); ok {
		return v.Values()
	}

	return nil
}
