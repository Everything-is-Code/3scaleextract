package stats

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

func TestExportMultiProduct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/usage.json"):
			_, _ = w.Write([]byte(`{"total":10}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewClient(Options{BaseURL: srv.URL, Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	writer, err := output.NewWriter(root)
	if err != nil {
		t.Fatal(err)
	}

	products := []ProductRef{
		{ID: 1, SystemName: "api"},
		{ID: 2, SystemName: "billing"},
		{ID: 3, SystemName: "inventory"},
	}
	meta, err := Export(context.Background(), client, writer, products, ExportOptions{
		Since: "2026-06-01", Until: "2026-07-01", Granularity: "day", MetricName: "hits",
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta == nil || len(meta.Products) != 3 {
		t.Fatalf("meta = %+v", meta)
	}

	for _, name := range []string{"api", "billing", "inventory"} {
		path := filepath.Join(root, "stats", "products", name, "hits.json")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !json.Valid(data) {
			t.Fatalf("invalid json in %s", path)
		}
	}

	queryPath := filepath.Join(root, "stats", "query.json")
	data, err := os.ReadFile(queryPath)
	if err != nil {
		t.Fatal(err)
	}
	var saved QueryMeta
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Since != "2026-06-01" || saved.Until != "2026-07-01" {
		t.Fatalf("query meta = %+v", saved)
	}
}
