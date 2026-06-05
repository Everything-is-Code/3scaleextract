package seed

import (
	"encoding/json"
	"testing"
)

func TestBuildPolicyChain(t *testing.T) {
	raw, err := buildPolicyChain([]string{"cors"})
	if err != nil {
		t.Fatal(err)
	}
	var chain []policyEntry
	if err := json.Unmarshal([]byte(raw), &chain); err != nil {
		t.Fatal(err)
	}
	if len(chain) != 2 {
		t.Fatalf("chain len = %d", len(chain))
	}
	if chain[0].Name != "cors" || chain[1].Name != "apicast" {
		t.Fatalf("unexpected chain: %+v", chain)
	}
}
