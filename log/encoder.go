package log

import (
	"fmt"
	"sync"
	"time"
)

// entry is one log event, without its fields.
type entry struct {
	lvl    level
	msg    string
	name   string
	ts     time.Time
	caller string
}

// encoder turns an entry and its fields into a complete output line,
// terminating newline included. It appends to dst and returns the extended
// slice so the caller can reuse a pooled buffer.
type encoder interface {
	encode(dst []byte, e entry, fields []Field) []byte
}

const initialBufSize = 512

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, initialBufSize)

		return &b
	},
}

func getBuf() *[]byte {
	return bufPool.Get().(*[]byte)
}

// putBuf returns a buffer to the pool. Buffers that grew very large are
// dropped so that one enormous log line does not pin memory forever.
func putBuf(b *[]byte) {
	const maxRetained = 16 << 10
	if cap(*b) > maxRetained {
		return
	}

	*b = (*b)[:0]
	bufPool.Put(b)
}

// formatAny renders an arbitrary value for output. A String() or Error()
// method can panic on a nil receiver or bad state, and a logging call must
// never take down the caller, so the panic is caught and rendered instead.
// Both things this replaced did the same: zapcore wrapped these in a
// recover, and fmt's %v prints %!v(PANIC=...).
func formatAny(v any) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("%%!v(PANIC=%v)", r)
		}
	}()

	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case error:
		return t.Error()
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// safeErrorString calls err.Error(), recovering if the method panics on a
// nil receiver or bad internal state. Used wherever an errorType field is
// rendered directly instead of through formatAny, so the same guarantee
// applies: a bad Error() implementation degrades the log line, it does not
// take down the caller.
func safeErrorString(err error) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = fmt.Sprintf("%%!v(PANIC=%v)", r)
		}
	}()

	return err.Error()
}
