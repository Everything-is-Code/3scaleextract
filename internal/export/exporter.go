package export

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

type Options struct {
	AdminURL            string
	Token               string
	OutDir              string
	IncludeApplications bool
	RedactSecrets       bool
	MaxConcurrent       int
	PerPage             int
}

type Exporter interface {
	Export(ctx context.Context, opts Options) (*output.Manifest, error)
}

type Service struct {
	client  admin.Client
	toolbox ProductExporter
}

func NewService(client admin.Client, toolbox ProductExporter) *Service {
	return &Service{client: client, toolbox: toolbox}
}

type serviceListResponse struct {
	Services []serviceEntry `json:"services"`
}

type serviceEntry struct {
	Service serviceRef `json:"service"`
}

type serviceRef struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	SystemName string `json:"system_name"`
}

type backendListResponse struct {
	BackendAPIs []backendEntry `json:"backend_apis"`
}

type backendEntry struct {
	BackendAPI json.RawMessage `json:"backend_api"`
}

type applicationEnvelope struct {
	Application applicationRef `json:"application"`
}

type applicationRef struct {
	ID        int `json:"id"`
	AccountID int `json:"account_id"`
}

func (s *Service) Export(ctx context.Context, opts Options) (*output.Manifest, error) {
	writer, err := output.NewWriter(opts.OutDir)
	if err != nil {
		return nil, err
	}
	if err := writer.EnsureLayout(); err != nil {
		return nil, err
	}

	manifest := &output.Manifest{
		SchemaVersion:       output.SchemaVersion,
		ExportedAt:          time.Now().UTC().Format(time.RFC3339),
		AdminURL:            opts.AdminURL,
		IncludeApplications: opts.IncludeApplications,
	}

	services, err := s.listServices(ctx)
	if err != nil {
		return nil, err
	}
	manifest.ProductCount = len(services)

	backendCount, err := s.exportBackends(ctx, writer)
	if err != nil {
		manifest.Incomplete = true
		return manifest, err
	}
	manifest.BackendCount = backendCount

	if err := s.exportPolicyCatalog(ctx, writer); err != nil {
		manifest.Incomplete = true
		return manifest, err
	}

	for _, svc := range services {
		if err := s.exportService(ctx, writer, opts, svc); err != nil {
			manifest.Incomplete = true
			return manifest, fmt.Errorf("export service %q: %w", svc.SystemName, err)
		}
	}

	if opts.IncludeApplications {
		count, err := s.exportApplications(ctx, writer, opts.PerPage)
		if err != nil {
			manifest.Incomplete = true
			return manifest, err
		}
		manifest.ApplicationCount = count
	}

	if opts.RedactSecrets {
		if err := RedactDirectory(writer.Root()); err != nil {
			return manifest, err
		}
	}

	if err := writer.WriteManifest(manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

func (s *Service) listServices(ctx context.Context) ([]serviceRef, error) {
	var resp serviceListResponse
	if err := s.client.Get(ctx, "/services", &resp); err != nil {
		return nil, err
	}
	out := make([]serviceRef, 0, len(resp.Services))
	for _, entry := range resp.Services {
		if entry.Service.SystemName != "" {
			out = append(out, entry.Service)
		}
	}
	return out, nil
}

func (s *Service) fetchBackends(ctx context.Context) ([]json.RawMessage, error) {
	var resp backendListResponse
	if err := s.client.Get(ctx, "/backend_apis", &resp); err != nil {
		return nil, err
	}
	items := make([]json.RawMessage, 0, len(resp.BackendAPIs))
	for _, entry := range resp.BackendAPIs {
		items = append(items, entry.BackendAPI)
	}
	return items, nil
}

func (s *Service) exportBackends(ctx context.Context, writer *output.Writer) (int, error) {
	items, err := s.fetchBackends(ctx)
	if err != nil {
		return 0, err
	}
	for i, item := range items {
		name, _ := jsonStringField(item, "system_name")
		if name == "" {
			name = fmt.Sprintf("backend-%d", i+1)
		}
		if err := writer.WriteRawJSON(filepath.Join("backends", name+".json"), item); err != nil {
			return 0, err
		}
	}
	return len(items), nil
}

func (s *Service) exportPolicyCatalog(ctx context.Context, writer *output.Writer) error {
	var catalog map[string]json.RawMessage
	if err := s.client.Get(ctx, "/policies", &catalog); err != nil {
		return err
	}
	return writer.WriteJSON("policies/catalog.json", catalog)
}

func (s *Service) exportService(ctx context.Context, writer *output.Writer, opts Options, svc serviceRef) error {
	yamlData, err := s.toolbox.ExportProduct(ctx, opts.AdminURL, opts.Token, svc.SystemName)
	if err != nil {
		return err
	}
	if err := writer.WriteBytes(filepath.Join("products", svc.SystemName+".yaml"), appendYAMLNewline(yamlData)); err != nil {
		return err
	}

	fetches := []struct {
		file string
		path string
	}{
		{"proxy.json", fmt.Sprintf("/services/%d/proxy", svc.ID)},
		{"policies.json", fmt.Sprintf("/services/%d/proxy/policies", svc.ID)},
		{"oidc_configuration.json", fmt.Sprintf("/services/%d/proxy/oidc_configuration", svc.ID)},
		{"application_plans.json", fmt.Sprintf("/services/%d/application_plans", svc.ID)},
		{"backend_usages.json", fmt.Sprintf("/services/%d/backend_usages", svc.ID)},
		{"metrics.json", fmt.Sprintf("/services/%d/metrics", svc.ID)},
	}

	for _, f := range fetches {
		var payload json.RawMessage
		if err := s.client.Get(ctx, f.path, &payload); err != nil {
			continue
		}
		rel := filepath.Join("products", svc.SystemName, f.file)
		if err := writer.WriteRawJSON(rel, payload); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) exportApplications(ctx context.Context, writer *output.Writer, perPage int) (int, error) {
	pageNum := 0
	total := 0
	seenAccounts := sync.Map{}

	err := s.client.GetAllPages(ctx, "/applications", perPage, func(items []json.RawMessage) error {
		pageNum++
		pageObjs := make([]any, 0, len(items))
		for _, item := range items {
			total++
			var obj any
			if err := json.Unmarshal(item, &obj); err == nil {
				pageObjs = append(pageObjs, obj)
			}
			var env applicationEnvelope
			if err := json.Unmarshal(item, &env); err != nil {
				continue
			}
			if env.Application.AccountID > 0 {
				accountID := env.Application.AccountID
				if _, loaded := seenAccounts.LoadOrStore(accountID, struct{}{}); !loaded {
					if err := s.exportAccount(ctx, writer, accountID); err != nil {
						return err
					}
				}
			}
		}
		return writer.WriteJSON(filepath.Join("applications", fmt.Sprintf("page-%d.json", pageNum)), pageObjs)
	})
	return total, err
}

func (s *Service) exportAccount(ctx context.Context, writer *output.Writer, accountID int) error {
	var account json.RawMessage
	if err := s.client.Get(ctx, fmt.Sprintf("/accounts/%d", accountID), &account); err != nil {
		return err
	}
	return writer.WriteRawJSON(filepath.Join("accounts", fmt.Sprintf("%d.json", accountID)), account)
}

func jsonStringField(raw json.RawMessage, key string) (string, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return "", err
	}
	var val string
	if err := json.Unmarshal(obj[key], &val); err != nil {
		return "", err
	}
	return val, nil
}

func appendYAMLNewline(data []byte) []byte {
	if len(data) == 0 || data[len(data)-1] == '\n' {
		return data
	}
	out := make([]byte, len(data)+1)
	copy(out, data)
	out[len(data)] = '\n'
	return out
}
