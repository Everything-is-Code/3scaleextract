package export

import (
	"context"
	"encoding/json"
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
				"proxy": map[string]any{"api_key": "secret-value"},
			},
		},
	}
	toolbox := &mockToolbox{outputs: map[string][]byte{
		"api": []byte("credentials:\n  client_secret: yaml-secret\n"),
	}}

	svc := NewService(client, toolbox)
	_, err := svc.Export(context.Background(), Options{
		AdminURL:      "https://tenant.example.com",
		Token:         "tok",
		OutDir:        dir,
		RedactSecrets: true,
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
	yamlData, err := os.ReadFile(filepath.Join(dir, "products", "api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(yamlData), "yaml-secret") {
		t.Fatalf("yaml still has secret: %s", yamlData)
	}
}
