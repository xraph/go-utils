package log

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"runtime"
	"time"
)

type fieldType uint8

const (
	unknownType fieldType = iota
	stringType
	int64Type
	uint64Type
	float64Type
	boolType
	durationType
	timeType
	timeFullType
	errorType
	stringerType
	stringsType
	anyType
	lazyType
)

// Field is a single structured key and value. It is a struct rather than an
// interface so that scalar values never box and never reach a map, which is
// what keeps field order deterministic and the hot path allocation free.
type Field struct {
	key   string
	typ   fieldType
	num   int64
	str   string
	iface any
}

// Key returns the field's key.
func (f Field) Key() string { return f.key }

// Value returns the field's value, reconstructed from the type tag.
func (f Field) Value() any {
	switch f.typ {
	case stringType:
		return f.str
	case int64Type:
		return f.num
	case uint64Type:
		return uint64(f.num)
	case float64Type:
		return math.Float64frombits(uint64(f.num))
	case boolType:
		return f.num == 1
	case durationType:
		return time.Duration(f.num)
	case timeType:
		return time.Unix(0, f.num).UTC()
	case timeFullType:
		t, _ := f.iface.(time.Time)

		return t
	case lazyType:
		if fn, ok := f.iface.(func() any); ok {
			return fn()
		}

		return nil
	default:
		return f.iface
	}
}

var (
	String = func(key, val string) Field {
		return Field{key: key, typ: stringType, str: val}
	}
	Int    = func(key string, val int) Field { return Int64(key, int64(val)) }
	Int8   = func(key string, val int8) Field { return Int64(key, int64(val)) }
	Int16  = func(key string, val int16) Field { return Int64(key, int64(val)) }
	Int32  = func(key string, val int32) Field { return Int64(key, int64(val)) }
	Int64  = func(key string, val int64) Field { return Field{key: key, typ: int64Type, num: val} }
	Uint   = func(key string, val uint) Field { return Uint64(key, uint64(val)) }
	Uint8  = func(key string, val uint8) Field { return Uint64(key, uint64(val)) }
	Uint16 = func(key string, val uint16) Field { return Uint64(key, uint64(val)) }
	Uint32 = func(key string, val uint32) Field { return Uint64(key, uint64(val)) }
	Uint64 = func(key string, val uint64) Field {
		return Field{key: key, typ: uint64Type, num: int64(val)}
	}
	Float32 = func(key string, val float32) Field { return Float64(key, float64(val)) }
	Float64 = func(key string, val float64) Field {
		return Field{key: key, typ: float64Type, num: int64(math.Float64bits(val))}
	}
	Bool = func(key string, val bool) Field {
		var n int64
		if val {
			n = 1
		}

		return Field{key: key, typ: boolType, num: n}
	}
	// Time stores the value as UnixNano when it fits, which keeps the field
	// allocation-free. Outside roughly 1678-2262 UnixNano silently wraps, so
	// those values (the zero time.Time among them) keep the whole time.Time in
	// the interface slot instead.
	Time = func(key string, val time.Time) Field {
		if val.Before(time.Unix(0, math.MinInt64)) || val.After(time.Unix(0, math.MaxInt64)) {
			return Field{key: key, typ: timeFullType, iface: val}
		}

		return Field{key: key, typ: timeType, num: val.UnixNano()}
	}
	Duration = func(key string, val time.Duration) Field {
		return Field{key: key, typ: durationType, num: int64(val)}
	}
	Error = func(err error) Field {
		return Field{key: "error", typ: errorType, iface: err}
	}
	Stringer = func(key string, val fmt.Stringer) Field {
		return Field{key: key, typ: stringerType, iface: val}
	}
	Strings = func(key string, val []string) Field {
		return Field{key: key, typ: stringsType, iface: val}
	}
	Any = func(key string, val any) Field {
		return Field{key: key, typ: anyType, iface: val}
	}
	Custom = func(key string, value any) Field { return Any(key, value) }
	Lazy   = func(key string, valueFunc func() any) Field {
		return Field{key: key, typ: lazyType, iface: valueFunc}
	}
	Binary     = func(key string, val []byte) Field { return Any(key, val) }
	ByteString = func(key string, val []byte) Field { return String(key, string(val)) }
	Reflect    = func(key string, val any) Field { return Any(key, val) }
	Complex64  = func(key string, val complex64) Field { return Any(key, val) }
	Complex128 = func(key string, val complex128) Field { return Any(key, val) }
	Namespace  = func(key string) Field { return String("namespace", key) }

	// Stack captures the current goroutine's stack at call time.
	Stack = func(key string) Field {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)

		return String(key, string(buf[:n]))
	}

	Conditional = func(condition bool, key string, value any) Field {
		if !condition {
			return Field{key: key, typ: unknownType}
		}

		return Any(key, value)
	}
	Nullable = func(key string, value any) Field {
		if value == nil {
			return String(key, "null")
		}

		return Any(key, value)
	}
)

// The dotted key names below are the ones this library has always emitted.
// They are part of the observable contract: dashboards, log queries and
// alert rules match on these strings, and renaming one breaks them
// silently, with no compile error anywhere. Do not "modernise" them.
var (
	HTTPMethod = func(method string) Field { return String("http.method", method) }
	HTTPStatus = func(status int) Field { return Int("http.status", status) }
	HTTPPath   = func(path string) Field { return String("http.path", path) }
	HTTPURL    = func(u *url.URL) Field {
		if u == nil {
			return String("http.url", "")
		}

		return String("http.url", u.String())
	}
	HTTPUserAgent = func(userAgent string) Field { return String("http.user_agent", userAgent) }

	DatabaseQuery = func(query string) Field { return String("db.query", query) }
	DatabaseTable = func(table string) Field { return String("db.table", table) }
	DatabaseRows  = func(rows int64) Field { return Int64("db.rows", rows) }

	ServiceName        = func(name string) Field { return String("service.name", name) }
	ServiceVersion     = func(version string) Field { return String("service.version", version) }
	ServiceEnvironment = func(env string) Field { return String("service.environment", env) }

	LatencyMs   = func(latency time.Duration) Field { return Float64("latency.ms", float64(latency.Nanoseconds())/1e6) }
	MemoryUsage = func(bytes int64) Field { return Int64("memory.usage", bytes) }

	RequestID = func(ctx context.Context) Field {
		id := RequestIDFromContext(ctx)
		if id == "" {
			return Field{key: "request_id", typ: unknownType}
		}

		return String("request_id", id)
	}
	TraceID = func(ctx context.Context) Field {
		id := TraceIDFromContext(ctx)
		if id == "" {
			return Field{key: "trace_id", typ: unknownType}
		}

		return String("trace_id", id)
	}
	UserID = func(ctx context.Context) Field {
		id := UserIDFromContext(ctx)
		if id == "" {
			return Field{key: "user_id", typ: unknownType}
		}

		return String("user_id", id)
	}

	ContextFields = func(ctx context.Context) []Field {
		fields := make([]Field, 0, 3)
		if id := RequestIDFromContext(ctx); id != "" {
			fields = append(fields, String("request_id", id))
		}

		if id := TraceIDFromContext(ctx); id != "" {
			fields = append(fields, String("trace_id", id))
		}

		if id := UserIDFromContext(ctx); id != "" {
			fields = append(fields, String("user_id", id))
		}

		return fields
	}
)
