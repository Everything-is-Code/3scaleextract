package visualize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInferAuthTypeFromProxyLegacyFields(t *testing.T) {
	cases := []struct {
		name     string
		userKey  string
		appID    string
		appKey   string
		oidcURL  string
		want     string
	}{
		{name: "api key booleans", userKey: "true", appID: "false", want: "api_key"},
		{name: "app id booleans", userKey: "false", appID: "true", appKey: "app_key", want: "app_id_and_app_key"},
		{name: "copec param names", userKey: "user_key", appID: "app_id", appKey: "app_key", want: "app_id_and_app_key"},
		{name: "oidc issuer", oidcURL: "https://sso.example.com/realm/demo", want: "oidc"},
		{name: "explicit auth type", userKey: "true", appID: "false", want: "api_key"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferAuthTypeFromProxy("", tc.userKey, tc.appID, tc.appKey, "keycloak", tc.oidcURL, nil)
			if got != tc.want {
				t.Fatalf("inferAuthTypeFromProxy() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInferAuthTypeFromProxyDefaultCredentialsPolicy(t *testing.T) {
	raw := []byte(`[
	  {"name":"apicast","enabled":true,"configuration":{}},
	  {"name":"default_credentials","enabled":true,"configuration":{"auth_type":"user_key"}}
	]`)
	got := inferAuthTypeFromProxy("", "app_id", "app_key", "user_key", "", "", raw)
	if got != "api_key" {
		t.Fatalf("got %q, want api_key", got)
	}
}

func TestReadAuthTypeFromYAMLListProduct(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "B2B-IS.yaml")
	content := `apiVersion: v1
kind: List
items:
- apiVersion: capabilities.3scale.net/v1beta1
  kind: Product
  spec:
    deployment:
      apicastHosted:
        authentication:
          appKeyAppID:
            appID: app_id
            appKey: app_key
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readAuthTypeFromYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "app_id_and_app_key" {
		t.Fatalf("got %q, want app_id_and_app_key", got)
	}
}

func TestReadAuthTypeFromYAMLUserKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Comisiones.yaml")
	content := `apiVersion: v1
kind: List
items:
- kind: Product
  spec:
    deployment:
      apicastHosted:
        authentication:
          userkey:
            authUserKey: user_key
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readAuthTypeFromYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "api_key" {
		t.Fatalf("got %q, want api_key", got)
	}
}

func TestLoadExportCopecStyleProxy(t *testing.T) {
	dir := t.TempDir()
	writeMinimalManifest(t, dir, false)
	writeJSON(t, filepath.Join(dir, "backends", "billing.json"), map[string]any{
		"id": 1, "system_name": "billing", "name": "billing",
	})
	productDir := filepath.Join(dir, "products", "demo")
	if err := os.MkdirAll(productDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, filepath.Join(productDir, "proxy.json"), map[string]any{
		"proxy": map[string]any{
			"service_id":    10,
			"auth_app_id":   "app_id",
			"auth_app_key":  "app_key",
			"auth_user_key": "user_key",
			"endpoint":      "https://demo.example.com",
		},
	})
	yaml := `apiVersion: v1
kind: List
items:
- kind: Product
  spec:
    name: Demo Product
    systemName: demo
    deployment:
      apicastHosted:
        authentication:
          userkey:
            authUserKey: user_key
`
	if err := os.WriteFile(filepath.Join(dir, "products", "demo.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	tenant, err := LoadExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	product := findProduct(t, tenant, "demo")
	if product.AuthType != "api_key" {
		t.Fatalf("AuthType = %q, want api_key", product.AuthType)
	}
}

func writeMinimalManifest(t *testing.T, dir string, includeApps bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "backends"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "products"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schema_version":        "1.0",
		"exported_at":           "2026-06-05T12:00:00Z",
		"admin_url":             "https://tenant-admin.example.com",
		"product_count":         1,
		"backend_count":         1,
		"include_applications":  includeApps,
		"incomplete":            false,
	}
	writeJSON(t, filepath.Join(dir, "manifest.json"), manifest)
}

func writeJSON(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
