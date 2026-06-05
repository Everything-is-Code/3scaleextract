package seed

import "fmt"

// Fixture catalog for export testing. Each product exercises different export dimensions.

type BackendFixture struct {
	SystemName      string
	Name            string
	PrivateEndpoint string
	Description     string
}

type PlanFixture struct {
	SystemName string
	Name       string
	LimitValue int
	PriceRules int // number of pricing rules to create
}

type ApplicationFixture struct {
	Name        string
	Plan        string // plan system_name
	RedirectURL string // OIDC callback URL (RH-SSO / Keycloak)
}

type ProductFixture struct {
	SystemName   string
	Name         string
	Description  string
	AuthMode     string // api_key, app_id, oidc
	BackendRefs  []string
	PolicyNames  []string
	Plans        []PlanFixture
	Applications []ApplicationFixture
	OIDC         *OIDCFixture
}

// OIDCFixture simulates Red Hat Single Sign-On (Keycloak) integration via Zync.
type OIDCFixture struct {
	// Realm URL without credentials, e.g. https://sso.example.com/auth/realms/seed-demo
	RealmURL string
	// Zync service-account credentials embedded in the issuer endpoint.
	ZyncClientID     string
	ZyncClientSecret string
	RedirectURL      string
	StandardFlow     bool
	ServiceAccounts  bool
}

func (o OIDCFixture) IssuerEndpoint() string {
	host := o.RealmURL
	if len(host) > 8 && host[:8] == "https://" {
		host = host[8:]
	}
	return fmt.Sprintf("https://%s:%s@%s", o.ZyncClientID, o.ZyncClientSecret, host)
}

type AccountFixture struct {
	OrgName  string
	Username string
	Email    string
	Password string
}

func DefaultFixtures() ([]BackendFixture, AccountFixture, []ProductFixture) {
	backends := []BackendFixture{
		{
			SystemName:      "seed_payments",
			Name:            "Seed Payments Backend",
			PrivateEndpoint: "https://payments.example.com:443",
			Description:     "Dummy payments upstream for export tests",
		},
		{
			SystemName:      "seed_inventory",
			Name:            "Seed Inventory Backend",
			PrivateEndpoint: "https://inventory.example.com:443",
			Description:     "Dummy inventory upstream for export tests",
		},
		{
			SystemName:      "seed_billing",
			Name:            "Seed Billing Backend",
			PrivateEndpoint: "https://billing.example.com:443",
			Description:     "Dummy billing upstream for export tests",
		},
	}

	account := AccountFixture{
		OrgName:  "Seed Demo Organization",
		Username: "seed_demo_user",
		Email:    "seed-demo@example.com",
		Password: "SeedDemoPass123!",
	}

	rhssoRealm := "https://sso.example.com/auth/realms/seed-demo"

	products := []ProductFixture{
		{
			SystemName:  "seed_api_key",
			Name:        "Seed API Key Product",
			Description: "Product with API Key auth, single backend, CORS policy",
			AuthMode:    "api_key",
			BackendRefs: []string{"seed_payments"},
			PolicyNames: []string{"cors"},
			Plans: []PlanFixture{
				{SystemName: "basic", Name: "Basic Plan", LimitValue: 1000},
			},
			Applications: []ApplicationFixture{
				{Name: "seed-app-key-01", Plan: "basic"},
				{Name: "seed-app-key-02", Plan: "basic"},
			},
		},
		{
			SystemName:  "seed_oidc",
			Name:        "Seed OIDC Product",
			Description: "Product with OIDC auth (RH-SSO) and JWT claim check policy",
			AuthMode:    "oidc",
			BackendRefs: []string{"seed_billing"},
			PolicyNames: []string{"jwt_claim_check", "cors"},
			OIDC: &OIDCFixture{
				RealmURL:         rhssoRealm,
				ZyncClientID:     "zync-admin",
				ZyncClientSecret: "ZyncSecret123-for-rhsso",
				RedirectURL:      "https://seed-oidc.example.com/callback",
				StandardFlow:     true,
				ServiceAccounts:  true,
			},
			Plans: []PlanFixture{
				{SystemName: "enterprise", Name: "Enterprise Plan", LimitValue: 10000, PriceRules: 6},
			},
			Applications: []ApplicationFixture{
				{
					Name:        "seed-oidc-app-01",
					Plan:        "enterprise",
					RedirectURL: "https://seed-oidc.example.com/callback",
				},
			},
		},
		{
			SystemName:  "seed_app_id",
			Name:        "Seed App ID Product",
			Description: "Product with App ID + App Key auth, two backends, IP check",
			AuthMode:    "app_id",
			BackendRefs: []string{"seed_payments", "seed_inventory"},
			PolicyNames: []string{"ip_check", "cors"},
			Plans: []PlanFixture{
				{SystemName: "pro", Name: "Pro Plan", LimitValue: 5000, PriceRules: 3},
			},
			Applications: []ApplicationFixture{
				{Name: "seed-app-id-01", Plan: "pro"},
				{Name: "seed-app-id-02", Plan: "pro"},
				{Name: "seed-app-id-03", Plan: "pro"},
			},
		},
		{
			SystemName:  "seed_multi_backend",
			Name:        "Seed Multi-Backend Product",
			Description: "Product with three backends and edge limiting policy",
			AuthMode:    "api_key",
			BackendRefs: []string{"seed_payments", "seed_inventory", "seed_billing"},
			PolicyNames: []string{"edge_limit", "url_rewriting"},
			Plans: []PlanFixture{
				{SystemName: "starter", Name: "Starter Plan", LimitValue: 500},
				{SystemName: "scale", Name: "Scale Plan", LimitValue: 20000, PriceRules: 2},
			},
			Applications: []ApplicationFixture{
				{Name: "seed-multi-01", Plan: "starter"},
				{Name: "seed-multi-02", Plan: "starter"},
				{Name: "seed-multi-03", Plan: "scale"},
				{Name: "seed-multi-04", Plan: "scale"},
				{Name: "seed-multi-05", Plan: "scale"},
			},
		},
	}

	return backends, account, products
}

// CoverageMatrix documents what each fixture validates in export output.
var CoverageMatrix = map[string][]string{
	"seed_api_key": {
		"auth: api_key", "backends: 1", "policies: cors", "plans: 1", "applications: 2",
	},
	"seed_app_id": {
		"auth: app_id", "backends: 2", "policies: ip_check,cors", "plans: 1 + pricing_rules", "applications: 3",
	},
	"seed_oidc": {
		"auth: oidc (RH-SSO)", "oidc_configuration.json", "policies: jwt_claim_check,cors", "pricing_rules: 6", "applications: client_id + client_secret",
	},
	"seed_multi_backend": {
		"backends: 3", "backend_usages", "policies: edge_limit,url_rewriting", "plans: 2", "applications: 5",
	},
}
