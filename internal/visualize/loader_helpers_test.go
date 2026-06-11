package visualize

import (
	"encoding/json"
	"testing"
)

func TestParseBackendUsagesArrayFormat(t *testing.T) {
	data := []byte(`[
	  {"backend_usage":{"backend_id":495,"path":"/legacy/seed_alpha/create_compensation"}}
	]`)
	usages := parseBackendUsages(data)
	if len(usages) != 1 {
		t.Fatalf("len = %d", len(usages))
	}
	if usages[0].BackendID != 495 || usages[0].Path != "/legacy/seed_alpha/create_compensation" {
		t.Fatalf("usage = %+v", usages[0])
	}
}

func TestParseBackendUsagesWrappedFormat(t *testing.T) {
	data := []byte(`{"backend_usages":[{"backend_usage":{"backend_id":100,"path":"/payments"}}]}`)
	usages := parseBackendUsages(data)
	if len(usages) != 1 || usages[0].BackendID != 100 {
		t.Fatalf("usage = %+v", usages)
	}
}

func TestParseOIDCConfigurationNested(t *testing.T) {
	data := []byte(`{"oidc":{"issuer_type":"keycloak","issuer_endpoint":"https://sso.example.com/realm/demo"}}`)
	cfg := parseOIDCConfiguration(data)
	if cfg == nil {
		t.Fatal("expected OIDC config")
	}
	if cfg.IssuerType != "keycloak" {
		t.Fatalf("IssuerType = %q", cfg.IssuerType)
	}
	if cfg.IssuerEndpoint != "https://sso.example.com/realm/demo" {
		t.Fatalf("IssuerEndpoint = %q", cfg.IssuerEndpoint)
	}
}

func TestParseOIDCConfigurationFlat(t *testing.T) {
	data := []byte(`{"issuer_type":"keycloak","issuer_endpoint":"https://sso.example.com","client_id":"demo"}`)
	cfg := parseOIDCConfiguration(data)
	if cfg == nil {
		t.Fatal("expected OIDC config")
	}
	if cfg.IssuerType != "keycloak" {
		t.Fatalf("IssuerType = %q", cfg.IssuerType)
	}
	if cfg.Fields["client_id"] != `"demo"` {
		t.Fatalf("Fields = %#v", cfg.Fields)
	}
}

func TestParseOIDCConfigurationInvalid(t *testing.T) {
	if cfg := parseOIDCConfiguration([]byte(`not-json`)); cfg != nil {
		t.Fatalf("expected nil, got %#v", cfg)
	}
	if cfg := parseOIDCConfiguration([]byte(`{"other":"value"}`)); cfg != nil {
		t.Fatalf("expected nil, got %#v", cfg)
	}
}

func TestStringifyMap(t *testing.T) {
	raw := map[string]json.RawMessage{
		"issuer_type": json.RawMessage(`"oidc"`),
		"enabled":     json.RawMessage(`true`),
	}
	out := stringifyMap(raw)
	if out["issuer_type"] != `"oidc"` {
		t.Fatalf("issuer_type = %q", out["issuer_type"])
	}
	if out["enabled"] != "true" {
		t.Fatalf("enabled = %q", out["enabled"])
	}
}

func TestStringJSONAndIntJSON(t *testing.T) {
	if got := stringJSON(json.RawMessage(`"hello"`)); got != "hello" {
		t.Fatalf("stringJSON = %q", got)
	}
	if got := stringJSON(json.RawMessage(`123`)); got != "" {
		t.Fatalf("stringJSON invalid = %q", got)
	}
	if got := intJSON(json.RawMessage(`42`)); got != 42 {
		t.Fatalf("intJSON = %d", got)
	}
	if got := intJSON(json.RawMessage(`"nope"`)); got != 0 {
		t.Fatalf("intJSON invalid = %d", got)
	}
}

func TestTenantLookupHelpersNil(t *testing.T) {
	var tenant *Tenant
	if tenant.BackendByID(1) != nil {
		t.Fatal("expected nil backend")
	}
	if tenant.ProductByServiceID(1) != nil {
		t.Fatal("expected nil product")
	}
	if tenant.PlanByID(1) != nil {
		t.Fatal("expected nil plan")
	}
}

func TestAuthLabel(t *testing.T) {
	cases := map[string]string{
		"api_key":              "API Key",
		"api_key_and_app_id":   "API Key",
		"app_key":              "App ID + App Key",
		"app_id_and_app_key":   "App ID + App Key",
		"oidc":                 "OIDC",
		"":                     "unknown",
		"user_key":              "API Key",
		"userkey":               "API Key",
	}
	for in, want := range cases {
		if got := authLabel(in); got != want {
			t.Fatalf("authLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
