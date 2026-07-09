package visualize

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
	"gopkg.in/yaml.v3"
)

// LoadExport reads a threescale-export v1.0 directory into a Tenant model.
func LoadExport(root string) (*Tenant, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("export directory is required")
	}

	manifestPath := filepath.Join(root, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest.json: %w", err)
	}
	manifestData = stripUTF8BOM(manifestData)

	var manifest output.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest.json: %w", err)
	}
	if manifest.SchemaVersion != output.SchemaVersion {
		return nil, fmt.Errorf("unsupported schema_version %q (expected %q)", manifest.SchemaVersion, output.SchemaVersion)
	}

	backends, err := loadBackends(filepath.Join(root, "backends"))
	if err != nil {
		return nil, err
	}

	products, err := loadProducts(filepath.Join(root, "products"))
	if err != nil {
		return nil, err
	}

	tenant := &Tenant{
		Manifest: manifest,
		Products: products,
		Backends: backends,
	}
	tenant.index()
	tenant.resolveReferences()

	if manifest.IncludeApplications {
		apps, err := loadApplications(filepath.Join(root, "applications"))
		if err != nil {
			return nil, err
		}
		tenant.Applications = apps
		tenant.resolveReferences()
	}

	return tenant, nil
}

func loadBackends(dir string) ([]Backend, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backends directory: %w", err)
	}

	var backends []Backend
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read backend %s: %w", entry.Name(), err)
		}
		backend, err := parseBackend(stripUTF8BOM(data))
		if err != nil {
			return nil, fmt.Errorf("parse backend %s: %w", entry.Name(), err)
		}
		backends = append(backends, backend)
	}

	sort.Slice(backends, func(i, j int) bool {
		return backends[i].SystemName < backends[j].SystemName
	})
	return backends, nil
}

func parseBackend(data []byte) (Backend, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Backend{}, err
	}
	return Backend{
		ID:              intJSON(raw["id"]),
		SystemName:      stringJSON(raw["system_name"]),
		Name:            stringJSON(raw["name"]),
		PrivateEndpoint: stringJSON(raw["private_endpoint"]),
	}, nil
}

func loadProducts(dir string) ([]Product, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read products directory: %w", err)
	}

	systemNames := make(map[string]struct{})
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			systemNames[name] = struct{}{}
			continue
		}
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			systemNames[strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")] = struct{}{}
		}
	}

	names := make([]string, 0, len(systemNames))
	for name := range systemNames {
		names = append(names, name)
	}
	sort.Strings(names)

	products := make([]Product, 0, len(names))
	for _, systemName := range names {
		product, err := loadProduct(dir, systemName)
		if err != nil {
			return nil, fmt.Errorf("load product %q: %w", systemName, err)
		}
		products = append(products, product)
	}
	return products, nil
}

func loadProduct(productsDir, systemName string) (Product, error) {
	product := Product{SystemName: systemName, DisplayName: systemName}
	productDir := filepath.Join(productsDir, systemName)

	if displayName, err := readProductDisplayName(filepath.Join(productsDir, systemName+".yaml")); err == nil && displayName != "" {
		product.DisplayName = displayName
	} else if err != nil && !os.IsNotExist(err) {
		// Product YAML may contain redacted values invalid for strict parsing; keep systemName.
		_ = err
	}

	if err := readOptionalJSON(filepath.Join(productDir, "proxy.json"), func(data []byte) error {
		return parseProxy(data, &product)
	}); err != nil {
		return Product{}, err
	}
	if err := readOptionalJSON(filepath.Join(productDir, "policies.json"), func(data []byte) error {
		product.Policies = parsePolicies(data)
		return nil
	}); err != nil {
		return Product{}, err
	}
	if len(product.Policies) == 0 {
		if err := readOptionalJSON(filepath.Join(productDir, "proxy.json"), func(data []byte) error {
			product.Policies = policyNamesFromProxyFile(data)
			return nil
		}); err != nil {
			return Product{}, err
		}
	}
	if err := readOptionalJSON(filepath.Join(productDir, "backend_usages.json"), func(data []byte) error {
		product.BackendUsages = parseBackendUsages(data)
		return nil
	}); err != nil {
		return Product{}, err
	}
	if err := readOptionalJSON(filepath.Join(productDir, "application_plans.json"), func(data []byte) error {
		product.Plans = parseApplicationPlans(data)
		return nil
	}); err != nil {
		return Product{}, err
	}

	oidcPath := filepath.Join(productDir, "oidc_configuration.json")
	if err := readOptionalJSON(oidcPath, func(data []byte) error {
		product.OIDC = parseOIDCConfiguration(data)
		return nil
	}); err != nil {
		return Product{}, err
	}
	resolveProductAuthType(&product, filepath.Join(productsDir, systemName+".yaml"))
	if product.AuthType == "oidc" && product.OIDC == nil {
		if _, err := os.Stat(oidcPath); os.IsNotExist(err) {
			product.MissingFiles = append(product.MissingFiles, "oidc_configuration.json")
		}
	}

	return product, nil
}

