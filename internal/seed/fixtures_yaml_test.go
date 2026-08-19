package seed

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadFixturesFile_sampleCatalog(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "testdata", "sample-catalog.yaml")

	backends, account, products, coverage, err := LoadFixturesFile(path)
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
