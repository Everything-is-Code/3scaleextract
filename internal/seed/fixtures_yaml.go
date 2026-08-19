package seed

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CatalogFile is the on-disk YAML shape for external fixture catalogs
// (e.g. migration-toolkit-rhcl/testdata/seed/catalog.yaml).
type CatalogFile struct {
	Account  AccountFixtureYAML   `yaml:"account"`
	Backends []BackendFixtureYAML `yaml:"backends"`
	Products []ProductFixtureYAML `yaml:"products"`
}

type AccountFixtureYAML struct {
	OrgName  string `yaml:"org_name"`
	Username string `yaml:"username"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

type BackendFixtureYAML struct {
	SystemName      string `yaml:"system_name"`
	Name            string `yaml:"name"`
	PrivateEndpoint string `yaml:"private_endpoint"`
	Description     string `yaml:"description"`
}

type PlanFixtureYAML struct {
	SystemName string `yaml:"system_name"`
	Name       string `yaml:"name"`
	LimitValue int    `yaml:"limit_value"`
	PriceRules int    `yaml:"price_rules"`
}

type ApplicationFixtureYAML struct {
	Name        string `yaml:"name"`
	Plan        string `yaml:"plan"`
	RedirectURL string `yaml:"redirect_url"`
}

type OIDCFixtureYAML struct {
	RealmURL         string `yaml:"realm_url"`
	ZyncClientID     string `yaml:"zync_client_id"`
	ZyncClientSecret string `yaml:"zync_client_secret"`
	RedirectURL      string `yaml:"redirect_url"`
	StandardFlow     bool   `yaml:"standard_flow"`
	ServiceAccounts  bool   `yaml:"service_accounts"`
}

type ProductFixtureYAML struct {
	SystemName   string                   `yaml:"system_name"`
	Name         string                   `yaml:"name"`
	Description  string                   `yaml:"description"`
	AuthMode     string                   `yaml:"auth_mode"`
	BackendRefs  []string                 `yaml:"backend_refs"`
	PolicyNames  []string                 `yaml:"policy_names"`
	Plans        []PlanFixtureYAML        `yaml:"plans"`
	Applications []ApplicationFixtureYAML `yaml:"applications"`
	OIDC         *OIDCFixtureYAML         `yaml:"oidc"`
	Coverage     []string                 `yaml:"coverage"`
}

// LoadFixturesFile reads a YAML catalog and returns seed fixtures.
func LoadFixturesFile(path string) ([]BackendFixture, AccountFixture, []ProductFixture, map[string][]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, AccountFixture{}, nil, nil, fmt.Errorf("read fixtures file: %w", err)
	}
	var cat CatalogFile
	if err := yaml.Unmarshal(raw, &cat); err != nil {
		return nil, AccountFixture{}, nil, nil, fmt.Errorf("parse fixtures YAML: %w", err)
	}
	if len(cat.Backends) == 0 {
		return nil, AccountFixture{}, nil, nil, fmt.Errorf("fixtures file %q: no backends", path)
	}
	if len(cat.Products) == 0 {
		return nil, AccountFixture{}, nil, nil, fmt.Errorf("fixtures file %q: no products", path)
	}

	backends := make([]BackendFixture, 0, len(cat.Backends))
	for _, b := range cat.Backends {
		backends = append(backends, BackendFixture{
			SystemName:      b.SystemName,
			Name:            b.Name,
			PrivateEndpoint: b.PrivateEndpoint,
			Description:     b.Description,
		})
	}

	account := AccountFixture{
		OrgName:  cat.Account.OrgName,
		Username: cat.Account.Username,
		Email:    cat.Account.Email,
		Password: cat.Account.Password,
	}
	if account.OrgName == "" {
		account.OrgName = "Seed Organization"
	}
	if account.Username == "" {
		account.Username = "seed_user"
	}
	if account.Email == "" {
		account.Email = "seed@example.com"
	}
	if account.Password == "" {
		account.Password = "SeedPass123!"
	}

	coverage := map[string][]string{}
	products := make([]ProductFixture, 0, len(cat.Products))
	for _, p := range cat.Products {
		pf := ProductFixture{
			SystemName:  p.SystemName,
			Name:        p.Name,
			Description: p.Description,
			AuthMode:    p.AuthMode,
			BackendRefs: p.BackendRefs,
			PolicyNames: p.PolicyNames,
		}
		if pf.AuthMode == "" {
			pf.AuthMode = "api_key"
		}
		for _, plan := range p.Plans {
			pf.Plans = append(pf.Plans, PlanFixture{
				SystemName: plan.SystemName,
				Name:       plan.Name,
				LimitValue: plan.LimitValue,
				PriceRules: plan.PriceRules,
			})
		}
		if len(pf.Plans) == 0 {
			pf.Plans = []PlanFixture{{SystemName: "basic", Name: "Basic", LimitValue: 1000}}
		}
		for _, app := range p.Applications {
			pf.Applications = append(pf.Applications, ApplicationFixture{
				Name:        app.Name,
				Plan:        app.Plan,
				RedirectURL: app.RedirectURL,
			})
		}
		if p.OIDC != nil {
			pf.OIDC = &OIDCFixture{
				RealmURL:         p.OIDC.RealmURL,
				ZyncClientID:     p.OIDC.ZyncClientID,
				ZyncClientSecret: p.OIDC.ZyncClientSecret,
				RedirectURL:      p.OIDC.RedirectURL,
				StandardFlow:     p.OIDC.StandardFlow,
				ServiceAccounts:  p.OIDC.ServiceAccounts,
			}
		}
		if len(p.Coverage) > 0 {
			coverage[p.SystemName] = append([]string{}, p.Coverage...)
		}
		products = append(products, pf)
	}

	return backends, account, products, coverage, nil
}
