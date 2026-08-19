package seed

import (
	"testing"
)

func TestLoadFixturesFile_rhclCatalog(t *testing.T) {
	const path = "/home/fmeneses/Documents/demos/migration-toolkit-rhcl/testdata/seed/catalog.yaml"
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
	if len(products) < 18 {
		t.Fatalf("products=%d want >= 18", len(products))
	}
	if coverage["rhcl_seed_cors"] == nil {
		t.Fatal("missing rhcl_seed_cors coverage")
	}
	foundEdge := false
	foundRetry := false
	foundMulti := false
	for _, p := range products {
		if p.SystemName == "rhcl_seed_multi_backend" {
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
		t.Fatal("expected rhcl_seed_multi_backend product in catalog")
	}
}
