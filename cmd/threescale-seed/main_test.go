package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestSeedHelp(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	if got := executeSeed([]string{"--help"}); got != 0 {
		t.Fatalf("executeSeed(--help) = %d, want 0", got)
	}
}

func TestListFixtures(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	out := captureStdout(func() {
		if got := executeSeed([]string{"--list-fixtures"}); got != 0 {
			t.Fatalf("executeSeed(--list-fixtures) = %d, want 0", got)
		}
	})
	if !strings.Contains(out, "seed_api_key") {
		t.Fatalf("stdout missing seed_api_key:\n%s", out)
	}
}

func TestExecuteDryRun(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "https://tenant.example.com")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "secret")
	out := captureStdout(func() {
		if got := executeSeed([]string{"--dry-run"}); got != 0 {
			t.Fatalf("executeSeed(--dry-run) = %d, want 0", got)
		}
	})
	if !strings.Contains(out, "seed_api_key") {
		t.Fatalf("stdout missing fixtures:\n%s", out)
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatalf("stdout missing dry-run note:\n%s", out)
	}
}

func TestExecuteMissingAuth(t *testing.T) {
	t.Setenv("THREESCALE_ADMIN_URL", "")
	t.Setenv("THREESCALE_ACCESS_TOKEN", "")
	if got := executeSeed(nil); got != 1 {
		t.Fatalf("executeSeed() = %d, want 1", got)
	}
}
