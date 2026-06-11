package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Everything-is-Code/3scaleextract/internal/export"
	"github.com/spf13/pflag"
)

const (
	DefaultPerPage       = 500
	DefaultMaxConcurrent = 4
)

var (
	ErrMissingAdminURL = errors.New("admin URL is required: set THREESCALE_ADMIN_URL or use --admin-url")
	ErrMissingToken    = errors.New("access token is required: set THREESCALE_ACCESS_TOKEN or use --token")
)

type ExportConfig struct {
	AuthConfig
	OutDir              string
	IncludeApplications bool
	RedactSecrets       bool
	PerPage             int
	MaxConcurrent       int
	ToolboxImage        string
	ToolboxRuntime      string
	ToolboxNativeBinary string
	ToolboxCertFile     string
}

func LoadExportFromEnv() (ExportConfig, error) {
	cfg := ExportConfig{
		AuthConfig:            LoadAuthFromEnv(),
		OutDir:                strings.TrimSpace(os.Getenv("THREESCALE_OUTPUT_DIR")),
		PerPage:               DefaultPerPage,
		MaxConcurrent:         DefaultMaxConcurrent,
		ToolboxImage:          strings.TrimSpace(os.Getenv("THREESCALE_TOOLBOX_IMAGE")),
		ToolboxRuntime:        strings.TrimSpace(os.Getenv("THREESCALE_TOOLBOX_RUNTIME")),
		ToolboxNativeBinary:   strings.TrimSpace(os.Getenv("THREESCALE_TOOLBOX_BINARY")),
		ToolboxCertFile:       strings.TrimSpace(os.Getenv("THREESCALE_TOOLBOX_TLS_CERT")),
	}
	return cfg, cfg.ValidateAuth()
}

func (c ExportConfig) ValidateOutput() error {
	if err := c.ValidateAuth(); err != nil {
		return err
	}
	if strings.TrimSpace(c.OutDir) == "" {
		return errors.New("output directory is required: use --output")
	}
	return nil
}

func BindExportFlags(fs *pflag.FlagSet, cfg *ExportConfig) {
	BindAuthFlags(fs, &cfg.AuthConfig)
	fs.StringVar(&cfg.OutDir, "output", cfg.OutDir, "export output directory")
	fs.BoolVar(&cfg.IncludeApplications, "include-applications", cfg.IncludeApplications, "export applications and linked accounts")
	fs.BoolVar(&cfg.RedactSecrets, "redact-secrets", cfg.RedactSecrets, "mask API keys and OIDC secrets in output")
	fs.IntVar(&cfg.PerPage, "per-page", cfg.PerPage, "Admin API page size (max 500)")
	fs.IntVar(&cfg.MaxConcurrent, "concurrency", cfg.MaxConcurrent, "max concurrent Admin API requests")
	fs.StringVar(&cfg.ToolboxImage, "toolbox-image", cfg.ToolboxImage, "3scale toolbox container image (Red Hat official)")
	fs.StringVar(&cfg.ToolboxRuntime, "toolbox-runtime", cfg.ToolboxRuntime, "container runtime for toolbox (docker or podman; auto-detects if empty)")
	fs.StringVar(&cfg.ToolboxNativeBinary, "toolbox-binary", cfg.ToolboxNativeBinary, "optional local 3scale binary instead of container")
	fs.StringVar(&cfg.ToolboxCertFile, "toolbox-tls-cert", cfg.ToolboxCertFile, "CA/cert file mounted into toolbox container for TLS")
	if cfg.ToolboxImage == "" {
		cfg.ToolboxImage = export.DefaultToolboxImage
	}
}

func NormalizeAdminURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return "", ErrMissingAdminURL
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", fmt.Errorf("admin URL must start with http:// or https://: %q", raw)
	}
	return raw, nil
}
