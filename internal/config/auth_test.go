package config

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
)

func TestLoadAuthFromEnv(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "https://tenant.example.com")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "secret")

	cfg := LoadAuthFromEnv()
	if cfg.AdminURL != "https://tenant.example.com" {
		t.Fatalf("AdminURL = %q", cfg.AdminURL)
	}
	if cfg.Token != "secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
}

func TestAuthValidateAuthMissingURL(t *testing.T) {
	cfg := AuthConfig{Token: "tok"}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingAdminURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestAuthValidateAuthMissingToken(t *testing.T) {
	cfg := AuthConfig{AdminURL: "https://tenant.example.com"}
	err := cfg.ValidateAuth()
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("err = %v", err)
	}
}

func TestBindAuthFlags(t *testing.T) {
	cfg := AuthConfig{}
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindAuthFlags(fs, &cfg)

	if err := fs.Parse([]string{
		"--admin-url", "https://flag.example.com",
		"--token", "flag-token",
		"--insecure",
	}); err != nil {
		t.Fatal(err)
	}
	if cfg.AdminURL != "https://flag.example.com" {
		t.Fatalf("AdminURL = %q", cfg.AdminURL)
	}
	if cfg.Token != "flag-token" {
		t.Fatalf("Token = %q", cfg.Token)
	}
	if !cfg.InsecureTLS {
		t.Fatal("expected InsecureTLS")
	}
}
