package visualize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProductsCatalog(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteProductsCatalog(tenant, out); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(out, "products-catalog.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{
		"# Product catalog",
		"| Product | Category | Auth | Backends | Apps | Policies | Policy names |",
		"seed_alpha",
		"seed_multi_backend",
		"cors",
		"edge_limit",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("products-catalog.md missing %q", want)
		}
	}
}

func TestWriteReportIncludesHTMLLinkWhenRequested(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteReport(tenant, out, true); err != nil {
		t.Fatal(err)
	}
	index, err := os.ReadFile(filepath.Join(out, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "topology.html") {
		t.Fatal("expected topology.html link in index when includeHTMLLink is true")
	}
}
