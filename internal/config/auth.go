package config

import (
	"os"
	"strings"

	"github.com/spf13/pflag"
)

// AuthConfig holds credentials shared by export and seed tools.
type AuthConfig struct {
	AdminURL    string
	Token       string
	InsecureTLS bool
}

func LoadAuthFromEnv() AuthConfig {
	return AuthConfig{
		AdminURL: strings.TrimSpace(os.Getenv("THREESCALE_ADMIN_URL")),
		Token:    strings.TrimSpace(os.Getenv("THREESCALE_ACCESS_TOKEN")),
	}
}

func BindAuthFlags(fs *pflag.FlagSet, cfg *AuthConfig) {
	fs.StringVar(&cfg.AdminURL, "admin-url", cfg.AdminURL, "3scale Admin Portal base URL")
	fs.StringVar(&cfg.Token, "token", cfg.Token, "3scale Personal Access Token")
	fs.BoolVar(&cfg.InsecureTLS, "insecure", cfg.InsecureTLS, "skip TLS certificate verification")
}

func (c AuthConfig) ValidateAuth() error {
	if c.AdminURL == "" {
		return ErrMissingAdminURL
	}
	if c.Token == "" {
		return ErrMissingToken
	}
	return nil
}
