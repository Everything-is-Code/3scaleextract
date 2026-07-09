package visualize

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExportMinimal(t *testing.T) {
	root := filepath.Join("testdata", "export-minimal")
	tenant, err := LoadExport(root)
	if err != nil {
		t.Fatal(err)
	}

	if tenant.Manifest.SchemaVersion != "1.0" {
		t.Fatalf("SchemaVersion = %q", tenant.Manifest.SchemaVersion)
	}
	if len(tenant.Products) != 2 {
		t.Fatalf("Products = %d, want 2", len(tenant.Products))
	}
	if len(tenant.Backends) != 2 {
		t.Fatalf("Backends = %d, want 2", len(tenant.Backends))
	}
	if len(tenant.Applications) != 2 {
		t.Fatalf("Applications = %d, want 2", len(tenant.Applications))
	}

	alpha := findProduct(t, tenant, "seed_alpha")
	if alpha == nil {
		t.Fatal("seed_alpha not found")
	}
	if alpha.DisplayName != "Seed Alpha Product" {
		t.Fatalf("DisplayName = %q", alpha.DisplayName)
	}
	if alpha.AuthType != "api_key" {
		t.Fatalf("AuthType = %q", alpha.AuthType)
	}
	if len(alpha.BackendUsages) != 1 {
		t.Fatalf("BackendUsages = %d", len(alpha.BackendUsages))
	}
	if alpha.BackendUsages[0].Backend == nil || alpha.BackendUsages[0].Backend.SystemName != "shared_payments" {
		t.Fatalf("backend usage not resolved: %+v", alpha.BackendUsages[0])
	}
	if len(alpha.Policies) != 1 || alpha.Policies[0].Name != "cors" {
		t.Fatalf("Policies = %+v", alpha.Policies)
	}
}

func TestLoadExportMultiBackendJoins(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	multi := findProduct(t, tenant, "seed_multi_backend")
	if multi == nil {
		t.Fatal("seed_multi_backend not found")
	}
	if len(multi.BackendUsages) != 3 {
		t.Fatalf("BackendUsages = %d, want 3", len(multi.BackendUsages))
	}

	wantPaths := map[string]string{
		"shared_payments": "/payments",
		"billing_api":     "/billing",
	}
	seen := make(map[string]string)
	for _, usage := range multi.BackendUsages {
		if usage.Backend == nil {
			t.Fatalf("unresolved usage: %+v", usage)
		}
		seen[usage.Backend.SystemName] = usage.Path
	}
	if seen["billing_api"] != "/invoices" && seen["billing_api"] != "/billing" {
		t.Fatalf("billing paths = %v", seen)
	}
	if seen["shared_payments"] != wantPaths["shared_payments"] {
		t.Fatalf("shared_payments path = %q", seen["shared_payments"])
	}

	shared := findBackend(t, tenant, "shared_payments")
	if shared == nil {
		t.Fatal("shared_payments backend not found")
	}
	if len(shared.ReferencedBy) != 2 {
		t.Fatalf("ReferencedBy = %v, want 2 products", shared.ReferencedBy)
	}
}

func TestLoadExportMissingOIDC(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	multi := findProduct(t, tenant, "seed_multi_backend")
	if multi == nil {
		t.Fatal("seed_multi_backend not found")
	}
	if multi.AuthType != "oidc" {
		t.Fatalf("AuthType = %q", multi.AuthType)
	}
	if multi.OIDC != nil {
		t.Fatal("expected no OIDC configuration")
	}
	if len(multi.MissingFiles) != 1 || multi.MissingFiles[0] != "oidc_configuration.json" {
		t.Fatalf("MissingFiles = %v", multi.MissingFiles)
	}
}

func TestLoadExportNoApplications(t *testing.T) {
	dir := t.TempDir()
	copyExportTree(t, filepath.Join("testdata", "export-minimal"), dir)

	manifestPath := filepath.Join(dir, "manifest.json")
	updated := []byte(`{
  "schema_version": "1.0",
  "exported_at": "2026-06-05T12:00:00Z",
  "admin_url": "https://tenant-admin.example.com",
  "product_count": 2,
  "backend_count": 2,
  "include_applications": false,
  "incomplete": false
}
`)
	if err := os.WriteFile(manifestPath, updated, 0o644); err != nil {
		t.Fatal(err)
	}

	tenant, err := LoadExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tenant.Applications) != 0 {
		t.Fatalf("Applications = %d, want 0", len(tenant.Applications))
	}
}

func TestLoadExportApplicationJoins(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	for _, app := range tenant.Applications {
		if app.ProductName == "" {
			t.Fatalf("application %d missing ProductName", app.ID)
		}
		if app.PlanName == "" {
			t.Fatalf("application %d missing PlanName", app.ID)
		}
	}

	alphaApp := findApplication(t, tenant, 1)
	if alphaApp == nil || alphaApp.ProductName != "seed_alpha" || alphaApp.PlanName != "Basic" {
		t.Fatalf("alpha app = %+v", alphaApp)
	}
}

func TestLoadExportUnsupportedSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"schema_version":"2.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadExport(dir); err == nil {
		t.Fatal("expected error for unsupported schema")
	}
}

func TestLoadExportMissingManifest(t *testing.T) {
	if _, err := LoadExport(t.TempDir()); err == nil {
		t.Fatal("expected error for missing manifest")
	}
}

func findProduct(t *testing.T, tenant *Tenant, systemName string) *Product {
	t.Helper()
	for i := range tenant.Products {
		if tenant.Products[i].SystemName == systemName {
			return &tenant.Products[i]
		}
	}
	return nil
}

func findBackend(t *testing.T, tenant *Tenant, systemName string) *Backend {
	t.Helper()
	for i := range tenant.Backends {
		if tenant.Backends[i].SystemName == systemName {
			return &tenant.Backends[i]
		}
	}
	return nil
}

func findApplication(t *testing.T, tenant *Tenant, id int) *Application {
	t.Helper()
	for i := range tenant.Applications {
		if tenant.Applications[i].ID == id {
			return &tenant.Applications[i]
		}
	}
	return nil
}

func copyExportTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestStripUTF8BOM(t *testing.T) {
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"proxy":{}}`)...)
	got := stripUTF8BOM(raw)
	if string(got) != `{"proxy":{}}` {
		t.Fatalf("stripUTF8BOM() = %q", got)
	}
}

func TestLoadExportWithUTF8BOM(t *testing.T) {
	root := t.TempDir()
	copyExportTree(t, filepath.Join("testdata", "export-minimal"), root)

	proxyPath := filepath.Join(root, "products", "seed_alpha", "proxy.json")
	data, err := os.ReadFile(proxyPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proxyPath, append([]byte{0xEF, 0xBB, 0xBF}, data...), 0o644); err != nil {
		t.Fatal(err)
	}

	tenant, err := LoadExport(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(tenant.Products) != 2 {
		t.Fatalf("Products = %d, want 2", len(tenant.Products))
	}
}
