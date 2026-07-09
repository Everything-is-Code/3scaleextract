package export

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportStrictFailsOnMissingSidecar(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 10, SystemName: "payments"}}},
			},
			"/backend_apis": backendListResponse{},
			"/policies":     map[string]any{},
			"/services/10/proxy": map[string]any{
				"proxy": map[string]any{"auth_type": "oidc"},
			},
			"/services/10/proxy/policies":    map[string]any{"policies": []any{}},
			"/services/10/application_plans": map[string]any{"plans": []any{}},
			"/services/10/backend_usages":    map[string]any{"backend_usages": []any{}},
			"/services/10/metrics":           map[string]any{"metrics": []any{}},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{"payments": []byte("apiVersion: v1\n")}}

	svc := NewService(client, toolbox)
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
		Strict:   true,
	})
	if err == nil {
		t.Fatal("expected strict export error")
	}
	if !errors.Is(err, ErrStrictSidecar) {
		t.Fatalf("err = %v", err)
	}
	if manifest == nil || !manifest.Incomplete {
		t.Fatal("expected incomplete manifest on strict failure")
	}
}

func TestVerifyExportLayout(t *testing.T) {
	dir := t.TempDir()
	client := defaultScopeMockClient()
	toolbox := &mockToolbox{outputs: map[string][]byte{"payments": []byte("apiVersion: v1\nkind: Product\n")}}

	svc := NewService(client, toolbox)
	if _, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
	}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExport(dir); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyExportWithMetrics(t *testing.T) {
	statsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stats/services/10/usage.json" {
			_, _ = w.Write([]byte(`{"periods":[]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer statsSrv.Close()

	dir := t.TempDir()
	svc := NewService(defaultScopeMockClient(), &mockToolbox{
		outputs: map[string][]byte{"payments": []byte("apiVersion: v1\nkind: Product\n")},
	})
	if _, err := svc.Export(context.Background(), Options{
		AdminURL:           statsSrv.URL,
		Token:              "tok",
		OutDir:             dir,
		IncludeMetrics:     true,
		MetricsSince:       "2026-01-01",
		MetricsUntil:       "2026-01-31",
		MetricsGranularity: "day",
		MetricsMetric:      "hits",
	}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyExport(dir); err != nil {
		t.Fatal(err)
	}
}

func TestExportGoldenLayout(t *testing.T) {
	dir := t.TempDir()
	svc := NewService(defaultScopeMockClient(), &mockToolbox{
		outputs: map[string][]byte{"payments": []byte("apiVersion: v1\nkind: Product\n")},
	})
	if _, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := ListExportPaths(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantRaw, err := os.ReadFile(filepath.Join("testdata", "golden-export-layout.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, line := range strings.Split(strings.TrimSpace(string(wantRaw)), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			want = append(want, line)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("paths = %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("path[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func defaultScopeMockClient() *mockClient {
	return &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 10, SystemName: "payments"}}},
			},
			"/backend_apis": backendListResponse{
				BackendAPIs: []backendEntry{{BackendAPI: []byte(`{"system_name":"billing-backend"}`)}},
			},
			"/policies": map[string]any{"apicast": map[string]any{"version": "builtin"}},
			"/services/10/proxy": map[string]any{
				"proxy": map[string]any{"auth_user_key": "key", "auth_type": "api_key"},
			},
			"/services/10/proxy/policies":            map[string]any{"policies": []any{}},
			"/services/10/proxy/oidc_configuration":    map[string]any{},
			"/services/10/application_plans":           map[string]any{"plans": []any{}},
			"/services/10/backend_usages":              map[string]any{"backend_usages": []any{}},
			"/services/10/metrics":                     map[string]any{"metrics": []any{}},
		},
	}
}

func TestVerifyNoCleartextSecretsClean(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clean.json"), []byte(`{"client_secret":"***REDACTED***"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyNoCleartextSecrets(dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyNoCleartextSecretsFailsWithPath(t *testing.T) {
	dir := t.TempDir()
	rel := filepath.Join("products", "api", "proxy.json")
	if err := os.MkdirAll(filepath.Join(dir, "products", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(`{"proxy":{"client_secret":"not-redacted"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := VerifyNoCleartextSecrets(dir)
	if err == nil {
		t.Fatal("expected cleartext gate error")
	}
	if !strings.Contains(err.Error(), "products/api/proxy.json") {
		t.Fatalf("error = %v, want path-qualified message", err)
	}
}
