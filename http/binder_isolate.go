package http

import (
	"reflect"
	"sync"
)

// Request bodies must never reach fields that bind from the path, the query
// string or a header.
//
// BindRequest binds those fields first and then decodes the body into the
// same struct. encoding/json matches untagged field names case-insensitively,
// so a struct like
//
//	type TransferRequest struct {
//	    TenantID string `path:"tenantId"`
//	    UserID   string `header:"X-User-Id"`
//	    Amount   int    `json:"amount"`
//	}
//
// let a body of {"TenantID":"victim"} replace the tenant taken from the URL
// after the router, and any middleware that checked the path, had already
// approved it. The same holds for a field tagged with both json:"..." and a
// parameter tag, and for XML bodies.
//
// The body is therefore decoded into a scratch copy of the struct in which
// every parameter field is detached (zeroed, with pointer embedded structs
// copied so the decoder cannot write through them into the caller's value).
// The parameter fields are then put back from the caller's value and the
// copy is stored over it. Parameter fields end up holding exactly what the
// path, query and headers bound, or whatever the caller had put there before
// binding when the parameter was absent. Nothing in the body can change them.
//
// Fields carrying both a parameter tag and a json tag bind from the parameter
// only. The json tag still names the field in responses and in validation
// errors.

// bodyBindPlan is what BindRequest needs to know about a request type before
// it decodes a body into it. It depends only on the type, so it is built once
// per type and cached.
type bodyBindPlan struct {
	// params lists the fields a body must not reach. Nil when the type has
	// none, in which case the body is decoded straight into the caller's
	// value as before.
	params *paramFieldSet

	// hasBodyFields reports whether any top level field is tagged json or
	// body and is not a parameter. Without one the body is not read at all.
	hasBodyFields bool
}

// paramFieldSet lists the parameter fields of one struct level, and the
// flattened embedded structs below it that contain parameter fields.
type paramFieldSet struct {
	fields   []int
	embedded []embeddedParamSet
}

type embeddedParamSet struct {
	set     *paramFieldSet
	index   int
	pointer bool
}

// bodyBindPlans caches *bodyBindPlan by reflect.Type.
var bodyBindPlans sync.Map

func bodyBindPlanFor(rt reflect.Type) *bodyBindPlan {
	if cached, ok := bodyBindPlans.Load(rt); ok {
		return cached.(*bodyBindPlan) //nolint:forcetypeassert // only *bodyBindPlan is stored
	}

	plan := &bodyBindPlan{
		params:        collectParamFields(rt),
		hasBodyFields: structHasBodyFields(rt),
	}

	actual, _ := bodyBindPlans.LoadOrStore(rt, plan)

	return actual.(*bodyBindPlan) //nolint:forcetypeassert // only *bodyBindPlan is stored
}

// isParamField reports whether bindField binds this field from the path, the
// query string or a header.
func isParamField(field reflect.StructField) bool {
	return field.Tag.Get("path") != "" ||
		field.Tag.Get("query") != "" ||
		field.Tag.Get("header") != ""
}

// collectParamFields walks rt the same way bindStructFields does and records
// every field it would bind from a parameter. It returns nil when there are
// none.
func collectParamFields(rt reflect.Type) *paramFieldSet {
	var set paramFieldSet

	for i := range rt.NumField() {
		field := rt.Field(i)

		// encoding/json still fills the exported fields of an unexported
		// embedded struct, so those are walked as well. bindStructFields
		// skips them, which leaves such a parameter unbound, but the body
		// must not be able to supply it either. An unexported embedded
		// pointer cannot be set through reflection at all, and the decoder
		// refuses to allocate one, so it is left alone.
		if !field.IsExported() && (!field.Anonymous || field.Type.Kind() != reflect.Struct) {
			continue
		}

		// Embedded structs without an explicit parameter or form tag are
		// flattened by bindStructFields, so their fields are walked too.
		if field.Anonymous && ((!isParamField(field) && field.Tag.Get("form") == "") || !field.IsExported()) {
			embedded := field.Type
			pointer := embedded.Kind() == reflect.Pointer

			if pointer {
				embedded = embedded.Elem()
			}

			if embedded.Kind() == reflect.Struct {
				if sub := collectParamFields(embedded); sub != nil {
					set.embedded = append(set.embedded, embeddedParamSet{set: sub, index: i, pointer: pointer})
				}

				continue
			}
		}

		if isParamField(field) {
			set.fields = append(set.fields, i)
		}
	}

	if len(set.fields) == 0 && len(set.embedded) == 0 {
		return nil
	}

	return &set
}

