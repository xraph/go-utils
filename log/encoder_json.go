package log

import (
	"math"
	"strconv"
	"time"
	"unicode/utf8"
)

type jsonEncoder struct{}

func (j *jsonEncoder) encode(dst []byte, e entry, fields []Field) []byte {
	dst = append(dst, '{')

	dst = append(dst, `"level":`...)
	dst = appendJSONString(dst, e.lvl.String())

	// AppendFormat writes straight into dst. e.ts.Format(...) would build a
	// throwaway string on every single log line, which is the most expensive
	// allocation in the encoder because it fires unconditionally. RFC3339Nano
	// only ever emits digits, '-', ':', '.', 'T', 'Z' and '+', none of which
	// need JSON escaping, so the quotes can go on by hand.
	dst = append(dst, `,"ts":"`...)
	dst = e.ts.AppendFormat(dst, time.RFC3339Nano)
	dst = append(dst, '"')

	if e.name != "" {
		dst = append(dst, `,"logger":`...)
		dst = appendJSONString(dst, e.name)
	}

	if e.caller != "" {
		dst = append(dst, `,"caller":`...)
		dst = appendJSONString(dst, e.caller)
	}

	dst = append(dst, `,"msg":`...)
	dst = appendJSONString(dst, e.msg)

	for i := range fields {
		f := &fields[i]
		if f.typ == unknownType {
			continue
		}

		dst = append(dst, ',')
		dst = appendJSONString(dst, f.key)
		dst = append(dst, ':')
		dst = appendJSONValue(dst, f)
	}

	dst = append(dst, '}', '\n')

	return dst
}

func appendJSONValue(dst []byte, f *Field) []byte {
	switch f.typ {
	case stringType:
		return appendJSONString(dst, f.str)
	case int64Type:
		return strconv.AppendInt(dst, f.num, 10)
	case uint64Type:
		return strconv.AppendUint(dst, uint64(f.num), 10)
	case float64Type:
		// Read the bits directly. f.Value() would box the float into an any and
		// immediately unbox it, costing an allocation per float field.
		v := math.Float64frombits(uint64(f.num))
		// JSON has no NaN or Inf, so those become strings.
		if v != v || v > 1.7976931348623157e308 || v < -1.7976931348623157e308 {
			return appendJSONString(dst, strconv.FormatFloat(v, 'g', -1, 64))
		}

		return strconv.AppendFloat(dst, v, 'g', -1, 64)
	case boolType:
		return strconv.AppendBool(dst, f.num == 1)
	case durationType:
		return appendJSONString(dst, time.Duration(f.num).String())
	case timeType:
		dst = append(dst, '"')
		dst = time.Unix(0, f.num).UTC().AppendFormat(dst, time.RFC3339Nano)

		return append(dst, '"')
	case timeFullType:
		t, _ := f.iface.(time.Time)
		dst = append(dst, '"')
		dst = t.AppendFormat(dst, time.RFC3339Nano)

		return append(dst, '"')
	case errorType:
		err, _ := f.iface.(error)
		if err == nil {
			return append(dst, "null"...)
		}

		return appendJSONString(dst, safeErrorString(err))
	case stringsType:
		vals, _ := f.iface.([]string)

		dst = append(dst, '[')

		for i, v := range vals {
			if i > 0 {
				dst = append(dst, ',')
			}

			dst = appendJSONString(dst, v)
		}

		return append(dst, ']')
	default:
		// stringerType, anyType, lazyType and anything else fall back to the
		// value's default formatting, rendered as a JSON string.
		return appendJSONString(dst, formatAny(f.Value()))
	}
}

// appendJSONString writes a correctly escaped JSON string, quotes included.
func appendJSONString(dst []byte, s string) []byte {
	dst = append(dst, '"')

	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			switch c {
			case '"':
				dst = append(dst, '\\', '"')
			case '\\':
				dst = append(dst, '\\', '\\')
			case '\n':
				dst = append(dst, '\\', 'n')
			case '\r':
				dst = append(dst, '\\', 'r')
			case '\t':
				dst = append(dst, '\\', 't')
			default:
				if c < 0x20 {
					dst = append(dst, '\\', 'u', '0', '0',
						hexDigits[c>>4], hexDigits[c&0xf])
				} else {
					dst = append(dst, c)
				}
			}

			i++

			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, `�`...)
			i++

			continue
		}

		dst = append(dst, s[i:i+size]...)
		i += size
	}

	return append(dst, '"')
}

const hexDigits = "0123456789abcdef"
