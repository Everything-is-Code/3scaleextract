package visualize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteReportBundle(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := WriteReport(tenant, dir, false); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		"index.md",
		"products-catalog.md",
		"backends.md",
		"products/seed_alpha.md",
		"products/seed_multi_backend.md",
		"applications.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}

	index, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	indexText := string(index)
	for _, want := range []string{"API Key", "OIDC", "flowchart LR", "seed_alpha", "shared_payments", "Product catalog", "products-catalog.md"} {
		if !strings.Contains(indexText, want) {
			t.Fatalf("index.md missing %q", want)
		}
	}
	if strings.Contains(indexText, "topology.html") {
		t.Fatal("index should not link topology.html when includeHTMLLink is false")
	}
	if !strings.Contains(indexText, `-->|"/payments"|`) && !strings.Contains(indexText, `-->|"/payments"| shared_payments`) {
		if !strings.Contains(indexText, "/payments") {
			t.Fatalf("index.md missing backend path edge: %s", indexText)
		}
	}

	product, err := os.ReadFile(filepath.Join(dir, "products", "seed_multi_backend.md"))
	if err != nil {
		t.Fatal(err)
	}
	productText := string(product)
	if !strings.Contains(productText, "OIDC configuration unavailable") {
		t.Fatalf("expected missing OIDC note: %s", productText)
	}
	if !strings.Contains(productText, "edge_limit") || !strings.Contains(productText, "url_rewriting") {
		t.Fatalf("expected policy names: %s", productText)
	}
}

func TestWriteReportSkipsApplicationsWhenNotIncluded(t *testing.T) {
	dir := t.TempDir()
	copyExportTree(t, filepath.Join("testdata", "export-minimal"), dir)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{
  "schema_version": "1.0",
  "exported_at": "2026-06-05T12:00:00Z",
  "admin_url": "https://tenant-admin.example.com",
  "product_count": 2,
  "backend_count": 2,
  "include_applications": false,
  "incomplete": false
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	tenant, err := LoadExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteReport(tenant, out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "applications.md")); !os.IsNotExist(err) {
		t.Fatalf("applications.md should not exist: %v", err)
	}
}

func TestWriteReportRedactionPassthrough(t *testing.T) {
	dir := t.TempDir()
	copyExportTree(t, filepath.Join("testdata", "export-minimal"), dir)
	oidc := []byte(`{
  "oidc": {
    "issuer_type": "keycloak",
    "issuer_endpoint": "***REDACTED***"
  }
}
`)
	if err := os.WriteFile(filepath.Join(dir, "products", "seed_multi_backend", "oidc_configuration.json"), oidc, 0o644); err != nil {
		t.Fatal(err)
	}

	tenant, err := LoadExport(dir)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteReport(tenant, out, false); err != nil {
		t.Fatal(err)
	}

	product, err := os.ReadFile(filepath.Join(out, "products", "seed_multi_backend.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(product), "***REDACTED***") {
		t.Fatal("expected redaction marker in product page")
	}
}

func TestWriteReportIncompleteBanner(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	tenant.Manifest.Incomplete = true

	out := t.TempDir()
	if err := WriteReport(tenant, out, false); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "incomplete") {
		t.Fatal("expected incomplete banner")
	}
}

func TestWriteReportManifestWarnings(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	tenant.Manifest.Incomplete = true
	tenant.Manifest.Warnings = []string{
		"product payments: skipped oidc_configuration.json (/services/10/proxy/oidc_configuration: unrecoverable)",
	}

	out := t.TempDir()
	if err := WriteReport(tenant, out, false); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(index)
	if !strings.Contains(text, "## Export warnings") {
		t.Fatalf("missing warnings section: %s", text)
	}
	if !strings.Contains(text, "oidc_configuration.json") {
		t.Fatalf("missing warning detail: %s", text)
	}
}

func TestWriteReportValidation(t *testing.T) {
	if err := WriteReport(nil, t.TempDir(), false); err == nil {
		t.Fatal("expected error for nil tenant")
	}
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteReport(tenant, "", false); err == nil {
		t.Fatal("expected error for empty output dir")
	}
}

func TestWriteReportApplicationsJoin(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteReport(tenant, out, false); err != nil {
		t.Fatal(err)
	}

	apps, err := os.ReadFile(filepath.Join(out, "applications.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(apps)
	if !strings.Contains(text, "Alpha App") || !strings.Contains(text, "seed_alpha") {
		t.Fatalf("applications join failed: %s", text)
	}
}
