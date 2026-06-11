package config

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
)

func TestValidateAuthMissingURL(t *testing.T) {
	cfg := ExportConfig{AuthConfig: AuthConfig{Token: "tok"}}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingAdminURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateAuthMissingToken(t *testing.T) {
	cfg := ExportConfig{AuthConfig: AuthConfig{AdminURL: "https://tenant.example.com"}}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateOutputRequiresDir(t *testing.T) {
	cfg := ExportConfig{
		AuthConfig: AuthConfig{
			AdminURL: "https://tenant.example.com",
			Token:    "tok",
		},
	}
	err := cfg.ValidateOutput()
	if err == nil || err.Error() != "output directory is required: use --output" {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeAdminURL(t *testing.T) {
	got, err := NormalizeAdminURL("https://tenant.example.com/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://tenant.example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeAdminURLEmpty(t *testing.T) {
	_, err := NormalizeAdminURL("")
	if !errors.Is(err, ErrMissingAdminURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeAdminURLInvalidScheme(t *testing.T) {
	_, err := NormalizeAdminURL("tenant.example.com")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateOutputPropagatesAuthError(t *testing.T) {
	cfg := ExportConfig{OutDir: "./out"}
	err := cfg.ValidateOutput()
	if !errors.Is(err, ErrMissingAdminURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadExportFromEnv(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "https://tenant.example.com")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "secret")

	cfg, err := LoadExportFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminURL != "https://tenant.example.com" {
		t.Fatalf("AdminURL = %q", cfg.AdminURL)
	}
	if cfg.Token != "secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
}

func TestBindExportFlags(t *testing.T) {
	cfg := ExportConfig{PerPage: DefaultPerPage, MaxConcurrent: DefaultMaxConcurrent}
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindExportFlags(fs, &cfg)

	if err := fs.Parse([]string{
		"--admin-url", "https://export.example.com",
		"--token", "export-token",
		"--output", "./out",
		"--include-applications",
		"--redact-secrets",
		"--insecure",
		"--toolbox-runtime", "docker",
		"--toolbox-binary", "/usr/bin/3scale",
	}); err != nil {
		t.Fatal(err)
	}
	if cfg.AdminURL != "https://export.example.com" {
		t.Fatalf("AdminURL = %q", cfg.AdminURL)
	}
	if cfg.OutDir != "./out" {
		t.Fatalf("OutDir = %q", cfg.OutDir)
	}
	if !cfg.IncludeApplications || !cfg.RedactSecrets || !cfg.InsecureTLS {
		t.Fatal("expected bool flags set")
	}
	if cfg.ToolboxRuntime != "docker" || cfg.ToolboxNativeBinary != "/usr/bin/3scale" {
		t.Fatalf("toolbox cfg = %#v", cfg)
	}
}

func TestBindExportFlagsDefaultToolboxImage(t *testing.T) {
	cfg := ExportConfig{}
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindExportFlags(fs, &cfg)
	if cfg.ToolboxImage == "" {
		t.Fatal("expected default toolbox image")
	}
}
