package export

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
)

type mockClient struct {
	responses map[string]any
	pages     map[string][][]json.RawMessage
}

func (m *mockClient) Get(_ context.Context, path string, dst any) error {
	val, ok := m.responses[path]
	if !ok {
		return admin.ErrUnrecoverable
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}

func (m *mockClient) GetAllPages(_ context.Context, path string, _ int, fn func([]json.RawMessage) error) error {
	pages := m.pages[path]
	for _, page := range pages {
		if err := fn(page); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockClient) PostForm(_ context.Context, _ string, _ url.Values, _ any) error {
	return nil
}

func (m *mockClient) PutForm(_ context.Context, _ string, _ url.Values, _ any) error {
	return nil
}

func (m *mockClient) PutJSON(_ context.Context, _ string, _ any, _ any) error {
	return nil
}

func (m *mockClient) Delete(_ context.Context, _ string) error {
	return nil
}

type mockToolbox struct {
	outputs map[string][]byte
}

func (m *mockToolbox) ExportProduct(_ context.Context, _, _, systemName string) ([]byte, error) {
	out, ok := m.outputs[systemName]
	if !ok {
		return nil, ErrToolboxFailed
	}
	return out, nil
}

func TestExportDefaultScope(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 10, SystemName: "payments"}}},
			},
			"/backend_apis": backendListResponse{
				BackendAPIs: []backendEntry{{BackendAPI: json.RawMessage(`{"system_name":"billing-backend"}`)}},
			},
			"/policies": map[string]any{"apicast": map[string]any{"version": "builtin"}},
			"/services/10/proxy": map[string]any{
				"proxy": map[string]any{"auth_user_key": "key", "auth_type": "api_key"},
			},
			"/services/10/proxy/policies": map[string]any{"policies": []any{}},
			"/services/10/proxy/oidc_configuration": map[string]any{},
			"/services/10/application_plans": map[string]any{"plans": []any{}},
			"/services/10/backend_usages": map[string]any{"backend_usages": []any{}},
			"/services/10/metrics": map[string]any{"metrics": []any{}},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{"payments": []byte("apiVersion: v1\nkind: Product\n")}}

	svc := NewService(client, toolbox)
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
		PerPage:  500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ProductCount != 1 {
		t.Fatalf("ProductCount = %d", manifest.ProductCount)
	}
	if _, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "products", "payments.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "products", "payments", "proxy.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "backends", "billing-backend.json")); err != nil {
		t.Fatal(err)
	}
}

func TestExportIncludeMetrics(t *testing.T) {
	statsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/stats/services/10/usage.json" {
			_, _ = w.Write([]byte(`{"periods":[{"since":"2026-01-01","until":"2026-01-02","values":{"hits":42}}]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer statsSrv.Close()

	dir := t.TempDir()
	client := defaultScopeMockClient()
	toolbox := &mockToolbox{outputs: map[string][]byte{"payments": []byte("apiVersion: v1\nkind: Product\n")}}

	svc := NewService(client, toolbox)
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL:           statsSrv.URL,
		Token:              "tok",
		OutDir:             dir,
		IncludeMetrics:     true,
		MetricsSince:       "2026-01-01",
		MetricsUntil:       "2026-01-31",
		MetricsGranularity: "day",
		MetricsMetric:      "hits",
		PerPage:            500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.IncludeMetrics {
		t.Fatal("expected include_metrics on manifest")
	}
	if manifest.MetricsSince != "2026-01-01" || manifest.MetricsUntil != "2026-01-31" {
		t.Fatalf("metrics window = %q..%q", manifest.MetricsSince, manifest.MetricsUntil)
	}
	if _, err := os.Stat(filepath.Join(dir, "stats", "query.json")); err != nil {
		t.Fatal(err)
	}
	hitsPath := filepath.Join(dir, "stats", "products", "payments", "hits.json")
	data, err := os.ReadFile(hitsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"hits"`) || !strings.Contains(string(data), "42") {
		t.Fatalf("unexpected hits payload: %s", data)
	}
	if err := VerifyExport(dir); err != nil {
		t.Fatal(err)
	}
}

func TestExportIncludeApplications(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{},
			"/backend_apis": backendListResponse{},
			"/policies": map[string]any{},
			"/accounts/42": map[string]any{"account": map[string]any{"id": 42}},
		},
		pages: map[string][][]json.RawMessage{
			"/applications": {
				{json.RawMessage(`{"application":{"id":1,"account_id":42}}`)},
				{json.RawMessage(`{"application":{"id":2,"account_id":42}}`)},
			},
		},
	}

	svc := NewService(client, &mockToolbox{outputs: map[string][]byte{}})
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL:            "https://tenant.example.com",
		Token:               "tok",
		OutDir:              dir,
		IncludeApplications: true,
		PerPage:             500,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ApplicationCount != 2 {
		t.Fatalf("ApplicationCount = %d", manifest.ApplicationCount)
	}
	if _, err := os.Stat(filepath.Join(dir, "applications", "page-1.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "accounts", "42.json")); err != nil {
		t.Fatal(err)
	}
}

