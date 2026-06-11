package config

import (
	"testing"
	"time"
)

func TestNewHTTPClientInsecure(t *testing.T) {
	client := NewHTTPClient(true, 30*time.Second)
	if client.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v", client.Timeout)
	}
	if client.Transport == nil {
		t.Fatal("expected transport for insecure TLS")
	}
}

func TestNewHTTPClientSecure(t *testing.T) {
	client := NewHTTPClient(false, 60*time.Second)
	if client.Transport != nil {
		t.Fatal("expected nil transport for secure client")
	}
}
