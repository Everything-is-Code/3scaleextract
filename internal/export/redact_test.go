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
