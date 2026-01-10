package log

import (
	"time"
)

// PerformanceMonitor helps monitor performance metrics.
type PerformanceMonitor struct {
	logger Logger
	start  time.Time
	fields []Field
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor(logger Logger, operation string) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger: logger,
		start:  time.Now(),
		fields: []Field{String("operation", operation)},
	}
}

// WithField adds a field to the performance monitor.
func (pm *PerformanceMonitor) WithField(field Field) *PerformanceMonitor {
	pm.fields = append(pm.fields, field)

	return pm
}

// WithFields adds multiple fields to the performance monitor.
func (pm *PerformanceMonitor) WithFields(fields ...Field) *PerformanceMonitor {
	pm.fields = append(pm.fields, fields...)

	return pm
}

// Finish logs the completion of the monitored operation.
func (pm *PerformanceMonitor) Finish() {
	duration := time.Since(pm.start)
	pm.fields = append(pm.fields,
		Duration("duration", duration),
		LatencyMs(duration),
	)

	// Log at different levels based on duration
	switch {
	case duration > 5*time.Second:
		pm.logger.Warn("Operation completed with high latency", pm.fields...)
	case duration > 1*time.Second:
		pm.logger.Info("Operation completed with moderate latency", pm.fields...)
	default:
		pm.logger.Debug("Operation completed", pm.fields...)
	}
}

// FinishWithError logs the completion of the monitored operation with an error.
func (pm *PerformanceMonitor) FinishWithError(err error) {
	duration := time.Since(pm.start)
	pm.fields = append(pm.fields,
		Duration("duration", duration),
		LatencyMs(duration),
		Error(err),
	)
	pm.logger.Error("Operation failed", pm.fields...)
}
