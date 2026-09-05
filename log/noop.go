package log

import "context"

// noopLogger discards everything. Every method body is empty so the compiler
// can inline the call and, in most cases, prove the variadic field slice never
// escapes, which makes logging genuinely free rather than merely cheap.
type noopLogger struct{}

// NewNoopLogger returns a logger that discards every message.
func NewNoopLogger() Logger { return noopLogger{} }

func (noopLogger) Debug(string, ...Field)               {}
func (noopLogger) Info(string, ...Field)                {}
func (noopLogger) Warn(string, ...Field)                {}
func (noopLogger) Error(string, ...Field)               {}
func (noopLogger) Fatal(string, ...Field)               {}
func (noopLogger) Debugf(string, ...any)                {}
func (noopLogger) Infof(string, ...any)                 {}
func (noopLogger) Warnf(string, ...any)                 {}
func (noopLogger) Errorf(string, ...any)                {}
func (noopLogger) Fatalf(string, ...any)                {}
func (n noopLogger) With(...Field) Logger               { return n }
func (n noopLogger) WithContext(context.Context) Logger { return n }
func (n noopLogger) Named(string) Logger                { return n }
func (noopLogger) Sugar() SugarLogger                   { return noopSugarLogger{} }
func (noopLogger) Sync() error                          { return nil }

type noopSugarLogger struct{}

func (noopSugarLogger) Debugw(string, ...any)     {}
func (noopSugarLogger) Infow(string, ...any)      {}
func (noopSugarLogger) Warnw(string, ...any)      {}
func (noopSugarLogger) Errorw(string, ...any)     {}
func (noopSugarLogger) Fatalw(string, ...any)     {}
func (n noopSugarLogger) With(...any) SugarLogger { return n }
