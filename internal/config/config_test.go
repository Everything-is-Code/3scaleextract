package config

import (
	"errors"
	"testing"
)

func TestValidateAuthMissingURL(t *testing.T) {
	cfg := ExportConfig{Token: "tok"}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingAdminURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateAuthMissingToken(t *testing.T) {
	cfg := ExportConfig{AdminURL: "https://tenant.example.com"}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateOutputRequiresDir(t *testing.T) {
	cfg := ExportConfig{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
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
