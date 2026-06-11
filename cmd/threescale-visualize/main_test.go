package main

import (
	"testing"
)

func TestVisualizeHelp(t *testing.T) {
	if got := executeVisualize([]string{"--help"}); got != 0 {
		t.Fatalf("executeVisualize(--help) = %d, want 0", got)
	}
}
