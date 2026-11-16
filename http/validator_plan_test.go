package http

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/go-utils/val"
)

func validateCustom(t *testing.T, v any) *val.ValidationError {
	t.Helper()

	var errs val.ValidationError

	rv := reflect.ValueOf(v).Elem()
	(&Ctx{}).validateCustomTags(rv, rv.Type(), &errs)

	return &errs
}

// A pattern that does not compile used to fail every value, because
// regexp.MatchString reported the compile error as no match. Caching the
// compiled form must keep that, and keep skipping empty optional fields.
func TestValidateCustomTags_InvalidPatternStillFails(t *testing.T) {
	type req struct {
		Code     string `json:"code"     pattern:"[a-"`
		Optional string `json:"optional" optional:"true" pattern:"[a-"`
	}

	for range 3 { // the second and third runs hit the cache
		errs := validateCustom(t, &req{Code: "anything"})
		require.Len(t, errs.Errors, 1)
		assert.Equal(t, "code", errs.Errors[0].Field)
		assert.Equal(t, val.ErrCodePattern, errs.Errors[0].Code)
	}

	errs := validateCustom(t, &req{Code: "x", Optional: "y"})
	assert.Len(t, errs.Errors, 2, "a non-empty optional value is checked against the broken pattern too")
}

// Every bound is parsed once per type now. Check each still applies, with the
// same messages, on repeated use of the cached plan.
func TestValidateCustomTags_CachedBoundsApply(t *testing.T) {
	type req struct {
		Slug  string  `json:"slug"   maxLength:"5"        minLength:"3" pattern:"^[a-z]+$"`
		Count int     `json:"count"  maximum:"100"        minimum:"10"  multipleOf:"5"`
		Ratio float64 `json:"ratio"  maximum:"1.5"        minimum:"0.5"`
		Kind  string  `enum:"a, b,c" json:"kind"`
		Day   string  `format:"date" json:"day"`
		Bad   int     `json:"bad"    minimum:"notanumber"`
	}

	for range 2 {
		ok := validateCustom(t, &req{Slug: "abcd", Count: 50, Ratio: 1, Kind: "b", Day: "2026-11-15", Bad: -5})
		assert.Empty(t, ok.Errors, "valid values, and an unparseable bound is ignored as before")

		bad := validateCustom(t, &req{Slug: "AB", Count: 7, Ratio: 2, Kind: "z", Day: "15/11/2026", Bad: -5})

		got := map[string][]string{}
		for _, e := range bad.Errors {
			got[e.Field] = append(got[e.Field], e.Message)
		}

		assert.ElementsMatch(t, []string{"must be at least 3 characters", "does not match required pattern"}, got["slug"])
		assert.ElementsMatch(t, []string{"must be at least 10", "must be a multiple of 5"}, got["count"])
		assert.ElementsMatch(t, []string{"must be at most 1.5"}, got["ratio"])
		assert.ElementsMatch(t, []string{"must be one of: a, b, c"}, got["kind"])
		assert.ElementsMatch(t, []string{"must be a valid date (YYYY-MM-DD)"}, got["day"])
		assert.NotContains(t, got, "bad")
	}
}

type selfEmbedding struct {
	*selfEmbedding

	Name string `json:"name" minLength:"2"`
}

type SelfEmbeddingExported struct {
	*SelfEmbeddingExported

	Name string `json:"name" minLength:"2"`
}

// A type that embeds a pointer to itself must not send plan building into
// endless recursion.
func TestValidateCustomTags_SelfEmbeddingType(t *testing.T) {
	errs := validateCustom(t, &SelfEmbeddingExported{
		Name:                  "ok",
		SelfEmbeddingExported: &SelfEmbeddingExported{Name: "x"},
	})
	require.Len(t, errs.Errors, 1)
	assert.Equal(t, "name", errs.Errors[0].Field)

	assert.Empty(t, validateCustom(t, &selfEmbedding{Name: "ok"}).Errors)
}

// Plans and compiled patterns are shared between requests. Run under -race.
func TestValidateCustomTags_ConcurrentUse(t *testing.T) {
	type req struct {
		Code string `json:"code" pattern:"^[A-Z]{3}$"`
		N    int    `json:"n"    maximum:"9"          minimum:"1"`
	}

	var wg sync.WaitGroup

	for i := range 16 {
		wg.Go(func() {
			r := req{Code: "ABC", N: 5}
			if i%2 == 1 {
				r = req{Code: "abc", N: 0}
			}

			var errs val.ValidationError

			rv := reflect.ValueOf(&r).Elem()
			(&Ctx{}).validateCustomTags(rv, rv.Type(), &errs)

			if i%2 == 1 {
				assert.Len(t, errs.Errors, 2)
			} else {
				assert.Empty(t, errs.Errors)
			}
		})
	}

	wg.Wait()
}
