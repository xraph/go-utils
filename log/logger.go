package log

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// logger is the single Logger implementation.
type logger struct {
	lvl       *atomic.Int32
	enc       encoder
	out       *syncWriter
	name      string
	fields    []Field
	addCaller bool
}

func newLogger(lvl level, enc encoder, w *syncWriter, name string, addCaller bool) *logger {
	var l atomic.Int32
	l.Store(int32(lvl))

	return &logger{lvl: &l, enc: enc, out: w, name: name, addCaller: addCaller}
}

// setLevel changes the level of this logger and every logger sharing its level
// cell (that is, all of its With/Named descendants).
func (l *logger) setLevel(lv level) { l.lvl.Store(int32(lv)) }

func (l *logger) enabled(lv level) bool { return int32(lv) >= l.lvl.Load() }

// callerSkip counts the frames between runtime.Caller and the user's call
// site: 0 is caller(), 1 is log(), 2 is the exported method that called log()
// directly, 3 is the user. Getting this wrong is finding 1 in the review,
// where the old code overshot by two and printed runtime internals on every
// line.
//
// Every exported entry point (Debug/Info/.../Fatal, their f-variants, and the
// sugar logger's w-variants) calls l.log() directly from its own frame, so
// they all sit at the same depth and all pass plain callerSkip. log still
// takes skip as a parameter rather than hardcoding it, in case a future
// entry point adds an indirection and needs callerSkip+1.
const callerSkip = 3

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}

	return trimPath(file) + ":" + strconv.Itoa(line)
}

// trimPath keeps the last two path segments, which is enough to identify a file
// without printing the whole GOPATH.
func trimPath(file string) string {
	idx := strings.LastIndexByte(file, '/')
	if idx < 0 {
		return file
	}

	prev := strings.LastIndexByte(file[:idx], '/')
	if prev < 0 {
		return file
	}

	return file[prev+1:]
}

func (l *logger) log(lv level, msg string, fields []Field, skip int) {
	e := entry{lvl: lv, msg: msg, name: l.name, ts: time.Now()}
	if l.addCaller {
		e.caller = caller(skip)
	}

	// Own fields first, then per-call fields, so a call can shadow context.
	// The merged slice comes from a pool: any logger built with With() has
	// accumulated fields, which is the common pattern, so allocating a fresh
	// slice here would mean one allocation on every line those loggers write.
	all := fields

	var pooled *[]Field
	if len(l.fields) > 0 {
		pooled = fieldsPool.Get().(*[]Field)
		*pooled = append((*pooled)[:0], l.fields...)
		*pooled = append(*pooled, fields...)
		all = *pooled
	}

	buf := getBuf()
	*buf = l.enc.encode((*buf)[:0], e, all)
	_, _ = l.out.Write(*buf)
	putBuf(buf)

	if pooled != nil {
		fieldsPool.Put(pooled)
	}
}

// fieldsPool holds the merged parent+call field slices used by log.
var fieldsPool = sync.Pool{
	New: func() any {
		s := make([]Field, 0, 16)

		return &s
	},
}

func (l *logger) Debug(msg string, fields ...Field) {
	if !l.enabled(debugLevel) {
		return
	}

	l.log(debugLevel, msg, fields, callerSkip)
}

func (l *logger) Info(msg string, fields ...Field) {
	if !l.enabled(infoLevel) {
		return
	}

	l.log(infoLevel, msg, fields, callerSkip)
}

func (l *logger) Warn(msg string, fields ...Field) {
	if !l.enabled(warnLevel) {
		return
	}

	l.log(warnLevel, msg, fields, callerSkip)
}

func (l *logger) Error(msg string, fields ...Field) {
	if !l.enabled(errorLevel) {
		return
	}

	l.log(errorLevel, msg, fields, callerSkip)
}

func (l *logger) Fatal(msg string, fields ...Field) {
	if l.enabled(fatalLevel) {
		l.log(fatalLevel, msg, fields, callerSkip)
	}

	_ = l.out.Sync()

	os.Exit(1)
}

