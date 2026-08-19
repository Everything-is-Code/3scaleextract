package seed

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func sampleCatalogPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(thisFile), "testdata", "sample-catalog.yaml")
}

func TestLoadFixturesFile_sampleCatalog(t *testing.T) {
	backends, account, products, coverage, err := LoadFixturesFile(sampleCatalogPath(t))
	if err != nil {
		t.Fatalf("LoadFixturesFile: %v", err)
	}
	if account.Username == "" {
		t.Fatal("empty account username")
	}
	if len(backends) < 1 {
		t.Fatalf("backends=%d want >= 1", len(backends))
	}
	if len(products) < 4 {
		t.Fatalf("products=%d want >= 4", len(products))
	}
	if coverage["test_cors"] == nil {
		t.Fatal("missing test_cors coverage")
	}
	foundEdge := false
	foundRetry := false
	foundMulti := false
	for _, p := range products {
		if p.SystemName == "test_multi_backend" {
			foundMulti = true
		}
		for _, name := range p.PolicyNames {
			if name == "edge_limiting" {
				foundEdge = true
			}
			if name == "retry" {
				foundRetry = true
			}
		}
	}
	if !foundEdge {
		t.Fatal("expected edge_limiting product in catalog")
	}
	if !foundRetry {
		t.Fatal("expected retry product in catalog")
	}
	if !foundMulti {
		t.Fatal("expected test_multi_backend product in catalog")
	}
}

func TestLoadFixturesFile_defaultsAndOIDC(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.yaml")
	content := `
backends:
  - system_name: be
    name: BE
    private_endpoint: "https://example.com:443"
products:
  - system_name: bare
    name: Bare
    backend_refs: [be]
  - system_name: with_oidc
    name: With OIDC
    auth_mode: oidc
    backend_refs: [be]
    oidc:
      realm_url: "https://sso.example.com/auth/realms/demo"
      zync_client_id: zync
      zync_client_secret: secret
      redirect_url: "https://app.example.com/cb"
      standard_flow: true
      service_accounts: true
    applications:
      - name: oidc-app
        plan: basic
        redirect_url: "https://app.example.com/cb"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	backends, account, products, _, err := LoadFixturesFile(path)
	if err != nil {
		t.Fatalf("LoadFixturesFile: %v", err)
	}
	if len(backends) != 1 {
		t.Fatalf("backends=%d", len(backends))
	}
	if account.OrgName != "Seed Organization" || account.Username != "seed_user" {
		t.Fatalf("account defaults not applied: %+v", account)
	}
	if len(products) != 2 {
		t.Fatalf("products=%d", len(products))
	}
	if products[0].AuthMode != "api_key" {
		t.Fatalf("default auth_mode=%q", products[0].AuthMode)
	}
	if len(products[0].Plans) != 1 || products[0].Plans[0].SystemName != "basic" {
		t.Fatalf("default plan missing: %+v", products[0].Plans)
	}
	if products[1].OIDC == nil || products[1].OIDC.RealmURL == "" {
		t.Fatal("expected OIDC on second product")
	}
	if len(products[1].Applications) != 1 || products[1].Applications[0].RedirectURL == "" {
		t.Fatalf("expected OIDC app redirect: %+v", products[1].Applications)
	}
}

func TestLoadFixturesFile_errors(t *testing.T) {
	if _, _, _, _, err := LoadFixturesFile(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
	dir := t.TempDir()
	badYAML := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(badYAML, []byte(":::"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := LoadFixturesFile(badYAML); err == nil {
		t.Fatal("expected error for invalid yaml")
	}
	noBackends := filepath.Join(dir, "no-backends.yaml")
	if err := os.WriteFile(noBackends, []byte("products:\n  - system_name: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := LoadFixturesFile(noBackends); err == nil {
		t.Fatal("expected error for empty backends")
	}
	noProducts := filepath.Join(dir, "no-products.yaml")
	if err := os.WriteFile(noProducts, []byte("backends:\n  - system_name: b\n    name: B\n    private_endpoint: https://x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := LoadFixturesFile(noProducts); err == nil {
		t.Fatal("expected error for empty products")
	}
}
