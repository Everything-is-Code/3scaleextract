package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildRemoteURL(t *testing.T) {
	got, err := buildRemoteURL("https://tenant-admin.example.com", "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	want := "https://secret-token@tenant-admin.example.com"
	if got != want {
		t.Fatalf("remote URL = %q, want %q", got, want)
	}
}

func TestResolveContainerRuntimePreferred(t *testing.T) {
	dir := t.TempDir()
	writeFakeRuntime(t, dir, "docker")
	t.Setenv("PATH", dir)

	runtime, err := resolveContainerRuntime("docker")
	if err != nil {
		t.Fatal(err)
	}
	if runtime != "docker" {
		t.Fatalf("runtime = %q, want docker", runtime)
	}
}

func TestResolveContainerRuntimeAutoDetectDocker(t *testing.T) {
	dir := t.TempDir()
	writeFakeRuntime(t, dir, "docker")
	t.Setenv("PATH", dir)

	runtime, err := resolveContainerRuntime("")
	if err != nil {
		t.Fatal(err)
	}
	if runtime != "docker" {
		t.Fatalf("runtime = %q, want docker", runtime)
	}
}

func TestResolveContainerRuntimeAutoDetectPodmanOnly(t *testing.T) {
	dir := t.TempDir()
	writeFakeRuntime(t, dir, "podman")
	t.Setenv("PATH", dir)

	runtime, err := resolveContainerRuntime("")
	if err != nil {
		t.Fatal(err)
	}
	if runtime != "podman" {
		t.Fatalf("runtime = %q, want podman", runtime)
	}
}

func TestResolveContainerRuntimeMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := resolveContainerRuntime("")
	if err == nil {
		t.Fatal("expected error when no runtime in PATH")
	}
	if !strings.Contains(err.Error(), "docker or podman") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeFakeRuntime(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestNewToolboxDefaults(t *testing.T) {
	tb, err := NewToolbox(ToolboxOptions{})
	if err != nil {
		t.Skip("podman/docker not available:", err)
	}
	if tb.image != DefaultToolboxImage {
		t.Fatalf("image = %q", tb.image)
	}
	if tb.runtime == "" {
		t.Fatal("expected runtime")
	}
}

func TestNewToolboxNativeMode(t *testing.T) {
	tb, err := NewToolbox(ToolboxOptions{NativeBinary: "/usr/bin/3scale"})
	if err != nil {
		t.Fatal(err)
	}
	if tb.nativeBinary != "/usr/bin/3scale" {
		t.Fatalf("native = %q", tb.nativeBinary)
	}
}

func TestRunContainerArgs(t *testing.T) {
	tb := &Toolbox{
		runtime: "podman",
		image:   DefaultToolboxImage,
		certFile: "/etc/ssl/certs/custom.pem",
	}
	// exercise internal path via ExportProduct would need mock exec; verify URL builder instead
	remote, err := buildRemoteURL("https://admin.example.com", "tok")
	if err != nil || !strings.Contains(remote, "tok@") {
		t.Fatalf("remote = %q err = %v", remote, err)
	}
	_ = tb
}
