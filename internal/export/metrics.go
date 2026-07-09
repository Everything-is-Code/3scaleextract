package export

import (
	"context"
	"net/http"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
	"github.com/Everything-is-Code/3scaleextract/internal/stats"
)

// ListProducts returns API products from the Admin API /services listing.
func (s *Service) ListProducts(ctx context.Context) ([]serviceRef, error) {
	return s.listServices(ctx)
}

// ExportMetrics fetches Analytics API usage for each product and writes stats/ artifacts.
func ExportMetrics(
	ctx context.Context,
	adminURL, token string,
	httpClient *http.Client,
	maxConcurrent int,
	writer *output.Writer,
	services []serviceRef,
	since, until, granularity, metricName string,
) error {
	statsBase, err := stats.DeriveStatsBaseURL(adminURL)
	if err != nil {
		return err
	}
	statsClient, err := stats.NewClient(stats.Options{
		BaseURL:       statsBase,
		Token:         token,
		HTTPClient:    httpClient,
		MaxConcurrent: maxConcurrent,
	})
	if err != nil {
		return err
	}

	products := make([]stats.ProductRef, 0, len(services))
	for _, svc := range services {
		products = append(products, stats.ProductRef{
			ID:         svc.ID,
			SystemName: svc.SystemName,
		})
	}
	_, err = stats.Export(ctx, statsClient, writer, products, stats.ExportOptions{
		Since:         since,
		Until:         until,
		Granularity:   granularity,
		MetricName:    metricName,
		MaxConcurrent: maxConcurrent,
	})
	return err
}

func applyMetricsManifest(manifest *output.Manifest, since, until, granularity, metricName string) {
	if manifest == nil {
		return
	}
	manifest.IncludeMetrics = true
	manifest.MetricsSince = since
	manifest.MetricsUntil = until
	manifest.MetricsGranularity = granularity
	manifest.MetricsMetric = metricName
}
