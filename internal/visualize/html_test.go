package visualize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTopologyHTMLFromMinimalFixture(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "topology.html")
	if err := WriteTopologyHTML(tenant, out); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Contains(text, htmlDataPlaceholder) {
		t.Fatal("html placeholder was not replaced")
	}
	payload := extractHTMLJSON(t, text)
	if !json.Valid([]byte(payload)) {
		t.Fatal("embedded html DATA is not valid JSON")
	}
	for _, want := range []string{
		"<!DOCTYPE html>",
		"seed_alpha",
		"Product catalog",
		"chart.js",
		"Policy names",
		"domainShowPercent",
		"Show percentages",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in html output", want)
		}
	}
}

func extractHTMLJSON(t *testing.T, content string) string {
	t.Helper()
	const prefix = "const DATA = "
	start := strings.Index(content, prefix)
	if start < 0 {
		t.Fatal("DATA assignment not found")
	}
	start += len(prefix)
	rest := content[start:]
	end := strings.Index(rest, ";\n")
	if end < 0 {
		t.Fatal("DATA statement terminator not found")
	}
	return rest[:end]
}