func TestExportWithoutRedactPreservesSecrets(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 1, SystemName: "api"}}},
			},
			"/backend_apis": backendListResponse{},
			"/policies":     map[string]any{},
			"/services/1/proxy": map[string]any{
				"proxy": map[string]any{"api_key": "secret-value"},
			},
			"/services/1/proxy/policies": map[string]any{"policies": []any{}},
			"/services/1/proxy/oidc_configuration": map[string]any{
				"oidc_configuration": map[string]any{
					"oidc_issuer_endpoint": "https://zync-user:zync-pass@idp.example.com/realms/demo",
					"client_id":            "oidc-client-id",
				},
			},
			"/services/1/application_plans": map[string]any{"plans": []any{}},
			"/services/1/backend_usages":    map[string]any{"backend_usages": []any{}},
			"/services/1/metrics":           map[string]any{"metrics": []any{}},
			"/accounts/7": map[string]any{
				"account": map[string]any{"id": 7},
			},
		},
		pages: map[string][][]json.RawMessage{
			"/applications": {
				{json.RawMessage(`{"application":{"id":1,"account_id":7,"provider_verification_key":"pvk-live","client_id":"app-client","app_id":"app-99"}}`)},
			},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{
		"api": []byte("credentials:\n  client_secret: yaml-secret\n  client_id: yaml-client\n"),
	}}

	svc := NewService(client, toolbox)
	_, err := svc.Export(context.Background(), Options{
		AdminURL:            "https://tenant.example.com",
		Token:               "tok",
		OutDir:              dir,
		IncludeApplications: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	proxyData, err := os.ReadFile(filepath.Join(dir, "products", "api", "proxy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proxyData), "secret-value") {
		t.Fatalf("expected cleartext api_key in proxy: %s", proxyData)
	}

	oidcData, err := os.ReadFile(filepath.Join(dir, "products", "api", "oidc_configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	oidcStr := string(oidcData)
	if !strings.Contains(oidcStr, "zync-user") || !strings.Contains(oidcStr, "oidc-client-id") {
		t.Fatalf("expected cleartext OIDC fields: %s", oidcData)
	}

	appData, err := os.ReadFile(filepath.Join(dir, "applications", "page-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	appStr := string(appData)
	for _, want := range []string{"pvk-live", "app-client", "app-99"} {
		if !strings.Contains(appStr, want) {
			t.Fatalf("expected cleartext %q in applications: %s", want, appData)
		}
	}

	yamlData, err := os.ReadFile(filepath.Join(dir, "products", "api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	yamlStr := string(yamlData)
	if !strings.Contains(yamlStr, "yaml-secret") || !strings.Contains(yamlStr, "yaml-client") {
		t.Fatalf("expected cleartext YAML credentials: %s", yamlData)
	}
	if strings.Contains(yamlStr, "***REDACTED***") {
		t.Fatalf("unexpected redaction marker without --redact-secrets: %s", yamlData)
	}
}

func TestExportRedactSecrets(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 1, SystemName: "api"}}},
			},
			"/backend_apis": backendListResponse{},
			"/policies":     map[string]any{},
			"/services/1/proxy": map[string]any{
				"proxy": map[string]any{
					"api_key":       "secret-value",
					"auth_user_key": "user_key",
					"auth_app_id":   "app_id",
					"auth_app_key":  "app_key",
				},
			},
			"/services/1/proxy/policies": map[string]any{"policies": []any{}},
			"/services/1/proxy/oidc_configuration": map[string]any{
				"oidc_configuration": map[string]any{
					"oidc_issuer_endpoint": "https://zync-user:zync-pass@idp.example.com/realms/demo",
					"client_id":            "oidc-client-id",
				},
			},
			"/services/1/application_plans": map[string]any{"plans": []any{}},
			"/services/1/backend_usages":    map[string]any{"backend_usages": []any{}},
			"/services/1/metrics":           map[string]any{"metrics": []any{}},
			"/accounts/7": map[string]any{
				"account": map[string]any{"id": 7},
			},
		},
		pages: map[string][][]json.RawMessage{
			"/applications": {
				{json.RawMessage(`{"application":{"id":1,"account_id":7,"provider_verification_key":"pvk-live","client_id":"app-client","app_id":"app-99"}}`)},
			},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{
		"api": []byte("credentials:\n  client_secret: yaml-secret\n  client_id: yaml-client\n"),
	}}

	svc := NewService(client, toolbox)
	_, err := svc.Export(context.Background(), Options{
		AdminURL:            "https://tenant.example.com",
		Token:               "tok",
		OutDir:              dir,
		RedactSecrets:       true,
		IncludeApplications: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	proxyData, err := os.ReadFile(filepath.Join(dir, "products", "api", "proxy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ContainsCleartextSecret(proxyData) {
		t.Fatalf("proxy still has secret: %s", proxyData)
	}
	proxyStr := string(proxyData)
	for _, want := range []string{`"auth_user_key": "user_key"`, `"auth_app_id": "app_id"`, `"auth_app_key": "app_key"`} {
		if !strings.Contains(proxyStr, want) {
			t.Fatalf("auth flag missing %q in %s", want, proxyData)
		}
	}

	oidcData, err := os.ReadFile(filepath.Join(dir, "products", "api", "oidc_configuration.json"))
	if err != nil {
		t.Fatal(err)
	}
	oidcStr := string(oidcData)
	if strings.Contains(oidcStr, "zync-user") || strings.Contains(oidcStr, "zync-pass") {
		t.Fatalf("issuer userinfo still present: %s", oidcData)
	}
	if !strings.Contains(oidcStr, "https://idp.example.com/realms/demo") {
		t.Fatalf("expected stripped issuer host in %s", oidcData)
	}
	if strings.Contains(oidcStr, "oidc-client-id") {
		t.Fatalf("client_id not redacted: %s", oidcData)
	}

	appData, err := os.ReadFile(filepath.Join(dir, "applications", "page-1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if ContainsCleartextSecret(appData) {
		t.Fatalf("applications still have secrets: %s", appData)
	}

	yamlData, err := os.ReadFile(filepath.Join(dir, "products", "api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(yamlData), "yaml-secret") || strings.Contains(string(yamlData), "yaml-client") {
		t.Fatalf("yaml still has secret: %s", yamlData)
	}

	if err := VerifyNoCleartextSecrets(dir); err != nil {
		t.Fatalf("post-export verify gate failed: %v", err)
	}
}

func TestExportRedactSecretsVerifyGateFailsOnResidual(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "products", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "products", "api", "proxy.json"),
		[]byte(`{"proxy":{"client_secret":"not-redacted"}}`),
		0o644,
	); err != nil {
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

func TestExportIncompleteOnBackendFailure(t *testing.T) {
	dir := t.TempDir()
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{},
		},
	}
	svc := NewService(client, &mockToolbox{outputs: map[string][]byte{}})
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if manifest == nil || !manifest.Incomplete {
		t.Fatal("expected incomplete manifest")
	}
}

func TestAppendYAMLNewline(t *testing.T) {
	if got := string(appendYAMLNewline([]byte("x"))); got != "x\n" {
		t.Fatalf("got %q", got)
	}
	if got := string(appendYAMLNewline([]byte("x\n"))); got != "x\n" {
		t.Fatalf("got %q", got)
	}
}

func TestExportRecordsWarningsOnSkippedSidecars(t *testing.T) {
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
			"/services/10/proxy/policies":      map[string]any{"policies": []any{}},
			"/services/10/application_plans":   map[string]any{"plans": []any{}},
			"/services/10/backend_usages":      map[string]any{"backend_usages": []any{}},
			"/services/10/metrics":             map[string]any{"metrics": []any{}},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{"payments": []byte("apiVersion: v1\n")}}

	svc := NewService(client, toolbox)
	manifest, err := svc.Export(context.Background(), Options{
		AdminURL: "https://tenant.example.com",
		Token:    "tok",
		OutDir:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.Incomplete {
		t.Fatal("expected incomplete manifest")
	}
	if len(manifest.Warnings) != 1 {
		t.Fatalf("Warnings = %#v, want 1 entry", manifest.Warnings)
	}
	if !strings.Contains(manifest.Warnings[0], "oidc_configuration.json") {
		t.Fatalf("warning = %q", manifest.Warnings[0])
	}
	if _, err := os.Stat(filepath.Join(dir, "products", "payments", "proxy.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "products", "payments", "oidc_configuration.json")); !os.IsNotExist(err) {
		t.Fatalf("oidc_configuration.json should be absent: %v", err)
	}
}
