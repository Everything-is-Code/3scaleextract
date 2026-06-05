package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("access_token"); got != "secret" {
			t.Fatalf("token = %q", got)
		}
		if !strings.HasSuffix(r.URL.Path, "/services.json") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"services":[{"id":1}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret", MaxConcurrent: 2})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	if err := client.Get(context.Background(), "/services", &resp); err != nil {
		t.Fatal(err)
	}
}

func TestGetUnrecoverable401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid token"}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "bad"})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	err = client.Get(context.Background(), "/services", &resp)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAllPagesPagination(t *testing.T) {
	var page int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("page")
		perPage := r.URL.Query().Get("per_page")
		if perPage != "2" {
			t.Fatalf("per_page = %q", perPage)
		}
		switch p {
		case "1":
			atomic.StoreInt32(&page, 1)
			_, _ = w.Write([]byte(`{"applications":[{"id":1},{"id":2}]}`))
		case "2":
			atomic.StoreInt32(&page, 2)
			_, _ = w.Write([]byte(`{"applications":[{"id":3}]}`))
		default:
			t.Fatalf("unexpected page %q", p)
		}
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var total int
	err = client.GetAllPages(context.Background(), "/applications", 2, func(items []json.RawMessage) error {
		total += len(items)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Fatalf("total items = %d", total)
	}
	if atomic.LoadInt32(&page) != 2 {
		t.Fatalf("expected page 2, got %d", page)
	}
}

func TestRetryOn429ThenSuccess(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{
		BaseURL:    srv.URL,
		Token:      "secret",
		MaxRetries: 5,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	if err := client.Get(context.Background(), "/services", &resp); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&attempts) < 3 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestSemaphoreLimitsConcurrency(t *testing.T) {
	var inFlight int32
	var maxInFlight int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			prev := atomic.LoadInt32(&maxInFlight)
			if cur <= prev || atomic.CompareAndSwapInt32(&maxInFlight, prev, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)
		_, _ = w.Write([]byte(`{"services":[]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret", MaxConcurrent: 2})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	for i := 0; i < 6; i++ {
		go func(n int) {
			path := fmt.Sprintf("/services/%d", n)
			_ = client.Get(ctx, path, &map[string]any{})
		}(i)
	}
	time.Sleep(150 * time.Millisecond)
	if got := atomic.LoadInt32(&maxInFlight); got > 2 {
		t.Fatalf("max in-flight = %d, want <= 2", got)
	}
}

func TestBuildURLAddsJSONSuffix(t *testing.T) {
	client := &HTTPClient{baseURL: "https://tenant.example.com", token: "tok"}
	u, err := client.buildURL("/services", 1, 500)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "/admin/api/services.json") {
		t.Fatalf("url = %s", u)
	}
	if !strings.Contains(u, "access_token=tok") {
		t.Fatalf("missing token in %s", u)
	}
}
