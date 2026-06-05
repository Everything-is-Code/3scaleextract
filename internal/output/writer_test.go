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
