package visualize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCanvasDataFromMinimalFixture(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	data := BuildCanvasData(tenant)
	if data.Manifest["product_count"] != 2 {
		t.Fatalf("product_count = %v, want 2", data.Manifest["product_count"])
	}
	if len(data.Products) != 2 {
		t.Fatalf("products = %d, want 2", len(data.Products))
	}
	if len(data.Backends) != 2 {
		t.Fatalf("backends = %d, want 2", len(data.Backends))
	}

	var canvasAlpha *CanvasProduct
	for i := range data.Products {
		if data.Products[i].Name == "seed_alpha" {
			canvasAlpha = &data.Products[i]
			break
		}
	}
	if canvasAlpha == nil {
		t.Fatal("seed_alpha missing from canvas products")
	}
	if canvasAlpha.Auth != "API Key" {
		t.Fatalf("auth = %q", canvasAlpha.Auth)
	}
	if len(canvasAlpha.Edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(canvasAlpha.Edges))
	}
	if len(canvasAlpha.PolicyNames) != 1 || canvasAlpha.PolicyNames[0] != "cors" {
		t.Fatalf("policies = %v", canvasAlpha.PolicyNames)
	}
	if len(canvasAlpha.Apps) != 1 {
		t.Fatalf("apps = %d, want 1", len(canvasAlpha.Apps))
	}
}

func TestWriteCanvasTSXFromMinimalFixture(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(t.TempDir(), "topology.canvas.tsx")
	if err := WriteCanvasTSX(tenant, out); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Contains(text, "__CANVAS_DATA_JSON__") {
		t.Fatal("canvas placeholder was not replaced")
	}
	payload := extractCanvasJSON(t, text)
	if !json.Valid([]byte(payload)) {
		t.Fatal("embedded canvas DATA is not valid JSON")
	}
	for _, want := range []string{"TopologyCanvas", "seed_alpha", "cursor/canvas", "Policy names"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in canvas output", want)
		}
	}
}

func TestPolicyNamesFromProxyFile(t *testing.T) {
	raw := []byte(`{"proxy":{"policies_config":[{"name":"apicast"},{"name":"headers"}]}}`)
	got := policyNamesFromProxyFile(raw)
	if len(got) != 2 || got[0].Name != "apicast" || got[1].Name != "headers" {
		t.Fatalf("policies = %+v", got)
	}
}

func extractCanvasJSON(t *testing.T, content string) string {
	t.Helper()
	const prefix = "const DATA = "
	start := strings.Index(content, prefix)
	if start < 0 {
		t.Fatal("DATA assignment not found")
	}
	start += len(prefix)
	rest := content[start:]
	end := strings.Index(rest, " as TopologyData;")
	if end < 0 {
		t.Fatal("TopologyData suffix not found")
	}
	return rest[:end]
}
