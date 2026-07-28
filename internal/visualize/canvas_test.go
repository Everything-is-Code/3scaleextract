package visualize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTopologyDataFromMinimalFixture(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}

	data := BuildTopologyData(tenant)
	if data.Manifest["product_count"] != 2 {
		t.Fatalf("product_count = %v, want 2", data.Manifest["product_count"])
	}
	if len(data.Products) != 2 {
		t.Fatalf("products = %d, want 2", len(data.Products))
	}
	if len(data.Backends) != 2 {
		t.Fatalf("backends = %d, want 2", len(data.Backends))
	}

	var alpha *TopologyProduct
	for i := range data.Products {
		if data.Products[i].Name == "seed_alpha" {
			alpha = &data.Products[i]
			break
		}
	}
	if alpha == nil {
		t.Fatal("seed_alpha missing from topology products")
	}
	if alpha.Auth != "API Key" {
		t.Fatalf("auth = %q", alpha.Auth)
	}
	if len(alpha.Edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(alpha.Edges))
	}
	if len(alpha.PolicyNames) != 1 || alpha.PolicyNames[0] != "cors" {
		t.Fatalf("policies = %v", alpha.PolicyNames)
	}
	if len(alpha.Apps) != 1 {
		t.Fatalf("apps = %d, want 1", len(alpha.Apps))
	}
}

func TestBuildTopologyDataEmptySharedMarshalsAsArray(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	for i := range tenant.Products {
		tenant.Products[i].BackendUsages = nil
	}
	data := BuildTopologyData(tenant)
	if data.Shared == nil {
		t.Fatal("Shared should be empty slice, not nil")
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"shared":null`) {
		t.Fatalf("shared must marshal as [], got null in %s", raw)
	}
	if !strings.Contains(string(raw), `"shared":[]`) {
		t.Fatalf("shared must marshal as [], payload=%s", raw)
	}
}

func TestBuildTopologyDataIncludesOneToOneBackends(t *testing.T) {
	tenant, err := LoadExport(filepath.Join("testdata", "export-minimal"))
	if err != nil {
		t.Fatal(err)
	}
	data := BuildTopologyData(tenant)

	byBackend := map[string]TopologyShared{}
	for _, s := range data.Shared {
		byBackend[s.Backend] = s
	}
	oneToOne, ok := byBackend["billing_api"]
	if !ok {
		t.Fatal("billing_api (1:1) missing from Shared")
	}
	if oneToOne.Count != 1 {
		t.Fatalf("billing_api count = %d, want 1", oneToOne.Count)
	}
	shared, ok := byBackend["shared_payments"]
	if !ok {
		t.Fatal("shared_payments missing from Shared")
	}
	if shared.Count != 2 {
		t.Fatalf("shared_payments count = %d, want 2", shared.Count)
	}
	if data.Shared[0].Backend != "shared_payments" {
		t.Fatalf("first Shared = %q, want shared_payments (highest count first)", data.Shared[0].Backend)
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
	for _, want := range []string{"TopologyCanvas", "seed_alpha", "cursor/canvas", "Policy names", "domainShowPercent", "Toggle", "Show percentages", "Most referenced backends", "Backend usage detail"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in canvas output", want)
		}
	}
}

func TestPolicyNamesFromProxyFile(t *testing.T) {
	raw := []byte(`{"proxy":{"policies_config":[{"name":"apicast","enabled":true},{"name":"headers","enabled":true},{"name":"camel","enabled":false}]}}`)
	got := policyNamesFromProxyFile(raw)
	if len(got) != 1 || got[0].Name != "headers" {
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
