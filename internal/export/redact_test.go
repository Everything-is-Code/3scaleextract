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
	if strings.Contains(string(out), "https://example.com") {
		t.Fatalf("endpoint URL should be redacted: %s", out)
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
	}{
		{name: "provider_verification_key", raw: `{"provider_verification_key":"pvk-live"}`},
		{name: "client_id", raw: `{"client_id":"cid-99"}`},
		{name: "app_id", raw: `{"app_id":"aid-42"}`},
		{name: "apikey", raw: `{"apikey":"hdr-key"}`},
		{name: "password", raw: `{"password":"P@ss"}`},
		{name: "secret_token", raw: `{"secret_token":"tok"}`},
		{name: "certificate", raw: `{"certificate":"-----BEGIN CERTIFICATE-----\nMII\n-----END CERTIFICATE-----"}`},
		{name: "certificate_key", raw: `{"certificate_key":"-----BEGIN RSA PRIVATE KEY-----\nMII\n-----END RSA PRIVATE KEY-----"}`},
		{name: "private_endpoint", raw: `{"private_endpoint":"https://upstream.internal"}`},
		{name: "admin_url", raw: `{"admin_url":"https://tenant-admin.example.com"}`},
		{name: "stagingPublicBaseURL", raw: `{"stagingPublicBaseURL":"https://staging.example.com"}`},
		{name: "username", raw: `{"username":"svc-user"}`},
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
		{name: "apikey", raw: "headers:\n  apikey: eyJhdr\n", secret: "eyJhdr"},
		{name: "password", raw: "account:\n  password: SecretPass\n", secret: "SecretPass"},
		{name: "private_endpoint", raw: "backend:\n  private_endpoint: https://backend.internal\n", secret: "https://backend.internal"},
		{name: "issuerEndpoint", raw: "oidc:\n  issuerEndpoint: https://sso.example.com/realms/demo\n", secret: "https://sso.example.com/realms/demo"},
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

func TestRedactYAMLMultilineCertificatePEM(t *testing.T) {
	raw := []byte(`policies:
  - name: tls
    configuration:
      certificate: |
        -----BEGIN CERTIFICATE-----
        MIIFakePublicCertData
        -----END CERTIFICATE-----
      certificate_key: |
        -----BEGIN RSA PRIVATE KEY-----
        MIIFakePrivateKeyData
        -----END RSA PRIVATE KEY-----
`)
	if !ContainsCleartextSecret(raw) {
		t.Fatal("expected PEM cleartext detected before redaction")
	}
	out, err := RedactBytes(".yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, leak := range []string{
		"BEGIN CERTIFICATE",
		"BEGIN RSA PRIVATE KEY",
		"MIIFakePublicCertData",
		"MIIFakePrivateKeyData",
	} {
		if strings.Contains(s, leak) {
			t.Fatalf("PEM material still present (%s): %s", leak, s)
		}
	}
	if ContainsCleartextSecret(out) {
		t.Fatalf("cleartext still detected after redaction: %s", s)
	}
}

func TestRedactIssuerEndpointsFullyRedacted(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{
			name: "oidc_issuer_endpoint with userinfo",
			raw:  `{"oidc_issuer_endpoint":"https://zync-user:zync-pass@idp.example.com/realms/demo"}`,
		},
		{
			name: "issuer_endpoint clean",
			raw:  `{"issuer_endpoint":"https://sso.example.com/realms/prod"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := RedactBytes(".json", []byte(tt.raw))
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)
			if strings.Contains(s, "idp.example.com") || strings.Contains(s, "sso.example.com") ||
				strings.Contains(s, "zync-user") {
				t.Fatalf("issuer URL should be fully redacted: %s", s)
			}
			if !strings.Contains(s, "***REDACTED***") {
				t.Fatalf("missing redacted marker: %s", s)
			}
			if ContainsCleartextSecret(out) {
				t.Fatalf("cleartext still detected: %s", s)
			}
		})
	}
}

func TestRedactIssuerYAMLFullyRedacted(t *testing.T) {
	raw := []byte("oidc:\n  oidc_issuer_endpoint: https://zync-user:zync-pass@idp.example.com/realms/demo\n")
	out, err := RedactBytes(".yaml", raw)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "zync-user") || strings.Contains(s, "idp.example.com") {
		t.Fatalf("issuer still present: %s", s)
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

func TestContainsCleartextSecretExtendedKeys(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantHit bool
	}{
		{name: "provider_verification_key cleartext", raw: `{"provider_verification_key":"live"}`, wantHit: true},
		{name: "client_id redacted", raw: `{"client_id":"***REDACTED***"}`, wantHit: false},
		{name: "oidc issuer cleartext", raw: `{"oidc_issuer_endpoint":"https://u:p@host/path"}`, wantHit: true},
		{name: "oidc issuer redacted", raw: `{"oidc_issuer_endpoint":"***REDACTED***"}`, wantHit: false},
		{name: "apikey cleartext", raw: `{"apikey":"eyJ"}`, wantHit: true},
		{name: "endpoint cleartext", raw: `{"endpoint":"https://gw.example.com"}`, wantHit: true},
		{name: "password redacted", raw: `{"password":"***REDACTED***"}`, wantHit: false},
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

func TestRedactJSONSchemaObjectUnderCertificateKey(t *testing.T) {
	// Policy catalog may nest a schema object under "certificate"; do not wipe the object tree blindly.
	raw := []byte(`{"properties":{"certificate":{"type":"string","title":"PEM"},"other":{"api_key":"live"}}}`)
	out, err := RedactBytes(".json", raw)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"type": "string"`) {
		t.Fatalf("schema object under certificate was wiped: %s", s)
	}
	if strings.Contains(s, `"live"`) {
		t.Fatalf("nested api_key not redacted: %s", s)
	}
}
