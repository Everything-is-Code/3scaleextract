package cli

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fmeneses/3scaleextract/internal/admin"
	"github.com/fmeneses/3scaleextract/internal/config"
	"github.com/fmeneses/3scaleextract/internal/export"
	"github.com/fmeneses/3scaleextract/internal/version"
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

	httpClient := &http.Client{Timeout: 60 * time.Second}
	if cfg.InsecureTLS {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	client, err := admin.NewClient(admin.Options{
		BaseURL:       adminURL,
		Token:         cfg.Token,
		HTTPClient:    httpClient,
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
	})
	if err != nil {
		return err
	}
	exporter := export.NewService(client, tb)
	manifest, err := exporter.Export(ctx, export.Options{
		AdminURL:            adminURL,
		Token:               cfg.Token,
		OutDir:              cfg.OutDir,
		IncludeApplications: cfg.IncludeApplications,
		RedactSecrets:       cfg.RedactSecrets,
		MaxConcurrent:       cfg.MaxConcurrent,
		PerPage:             cfg.PerPage,
	})
	if err != nil {
		if manifest != nil && manifest.Incomplete {
			_, _ = fmt.Fprintf(os.Stderr, "export incomplete: wrote partial output to %s\n", cfg.OutDir)
		}
		return err
	}
	return nil
}

func Execute() int {
	if err := NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
