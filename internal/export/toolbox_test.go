package export

import (
	"context"
	"errors"
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
	var captured struct {
		command string
		args    []string
	}
	runner := &mockCommandRunner{
		fn: func(command string, args []string) ([]byte, []byte, error) {
			captured.command = command
			captured.args = append([]string(nil), args...)
			return []byte("apiVersion: v1\nkind: Product\n"), nil, nil
		},
	}
	tb := &Toolbox{
		runtime:  "podman",
		image:    DefaultToolboxImage,
		certFile: "/etc/ssl/certs/custom.pem",
		runner:   runner,
	}
	out, err := tb.ExportProduct(context.Background(), "https://admin.example.com", "tok", "payments")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "apiVersion") {
		t.Fatalf("output = %q", out)
	}
	if captured.command != "podman" {
		t.Fatalf("command = %q", captured.command)
	}
	joined := strings.Join(captured.args, " ")
	for _, want := range []string{
		"run", "--rm",
		"SSL_CERT_FILE=/tmp/3scale-toolbox-cert.pem",
		"/etc/ssl/certs/custom.pem:/tmp/3scale-toolbox-cert.pem:ro",
		DefaultToolboxImage,
		"3scale", "product", "export",
		"payments",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q: %v", want, captured.args)
		}
	}
	if !strings.Contains(joined, "tok@admin.example.com") {
		t.Fatalf("remote URL not embedded: %v", captured.args)
	}
}

func TestExportProductNativeUsesRunner(t *testing.T) {
	var captured struct {
		command string
		args    []string
	}
	runner := &mockCommandRunner{
		fn: func(command string, args []string) ([]byte, []byte, error) {
			captured.command = command
			captured.args = append([]string(nil), args...)
			return []byte("kind: Product\n"), nil, nil
		},
	}
	tb := &Toolbox{
		nativeBinary: "/usr/bin/3scale",
		runner:       runner,
	}
	if _, err := tb.ExportProduct(context.Background(), "https://tenant.example.com", "secret", "demo_api"); err != nil {
		t.Fatal(err)
	}
	if captured.command != "/usr/bin/3scale" {
		t.Fatalf("command = %q", captured.command)
	}
	if len(captured.args) != 4 || captured.args[0] != "product" || captured.args[3] != "demo_api" {
		t.Fatalf("args = %v", captured.args)
	}
}

func TestRunCommandEmptyOutput(t *testing.T) {
	tb := &Toolbox{
		nativeBinary: "/usr/bin/3scale",
		runner: &mockCommandRunner{
			fn: func(string, []string) ([]byte, []byte, error) {
				return nil, nil, nil
			},
		},
	}
	_, err := tb.ExportProduct(context.Background(), "https://tenant.example.com", "secret", "demo")
	if err == nil || !errors.Is(err, ErrToolboxFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunCommandStderrError(t *testing.T) {
	tb := &Toolbox{
		nativeBinary: "/usr/bin/3scale",
		runner: &mockCommandRunner{
			fn: func(string, []string) ([]byte, []byte, error) {
				return nil, []byte("connection refused"), errors.New("exit 1")
			},
		},
	}
	_, err := tb.ExportProduct(context.Background(), "https://tenant.example.com", "secret", "demo")
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("err = %v", err)
	}
}

type mockCommandRunner struct {
	fn func(command string, args []string) (stdout, stderr []byte, err error)
}

func (m *mockCommandRunner) Run(_ context.Context, command string, args []string) ([]byte, []byte, error) {
	if m.fn == nil {
		return nil, nil, errors.New("mock not configured")
	}
	return m.fn(command, args)
}
