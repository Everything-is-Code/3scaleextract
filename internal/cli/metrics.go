package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
	"github.com/Everything-is-Code/3scaleextract/internal/config"
	"github.com/Everything-is-Code/3scaleextract/internal/export"
	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

func RunMetrics(ctx context.Context, cfg config.ExportConfig) error {
	if err := cfg.ValidateAuth(); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.OutDir) == "" {
		return errors.New("output directory is required: use --output")
	}
	if err := cfg.ValidateMetrics(); err != nil {
		return err
	}

	adminURL, err := config.NormalizeAdminURL(cfg.AdminURL)
	if err != nil {
		return err
	}
	since, until, err := cfg.ResolveMetricsWindow()
	if err != nil {
		return err
	}
	granularity := cfg.ResolvedMetricsGranularity()
	metricName := cfg.ResolvedMetricsMetric()

	client, err := admin.NewClient(admin.Options{
		BaseURL:       adminURL,
		Token:         cfg.Token,
		HTTPClient:    config.NewHTTPClient(cfg.InsecureTLS, 60*time.Second),
		MaxConcurrent: cfg.MaxConcurrent,
	})
	if err != nil {
		return err
	}

	writer, err := output.NewWriter(cfg.OutDir)
	if err != nil {
		return err
	}
	if err := writer.EnsureLayout(); err != nil {
		return err
	}

	services, err := export.NewService(client, nil).ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("list products: %w", err)
	}
	if len(services) == 0 {
		return errors.New("no products found for metrics export")
	}

	httpClient := config.NewHTTPClient(cfg.InsecureTLS, 60*time.Second)
	return export.ExportMetrics(ctx, adminURL, cfg.Token, httpClient, cfg.MaxConcurrent, writer, services, since, until, granularity, metricName)
}
