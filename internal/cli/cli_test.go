package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/config"
)

func TestRunExportMissingAuth(t *testing.T) {
	err := RunExport(context.Background(), config.ExportConfig{OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, config.ErrMissingAdminURL) && !errors.Is(err, config.ErrMissingToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRootIsExportCommand(t *testing.T) {
	root := NewRoot()
	if root.Use != "threescale-export" {
		t.Fatalf("use = %s", root.Use)
	}
	if root.RunE == nil {
		t.Fatal("expected root to run export")
	}
}

func TestExportHelp(t *testing.T) {
	cmd := NewRoot()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteMissingAuth(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	t.Setenv("THREESCALE_OUTPUT_DIR", "")
	if got := Execute(); got != 1 {
		t.Fatalf("Execute() = %d, want 1", got)
	}
}

func TestRunExportHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/admin/api/services.json":
			_, _ = w.Write([]byte(`{"services":[{"service":{"id":10,"system_name":"payments"}}]}`))
		case r.URL.Path == "/admin/api/backend_apis.json":
			_, _ = w.Write([]byte(`{"backend_apis":[{"backend_api":{"system_name":"billing-backend"}}]}`))
		case r.URL.Path == "/admin/api/policies.json":
			_, _ = w.Write([]byte(`{"builtin":{}}`))
		case r.URL.Path == "/admin/api/services/10/proxy.json":
			_, _ = w.Write([]byte(`{"proxy":{"auth_type":"api_key"}}`))
		case r.URL.Path == "/admin/api/services/10/proxy/policies.json":
			_, _ = w.Write([]byte(`{"policies":[]}`))
		case r.URL.Path == "/admin/api/services/10/proxy/oidc_configuration.json":
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/admin/api/services/10/application_plans.json":
			_, _ = w.Write([]byte(`{"plans":[]}`))
		case r.URL.Path == "/admin/api/services/10/backend_usages.json":
			_, _ = w.Write([]byte(`{"backend_usages":[]}`))
		case r.URL.Path == "/admin/api/services/10/metrics.json":
			_, _ = w.Write([]byte(`{"metrics":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	toolboxBin := writeFakeToolboxScript(t, "payments")

	dir := t.TempDir()
	err := RunExport(context.Background(), config.ExportConfig{
		AuthConfig: config.AuthConfig{
			AdminURL: srv.URL,
			Token:    "secret",
		},
		OutDir:              dir,
		ToolboxNativeBinary: toolboxBin,
	})
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		ProductCount int `json:"product_count"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ProductCount != 1 {
		t.Fatalf("ProductCount = %d", manifest.ProductCount)
	}
	if _, err := os.Stat(filepath.Join(dir, "products", "payments.yaml")); err != nil {
		t.Fatal(err)
	}
}

func TestRunExportIncludeMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/admin/api/services.json":
			_, _ = w.Write([]byte(`{"services":[{"service":{"id":10,"system_name":"payments"}}]}`))
		case r.URL.Path == "/admin/api/backend_apis.json":
			_, _ = w.Write([]byte(`{"backend_apis":[{"backend_api":{"system_name":"billing-backend"}}]}`))
		case r.URL.Path == "/admin/api/policies.json":
			_, _ = w.Write([]byte(`{"builtin":{}}`))
		case r.URL.Path == "/admin/api/services/10/proxy.json":
			_, _ = w.Write([]byte(`{"proxy":{"auth_type":"api_key"}}`))
		case r.URL.Path == "/admin/api/services/10/proxy/policies.json":
			_, _ = w.Write([]byte(`{"policies":[]}`))
		case r.URL.Path == "/admin/api/services/10/proxy/oidc_configuration.json":
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/admin/api/services/10/application_plans.json":
			_, _ = w.Write([]byte(`{"plans":[]}`))
		case r.URL.Path == "/admin/api/services/10/backend_usages.json":
			_, _ = w.Write([]byte(`{"backend_usages":[]}`))
		case r.URL.Path == "/admin/api/services/10/metrics.json":
			_, _ = w.Write([]byte(`{"metrics":[]}`))
		case r.URL.Path == "/stats/services/10/usage.json":
			_, _ = w.Write([]byte(`{"periods":[]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	toolboxBin := writeFakeToolboxScript(t, "payments")
	dir := t.TempDir()
	err := RunExport(context.Background(), config.ExportConfig{
		AuthConfig: config.AuthConfig{
			AdminURL: srv.URL,
			Token:    "secret",
		},
		OutDir:              dir,
		IncludeMetrics:      true,
		MetricsSince:        "2026-01-01",
		MetricsUntil:        "2026-01-31",
		ToolboxNativeBinary: toolboxBin,
	})
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		IncludeMetrics bool `json:"include_metrics"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if !manifest.IncludeMetrics {
		t.Fatal("expected include_metrics in manifest")
	}
	if _, err := os.Stat(filepath.Join(dir, "stats", "products", "payments", "hits.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunMetricsHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/admin/api/services.json":
			_, _ = w.Write([]byte(`{"services":[{"service":{"id":10,"system_name":"payments"}}]}`))
		case r.URL.Path == "/stats/services/10/usage.json":
			_, _ = w.Write([]byte(`{"periods":[{"values":{"hits":7}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := RunMetrics(context.Background(), config.ExportConfig{
		AuthConfig: config.AuthConfig{
			AdminURL: srv.URL,
			Token:    "secret",
		},
		OutDir:         dir,
		MetricsSince:   "2026-01-01",
		MetricsUntil:   "2026-01-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "stats", "query.json")); err != nil {
		t.Fatal(err)
	}
	hits, err := os.ReadFile(filepath.Join(dir, "stats", "products", "payments", "hits.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hits), `"hits"`) || !strings.Contains(string(hits), "7") {
		t.Fatalf("unexpected hits: %s", hits)
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("metrics-only export should not write manifest.json: %v", err)
	}
}

func writeFakeToolboxScript(t *testing.T, systemName string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "3scale-fake")
	script := "#!/bin/sh\n" +
		"case \"$4\" in\n" +
		"  " + systemName + ") printf 'apiVersion: v1\\nkind: Product\\n' ;;\n" +
		"  *) exit 1 ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
