package config

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultMetricsGranularity = "day"
	DefaultMetricsMetric      = "hits"
	metricsDateLayout         = "2006-01-02"
)

var (
	ErrInvalidMetricsDateRange = errors.New("metrics since must be on or before metrics until")
	ErrInvalidMetricsGranularity = errors.New("metrics granularity must be day, hour, or month")
)

// ResolveMetricsWindow returns ISO date strings for the Analytics API query window.
// Empty MetricsUntil defaults to today UTC; empty MetricsSince defaults to 30 days before until.
func (c ExportConfig) ResolveMetricsWindow() (since, until string, err error) {
	until = strings.TrimSpace(c.MetricsUntil)
	since = strings.TrimSpace(c.MetricsSince)

	if until == "" {
		until = time.Now().UTC().Format(metricsDateLayout)
	}
	if _, err := time.Parse(metricsDateLayout, until); err != nil {
		return "", "", fmt.Errorf("metrics until: expected YYYY-MM-DD: %w", err)
	}

	if since == "" {
		t, err := time.Parse(metricsDateLayout, until)
		if err != nil {
			return "", "", err
		}
		since = t.AddDate(0, 0, -30).Format(metricsDateLayout)
	}
	if _, err := time.Parse(metricsDateLayout, since); err != nil {
		return "", "", fmt.Errorf("metrics since: expected YYYY-MM-DD: %w", err)
	}

	if since > until {
		return "", "", ErrInvalidMetricsDateRange
	}
	return since, until, nil
}

func ValidateMetricsGranularity(granularity string) error {
	granularity = strings.TrimSpace(granularity)
	if granularity == "" {
		return nil
	}
	switch granularity {
	case "day", "hour", "month":
		return nil
	default:
		return fmt.Errorf("%w: got %q", ErrInvalidMetricsGranularity, granularity)
	}
}

func (c ExportConfig) ValidateMetrics() error {
	if _, _, err := c.ResolveMetricsWindow(); err != nil {
		return err
	}
	granularity := strings.TrimSpace(c.MetricsGranularity)
	if granularity == "" {
		granularity = DefaultMetricsGranularity
	}
	return ValidateMetricsGranularity(granularity)
}

func (c ExportConfig) ResolvedMetricsGranularity() string {
	if g := strings.TrimSpace(c.MetricsGranularity); g != "" {
		return g
	}
	return DefaultMetricsGranularity
}

func (c ExportConfig) ResolvedMetricsMetric() string {
	if m := strings.TrimSpace(c.MetricsMetric); m != "" {
		return m
	}
	return DefaultMetricsMetric
}
