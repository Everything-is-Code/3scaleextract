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
	if _, err := os.Stat(filepath.Join(out, "backends.md")); err != nil {
		t.Fatalf("missing backends.md: %v", err)
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
