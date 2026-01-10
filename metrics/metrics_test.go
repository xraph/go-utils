package metrics

import (
	"encoding/json"
	"testing"
)

func TestMetricOption(t *testing.T) {
	t.Run("WithLabel", func(t *testing.T) {
		opts := &MetricOptions{}
		WithLabel("key", "value")(opts)

		if opts.Labels["key"] != "value" {
			t.Errorf("Labels[key] = %v, want value", opts.Labels["key"])
		}
	})

	t.Run("WithLabels", func(t *testing.T) {
		labels := map[string]string{
			"env":    "prod",
			"region": "us-east",
		}
		opts := &MetricOptions{}
		WithLabels(labels)(opts)

		if opts.Labels["env"] != "prod" {
			t.Errorf("Labels[env] = %v, want prod", opts.Labels["env"])
		}

		if opts.Labels["region"] != "us-east" {
			t.Errorf("Labels[region] = %v, want us-east", opts.Labels["region"])
		}
	})
}

func TestMetricsConfig(t *testing.T) {
	t.Run("MetricsFeatures", func(t *testing.T) {
		features := MetricsFeatures{
			SystemMetrics:  true,
			RuntimeMetrics: false,
			HTTPMetrics:    true,
		}

		data, err := json.Marshal(features)
		if err != nil {
			t.Fatalf("Marshal error = %v", err)
		}

		var parsed MetricsFeatures
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Unmarshal error = %v", err)
		}

		if parsed.SystemMetrics != features.SystemMetrics {
			t.Error("SystemMetrics not preserved")
		}
	})
}

func TestExporterStats(t *testing.T) {
	t.Run("ZeroValues", func(t *testing.T) {
		stats := ExporterStats{}

		if stats.ExportCount != 0 {
			t.Errorf("ExportCount = %v, want 0", stats.ExportCount)
		}
	})
}

func TestMetricType(t *testing.T) {
	t.Run("Values", func(t *testing.T) {
		if string(MetricTypeCounter) != "counter" {
			t.Errorf("MetricTypeCounter = %v, want counter", MetricTypeCounter)
		}
	})
}

func TestExportFormat(t *testing.T) {
	t.Run("Values", func(t *testing.T) {
		if string(ExportFormatJSON) != "json" {
			t.Errorf("ExportFormatJSON = %v, want json", ExportFormatJSON)
		}
	})
}
