//go:build !race

package log

import "testing"

// skipUnderRace is a no-op when the race detector is off.
func skipUnderRace(*testing.T) {}
