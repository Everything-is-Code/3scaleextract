package config

import (
	"testing"
	"time"
)

func TestResolveMetricsWindowDefaults(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	wantSince := time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02")

	cfg := ExportConfig{}
	since, until, err := cfg.ResolveMetricsWindow()
	if err != nil {
		t.Fatal(err)
	}
	if until != today {
		t.Fatalf("until = %q, want %q", until, today)
	}
	if since != wantSince {
		t.Fatalf("since = %q, want %q", since, wantSince)
	}
}

func TestResolveMetricsWindowInvalidRange(t *testing.T) {
	cfg := ExportConfig{
		MetricsSince: "2026-07-10",
		MetricsUntil: "2026-07-01",
	}
	_, _, err := cfg.ResolveMetricsWindow()
	if err != ErrInvalidMetricsDateRange {
		t.Fatalf("err = %v, want ErrInvalidMetricsDateRange", err)
	}
}

func TestValidateMetricsGranularity(t *testing.T) {
	if err := ValidateMetricsGranularity("day"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMetricsGranularity("week"); err == nil {
		t.Fatal("expected error for week")
	}
}

func TestValidateMetricsExplicitWindow(t *testing.T) {
	cfg := ExportConfig{
		MetricsSince:       "2026-01-01",
		MetricsUntil:       "2026-01-31",
		MetricsGranularity: "hour",
		MetricsMetric:      "hits",
	}
	if err := cfg.ValidateMetrics(); err != nil {
		t.Fatal(err)
	}
	since, until, err := cfg.ResolveMetricsWindow()
	if err != nil {
		t.Fatal(err)
	}
	if since != "2026-01-01" || until != "2026-01-31" {
		t.Fatalf("window = %q..%q", since, until)
	}
}
