package stats

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
	"github.com/Everything-is-Code/3scaleextract/internal/progress"
)

type ProductRef struct {
	ID         int
	SystemName string
}

type ExportOptions struct {
	Since         string
	Until         string
	Granularity   string
	MetricName    string
	MaxConcurrent int
	Reporter      progress.Reporter
}

type QueryMeta struct {
	Since       string       `json:"since"`
	Until       string       `json:"until"`
	Granularity string       `json:"granularity"`
	MetricName  string       `json:"metric_name"`
	ExportedAt  string       `json:"exported_at"`
	Products    []ProductRef `json:"products"`
}

func Export(ctx context.Context, client Client, writer *output.Writer, products []ProductRef, opts ExportOptions) (*QueryMeta, error) {
	if client == nil {
		return nil, fmt.Errorf("stats client is required")
	}
	if writer == nil {
		return nil, fmt.Errorf("output writer is required")
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("at least one product is required for stats export")
	}

	query := UsageQuery{
		Since:       opts.Since,
		Until:       opts.Until,
		Granularity: opts.Granularity,
		MetricName:  opts.MetricName,
	}

	meta := &QueryMeta{
		Since:       opts.Since,
		Until:       opts.Until,
		Granularity: opts.Granularity,
		MetricName:  opts.MetricName,
		ExportedAt:  time.Now().UTC().Format(time.RFC3339),
		Products:    append([]ProductRef(nil), products...),
	}

	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	rep := progress.OrNop(opts.Reporter)
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	errCh := make(chan error, len(products))
	var completed atomic.Int32
	total := len(products)

	for _, product := range products {
		wg.Add(1)
		go func(p ProductRef) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			raw, err := client.GetUsage(ctx, p.ID, query)
			if err != nil {
				errCh <- fmt.Errorf("product %q: %w", p.SystemName, err)
				return
			}
			rel := fmt.Sprintf("stats/products/%s/hits.json", p.SystemName)
			if err := writer.WriteRawJSON(rel, raw); err != nil {
				errCh <- fmt.Errorf("product %q: write hits: %w", p.SystemName, err)
				return
			}
			n := int(completed.Add(1))
			rep.Item(n, total, p.SystemName)
		}(product)
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	if err := writer.WriteJSON("stats/query.json", meta); err != nil {
		return nil, fmt.Errorf("write stats query metadata: %w", err)
	}
	return meta, nil
}
