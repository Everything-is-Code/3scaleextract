package main

import (
	"os"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/cli"
)

func TestExportHelp(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"threescale-export", "--help"}
	if got := cli.Execute(); got != 0 {
		t.Fatalf("Execute(--help) = %d, want 0", got)
	}
}

func TestExportMissingAuthExitCode(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	t.Setenv("THREESCALE_OUTPUT_DIR", "")
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"threescale-export", "--output", t.TempDir()}
	if got := cli.Execute(); got != 1 {
		t.Fatalf("Execute() = %d, want 1", got)
	}
}
