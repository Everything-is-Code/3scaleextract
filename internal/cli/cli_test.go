package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/config"
)

func TestRunExportMissingAuth(t *testing.T) {
	err := RunExport(context.Background(), config.ExportConfig{OutDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, config.ErrMissingAdminURL) && !errors.Is(err, config.ErrMissingToken) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRootIsExportCommand(t *testing.T) {
	root := NewRoot()
	if root.Use != "threescale-export" {
		t.Fatalf("use = %s", root.Use)
	}
	if root.RunE == nil {
		t.Fatal("expected root to run export")
	}
}

func TestExportHelp(t *testing.T) {
	cmd := NewRoot()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestExecuteMissingAuth(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	t.Setenv("THREESCALE_OUTPUT_DIR", "")
	if got := Execute(); got != 1 {
		t.Fatalf("Execute() = %d, want 1", got)
	}
}
