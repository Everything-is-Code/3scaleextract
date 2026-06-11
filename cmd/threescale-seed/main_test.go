package main

import (
	"testing"

	seedcli "github.com/Everything-is-Code/3scaleextract/internal/seed/cli"
)

func TestMainDelegatesToCLI(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	if got := seedcli.Execute(); got != 1 {
		t.Fatalf("Execute() = %d, want 1", got)
	}
}
