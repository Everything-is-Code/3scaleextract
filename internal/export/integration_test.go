//go:build integration

package export_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
	"github.com/Everything-is-Code/3scaleextract/internal/export"
)

func integrationEnv() (adminURL, token, outDir string, ok bool) {
	adminURL = os.Getenv("THREESCALE_ADMIN_URL")
	token = os.Getenv("THREESCALE_ACCESS_TOKEN")
	outDir = os.Getenv("THREESCALE_OUTPUT_DIR")
	ok = adminURL != "" && token != "" && outDir != ""
	return adminURL, token, outDir, ok
}

func integrationHTTPClient() *http.Client {
	client := &http.Client{Timeout: 60 * time.Second}
	if os.Getenv("THREESCALE_INSECURE_TLS") == "true" {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}
	return client
}

func integrationToolbox(t *testing.T) *export.Toolbox {
	t.Helper()
	tb, err := export.NewToolbox(export.ToolboxOptions{
		Runtime:      os.Getenv("THREESCALE_TOOLBOX_RUNTIME"),
		Image:        os.Getenv("THREESCALE_TOOLBOX_IMAGE"),
		NativeBinary: os.Getenv("THREESCALE_TOOLBOX_BINARY"),
		CertFile:     os.Getenv("THREESCALE_TOOLBOX_TLS_CERT"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func TestIntegrationExport(t *testing.T) {
	adminURL, token, outDir, ok := integrationEnv()
	if !ok {
		t.Skip("set THREESCALE_ADMIN_URL, THREESCALE_ACCESS_TOKEN, THREESCALE_OUTPUT_DIR")
	}

	client, err := admin.NewClient(admin.Options{
		BaseURL:    adminURL,
		Token:      token,
		HTTPClient: integrationHTTPClient(),
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := export.NewService(client, integrationToolbox(t))
	manifest, err := svc.Export(context.Background(), export.Options{
		AdminURL: adminURL,
		Token:    token,
		OutDir:   outDir,
		PerPage:  500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest == nil {
		t.Fatal("expected manifest")
	}
	if manifest.SchemaVersion != "1.0" {
		t.Fatalf("schema_version = %q", manifest.SchemaVersion)
	}
	if err := export.VerifyExport(outDir); err != nil {
		t.Fatal(err)
	}
}
