package metrics

import (
	"bytes"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// MaxLabelsPerMetric limits labels per metric to prevent cardinality explosion.
	MaxLabelsPerMetric = 20

	// MaxLabelKeyLength limits label key length.
	MaxLabelKeyLength = 128

	// MaxLabelValueLength limits label value length.
	MaxLabelValueLength = 256

	// MaxLabelCardinality limits unique label combinations.
	MaxLabelCardinality = 10000
)

// ReservedLabels are protected system labels.
var ReservedLabels = map[string]bool{
	"__name__":     true,
	"__instance__": true,
	"__job__":      true,
	"__replica__":  true,
	"__tenant__":   true,
	"job":          true,
	"instance":     true,
	"le":           true, // Prometheus histogram bucket
	"quantile":     true, // Prometheus summary quantile
}

// LabelValidationError represents a label validation error.
type LabelValidationError struct {
	Label  string
	Reason string
	Value  string
}

func (e *LabelValidationError) Error() string {
	return fmt.Sprintf("invalid label %q: %s (value: %q)", e.Label, e.Reason, e.Value)
}

func FormatDuration(d time.Duration) string {
	return d.String()
}

// TagsToString converts tags map to string representation.
func TagsToString(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}

	var parts []string
	for k, v := range tags {
		parts = append(parts, k+"="+v)
	}

	sort.Strings(parts)

	result := ""

	var resultSb186 strings.Builder

	for i, part := range parts {
		if i > 0 {
			resultSb186.WriteString(",")
		}

		resultSb186.WriteString(part)
	}

	result += resultSb186.String()

	return result
}

// ParseTags parses tags from string array.
func ParseTags(tags ...string) map[string]string {
	result := make(map[string]string)

	for i := 0; i < len(tags); i += 2 {
		if i+1 < len(tags) {
			result[tags[i]] = tags[i+1]
		}
	}

	return result
}

// ParseTagsOptions parses tags from options.
func ParseTagsOptions(defaultTags map[string]string, opts ...MetricOption) map[string]string {
	opts1 := MetricOptions{}
	for _, opt := range opts {
		opt(&opts1)
	}

	return MergeTags(defaultTags, opts1.Labels)
}

// MergeTags merges multiple tag maps with validation.
func MergeTags(tagMaps ...map[string]string) map[string]string {
	result := make(map[string]string)

	for _, tags := range tagMaps {
		maps.Copy(result, tags)
	}

	return result
}

// ValidateAndSanitizeTags validates and sanitizes tags for production use.
func ValidateAndSanitizeTags(tags map[string]string) (map[string]string, error) {
	if len(tags) > MaxLabelsPerMetric {
		return nil, &LabelValidationError{
			Label:  "count",
			Reason: fmt.Sprintf("exceeds maximum %d labels", MaxLabelsPerMetric),
		}
	}

	sanitized := make(map[string]string, len(tags))

	for key, value := range tags {
		// Validate key
		if err := ValidateLabelKey(key); err != nil {
			return nil, err
		}

		// Validate value
		if err := ValidateLabelValue(key, value); err != nil {
			return nil, err
		}

		// Sanitize key and value
		sanitizedKey := SanitizeLabelKey(key)
		sanitizedValue := SanitizeLabelValue(value)

		sanitized[sanitizedKey] = sanitizedValue
	}

	return sanitized, nil
}

// ValidateLabelKey validates a label key.
func ValidateLabelKey(key string) error {
	if key == "" {
		return &LabelValidationError{
			Label:  key,
			Reason: "empty label key",
		}
	}

	if len(key) > MaxLabelKeyLength {
		return &LabelValidationError{
			Label:  key,
			Reason: fmt.Sprintf("key exceeds maximum length %d", MaxLabelKeyLength),
		}
	}

	// Check reserved labels
	if ReservedLabels[key] {
		return &LabelValidationError{
			Label:  key,
			Reason: "reserved system label",
		}
	}

	// Validate format: must start with letter or underscore
	if key[0] >= '0' && key[0] <= '9' {
		return &LabelValidationError{
			Label:  key,
			Reason: "label key cannot start with a digit",
		}
	}

	// Check for valid characters
	for i, char := range key {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return &LabelValidationError{
				Label:  key,
				Reason: fmt.Sprintf("invalid character at position %d: must be alphanumeric or underscore", i),
			}
		}
	}

	return nil
}

// ValidateLabelValue validates a label value.
func ValidateLabelValue(key, value string) error {
	if len(value) > MaxLabelValueLength {
		return &LabelValidationError{
			Label:  key,
			Reason: fmt.Sprintf("value exceeds maximum length %d", MaxLabelValueLength),
			Value:  value[:50] + "...",
		}
	}

	// Check for null bytes and control characters
	for i, char := range value {
		if char == 0 || (char < 32 && char != '\t' && char != '\n' && char != '\r') {
			return &LabelValidationError{
				Label:  key,
				Reason: fmt.Sprintf("contains invalid control character at position %d", i),
				Value:  value,
			}
		}
	}

	return nil
}

