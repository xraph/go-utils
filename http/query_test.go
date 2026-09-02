package http

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanQueryValues_MatchesParseQuery(t *testing.T) {
	cases := []struct{ raw, key string }{
		{"", "a"},
		{"a=1", "a"},
		{"a=1&b=2", "b"},
		{"a=1&a=2&a=3", "a"},
		{"a", "a"},
		{"a=", "a"},
		{"=1", ""},
		{"a=1&", "a"},
		{"&a=1", "a"},
		{"a=1&&b=2", "b"},
		{"a+b=c+d", "a b"},
		{"a%20b=c%20d", "a b"},
		{"%61=%31", "a"},
		{"a=%zz", "a"},
		{"%zz=1", "a"},
		{"a=1;b=2", "a"},
		{"a=1&b=2;c=3", "a"},
		{"a=b=c", "a"},
		{"a=1&missing=2", "nope"},
		{"UPPER=1", "UPPER"},
		{"a=%2F%3F%23", "a"},
	}

	for _, tc := range cases {
		t.Run(tc.raw+" | "+tc.key, func(t *testing.T) {
			want, _ := url.ParseQuery(tc.raw)
			wantValues := want[tc.key]

			_, gotAll, _ := scanQueryValues(tc.raw, tc.key, true)
			assert.Equal(t, wantValues, gotAll, "all values must match url.ParseQuery")

			gotFirst, found := scanQueryValues2(tc.raw, tc.key)
			assert.Equal(t, len(wantValues) > 0, found, "presence must match")

			if len(wantValues) > 0 {
				assert.Equal(t, wantValues[0], gotFirst, "first value must match Values.Get")
			}
		})
	}
}

// scanQueryValues2 is the single-value form, spelled out so the test exercises
// the early-return path rather than only the collecting one.
func scanQueryValues2(raw, key string) (string, bool) {
	first, _, found := scanQueryValues(raw, key, false)

	return first, found
}

// The only way to trust a hand-rolled query parser is to make the standard
// library the oracle. Any disagreement with url.ParseQuery is a defect here.
func FuzzScanQueryValues(f *testing.F) {
	seeds := []struct{ raw, key string }{
		{"a=1&b=2", "a"},
		{"a=1&a=2", "a"},
		{"a+b=c+d", "a b"},
		{"%61=%31", "a"},
		{"a=%zz", "a"},
		{"a=1;b=2", "a"},
		{"", ""},
		{"=", ""},
		{"&&&", "a"},
		{"a=%2F", "a"},
	}
	for _, s := range seeds {
		f.Add(s.raw, s.key)
	}

	f.Fuzz(func(t *testing.T, raw, key string) {
		want, _ := url.ParseQuery(raw)

		_, gotAll, found := scanQueryValues(raw, key, true)

		require.Equalf(t, want[key], gotAll,
			"all-values disagreement for key %q in query %q", key, raw)

		gotFirst, _, gotFound := scanQueryValues(raw, key, false)
		require.Equalf(t, len(want[key]) > 0, gotFound,
			"presence disagreement for key %q in query %q", key, raw)
		require.Equalf(t, found, gotFound,
			"the all and first forms must agree on presence for %q in %q", key, raw)

		if len(want[key]) > 0 {
			require.Equalf(t, want.Get(key), gotFirst,
				"first-value disagreement for key %q in query %q", key, raw)
		}
	})
}
