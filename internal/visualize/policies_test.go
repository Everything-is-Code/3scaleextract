package visualize

import (
	"testing"
)

func TestVisiblePoliciesFromConfig(t *testing.T) {
	disabled := false
	enabled := true

	raw := []byte(`[
		{"name":"apicast","enabled":true},
		{"name":"headers","enabled":true},
		{"name":"camel","enabled":false},
		{"name":"cors"}
	]`)
	got := parsePolicyConfigJSONArray(raw)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Name != "headers" || got[1].Name != "cors" {
		t.Fatalf("policies = %+v", got)
	}

	entries := []policyConfigEntry{
		{Name: "apicast", Enabled: &enabled},
		{Name: "edge_limit", Enabled: &enabled},
		{Name: "url_rewriting", Enabled: &disabled},
	}
	got = visiblePoliciesFromEntries(entries)
	if len(got) != 1 || got[0].Name != "edge_limit" {
		t.Fatalf("filtered = %+v", got)
	}
}

func TestParsePoliciesPoliciesConfigRoot(t *testing.T) {
	raw := []byte(`{
		"policies_config": [
			{"name":"edge_limit","enabled":true},
			{"name":"camel","enabled":false},
			{"name":"apicast","enabled":true}
		]
	}`)
	got := parsePolicies(raw)
	if len(got) != 1 || got[0].Name != "edge_limit" {
		t.Fatalf("policies = %+v", got)
	}
}

func TestParsePoliciesWrappedFormatStillWorks(t *testing.T) {
	raw := []byte(`{
		"policies": [
			{"policy": {"name": "cors", "version": "builtin"}},
			{"policy": {"name": "apicast", "version": "builtin"}}
		]
	}`)
	got := parsePolicies(raw)
	if len(got) != 1 || got[0].Name != "cors" {
		t.Fatalf("policies = %+v", got)
	}
}
