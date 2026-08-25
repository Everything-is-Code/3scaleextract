package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
	"github.com/Everything-is-Code/3scaleextract/internal/config"
	"github.com/Everything-is-Code/3scaleextract/internal/export"
	"github.com/Everything-is-Code/3scaleextract/internal/output"
	"github.com/Everything-is-Code/3scaleextract/internal/progress"
	"github.com/Everything-is-Code/3scaleextract/internal/version"
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	cfg, _ := config.LoadExportFromEnv()
	cmd := &cobra.Command{
		Use:   "threescale-export",
		Short: "Export 3scale tenant configuration (products, backends, policies, applications)",
		Long: `Export all relevant 3scale Admin API data for a tenant: products, backends,
application plans, auth configuration, policy chains, and optionally applications.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunExport(cmd.Context(), cfg)
		},
	}
	config.BindExportFlags(cmd.Flags(), &cfg)

	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Export Analytics API hit traffic into stats/ only",
		Long:  `List products from the Admin API and fetch Analytics API usage metrics into stats/ without running a full configuration export.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunMetrics(cmd.Context(), cfg)
		},
	}
	config.BindAuthFlags(metricsCmd.Flags(), &cfg.AuthConfig)
	metricsCmd.Flags().StringVar(&cfg.OutDir, "output", cfg.OutDir, "export output directory")
	metricsCmd.Flags().IntVar(&cfg.MaxConcurrent, "concurrency", cfg.MaxConcurrent, "max concurrent Analytics API requests")
	metricsCmd.Flags().BoolVar(&cfg.Quiet, "quiet", cfg.Quiet, "suppress progress output on stderr")
	metricsCmd.Flags().BoolVar(&cfg.Verbose, "verbose", cfg.Verbose, "show detailed progress")
	config.BindMetricsFlags(metricsCmd.Flags(), &cfg)
	cmd.AddCommand(metricsCmd)

	cmd.Version = version.Version
	return cmd
}

func RunExport(ctx context.Context, cfg config.ExportConfig) error {
	if err := cfg.ValidateOutput(); err != nil {
		return err
	}
	adminURL, err := config.NormalizeAdminURL(cfg.AdminURL)
	if err != nil {
		return err
	}

	rep := progress.New(os.Stderr, cfg.Quiet, cfg.Verbose)

	client, err := admin.NewClient(admin.Options{
		BaseURL:       adminURL,
		Token:         cfg.Token,
		HTTPClient:    config.NewHTTPClient(cfg.InsecureTLS, 60*time.Second),
		MaxConcurrent: cfg.MaxConcurrent,
	})
	if err != nil {
		return err
	}

	tb, err := export.NewToolbox(export.ToolboxOptions{
		Runtime:      cfg.ToolboxRuntime,
		Image:        cfg.ToolboxImage,
		NativeBinary: cfg.ToolboxNativeBinary,
		CertFile:     cfg.ToolboxCertFile,
		Insecure:     cfg.InsecureTLS,
		OnVerbose:    verboseHook(rep, cfg.Verbose),
	})
	if err != nil {
		return err
	}
	exporter := export.NewService(client, tb)
	metricsSince := cfg.MetricsSince
	metricsUntil := cfg.MetricsUntil
	metricsGranularity := cfg.ResolvedMetricsGranularity()
	metricsMetric := cfg.ResolvedMetricsMetric()
	if cfg.IncludeMetrics {
		since, until, err := cfg.ResolveMetricsWindow()
		if err != nil {
			return err
		}
		metricsSince, metricsUntil = since, until
	}
	manifest, err := exporter.Export(ctx, export.Options{
		AdminURL:            adminURL,
		Token:               cfg.Token,
		OutDir:              cfg.OutDir,
		IncludeApplications: cfg.IncludeApplications,
		IncludeMetrics:      cfg.IncludeMetrics,
		MetricsSince:        metricsSince,
		MetricsUntil:        metricsUntil,
		MetricsGranularity:  metricsGranularity,
		MetricsMetric:       metricsMetric,
		MetricsHTTPClient:   config.NewHTTPClient(cfg.InsecureTLS, 60*time.Second),
		RedactSecrets:       cfg.RedactSecrets,
		Strict:              cfg.Strict,
		MaxConcurrent:       cfg.MaxConcurrent,
		PerPage:             cfg.PerPage,
		Reporter:            rep,
	})
	if err != nil {
		if manifest != nil && manifest.Incomplete {
			_, _ = fmt.Fprintf(os.Stderr, "export incomplete: wrote partial output to %s (%d warnings)\n", cfg.OutDir, len(manifest.Warnings))
		}
		return err
	}
	rep.Done(formatExportSummary(manifest, cfg.OutDir))
	return nil
}

func formatExportSummary(m *output.Manifest, outDir string) string {
	if m == nil {
		return fmt.Sprintf("Export complete → %s", outDir)
	}
	parts := []string{
		fmt.Sprintf("%d products", m.ProductCount),
		fmt.Sprintf("%d backends", m.BackendCount),
	}
	if m.ApplicationCount > 0 {
		parts = append(parts, fmt.Sprintf("%d applications", m.ApplicationCount))
	}
	summary := fmt.Sprintf("Export complete: %s → %s", strings.Join(parts, ", "), outDir)
	if len(m.Warnings) > 0 {
		summary += fmt.Sprintf(" (%d warnings)", len(m.Warnings))
	}
	return summary
}

func verboseHook(rep progress.Reporter, verbose bool) func(string) {
	if !verbose {
		return nil
	}
	return func(msg string) {
		rep.Verbose(msg)
	}
}

func Execute() int {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
