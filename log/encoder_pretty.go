package log

import (
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ANSI escapes. Kept here rather than in a colours file so that the pretty
// encoder is the only thing in the package that knows about terminals.
const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[90m"
	ansiCyan   = "\033[36m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[91m"
	ansiPurple = "\033[95m"
)

const (
	timestampLayout = "15:04:05.000"
	timestampWidth  = len(timestampLayout)
	levelWidth      = 5
	// minNameWidth is 12 because that is wide enough for the logger names this
	// framework actually produces (forge.http, forge.cache, forge.router,
	// forge.config all fit), so the common case aligns from the very first line
	// instead of stepping through two or three widths as names arrive. Measured:
	// at 8, six lines of realistic names produced three different message
	// columns; at 12, all six aligned immediately.
	minNameWidth = 12
	minMsgWidth  = 24
	maxNameWidth = 28
	maxMsgWidth  = 40
)

// prettyEncoder renders aligned, optionally coloured columns for a terminal.
//
// Column widths grow to fit and never shrink. You cannot know the widest
// logger name before you have seen it, and you cannot re-align bytes that are
// already on screen, so monotonic growth is the only stable option: the first
// few lines may sit tight, then the layout converges and stays put.
//
// encode mutates nameWidth, msgWidth and scratch, and the logger calls it
// OUTSIDE the writer's lock, so concurrent log calls would race on all three.
// The mutex here is what makes the encoder safe to share. Do not remove it on
// the grounds that the writer is already serialised: the writer serialises
// writes, not encoding.
type prettyEncoder struct {
	mu    sync.Mutex
	color bool
	width int

	nameWidth   int
	callerWidth int
	msgWidth    int
	scratch     []byte // reused per field; guarded by mu
}

func newPrettyEncoder(color bool, width int) *prettyEncoder {
	if width <= 0 {
		width = 120
	}

	return &prettyEncoder{
		color: color,
		width: width,
		// nameWidth starts at zero, not at minNameWidth: a logger with no name
		// should not pay for a name column at all. It jumps straight to
		// minNameWidth the first time a named entry arrives, and grows from
		// there. Same treatment as callerWidth below.
		nameWidth: 0,
		msgWidth:  minMsgWidth,
		scratch:   make([]byte, 0, 64),
	}
}

func levelColor(l level) string {
	switch l {
	case debugLevel:
		return ansiCyan
	case infoLevel:
		return ansiGreen
	case warnLevel:
		return ansiYellow
	case errorLevel:
		return ansiRed
	case fatalLevel:
		return ansiPurple
	default:
		return ansiReset
	}
}

func (p *prettyEncoder) paint(dst []byte, color, s string) []byte {
	// Painting an empty string emits a colour code and a reset around
	// nothing, which is invisible noise in a terminal and pure bloat in
	// a captured log.
	if !p.color || s == "" {
		return append(dst, s...)
	}

	dst = append(dst, color...)
	dst = append(dst, s...)

	return append(dst, ansiReset...)
}

// paintBytes is the []byte form. Callers hold field text in a byte slice, and
// converting it to a string just to call paint would allocate once per field.
func (p *prettyEncoder) paintBytes(dst []byte, color string, b []byte) []byte {
	if !p.color {
		return append(dst, b...)
	}

	dst = append(dst, color...)
	dst = append(dst, b...)

	return append(dst, ansiReset...)
}

func (p *prettyEncoder) encode(dst []byte, e entry, fields []Field) []byte {
	p.mu.Lock()
	defer p.mu.Unlock()

	name := truncateLeft(e.name, maxNameWidth)
	if name != "" {
		n := max(len(name), minNameWidth)

		if n > p.nameWidth {
			p.nameWidth = n
		}
	}

	if n := len(e.msg); n > p.msgWidth && n <= maxMsgWidth {
		p.msgWidth = n
	}

	// Timestamp. AppendFormat writes into dst; e.ts.Format would allocate a
	// throwaway string on every line.
	if p.color {
		dst = append(dst, ansiDim...)
	}

	dst = e.ts.AppendFormat(dst, timestampLayout)
	if p.color {
		dst = append(dst, ansiReset...)
	}

	dst = append(dst, ' ', ' ')

	// Level, padded to a fixed width so the following columns line up.
	lvl := e.lvl.String()
	dst = p.paint(dst, levelColor(e.lvl), lvl)
	dst = appendPad(dst, levelWidth-len(lvl))
	dst = append(dst, ' ')

	// Logger name. Reserved only once some entry has actually carried one, so
	// an unnamed logger does not waste a column of blanks on every line.
	nameCol := 0

	if p.nameWidth > 0 {
		nameCol = p.nameWidth + 1
		dst = p.paint(dst, ansiDim, name)
		dst = appendPad(dst, p.nameWidth-len(name))
		dst = append(dst, ' ')
	}

	// Caller sits between the name and the message and is an adaptive column
	// like the name, not a variable-width prefix. It has to be, because the
	// continuation lines of a wrapped field set hang-indent to the message
	// column: if the caller's width were not reserved, those lines would indent
	// to the wrong place and the wrap arithmetic would undercount, letting lines
	// run past the configured width.
	//
	// Once any line has carried a caller, the column stays reserved for every
	// later line, so a logger that sets callers on some lines and not others
	// still aligns.
	if e.caller != "" {
		if n := len(e.caller); n > p.callerWidth {
			p.callerWidth = n
		}
	}

	callerCol := 0
	if p.callerWidth > 0 {
		callerCol = p.callerWidth + 1
		dst = p.paint(dst, ansiDim, e.caller)
		dst = appendPad(dst, p.callerWidth-len(e.caller))
		dst = append(dst, ' ')
	}

	// Message.
	msgCol := timestampWidth + 2 + levelWidth + 1 + nameCol + callerCol

	dst = append(dst, e.msg...)
	dst = appendPad(dst, p.msgWidth-len(e.msg))

	// Fields, wrapping to the message column when the line runs long.
	col := msgCol + max(len(e.msg), p.msgWidth)

	for i := range fields {
		f := &fields[i]
		if f.typ == unknownType {
			continue
		}
		// Reuse the scratch buffer rather than allocating a fresh one per
		// field. Safe because encode holds p.mu.
		p.scratch = p.scratch[:0]
		p.scratch = append(p.scratch, f.key...)
		p.scratch = append(p.scratch, '=')
		p.scratch = appendPrettyValue(p.scratch, f)
		kvLen := len(p.scratch)

		if col+1+kvLen > p.width && col > msgCol {
			dst = append(dst, '\n')
			dst = appendPad(dst, msgCol)
			col = msgCol
		} else {
			dst = append(dst, ' ')
			col++
		}

		dst = p.paintBytes(dst, ansiDim, p.scratch)
		col += kvLen
	}

	return append(dst, '\n')
}

func appendPrettyValue(dst []byte, f *Field) []byte {
	switch f.typ {
	case stringType:
		return append(dst, f.str...)
	case int64Type:
		return strconv.AppendInt(dst, f.num, 10)
	case uint64Type:
		return strconv.AppendUint(dst, uint64(f.num), 10)
	case float64Type:
		// Read the bits directly; f.Value() would box and immediately unbox.
		return strconv.AppendFloat(dst, math.Float64frombits(uint64(f.num)), 'g', -1, 64)
	case boolType:
		return strconv.AppendBool(dst, f.num == 1)
	case durationType:
		return append(dst, time.Duration(f.num).String()...)
	case timeType:
		return time.Unix(0, f.num).UTC().AppendFormat(dst, time.RFC3339)
	case timeFullType:
		t, _ := f.iface.(time.Time)

		return t.AppendFormat(dst, time.RFC3339)
	case errorType:
		err, _ := f.iface.(error)
		if err == nil {
			return append(dst, "null"...)
		}

		return append(dst, safeErrorString(err)...)
	case stringsType:
		vals, _ := f.iface.([]string)

		return append(dst, strings.Join(vals, ",")...)
	default:
		return append(dst, formatAny(f.Value())...)
	}
}

func appendPad(dst []byte, n int) []byte {
	for ; n > 0; n-- {
		dst = append(dst, ' ')
	}

	return dst
}

// truncateLeft shortens s to at most limit characters, keeping the tail. A
// dotted logger name gets more specific from left to right, so the informative
// half is the end: forge.dashboard.contract.dispatcher becomes
// ...contract.dispatcher.
func truncateLeft(s string, limit int) string {
	if len(s) <= limit || limit < 4 {
		return s
	}

	return "..." + s[len(s)-(limit-3):]
}
