package log

import (
	"strconv"
	"time"
	"unicode/utf8"
)

type jsonEncoder struct{}

func (j *jsonEncoder) encode(dst []byte, e entry, fields []Field) []byte {
	dst = append(dst, '{')

	dst = append(dst, `"level":`...)
	dst = appendJSONString(dst, e.lvl.String())

	dst = append(dst, `,"ts":`...)
	dst = appendJSONString(dst, e.ts.Format(time.RFC3339Nano))

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
		v := f.Value().(float64)
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
		return appendJSONString(dst, time.Unix(0, f.num).UTC().Format(time.RFC3339Nano))
	case errorType:
		err, _ := f.iface.(error)
		if err == nil {
			return append(dst, "null"...)
		}
		return appendJSONString(dst, err.Error())
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