type productYAML struct {
	Spec struct {
		Name       string `yaml:"name"`
		SystemName string `yaml:"systemName"`
	} `yaml:"spec"`
}

func readProductDisplayName(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var doc productYAML
	if err := yaml.Unmarshal(stripUTF8BOM(data), &doc); err != nil {
		return "", err
	}
	if doc.Spec.Name != "" {
		return doc.Spec.Name, nil
	}
	return doc.Spec.SystemName, nil
}

func readOptionalJSON(path string, fn func([]byte) error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return fn(stripUTF8BOM(data))
}

func stripUTF8BOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

func parseProxy(data []byte, product *Product) error {
	var envelope struct {
		Proxy struct {
			ServiceID          int             `json:"service_id"`
			AuthType           string          `json:"auth_type"`
			AuthUserKey        string          `json:"auth_user_key"`
			AuthAppID          string          `json:"auth_app_id"`
			AuthAppKey         string          `json:"auth_app_key"`
			OIDCIssuerType     string          `json:"oidc_issuer_type"`
			OIDCIssuerEndpoint string          `json:"oidc_issuer_endpoint"`
			PoliciesConfig     json.RawMessage `json:"policies_config"`
			StagingEndpoint    string          `json:"staging_endpoint"`
			ProductionEndpoint string          `json:"endpoint"`
		} `json:"proxy"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	product.ServiceID = envelope.Proxy.ServiceID
	product.AuthType = inferAuthTypeFromProxy(
		envelope.Proxy.AuthType,
		envelope.Proxy.AuthUserKey,
		envelope.Proxy.AuthAppID,
		envelope.Proxy.AuthAppKey,
		envelope.Proxy.OIDCIssuerType,
		envelope.Proxy.OIDCIssuerEndpoint,
		envelope.Proxy.PoliciesConfig,
	)
	product.StagingEndpoint = envelope.Proxy.StagingEndpoint
	product.ProductionEndpoint = envelope.Proxy.ProductionEndpoint
	return nil
}

const hiddenPolicyApicast = "apicast"

type policyConfigEntry struct {
	Name    string `json:"name"`
	Enabled *bool  `json:"enabled"`
}

func policyVisible(name string, enabled *bool) bool {
	if name == "" || name == hiddenPolicyApicast {
		return false
	}
	if enabled != nil && !*enabled {
		return false
	}
	return true
}

func visiblePoliciesFromEntries(entries []policyConfigEntry) []Policy {
	out := make([]Policy, 0, len(entries))
	for _, entry := range entries {
		if policyVisible(entry.Name, entry.Enabled) {
			out = append(out, Policy{Name: entry.Name})
		}
	}
	return out
}

func parsePolicyConfigJSONArray(raw json.RawMessage) []Policy {
	if len(raw) == 0 {
		return nil
	}
	var entries []policyConfigEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		var encoded string
		if json.Unmarshal(raw, &encoded) != nil {
			return nil
		}
		if err := json.Unmarshal([]byte(encoded), &entries); err != nil {
			return nil
		}
	}
	return visiblePoliciesFromEntries(entries)
}

func policyNamesFromProxyFile(data []byte) []Policy {
	var envelope struct {
		Proxy struct {
			PoliciesConfig json.RawMessage `json:"policies_config"`
		} `json:"proxy"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}
	return parsePolicyConfigJSONArray(envelope.Proxy.PoliciesConfig)
}

