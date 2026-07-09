package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeriveStatsBaseURL(t *testing.T) {
	got, err := DeriveStatsBaseURL("https://tenant.example.com/admin/api")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://tenant.example.com/stats" {
		t.Fatalf("got %q", got)
	}

	got, err = DeriveStatsBaseURL("https://forum-developers-admin.3scale.net")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://forum-developers-admin.3scale.net/stats" {
		t.Fatalf("got %q", got)
	}
}

func TestGetUsageSuccess(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Query().Get("metric_name") != "hits" {
			t.Fatalf("metric_name = %q", r.URL.Query().Get("metric_name"))
		}
		_, _ = w.Write([]byte(`{"total":42,"values":[{"date":"2026-07-01","value":42}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := client.GetUsage(context.Background(), 42, UsageQuery{
		Since: "2026-07-01", Until: "2026-07-09", Granularity: "day", MetricName: "hits",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"total":42`) {
		t.Fatalf("raw = %s", raw)
	}
	if gotPath != "/services/42/usage.json" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestGetUsageForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "tok", MaxRetries: 0})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetUsage(context.Background(), 1, UsageQuery{MetricName: "hits"})
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetUsageRateLimitRetry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"total":1}`))
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "tok", MaxRetries: 2})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := client.GetUsage(context.Background(), 1, UsageQuery{MetricName: "hits"})
	if err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatalf("attempts = %d", attempts)
	}
	if !json.Valid(raw) {
		t.Fatalf("invalid json: %s", raw)
	}
}
