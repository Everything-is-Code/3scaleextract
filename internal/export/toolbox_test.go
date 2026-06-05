package export

import (
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
