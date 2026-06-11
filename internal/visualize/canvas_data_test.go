package visualize

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestWriteCanvasData regenerates embedded canvas DATA when WRITE_CANVAS_DATA=1.
func TestWriteCanvasData(t *testing.T) {
	if os.Getenv("WRITE_CANVAS_DATA") == "" {
		t.Skip("set WRITE_CANVAS_DATA=1 to regenerate topology canvas data")
	}

	root := os.Getenv("CANVAS_EXPORT_ROOT")
	if root == "" {
		root = "/tmp/example-export"
	}

	tenant, err := LoadExport(root)
	if err != nil {
		t.Fatal(err)
	}

	backendNames := make([]string, 0, len(tenant.Backends))
	backendIndex := map[string]int{}
	for _, b := range tenant.Backends {
		backendIndex[b.SystemName] = len(backendNames)
		backendNames = append(backendNames, b.SystemName)
	}

	refs := map[string]map[string]struct{}{}
	products := make([]map[string]any, 0, len(tenant.Products))
	for _, p := range tenant.Products {
		es := make([][2]any, 0, len(p.BackendUsages))
		for _, u := range p.BackendUsages {
			if u.Backend == nil {
				continue
			}
			idx := backendIndex[u.Backend.SystemName]
			es = append(es, [2]any{idx, u.Path})
			if refs[u.Backend.SystemName] == nil {
				refs[u.Backend.SystemName] = map[string]struct{}{}
			}
			refs[u.Backend.SystemName][p.SystemName] = struct{}{}
		}
		products = append(products, map[string]any{
			"n": p.SystemName,
			"c": canvasCategoryKey(p.SystemName),
			"a": authLabel(p.AuthType),
			"e": es,
		})
	}

	type sharedEntry struct {
		B string   `json:"b"`
		N int      `json:"n"`
		P []string `json:"p"`
	}
	var shared []sharedEntry
	for b, ps := range refs {
		if len(ps) < 2 {
			continue
		}
		names := make([]string, 0, len(ps))
		for name := range ps {
			names = append(names, name)
		}
		sort.Strings(names)
		limit := len(names)
		if limit > 6 {
			limit = 6
		}
		shared = append(shared, sharedEntry{B: b, N: len(names), P: names[:limit]})
	}
	sort.Slice(shared, func(i, j int) bool {
		if shared[i].N != shared[j].N {
			return shared[i].N > shared[j].N
		}
		return shared[i].B < shared[j].B
	})
	if len(shared) > 12 {
		shared = shared[:12]
	}

	catCounts := map[string]int{"I": 0, "B": 0, "S": 0, "P": 0}
	for _, p := range products {
		catCounts[p["c"].(string)]++
	}

	out := map[string]any{
		"m":         tenant.Manifest,
		"cat":       map[string]string{"I": "Integration (-IS)", "B": "Business API", "S": "SAP", "P": "Platform / misc"},
		"catCounts": catCounts,
		"backends":  backendNames,
		"products":  products,
		"shared":    shared,
	}

	outPath := os.Getenv("CANVAS_DATA_OUT")
	if outPath == "" {
		outPath = "/tmp/canvas-data.json"
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes)", outPath, len(data))
}

func canvasCategoryKey(name string) string {
	switch {
	case strings.HasSuffix(name, "-IS") || name == "Industrial-IS-int":
		return "I"
	case strings.Contains(name, "SAP") || strings.HasPrefix(name, "SalidaSap"):
		return "S"
	case name == "dummy-for-alerts" || name == "llm" || name == "ddrr":
		return "P"
	default:
		return "B"
	}
}
