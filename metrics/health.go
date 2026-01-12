package metrics

import (
	"context"
	"maps"
	"time"

	"encoding/json"
)

// HealthChecker performs health checks and provides status information.
type HealthChecker interface {
	// Check executes health checks for all registered services and returns a comprehensive health report.
	Check(ctx context.Context) *HealthReport

	// CheckOne initiates a health check for the specified check by name and returns its health result.
	CheckOne(ctx context.Context, name string) *HealthResult

	// Status returns the current overall health status of the system or service.
	Status() HealthStatus
}

// HealthCheckRegistry manages registration and retrieval of health checks.
type HealthCheckRegistry interface {
	// Register registers a health check. Returns an error if the check name is already registered or invalid.
	Register(check HealthCheck) error

	// RegisterFn registers a function-based health check under a provided name.
	RegisterFn(name string, check HealthCheckFn) error

	// Unregister removes a health check associated with the specified name.
	Unregister(name string) error

	// ListChecks retrieves a map of all registered health checks with their corresponding names as keys.
	ListChecks() map[string]HealthCheck
}

// HealthService defines the lifecycle management for health checking services.
type HealthService interface {
	// Name returns the name or identifier associated with the implementation.
	Name() string

	// Start initializes and starts the health service.
	Start(ctx context.Context) error

	// Stop gracefully stops the health service and releases resources.
	Stop(ctx context.Context) error

	// Health performs health checks for the service itself.
	Health(ctx context.Context) error
}

// HealthMetadata manages environment, version, and hostname information.
type HealthMetadata interface {
	// SetEnvironment sets the environment name for the health manager.
	SetEnvironment(name string)

	// SetVersion sets the version information.
	SetVersion(version string)

	// SetHostname sets the hostname.
	SetHostname(hostname string)

	// Environment returns the name of the deployment environment.
	Environment() string

	// Hostname returns the hostname of the system.
	Hostname() string

	// Version returns the version information.
	Version() string

	// StartTime returns the time when the service was initialized.
	StartTime() time.Time
}

// HealthReporter provides access to health reports and statistics.
type HealthReporter interface {
	// LastReport retrieves the most recent health report generated.
	LastReport() *HealthReport

	// Stats retrieves statistics about the health checker.
	Stats() *HealthCheckerStats
}

// HealthSubscriber manages health status change notifications.
type HealthSubscriber interface {
	// Subscribe registers a callback function that triggers when health status changes.
	Subscribe(callback HealthCallback) error
}

// HealthConfigurable allows runtime configuration updates.
type HealthConfigurable interface {
	// Reload reloads the health configuration at runtime.
	Reload(config *HealthConfig) error
}

// HealthManager is the composite interface providing full health management functionality.
// Implementations should satisfy all constituent interfaces.
type HealthManager interface {
	HealthService
	HealthChecker
	HealthCheckRegistry
	HealthReporter
	HealthMetadata
	HealthSubscriber
	HealthConfigurable
}

// HealthCheckFn represents a single health check.
type HealthCheckFn func(ctx context.Context) *HealthResult

// HealthCheck defines the interface for health checks.
type HealthCheck interface {
	// Name returns the name of the health check implementation.
	Name() string

	// Check performs the health check and returns the result as a HealthResult.
	Check(ctx context.Context) *HealthResult

	// Timeout returns the maximum duration allowed for the health check to complete before timing out.
	Timeout() time.Duration

	// Critical returns true if the health check is considered critical, meaning its failure impacts the overall system health.
	Critical() bool

	// Dependencies returns a slice of strings representing the dependencies for the health check.
	Dependencies() []string
}

// HealthStatus represents the health status of a service or component.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// String returns the string representation of the health status.
func (hs HealthStatus) String() string {
	return string(hs)
}

// IsHealthy returns true if the status is healthy.
func (hs HealthStatus) IsHealthy() bool {
	return hs == HealthStatusHealthy
}

// IsDegraded returns true if the status is degraded.
func (hs HealthStatus) IsDegraded() bool {
	return hs == HealthStatusDegraded
}

// IsUnhealthy returns true if the status is unhealthy.
func (hs HealthStatus) IsUnhealthy() bool {
	return hs == HealthStatusUnhealthy
}

// IsUnknown returns true if the status is unknown.
func (hs HealthStatus) IsUnknown() bool {
	return hs == HealthStatusUnknown
}

// Severity returns a numeric severity level for comparison.
func (hs HealthStatus) Severity() int {
	switch hs {
	case HealthStatusHealthy:
		return 0
	case HealthStatusDegraded:
		return 1
	case HealthStatusUnhealthy:
		return 2
	case HealthStatusUnknown:
		return 3
	default:
		return 3
	}
}

