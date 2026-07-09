package export

import (
	"context"
	"testing"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

func TestListProducts(t *testing.T) {
	client := &mockClient{
		responses: map[string]any{
			"/services": serviceListResponse{
				Services: []serviceEntry{{Service: serviceRef{ID: 10, SystemName: "payments"}}},
			},
		},
	}
	svc := NewService(client, nil)
	got, err := svc.ListProducts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SystemName != "payments" {
		t.Fatalf("got %#v", got)
	}
}

func TestApplyMetricsManifest(t *testing.T) {
	m := &output.Manifest{}
	applyMetricsManifest(m, "2026-01-01", "2026-01-31", "day", "hits")
	if !m.IncludeMetrics || m.MetricsSince != "2026-01-01" || m.MetricsMetric != "hits" {
		t.Fatalf("manifest = %#v", m)
	}
	applyMetricsManifest(nil, "2026-01-01", "2026-01-31", "day", "hits")
}
