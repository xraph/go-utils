package http

import (
	"net/url"
	"strings"
)

// Query parameters are read by scanning RawQuery for the one key that is
// wanted, rather than by building a url.Values for the whole string.
//
// url.ParseQuery allocates a map plus a slice per distinct key, which was the
// largest remaining cost in the bind path: about five allocations to answer a
// handful of by-name lookups. Nothing in this package exposes the map, so
// there is no reason to build one.
//
// The semantics below deliberately mirror net/url.parseQuery, including the
// parts that look odd:
//
//   - pairs are separated by "&" only,
//   - a pair containing ";" is skipped (net/url reports it as an error, and
//     URL.Query discards that error, so the pair is dropped),
//   - an empty pair is skipped,
//   - a pair whose key or value fails to unescape is skipped, even when the
//     key is the one being looked for.
//
// scanQueryValues is fuzzed against url.ParseQuery, which is the only way to
// be confident a hand-rolled parser agrees with the standard library.

// queryFirst returns the first value for name, and whether name was present.
func (c *Ctx) queryFirst(name string) (string, bool) {
	if c.request == nil {
		return "", false
	}

	first, _, found := scanQueryValues(c.request.URL.RawQuery, name, false)

	return first, found
}

// queryAll returns every value recorded for name, in order.
func (c *Ctx) queryAll(name string) []string {
	if c.request == nil {
		return nil
	}

	_, values, _ := scanQueryValues(c.request.URL.RawQuery, name, true)

	return values
}

// scanQueryValues walks raw looking for name.
//
// When all is false it stops at the first match, which is what a single-valued
// lookup needs. When all is true it collects every occurrence, which is what a
// slice-typed field needs so a repeated parameter is not silently narrowed to
// its first value.
func scanQueryValues(raw, name string, all bool) (first string, values []string, found bool) {
	for raw != "" {
		var pair string

		pair, raw, _ = strings.Cut(raw, "&")

		// net/url treats a semicolon as an error rather than a separator, and
		// drops the pair. Matching that matters: doing the friendly thing here
		// would make this parser disagree with the rest of the standard
		// library about what the request said.
		if pair == "" || strings.Contains(pair, ";") {
			continue
		}

		rawKey, rawValue, _ := strings.Cut(pair, "=")

		key, ok := unescapeQueryPart(rawKey)
		if !ok || key != name {
			continue
		}

		value, ok := unescapeQueryPart(rawValue)
		if !ok {
			continue
		}

		if !found {
			first, found = value, true

			if !all {
				return first, nil, true
			}
		}

		values = append(values, value)
	}

	return first, values, found
}

// unescapeQueryPart decodes one key or value, skipping the work entirely when
// there is nothing to decode. That fast path is where the allocations go: a
// plain key like "page" is returned as a slice of the original string.
func unescapeQueryPart(s string) (string, bool) {
	if !strings.ContainsAny(s, "%+") {
		return s, true
	}

	out, err := url.QueryUnescape(s)
	if err != nil {
		return "", false
	}

	return out, true
}