// HealthResult represents the result of a health check.
type HealthResult struct {
	Name      string            `json:"name"`
	Status    HealthStatus      `json:"status"`
	Message   string            `json:"message"`
	Details   map[string]any    `json:"details"`
	Timestamp time.Time         `json:"timestamp"`
	Duration  time.Duration     `json:"duration"`
	Error     string            `json:"error,omitempty"`
	Critical  bool              `json:"critical"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// NewHealthResult creates a new health result.
func NewHealthResult(name string, status HealthStatus, message string) *HealthResult {
	return &HealthResult{
		Name:      name,
		Status:    status,
		Message:   message,
		Details:   make(map[string]any),
		Timestamp: time.Now(),
		Tags:      make(map[string]string),
	}
}

// HealthResultOption is a functional option for configuring HealthResult.
type HealthResultOption func(*HealthResult)

// WithDetails adds details to the health result.
func WithDetails(details map[string]any) HealthResultOption {
	return func(hr *HealthResult) {
		maps.Copy(hr.Details, details)
	}
}

// WithDetail adds a single detail to the health result.
func WithDetail(key string, value any) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Details[key] = value
	}
}

// WithError adds an error to the health result and updates status if healthy.
func WithError(err error) HealthResultOption {
	return func(hr *HealthResult) {
		if err != nil {
			hr.Error = err.Error()
			if hr.Status == HealthStatusHealthy {
				hr.Status = HealthStatusUnhealthy
			}
		}
	}
}

// WithDuration sets the duration of the health check.
func WithDuration(duration time.Duration) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Duration = duration
	}
}

// WithCritical marks the health check as critical.
func WithCritical(critical bool) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Critical = critical
	}
}

// WithTags adds tags to the health result.
func WithTags(tags map[string]string) HealthResultOption {
	return func(hr *HealthResult) {
		maps.Copy(hr.Tags, tags)
	}
}

// WithTag adds a single tag to the health result.
func WithTag(key, value string) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Tags[key] = value
	}
}

func WithTimestamp(timestamp time.Time) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Timestamp = timestamp
	}
}

func WithTimestampNow() HealthResultOption {
	return func(hr *HealthResult) {
		hr.Timestamp = time.Now()
	}
}

func WithStatus(status HealthStatus) HealthResultOption {
	return func(hr *HealthResult) {
		hr.Status = status
	}
}

// WithDetails adds details to the health result.
func (hr *HealthResult) WithDetails(details map[string]any) *HealthResult {
	maps.Copy(hr.Details, details)

	return hr
}

// WithDetail adds a single detail to the health result.
func (hr *HealthResult) WithDetail(key string, value any) *HealthResult {
	hr.Details[key] = value

	return hr
}

// WithError adds an error to the health result.
func (hr *HealthResult) WithError(err error) *HealthResult {
	if err != nil {
		hr.Error = err.Error()
		if hr.Status == HealthStatusHealthy {
			hr.Status = HealthStatusUnhealthy
		}
	}

	return hr
}

// WithDuration sets the duration of the health check.
func (hr *HealthResult) WithDuration(duration time.Duration) *HealthResult {
	hr.Duration = duration

	return hr
}

// WithCritical marks the health check as critical.
func (hr *HealthResult) WithCritical(critical bool) *HealthResult {
	hr.Critical = critical

	return hr
}

func (hr *HealthResult) WithStatus(status HealthStatus) *HealthResult {
	hr.Status = status

	return hr
}

func (hr *HealthResult) WithMessage(message string) *HealthResult {
	hr.Message = message

	return hr
}

// WithTags adds tags to the health result.
func (hr *HealthResult) WithTags(tags map[string]string) *HealthResult {
	maps.Copy(hr.Tags, tags)

	return hr
}

// WithTag adds a single tag to the health result.
func (hr *HealthResult) WithTag(key, value string) *HealthResult {
	hr.Tags[key] = value

	return hr
}

func (hr *HealthResult) WithTimestamp(timestamp time.Time) *HealthResult {
	hr.Timestamp = timestamp

	return hr
}

func (hr *HealthResult) WithTimestampNow() *HealthResult {
	hr.Timestamp = time.Now()

	return hr
}

// With applies multiple options to the health result.
func (hr *HealthResult) With(opts ...HealthResultOption) *HealthResult {
	for _, opt := range opts {
		opt(hr)
	}

	return hr
}

func (hr *HealthResult) String() string {
	return hr.Name + ": " + hr.Status.String() + " - " + hr.Message
}

// IsHealthy returns true if the health result is healthy.
func (hr *HealthResult) IsHealthy() bool {
	return hr.Status.IsHealthy()
}

// IsDegraded returns true if the health result is degraded.
func (hr *HealthResult) IsDegraded() bool {
	return hr.Status.IsDegraded()
}

// IsUnhealthy returns true if the health result is unhealthy.
func (hr *HealthResult) IsUnhealthy() bool {
	return hr.Status.IsUnhealthy()
}

// IsCritical returns true if the health check is critical.
func (hr *HealthResult) IsCritical() bool {
	return hr.Critical
}

// HealthReport represents a comprehensive health report.
type HealthReport struct {
	Overall     HealthStatus             `json:"overall"`
	Services    map[string]*HealthResult `json:"services"`
	Timestamp   time.Time                `json:"timestamp"`
	Duration    time.Duration            `json:"duration"`
	Version     string                   `json:"version"`
	Environment string                   `json:"environment"`
	Hostname    string                   `json:"hostname"`
	Uptime      time.Duration            `json:"uptime"`
	Metadata    map[string]any           `json:"metadata,omitempty"`
}

// NewHealthReport creates a new health report.
func NewHealthReport() *HealthReport {
	return &HealthReport{
		Overall:   HealthStatusUnknown,
		Services:  make(map[string]*HealthResult),
		Timestamp: time.Now(),
		Metadata:  make(map[string]any),
	}
}

// AddResult adds a health result to the report.
func (hr *HealthReport) AddResult(result *HealthResult) {
	hr.Services[result.Name] = result
}

// AddResults adds multiple health results to the report.
func (hr *HealthReport) AddResults(results []*HealthResult) {
	for _, result := range results {
		hr.AddResult(result)
	}
}

// WithVersion sets the version information.
func (hr *HealthReport) WithVersion(version string) *HealthReport {
	hr.Version = version

	return hr
}

// WithEnvironment sets the environment information.
func (hr *HealthReport) WithEnvironment(environment string) *HealthReport {
	hr.Environment = environment

	return hr
}

// WithHostname sets the hostname information.
func (hr *HealthReport) WithHostname(hostname string) *HealthReport {
	hr.Hostname = hostname

	return hr
}

// WithUptime sets the application uptime.
func (hr *HealthReport) WithUptime(uptime time.Duration) *HealthReport {
	hr.Uptime = uptime

	return hr
}

// WithDuration sets the duration of the health check.
func (hr *HealthReport) WithDuration(duration time.Duration) *HealthReport {
	hr.Duration = duration

	return hr
}

// WithMetadata adds metadata to the health report.
func (hr *HealthReport) WithMetadata(metadata map[string]any) *HealthReport {
	maps.Copy(hr.Metadata, metadata)

	return hr
}

// IsHealthy returns true if the overall health is healthy.
func (hr *HealthReport) IsHealthy() bool {
	return hr.Overall.IsHealthy()
}

// IsDegraded returns true if the overall health is degraded.
func (hr *HealthReport) IsDegraded() bool {
	return hr.Overall.IsDegraded()
}

// IsUnhealthy returns true if the overall health is unhealthy.
func (hr *HealthReport) IsUnhealthy() bool {
	return hr.Overall.IsUnhealthy()
}

// ToJSON converts the health report to JSON.
func (hr *HealthReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(hr, "", "  ")
}

// FromJSON creates a health report from JSON.
func FromJSON(data []byte) (*HealthReport, error) {
	var report HealthReport

	err := json.Unmarshal(data, &report)

	return &report, err
}

// HealthReportAnalyzer provides analysis and querying capabilities for health reports.
// It separates data representation (HealthReport) from analysis operations.
type HealthReportAnalyzer struct {
	report *HealthReport
}

// NewHealthReportAnalyzer creates a new analyzer for a health report.
func NewHealthReportAnalyzer(report *HealthReport) *HealthReportAnalyzer {
	return &HealthReportAnalyzer{report: report}
}

// HealthyCount returns the number of healthy services.
func (a *HealthReportAnalyzer) HealthyCount() int {
	count := 0

	for _, result := range a.report.Services {
		if result.IsHealthy() {
			count++
		}
	}

	return count
}

// DegradedCount returns the number of degraded services.
func (a *HealthReportAnalyzer) DegradedCount() int {
	count := 0

	for _, result := range a.report.Services {
		if result.IsDegraded() {
			count++
		}
	}

	return count
}

// UnhealthyCount returns the number of unhealthy services.
func (a *HealthReportAnalyzer) UnhealthyCount() int {
	count := 0

	for _, result := range a.report.Services {
		if result.IsUnhealthy() {
			count++
		}
	}

	return count
}

// CriticalCount returns the number of critical services.
func (a *HealthReportAnalyzer) CriticalCount() int {
	count := 0

	for _, result := range a.report.Services {
		if result.IsCritical() {
			count++
		}
	}

	return count
}

// FailedCriticalCount returns the number of failed critical services.
func (a *HealthReportAnalyzer) FailedCriticalCount() int {
	count := 0

	for _, result := range a.report.Services {
		if result.IsCritical() && result.IsUnhealthy() {
			count++
		}
	}

	return count
}

// ServicesByStatus returns services filtered by status.
func (a *HealthReportAnalyzer) ServicesByStatus(status HealthStatus) []*HealthResult {
	var results []*HealthResult

	for _, result := range a.report.Services {
		if result.Status == status {
			results = append(results, result)
		}
	}

	return results
}

// CriticalServices returns all critical services.
func (a *HealthReportAnalyzer) CriticalServices() []*HealthResult {
	var results []*HealthResult

	for _, result := range a.report.Services {
		if result.IsCritical() {
			results = append(results, result)
		}
	}

	return results
}

// Summary returns a comprehensive summary of the health report.
func (a *HealthReportAnalyzer) Summary() map[string]any {
	return map[string]any{
		"overall":         a.report.Overall,
		"total_services":  len(a.report.Services),
		"healthy_count":   a.HealthyCount(),
		"degraded_count":  a.DegradedCount(),
		"unhealthy_count": a.UnhealthyCount(),
		"critical_count":  a.CriticalCount(),
		"failed_critical": a.FailedCriticalCount(),
		"timestamp":       a.report.Timestamp,
		"duration":        a.report.Duration,
		"version":         a.report.Version,
		"environment":     a.report.Environment,
		"hostname":        a.report.Hostname,
		"uptime":          a.report.Uptime,
	}
}

// HealthCallback is a callback function for health status changes.
type HealthCallback func(result *HealthResult)

// HealthReportCallback is a callback function for health report changes.
type HealthReportCallback func(report *HealthReport)

// HealthCheckerStats contains statistics about the health checker.
type HealthCheckerStats struct {
	RegisteredChecks int           `json:"registered_checks"`
	Subscribers      int           `json:"subscribers"`
	Started          bool          `json:"started"`
	Uptime           time.Duration `json:"uptime"`
	LastReportTime   time.Time     `json:"last_report_time"`
	OverallStatus    HealthStatus  `json:"overall_status"`
	LastReport       *HealthReport `json:"last_report,omitempty"`
}

// HealthFeatures configures which health check features are enabled.
type HealthFeatures struct {
	AutoDiscovery bool `json:"auto_discovery" yaml:"auto_discovery"`
	Persistence   bool `json:"persistence"    yaml:"persistence"`
	Alerting      bool `json:"alerting"       yaml:"alerting"`
	Aggregation   bool `json:"aggregation"    yaml:"aggregation"`
	Prediction    bool `json:"prediction"     yaml:"prediction"`
	Metrics       bool `json:"metrics"        yaml:"metrics"`
}

// HealthIntervals configures health check timing.
type HealthIntervals struct {
	Check  time.Duration `json:"check"  yaml:"check"`
	Report time.Duration `json:"report" yaml:"report"`
}

// HealthThresholds configures health status thresholds.
type HealthThresholds struct {
	Degraded  float64 `json:"degraded"  yaml:"degraded"`
	Unhealthy float64 `json:"unhealthy" yaml:"unhealthy"`
}

// HealthEndpoints configures health check HTTP endpoints.
type HealthEndpoints struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Prefix  string `json:"prefix"  yaml:"prefix"`
}

// HealthPerformance configures health check performance settings.
type HealthPerformance struct {
	MaxConcurrentChecks int           `json:"max_concurrent_checks" yaml:"max_concurrent_checks"`
	DefaultTimeout      time.Duration `json:"default_timeout"       yaml:"default_timeout"`
	HistorySize         int           `json:"history_size"          yaml:"history_size"`
}

// HealthConfig configures health checks.
type HealthConfig struct {
	Enabled          bool              `json:"enabled"           yaml:"enabled"`
	Features         HealthFeatures    `json:"features"          yaml:"features"`
	Intervals        HealthIntervals   `json:"intervals"         yaml:"intervals"`
	Thresholds       HealthThresholds  `json:"thresholds"        yaml:"thresholds"`
	Endpoints        HealthEndpoints   `json:"endpoints"         yaml:"endpoints"`
	Performance      HealthPerformance `json:"performance"       yaml:"performance"`
	CriticalServices []string          `json:"critical_services" yaml:"critical_services"`
	Tags             map[string]string `json:"tags"              yaml:"tags"`
	Version          string            `json:"version"           yaml:"version"`
	Environment      string            `json:"environment"       yaml:"environment"`
	AutoRegister     bool              `json:"auto_register"     yaml:"auto_register"`
	ExposeEndpoints  bool              `json:"expose_endpoints"  yaml:"expose_endpoints"`
}
