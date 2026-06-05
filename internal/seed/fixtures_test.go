package seed

import (
	"context"
	"testing"
)

func TestDefaultFixturesCoverage(t *testing.T) {
	backends, account, products := DefaultFixtures()
	if len(backends) != 3 {
		t.Fatalf("backends = %d", len(backends))
	}
	if account.Username == "" {
		t.Fatal("missing account")
	}
	if len(products) != 4 {
		t.Fatalf("products = %d", len(products))
	}
	for _, p := range products {
		if _, ok := CoverageMatrix[p.SystemName]; !ok {
			t.Fatalf("missing coverage for %q", p.SystemName)
		}
	}
}

func TestDryRunSeeder(t *testing.T) {
	s := NewSeeder(nil, Options{DryRun: true})
	result, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Backends) != 3 {
		t.Fatalf("backends = %d", len(result.Backends))
	}
}
