package log

import (
	"context"
	"strings"
	"time"
)

// Track logs the execution time of a function.
func Track(ctx context.Context, name string) func() {
	start := time.Now()
	logger := LoggerFromContext(ctx)

	return func() {
		duration := time.Since(start)
		logger.Debug("Function execution completed",
			String("function", name),
			Duration("duration", duration),
		)
	}
}

// TrackWithLogger logs the execution time using a specific logger.
func TrackWithLogger(logger Logger, name string) func() {
	start := time.Now()

	return func() {
		duration := time.Since(start)
		logger.Debug("Function execution completed",
			String("function", name),
			Duration("duration", duration),
		)
	}
}

// TrackWithFields logs the execution time with additional fields.
func TrackWithFields(ctx context.Context, name string, fields ...Field) func() {
	start := time.Now()
	logger := LoggerFromContext(ctx)

	return func() {
		duration := time.Since(start)
		fields = append(fields,
			String("function", name),
			Duration("duration", duration),
		)
		logger.Debug("Function execution completed", fields...)
	}
}

// LogPanic logs a panic with stack trace.
func LogPanic(logger Logger, recovered any) {
	logger.Error("Panic recovered",
		Any("panic", recovered),
		Stack("stacktrace"),
	)
}

// LogPanicWithFields logs a panic with additional fields.
func LogPanicWithFields(logger Logger, recovered any, fields ...Field) {
	fields = append(fields,
		Any("panic", recovered),
		Stack("stacktrace"),
	)
	logger.Error("Panic recovered", fields...)
}

// ConditionalLog logs only if condition is true.
func ConditionalLog(condition bool, logger Logger, level string, msg string, fields ...Field) {
	if !condition {
		return
	}

	switch strings.ToLower(level) {
	case "debug":
		logger.Debug(msg, fields...)
	case "info":
		logger.Info(msg, fields...)
	case "warn", "warning":
		logger.Warn(msg, fields...)
	case "error":
		logger.Error(msg, fields...)
	case "fatal":
		logger.Fatal(msg, fields...)
	}
}

// Must wraps a function call and logs any error fatally.
func Must(err error, logger Logger, msg string, fields ...Field) {
	if err != nil {
		fields = append(fields, Error(err))
		logger.Fatal(msg, fields...)
	}
}

// MustNotNil logs fatally if value is nil.
func MustNotNil(value any, logger Logger, msg string, fields ...Field) {
	if value == nil {
		logger.Fatal(msg, fields...)
	}
}

// ErrorHandler provides a callback-based error handler with logging.
type ErrorHandler struct {
	logger   Logger
	callback func(error)
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(logger Logger, callback func(error)) *ErrorHandler {
	return &ErrorHandler{
		logger:   logger,
		callback: callback,
	}
}

// Handle handles an error by logging it and calling the callback.
func (eh *ErrorHandler) Handle(err error, msg string, fields ...Field) {
	if err == nil {
		return
	}

	fields = append(fields, Error(err))
	eh.logger.Error(msg, fields...)

	if eh.callback != nil {
		eh.callback(err)
	}
}

// HandleWithLevel handles an error at a specific log level.
func (eh *ErrorHandler) HandleWithLevel(err error, level string, msg string, fields ...Field) {
	if err == nil {
		return
	}

	fields = append(fields, Error(err))

	switch strings.ToLower(level) {
	case "debug":
		eh.logger.Debug(msg, fields...)
	case "info":
		eh.logger.Info(msg, fields...)
	case "warn", "warning":
		eh.logger.Warn(msg, fields...)
	case "error":
		eh.logger.Error(msg, fields...)
	case "fatal":
		eh.logger.Fatal(msg, fields...)
	}

	if eh.callback != nil {
		eh.callback(err)
	}
}

// LoggingWriter is an io.Writer that logs each write.
type LoggingWriter struct {
	logger Logger
	level  string
}

// NewLoggingWriter creates a new logging writer.
func NewLoggingWriter(logger Logger, level string) *LoggingWriter {
	return &LoggingWriter{
		logger: logger,
		level:  level,
	}
}

// Write implements io.Writer.
func (lw *LoggingWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		ConditionalLog(true, lw.logger, lw.level, msg)
	}

	return len(p), nil
}

// HTTPRequestLogger creates a logger with HTTP request fields.
func HTTPRequestLogger(logger Logger, method, path, userAgent string, status int) Logger {
	group := HTTPRequestGroup(method, path, userAgent, status)

	return logger.With(group.Fields()...)
}

// DatabaseQueryLogger creates a logger with database query fields.
func DatabaseQueryLogger(logger Logger, query, table string, rows int64, duration time.Duration) Logger {
	group := DatabaseQueryGroup(query, table, rows, duration)

	return logger.With(group.Fields()...)
}

// ServiceLogger creates a logger with service information fields.
func ServiceLogger(logger Logger, name, version, environment string) Logger {
	group := ServiceInfoGroup(name, version, environment)

	return logger.With(group.Fields()...)
}
