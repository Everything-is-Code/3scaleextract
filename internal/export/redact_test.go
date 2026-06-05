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
