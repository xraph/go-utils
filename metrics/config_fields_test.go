package metrics

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMetricsConfigNewFieldsYAML(t *testing.T) {
	src := []byte(`
features:
  http_body_sizes: true
limits:
  max_series_per_metric: 500
`)

	var cfg MetricsConfig
	if err := yaml.Unmarshal(src, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !cfg.Features.HTTPBodySizes {
		t.Fatal("HTTPBodySizes = false, want true")
	}

	if cfg.Limits.MaxSeriesPerMetric != 500 {
		t.Fatalf("MaxSeriesPerMetric = %d, want 500", cfg.Limits.MaxSeriesPerMetric)
	}
}

func TestMetricsConfigNewFieldsAbsent(t *testing.T) {
	var cfg MetricsConfig
	if err := yaml.Unmarshal([]byte("enabled: true\n"), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if cfg.Features.HTTPBodySizes || cfg.Limits.MaxSeriesPerMetric != 0 {
		t.Fatalf("absent keys should leave zero values, got %+v %+v", cfg.Features, cfg.Limits)
	}
}

func TestMetricsConfigNewFieldsJSON(t *testing.T) {
	var cfg MetricsConfig
	if err := json.Unmarshal([]byte(`{"features":{"http_body_sizes":true},"limits":{"max_series_per_metric":7}}`), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !cfg.Features.HTTPBodySizes || cfg.Limits.MaxSeriesPerMetric != 7 {
		t.Fatalf("got %+v %+v", cfg.Features, cfg.Limits)
	}
}
