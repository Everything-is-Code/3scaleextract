package main

import (
	"testing"

	vizcli "github.com/Everything-is-Code/3scaleextract/internal/visualize/cli"
)

func TestMainDelegatesToCLI(t *testing.T) {
	if got := vizcli.Execute(); got != 1 {
		t.Fatalf("Execute() = %d, want 1", got)
	}
}