func (l *logger) Debugf(t string, a ...any) {
	if !l.enabled(debugLevel) {
		return
	}

	l.log(debugLevel, fmt.Sprintf(t, a...), nil, callerSkip)
}

func (l *logger) Infof(t string, a ...any) {
	if !l.enabled(infoLevel) {
		return
	}

	l.log(infoLevel, fmt.Sprintf(t, a...), nil, callerSkip)
}

func (l *logger) Warnf(t string, a ...any) {
	if !l.enabled(warnLevel) {
		return
	}

	l.log(warnLevel, fmt.Sprintf(t, a...), nil, callerSkip)
}

func (l *logger) Errorf(t string, a ...any) {
	if !l.enabled(errorLevel) {
		return
	}

	l.log(errorLevel, fmt.Sprintf(t, a...), nil, callerSkip)
}

func (l *logger) Fatalf(t string, a ...any) {
	if l.enabled(fatalLevel) {
		l.log(fatalLevel, fmt.Sprintf(t, a...), nil, callerSkip)
	}

	_ = l.out.Sync()

	os.Exit(1)
}

// clone copies the field slice so that two children never alias each other.
func (l *logger) clone() *logger {
	c := *l
	c.fields = make([]Field, len(l.fields), len(l.fields)+4)
	copy(c.fields, l.fields)

	return &c
}

func (l *logger) With(fields ...Field) Logger {
	if len(fields) == 0 {
		return l
	}

	c := l.clone()
	c.fields = append(c.fields, fields...)

	return c
}

func (l *logger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		return l
	}

	cf := ContextFields(ctx)
	if len(cf) == 0 {
		return l
	}

	c := l.clone()
	c.fields = append(c.fields, cf...)

	return c
}

func (l *logger) Named(name string) Logger {
	c := l.clone()
	if c.name == "" {
		c.name = name
	} else {
		c.name = c.name + "." + name
	}

	return c
}

func (l *logger) Sugar() SugarLogger { return &sugarLogger{l: l} }

func (l *logger) Sync() error { return l.out.Sync() }

// sugarLogger adapts the loosely typed key/value API onto Field.
type sugarLogger struct {
	l    *logger
	args []Field
}

func keysAndValuesToFields(kv ...any) []Field {
	fields := make([]Field, 0, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", kv[i])
		}

		fields = append(fields, Any(key, kv[i+1]))
	}
	// An odd trailing argument is dropped rather than panicking.
	return fields
}

func (s *sugarLogger) with(kv []any) []Field {
	f := keysAndValuesToFields(kv...)
	if len(s.args) == 0 {
		return f
	}

	out := make([]Field, 0, len(s.args)+len(f))
	out = append(out, s.args...)

	return append(out, f...)
}

func (s *sugarLogger) Debugw(msg string, kv ...any) {
	if s.l.enabled(debugLevel) {
		s.l.log(debugLevel, msg, s.with(kv), callerSkip)
	}
}

func (s *sugarLogger) Infow(msg string, kv ...any) {
	if s.l.enabled(infoLevel) {
		s.l.log(infoLevel, msg, s.with(kv), callerSkip)
	}
}

func (s *sugarLogger) Warnw(msg string, kv ...any) {
	if s.l.enabled(warnLevel) {
		s.l.log(warnLevel, msg, s.with(kv), callerSkip)
	}
}

func (s *sugarLogger) Errorw(msg string, kv ...any) {
	if s.l.enabled(errorLevel) {
		s.l.log(errorLevel, msg, s.with(kv), callerSkip)
	}
}

func (s *sugarLogger) Fatalw(msg string, kv ...any) {
	if s.l.enabled(fatalLevel) {
		s.l.log(fatalLevel, msg, s.with(kv), callerSkip)
	}

	_ = s.l.out.Sync()

	os.Exit(1)
}

func (s *sugarLogger) With(args ...any) SugarLogger {
	return &sugarLogger{l: s.l, args: s.with(args)}
}
