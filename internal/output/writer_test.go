package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterEnsureLayoutAndManifest(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []string{"products", "backends", "applications", "accounts", "policies"} {
		if _, err := os.Stat(filepath.Join(dir, sub)); err != nil {
			t.Fatalf("missing dir %s: %v", sub, err)
		}
	}

	m := &Manifest{ExportedAt: "2026-06-02T00:00:00Z", AdminURL: "https://tenant.example.com", ProductCount: 1}
	if err := w.WriteManifest(m); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.SchemaVersion != SchemaVersion {
		t.Fatalf("schema = %q", decoded.SchemaVersion)
	}
}

func TestWriteAtomicReplacesFile(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteJSON("nested/file.json", map[string]string{"a": "1"}); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteJSON("nested/file.json", map[string]string{"a": "2"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "nested", "file.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"a": "2"`) {
		t.Fatalf("unexpected content: %s", data)
	}
}

func TestWriteBytes(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("apiVersion: v1\nkind: Product\n")
	if err := w.WriteBytes(filepath.Join("products", "demo.yaml"), payload); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "products", "demo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("got %q", data)
	}
}

func TestWriteRawJSONValid(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{"proxy":{"auth_type":"api_key"}}`)
	if err := w.WriteRawJSON(filepath.Join("products", "demo", "proxy.json"), raw); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "products", "demo", "proxy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"auth_type"`) {
		t.Fatalf("content = %s", data)
	}
}

func TestWriteRawJSONEmpty(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.WriteRawJSON("empty.json", json.RawMessage{}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "empty.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "{}") {
		t.Fatalf("content = %s", data)
	}
}

func TestWriteRawJSONInvalidFallsBackToBytes(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`not-json`)
	if err := w.WriteRawJSON("broken.json", raw); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "broken.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "not-json" {
		t.Fatalf("content = %s", data)
	}
}

func TestWriterRoot(t *testing.T) {
	dir := t.TempDir()
	w, err := NewWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.Root() != dir {
		t.Fatalf("Root = %q", w.Root())
	}
}