func parsePolicies(data []byte) []Policy {
	var root struct {
		Policies []struct {
			Policy struct {
				Name    string `json:"name"`
				Enabled *bool  `json:"enabled"`
			} `json:"policy"`
		} `json:"policies"`
		PoliciesConfig json.RawMessage `json:"policies_config"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	if len(root.Policies) > 0 {
		entries := make([]policyConfigEntry, 0, len(root.Policies))
		for _, item := range root.Policies {
			entries = append(entries, policyConfigEntry{
				Name:    item.Policy.Name,
				Enabled: item.Policy.Enabled,
			})
		}
		return visiblePoliciesFromEntries(entries)
	}
	return parsePolicyConfigJSONArray(root.PoliciesConfig)
}

func parseBackendUsages(data []byte) []BackendUsage {
	type usageEntry struct {
		BackendUsage struct {
			BackendID int    `json:"backend_id"`
			Path      string `json:"path"`
		} `json:"backend_usage"`
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil
	}

	var entries []usageEntry
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			return nil
		}
	} else {
		var envelope struct {
			BackendUsages []usageEntry `json:"backend_usages"`
		}
		if err := json.Unmarshal(trimmed, &envelope); err != nil {
			return nil
		}
		entries = envelope.BackendUsages
	}

	out := make([]BackendUsage, 0, len(entries))
	for _, item := range entries {
		out = append(out, BackendUsage{
			BackendID: item.BackendUsage.BackendID,
			Path:      item.BackendUsage.Path,
		})
	}
	return out
}

func parseApplicationPlans(data []byte) []ApplicationPlan {
	var envelope struct {
		Plans []struct {
			ApplicationPlan struct {
				ID         int    `json:"id"`
				Name       string `json:"name"`
				SystemName string `json:"system_name"`
				State      string `json:"state"`
			} `json:"application_plan"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}
	out := make([]ApplicationPlan, 0, len(envelope.Plans))
	for _, item := range envelope.Plans {
		plan := item.ApplicationPlan
		if plan.ID == 0 && plan.Name == "" && plan.SystemName == "" {
			continue
		}
		out = append(out, ApplicationPlan{
			ID:         plan.ID,
			Name:       plan.Name,
			SystemName: plan.SystemName,
			State:      plan.State,
		})
	}
	return out
}

func parseOIDCConfiguration(data []byte) *OIDCConfiguration {
	var envelope struct {
		OIDC struct {
			IssuerType     string `json:"issuer_type"`
			IssuerEndpoint string `json:"issuer_endpoint"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil
	}
	if envelope.OIDC.IssuerType == "" && envelope.OIDC.IssuerEndpoint == "" {
		var flat map[string]json.RawMessage
		if err := json.Unmarshal(data, &flat); err != nil {
			return nil
		}
		issuerType := stringJSON(flat["issuer_type"])
		issuerEndpoint := stringJSON(flat["issuer_endpoint"])
		if issuerType == "" && issuerEndpoint == "" {
			return nil
		}
		return &OIDCConfiguration{
			IssuerType:     issuerType,
			IssuerEndpoint: issuerEndpoint,
			Fields:         stringifyMap(flat),
		}
	}
	return &OIDCConfiguration{
		IssuerType:     envelope.OIDC.IssuerType,
		IssuerEndpoint: envelope.OIDC.IssuerEndpoint,
	}
}

func loadApplications(dir string) ([]Application, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read applications directory: %w", err)
	}

	var applications []Application
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "page-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read applications %s: %w", entry.Name(), err)
		}
		pageApps, err := parseApplicationsPage(stripUTF8BOM(data))
		if err != nil {
			return nil, fmt.Errorf("parse applications %s: %w", entry.Name(), err)
		}
		applications = append(applications, pageApps...)
	}
	return applications, nil
}

func parseApplicationsPage(data []byte) ([]Application, error) {
	var pages []json.RawMessage
	if err := json.Unmarshal(data, &pages); err != nil {
		return nil, err
	}

	out := make([]Application, 0, len(pages))
	for _, item := range pages {
		var envelope struct {
			Application struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				ServiceID int    `json:"service_id"`
				PlanID    int    `json:"plan_id"`
				AccountID int    `json:"account_id"`
				State     string `json:"state"`
			} `json:"application"`
		}
		if err := json.Unmarshal(item, &envelope); err != nil {
			continue
		}
		app := envelope.Application
		if app.ID == 0 {
			continue
		}
		out = append(out, Application{
			ID:        app.ID,
			Name:      app.Name,
			ServiceID: app.ServiceID,
			PlanID:    app.PlanID,
			AccountID: app.AccountID,
			State:     app.State,
		})
	}
	return out, nil
}

func stringJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func intJSON(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0
	}
	return value
}

func stringifyMap(raw map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = strings.TrimSpace(string(value))
	}
	return out
}
