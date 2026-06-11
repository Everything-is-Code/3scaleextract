package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestPostFormSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("content-type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("name") != "demo" {
			t.Fatalf("form = %v", r.Form)
		}
		_, _ = w.Write([]byte(`{"backend_api":{"id":7}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var resp struct {
		BackendAPI struct {
			ID int `json:"id"`
		} `json:"backend_api"`
	}
	form := make(url.Values)
	form.Set("name", "demo")
	if err := client.PostForm(context.Background(), "/backend_apis", form, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.BackendAPI.ID != 7 {
		t.Fatalf("id = %d", resp.BackendAPI.ID)
	}
}

func TestPutFormSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"service":{"id":3}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	if err := client.PutForm(context.Background(), "/services/3", url.Values{"name": {"api"}}, &resp); err != nil {
		t.Fatal(err)
	}
}

func TestPutJSONSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("content-type = %q", ct)
		}
		_, _ = w.Write([]byte(`{"proxy":{"id":99}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	if err := client.PutJSON(context.Background(), "/services/1/proxy", map[string]string{"auth_type": "api_key"}, &resp); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Delete(context.Background(), "/services/5/applications/9"); err != nil {
		t.Fatal(err)
	}
}

func TestWriteMethodsUnrecoverable400(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	err = client.PostForm(context.Background(), "/backend_apis", url.Values{}, &map[string]any{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetAllPagesBackendAPIsKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"backend_apis":[{"id":1},{"id":2}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var total int
	err = client.GetAllPages(context.Background(), "/backend_apis", 10, func(items []json.RawMessage) error {
		total += len(items)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
}

func TestGetAllPagesItemsKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":9}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var total int
	if err := client.GetAllPages(context.Background(), "/custom", 10, func(items []json.RawMessage) error {
		total += len(items)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
}

func TestGetAllPagesEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"applications":[]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	called := false
	err = client.GetAllPages(context.Background(), "/applications", 10, func([]json.RawMessage) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected callback not to run for empty page")
	}
}

func TestPutJSONInvalidPayload(t *testing.T) {
	client, err := NewClient(Options{BaseURL: "https://example.com", Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	err = client.PutJSON(context.Background(), "/services/1", make(chan int), nil)
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestPostFormRetryOn500ThenSuccess(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"backend_api":{"id":1}}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret", MaxRetries: 3})
	if err != nil {
		t.Fatal(err)
	}

	var resp map[string]any
	if err := client.PostForm(context.Background(), "/backend_apis", url.Values{"name": {"demo"}}, &resp); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&attempts) < 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestGetAllPagesFallbackCollectionKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"widgets":[{"id":5}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	var total int
	if err := client.GetAllPages(context.Background(), "/widgets", 10, func(items []json.RawMessage) error {
		total += len(items)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("total = %d", total)
	}
}

func TestGetAllPagesInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	err = client.GetAllPages(context.Background(), "/applications", 10, func([]json.RawMessage) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAcquireRespectsContextCancel(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-block
	}))
	defer srv.Close()
	defer close(block)

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret", MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		_ = client.Get(context.Background(), "/services", &map[string]any{})
	}()
	time.Sleep(20 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = client.Get(ctx, "/services", &map[string]any{})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestPostFormEmptyResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.PostForm(context.Background(), "/backend_apis", nil, nil); err != nil {
		t.Fatal(err)
	}
}