// SanitizeLabelKey sanitizes a label key for safe use.
func SanitizeLabelKey(key string) string {
	if key == "" {
		return "unknown"
	}

	// Ensure it starts with letter or underscore
	if key[0] >= '0' && key[0] <= '9' {
		key = "_" + key
	}

	// Replace invalid characters with underscores
	sanitized := ""

	var sanitizedSb338 strings.Builder

	for _, char := range key {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' {
			sanitizedSb338.WriteRune(char)
		} else {
			sanitizedSb338.WriteString("_")
		}
	}

	sanitized += sanitizedSb338.String()

	// Truncate if too long
	if len(sanitized) > MaxLabelKeyLength {
		sanitized = sanitized[:MaxLabelKeyLength]
	}

	return sanitized
}

// SanitizeLabelValue sanitizes a label value for safe use.
func SanitizeLabelValue(value string) string {
	// Remove null bytes and control characters
	sanitized := ""

	var sanitizedSb361 strings.Builder

	for _, char := range value {
		if char >= 32 || char == '\t' || char == '\n' || char == '\r' {
			sanitizedSb361.WriteRune(char)
		}
	}

	sanitized += sanitizedSb361.String()

	// Truncate if too long
	if len(sanitized) > MaxLabelValueLength {
		sanitized = sanitized[:MaxLabelValueLength]
	}

	return sanitized
}

// CopyLabels creates a deep copy of labels map.
func CopyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}

	copied := make(map[string]string, len(labels))
	maps.Copy(copied, labels)

	return copied
}

// FilterReservedLabels removes reserved labels from a tag map.
func FilterReservedLabels(tags map[string]string) map[string]string {
	filtered := make(map[string]string)

	for k, v := range tags {
		if !ReservedLabels[k] {
			filtered[k] = v
		}
	}

	return filtered
}

// LabelCardinality tracks label cardinality to prevent metric explosion.
type LabelCardinality struct {
	mu             sync.RWMutex
	combinations   map[string]int
	maxCardinality int
}

// NewLabelCardinality creates a new label cardinality tracker.
func NewLabelCardinality(maxCardinality int) *LabelCardinality {
	if maxCardinality <= 0 {
		maxCardinality = MaxLabelCardinality
	}

	return &LabelCardinality{
		combinations:   make(map[string]int),
		maxCardinality: maxCardinality,
	}
}

// Check checks if adding this label combination would exceed cardinality limits.
func (lc *LabelCardinality) Check(metricName string, labels map[string]string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	key := lc.buildKey(metricName, labels)

	return len(lc.combinations) < lc.maxCardinality || lc.combinations[key] > 0
}

// Record records a label combination.
func (lc *LabelCardinality) Record(metricName string, labels map[string]string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	key := lc.buildKey(metricName, labels)

	if lc.combinations[key] == 0 && len(lc.combinations) >= lc.maxCardinality {
		return fmt.Errorf("label cardinality limit exceeded: %d combinations", lc.maxCardinality)
	}

	lc.combinations[key]++

	return nil
}

// GetCardinality returns current cardinality count.
func (lc *LabelCardinality) GetCardinality() int {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	return len(lc.combinations)
}

// Reset resets the cardinality tracker.
func (lc *LabelCardinality) Reset() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.combinations = make(map[string]int)
}

// buildKey builds a unique key for metric name and labels.
func (lc *LabelCardinality) buildKey(metricName string, labels map[string]string) string {
	// Sort labels for consistent key generation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString(metricName)
	buf.WriteString("{")

	for i, k := range keys {
		if i > 0 {
			buf.WriteString(",")
		}

		buf.WriteString(k)
		buf.WriteString("=")
		buf.WriteString(labels[k])
	}

	buf.WriteString("}")

	return buf.String()
}

// ValidateMetricName validates metric name format.
func ValidateMetricName(name string) bool {
	if name == "" {
		return false
	}

	// Basic validation - alphanumeric, underscore, dot, hyphen
	for _, char := range name {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '_' && char != '.' && char != '-' {
			return false
		}
	}

	return true
}

// NormalizeMetricName normalizes metric name.
func NormalizeMetricName(name string) string {
	// Replace invalid characters with underscore
	normalized := ""

	var normalizedSb503 strings.Builder

	for _, char := range name {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.' || char == '-' {
			normalizedSb503.WriteRune(char)
		} else {
			normalizedSb503.WriteString("_")
		}
	}

	normalized += normalizedSb503.String()

	return normalized
}
