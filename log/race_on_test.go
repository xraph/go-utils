//go:build race

package log

import "testing"

// skipUnderRace skips allocation-budget assertions. The race detector
// allocates its own bookkeeping per operation, so AllocsPerRun measures the
// detector as much as the code and the numbers are not comparable to a
// normal build. The budgets still run in CI's non-race pass.
func skipUnderRace(t *testing.T) {
	t.Helper()
	t.Skip("allocation counts are not meaningful under -race")
}