// structHasBodyFields reports whether rt has a top level field that the body
// fills: one tagged json or body that is not also a parameter.
func structHasBodyFields(rt reflect.Type) bool {
	for i := range rt.NumField() {
		field := rt.Field(i)

		if isParamField(field) {
			continue
		}

		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			return true
		}

		if tag := field.Tag.Get("body"); tag != "" && tag != "-" {
			return true
		}
	}

	return false
}

// decodeBodyIsolated decodes the request body into rv, which v points to,
// without letting it touch any field in params. See the comment at the top of
// this file.
func (c *Ctx) decodeBodyIsolated(v any, rv reflect.Value, params *paramFieldSet) error {
	if params == nil {
		return c.Bind(v)
	}

	scratch := reflect.New(rv.Type())
	sv := scratch.Elem()
	sv.Set(rv)

	detachParamFields(sv, params)

	// The decode result is applied even when it fails. A GET or DELETE with
	// no body reports io.EOF here and BindRequest ignores it, and the caller
	// previously saw whatever a partial decode had written. Only the
	// parameter fields are protected; everything else behaves as before.
	err := c.Bind(scratch.Interface())

	restoreParamFields(sv, rv, params)
	rv.Set(sv)

	return err
}

// detachParamFields zeroes every parameter field in v so the decoder starts
// from nothing for them. A pointer, slice or map shared with the caller's
// value would otherwise be decoded into in place. Pointer embedded structs
// are replaced with copies for the same reason.
func detachParamFields(v reflect.Value, set *paramFieldSet) {
	for _, i := range set.fields {
		v.Field(i).SetZero()
	}

	for _, e := range set.embedded {
		f := v.Field(e.index)

		if e.pointer {
			if f.IsNil() {
				continue
			}

			cp := reflect.New(f.Type().Elem())
			cp.Elem().Set(f.Elem())
			f.Set(cp)
			f = cp.Elem()
		}

		detachParamFields(f, e.set)
	}
}

// restoreParamFields copies every parameter field from src, the caller's
// untouched value, over dst, the decoded scratch value.
func restoreParamFields(dst, src reflect.Value, set *paramFieldSet) {
	for _, i := range set.fields {
		dst.Field(i).Set(src.Field(i))
	}

	for _, e := range set.embedded {
		d := dst.Field(e.index)
		s := src.Field(e.index)

		if e.pointer {
			switch {
			case d.IsNil():
				// Only a custom unmarshaler can clear an embedded pointer.
				// Keep the caller's struct so its parameters survive.
				d.Set(s)

				continue
			case s.IsNil():
				// The decoder allocated a struct the caller never had, so
				// nothing in it came from a parameter.
				clearParamFields(d.Elem(), e.set)

				continue
			}

			d = d.Elem()
			s = s.Elem()
		}

		restoreParamFields(d, s, e.set)
	}
}

// clearParamFields zeroes every parameter field reachable from v.
func clearParamFields(v reflect.Value, set *paramFieldSet) {
	for _, i := range set.fields {
		v.Field(i).SetZero()
	}

	for _, e := range set.embedded {
		f := v.Field(e.index)

		if e.pointer {
			if f.IsNil() {
				continue
			}

			f = f.Elem()
		}

		clearParamFields(f, e.set)
	}
}
