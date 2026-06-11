package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVisualizeHelp(t *testing.T) {
	if got := executeVisualize([]string{"--help"}); got != 0 {
		t.Fatalf("executeVisualize(--help) = %d, want 0", got)
	}
}

func TestVisualizeExportMinimalFixture(t *testing.T) {
	fixture := filepath.Join("..", "..", "internal", "visualize", "testdata", "export-minimal")
	out := t.TempDir()
	if got := executeVisualize([]string{fixture, "-o", out}); got != 0 {
		t.Fatalf("executeVisualize(fixture) = %d, want 0", got)
	}
	if _, err := os.Stat(filepath.Join(out, "index.md")); err != nil {
		t.Fatalf("missing index.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "backends.md")); err != nil {
		t.Fatalf("missing backends.md: %v", err)
	}
}
