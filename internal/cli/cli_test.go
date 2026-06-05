package cli

import (
	"errors"
	"testing"

	"github.com/fmeneses/3scaleextract/internal/config"
)

func TestRunExportMissingAuth(t *testing.T) {
	err := RunExport(t.Context(), config.ExportConfig{OutDir: t.TempDir()})
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
