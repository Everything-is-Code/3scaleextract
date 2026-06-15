package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactJSONRemovesCleartextSecrets(t *testing.T) {
	raw := []byte(`{"proxy":{"api_key":"abc123","endpoint":"https://example.com"},"client_secret":"shh"}`)
	out, err := RedactBytes(".json", raw)
	if err != nil {
		t.Fatal(err)
	}
	if ContainsCleartextSecret(out) {
		t.Fatalf("expected redacted output, got %s", out)
	}
	if !strings.Contains(string(out), "***REDACTED***") {
		t.Fatalf("missing redacted marker: %s", out)
	}
}

func TestRedactYAMLRemovesCleartextSecrets(t *testing.T) {
	raw := []byte("oidc:\n  client_secret: super-secret\n  issuer: https://idp.example.com\n")
	out, err := RedactBytes(".yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "super-secret") {
		t.Fatalf("secret still present: %s", out)
	}
}

func TestRedactDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.json")
	if err := os.WriteFile(path, []byte(`{"user_key":"live-key"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RedactDirectory(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if ContainsCleartextSecret(data) {
		t.Fatalf("file still has secret: %s", data)
	}
}

func TestContainsCleartextSecretNestedArray(t *testing.T) {
	raw := []byte(`{"items":[{"user_key":"live"}]}`)
	if !ContainsCleartextSecret(raw) {
		t.Fatal("expected secret detected")
	}
}

func TestContainsCleartextSecretYAML(t *testing.T) {
	raw := []byte("client_secret: still-there\n")
	if !ContainsCleartextSecret(raw) {
		t.Fatal("expected yaml secret detected")
	}
}

func TestContainsCleartextSecretAlreadyRedacted(t *testing.T) {
	raw := []byte(`{"client_secret":"***REDACTED***"}`)
	if ContainsCleartextSecret(raw) {
		t.Fatal("expected no cleartext secret")
	}
}

func TestRedactBytesUnsupportedExtension(t *testing.T) {
	_, err := RedactBytes(".txt", []byte("plain"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRedactJSONNestedStructures(t *testing.T) {
	raw := []byte(`{"nested":{"items":[{"api_key":"secret"}]}}`)
	out, err := RedactBytes(".json", raw)
	if err != nil {
		t.Fatal(err)
	}
	if ContainsCleartextSecret(out) {
		t.Fatalf("nested secret not redacted: %s", out)
	}
}

func TestRedactDirectorySkipsNonJSONYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RedactDirectory(dir); err != nil {
		t.Fatal(err)
	}
}

func TestRedactExtendedSensitiveKeys(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		key  string
	}{
		{name: "provider_verification_key", raw: `{"provider_verification_key":"pvk-live"}`, key: "provider_verification_key"},
		{name: "client_id", raw: `{"client_id":"cid-99"}`, key: "client_id"},
		{name: "app_id", raw: `{"app_id":"aid-42"}`, key: "app_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RedactBytes(".json", []byte(tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			if ContainsCleartextSecret(out) {
				t.Fatalf("expected redacted output, got %s", out)
			}
			if !strings.Contains(string(out), "***REDACTED***") {
				t.Fatalf("missing redacted marker: %s", out)
			}
			if strings.Contains(string(out), tt.key+`":"`) && !strings.Contains(string(out), "***REDACTED***") {
				t.Fatalf("cleartext value still present: %s", out)
			}
		})
	}
}

func TestRedactExtendedSensitiveKeysYAML(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		secret string
	}{
		{name: "provider_verification_key", raw: "application:\n  provider_verification_key: pvk-yaml\n", secret: "pvk-yaml"},
		{name: "client_id", raw: "credentials:\n  client_id: cid-yaml\n", secret: "cid-yaml"},
		{name: "app_id", raw: "credentials:\n  app_id: aid-yaml\n", secret: "aid-yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RedactBytes(".yaml", []byte(tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(out), tt.secret) {
				t.Fatalf("secret still present: %s", out)
			}
			if !strings.Contains(string(out), "***REDACTED***") {
				t.Fatalf("missing redacted marker: %s", out)
			}
		})
	}
}

func TestRedactIssuerStripsUserinfoJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "oidc_issuer_endpoint",
			raw:  `{"oidc_issuer_endpoint":"https://zync-user:zync-pass@idp.example.com/realms/demo"}`,
			want: "https://idp.example.com/realms/demo",
		},
		{
			name: "issuer_endpoint",
			raw:  `{"issuer_endpoint":"https://user:pass@sso.example.com/realms/prod"}`,
			want: "https://sso.example.com/realms/prod",
		},
		{
			name: "clean issuer unchanged",
			raw:  `{"issuer_endpoint":"https://sso.example.com/realms/prod"}`,
			want: "https://sso.example.com/realms/prod",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RedactBytes(".json", []byte(tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), tt.want) {
				t.Fatalf("expected %q in output, got %s", tt.want, out)
			}
			if strings.Contains(string(out), "zync-user") || strings.Contains(string(out), "zync-pass") ||
				strings.Contains(string(out), "user:pass") {
				t.Fatalf("userinfo still present: %s", out)
			}
			if ContainsCleartextSecret(out) {
				t.Fatalf("issuer userinfo detected as cleartext: %s", out)
			}
		})
	}
}

func TestRedactIssuerStripsUserinfoYAML(t *testing.T) {
	raw := []byte("oidc:\n  oidc_issuer_endpoint: https://zync-user:zync-pass@idp.example.com/realms/demo\n")
	out, err := RedactBytes(".yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "zync-user") || strings.Contains(string(out), "zync-pass") {
		t.Fatalf("userinfo still present: %s", out)
	}
	if !strings.Contains(string(out), "https://idp.example.com/realms/demo") {
		t.Fatalf("expected stripped host URL, got %s", out)
	}
}

func TestRedactPreservesAuthProxyFlags(t *testing.T) {
	raw := []byte(`{"proxy":{"auth_user_key":"user_key","auth_app_id":"app_id","auth_app_key":"app_key","api_key":"secret"}}`)
	out, err := RedactBytes(".json", raw)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, flag := range []string{`"auth_user_key": "user_key"`, `"auth_app_id": "app_id"`, `"auth_app_key": "app_key"`} {
		if !strings.Contains(s, flag) {
			t.Fatalf("auth flag altered or missing %q in %s", flag, out)
		}
	}
	if !strings.Contains(s, "***REDACTED***") {
		t.Fatalf("api_key not redacted: %s", out)
	}
}

func TestContainsCleartextSecretYAMLIssuerUserinfo(t *testing.T) {
	raw := []byte("oidc:\n  oidc_issuer_endpoint: https://user:pass@host/realms/demo\n")
	if !ContainsCleartextSecret(raw) {
		t.Fatal("expected issuer userinfo detected in YAML")
	}
	stripped := []byte("oidc:\n  oidc_issuer_endpoint: https://host/realms/demo\n")
	if ContainsCleartextSecret(stripped) {
		t.Fatal("expected no cleartext after issuer strip")
	}
}

func TestContainsCleartextSecretExtendedKeys(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantHit bool
	}{
		{name: "provider_verification_key cleartext", raw: `{"provider_verification_key":"live"}`, wantHit: true},
		{name: "client_id redacted", raw: `{"client_id":"***REDACTED***"}`, wantHit: false},
		{name: "issuer userinfo", raw: `{"oidc_issuer_endpoint":"https://u:p@host/path"}`, wantHit: true},
		{name: "issuer stripped", raw: `{"oidc_issuer_endpoint":"https://host/path"}`, wantHit: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCleartextSecret([]byte(tt.raw))
			if got != tt.wantHit {
				t.Fatalf("ContainsCleartextSecret() = %v, want %v for %s", got, tt.wantHit, tt.raw)
			}
		})
	}
}
