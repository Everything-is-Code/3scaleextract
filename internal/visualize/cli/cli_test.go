package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisualizeHelp(t *testing.T) {
	if got := execute([]string{"--help"}); got != 0 {
		t.Fatalf("execute(--help) = %d, want 0", got)
	}
}

func TestVisualizeExportMinimalFixture(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "export-minimal")
	out := t.TempDir()
	if got := execute([]string{fixture, "-o", out}); got != 0 {
		t.Fatalf("execute(fixture) = %d, want 0", got)
	}
	if _, err := os.Stat(filepath.Join(out, "index.md")); err != nil {
		t.Fatalf("missing index.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "products-catalog.md")); err != nil {
		t.Fatalf("missing products-catalog.md: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "backends.md")); err != nil {
		t.Fatalf("missing backends.md: %v", err)
	}
}

func TestVisualizeHTMLFlag(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "export-minimal")
	out := t.TempDir()
	if got := execute([]string{fixture, "-o", out, "--html"}); got != 0 {
		t.Fatalf("execute(html) = %d, want 0", got)
	}
	htmlPath := filepath.Join(out, "topology.html")
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read topology.html: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "seed_alpha") || !strings.Contains(text, "<!DOCTYPE html>") {
		t.Fatalf("unexpected html content")
	}
	index, err := os.ReadFile(filepath.Join(out, "index.md"))
	if err != nil {
		t.Fatalf("read index.md: %v", err)
	}
	if !strings.Contains(string(index), "topology.html") {
		t.Fatal("index.md should link topology.html when --html is set")
	}
}

func TestVisualizeCanvasFlag(t *testing.T) {
	fixture := filepath.Join("..", "testdata", "export-minimal")
	canvas := filepath.Join(t.TempDir(), "topology.canvas.tsx")
	if got := execute([]string{fixture, "--canvas", canvas}); got != 0 {
		t.Fatalf("execute(canvas) = %d, want 0", got)
	}
	content, err := os.ReadFile(canvas)
	if err != nil {
		t.Fatalf("read canvas: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "TopologyCanvas") || !strings.Contains(text, "seed_alpha") || !strings.Contains(text, "cursor/canvas") {
		t.Fatalf("unexpected canvas content")
	}
}
