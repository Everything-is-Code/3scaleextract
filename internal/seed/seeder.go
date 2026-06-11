package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
)

type Options struct {
	SkipExisting bool
	DryRun       bool
}

type Result struct {
	Backends     map[string]int
	Services     map[string]int
	Plans        map[string]int
	Applications []string
	AccountID    int
	Skipped      []string
}

type Seeder struct {
	client  admin.Client
	opts    Options
	skipped []string
}

func NewSeeder(client admin.Client, opts Options) *Seeder {
	return &Seeder{client: client, opts: opts}
}

func (s *Seeder) Run(ctx context.Context) (*Result, error) {
	backends, account, products := DefaultFixtures()
	result := &Result{
		Backends: make(map[string]int),
		Services: make(map[string]int),
		Plans:    make(map[string]int),
	}

	for _, b := range backends {
		id, err := s.ensureBackend(ctx, b)
		if err != nil {
			return result, fmt.Errorf("backend %q: %w", b.SystemName, err)
		}
		result.Backends[b.SystemName] = id
	}

	accountID, err := s.ensureAccount(ctx, account)
	if err != nil {
		return result, fmt.Errorf("account: %w", err)
	}
	result.AccountID = accountID

	for _, p := range products {
		if err := s.seedProduct(ctx, p, result, accountID); err != nil {
			return result, fmt.Errorf("product %q: %w", p.SystemName, err)
		}
	}
	result.Skipped = append(result.Skipped, s.skipped...)

	return result, nil
}

func (s *Seeder) ensureBackend(ctx context.Context, b BackendFixture) (int, error) {
	if s.opts.DryRun {
		return 0, nil
	}
	if id, ok := s.findBackend(ctx, b.SystemName); ok {
		if s.opts.SkipExisting {
			s.resultSkip("backend:" + b.SystemName)
			return id, nil
		}
	}
	if s.opts.DryRun {
		return 0, nil
	}
	form := url.Values{
		"name":             {b.Name},
		"system_name":       {b.SystemName},
		"private_endpoint":  {b.PrivateEndpoint},
		"description":       {b.Description},
	}
	var resp struct {
		Backend struct {
			ID int `json:"id"`
		} `json:"backend_api"`
	}
	if err := s.client.PostForm(ctx, "/backend_apis", form, &resp); err != nil {
		if isDuplicateError(err) {
			if id, ok := s.findBackend(ctx, b.SystemName); ok {
				s.resultSkip("backend:" + b.SystemName)
				return id, nil
			}
		}
		return 0, err
	}
	return resp.Backend.ID, nil
}

func (s *Seeder) ensureAccount(ctx context.Context, a AccountFixture) (int, error) {
	if s.opts.DryRun {
		return 0, nil
	}
	if id, ok := s.findAccountByUsername(ctx, a.Username); ok {
		if s.opts.SkipExisting {
			s.resultSkip("account:" + a.Username)
			return id, nil
		}
	}
	form := url.Values{
		"org_name": {a.OrgName},
		"username": {a.Username},
		"email":    {a.Email},
		"password": {a.Password},
	}
	var resp struct {
		Account struct {
			ID int `json:"id"`
		} `json:"account"`
	}
	// Developer accounts are created via signup, not POST /accounts (404 on many tenants).
	if err := s.client.PostForm(ctx, "/signup", form, &resp); err != nil {
		return 0, err
	}
	return resp.Account.ID, nil
}

func (s *Seeder) seedProduct(ctx context.Context, p ProductFixture, result *Result, accountID int) error {
	serviceID, err := s.ensureService(ctx, p)
	if err != nil {
		return err
	}
	result.Services[p.SystemName] = serviceID
	if s.opts.DryRun || serviceID == 0 {
		return nil
	}

	if err := s.configureAuth(ctx, serviceID, p); err != nil {
		return err
	}
	if err := s.ensureMappingRule(ctx, serviceID); err != nil {
		return err
	}
	if err := s.linkBackends(ctx, serviceID, p, result); err != nil {
		return err
	}
	if err := s.configurePolicies(ctx, serviceID, p); err != nil {
		return err
	}
	if err := s.deployProxy(ctx, serviceID); err != nil {
		return err
	}

	metricID, err := s.hitsMetricID(ctx, serviceID)
	if err != nil {
		return err
	}

	planIDs := map[string]int{}
	for _, plan := range p.Plans {
		planID, err := s.ensurePlan(ctx, serviceID, plan)
		if err != nil {
			return err
		}
		planIDs[plan.SystemName] = planID
		result.Plans[p.SystemName+"/"+plan.SystemName] = planID
		if plan.LimitValue > 0 {
			if err := s.createLimit(ctx, planID, metricID, plan.LimitValue); err != nil {
				fmt.Fprintf(os.Stderr, "warn: limit for %s/%s: %v\n", p.SystemName, plan.SystemName, err)
			}
		}
		for i := 0; i < plan.PriceRules; i++ {
			if err := s.createPricingRule(ctx, planID, metricID, i+1); err != nil {
				fmt.Fprintf(os.Stderr, "warn: pricing rule for %s/%s: %v\n", p.SystemName, plan.SystemName, err)
			}
		}
	}

	if p.AuthMode == "oidc" {
		if err := s.refreshOIDCApplications(ctx, accountID, serviceID); err != nil {
			fmt.Fprintf(os.Stderr, "warn: refresh oidc apps for %s: %v\n", p.SystemName, err)
		}
	}

	for _, app := range p.Applications {
		planID, ok := planIDs[app.Plan]
		if !ok {
			return fmt.Errorf("unknown plan %q for application %q", app.Plan, app.Name)
		}
		if err := s.ensureApplication(ctx, accountID, planID, app, p.AuthMode, result); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) ensureService(ctx context.Context, p ProductFixture) (int, error) {
	if s.opts.DryRun {
		return 0, nil
	}
	if id, ok := s.findService(ctx, p.SystemName); ok {
		if s.opts.SkipExisting {
			s.resultSkip("service:" + p.SystemName)
			return id, nil
		}
	}
	form := url.Values{
		"name":              {p.Name},
		"system_name":       {p.SystemName},
		"description":       {p.Description},
		"backend_version":   {backendVersion(p.AuthMode)},
		"deployment_option": {"hosted"},
	}
	var resp struct {
		Service struct {
			ID int `json:"id"`
		} `json:"service"`
	}
	if err := s.client.PostForm(ctx, "/services", form, &resp); err != nil {
		if isDuplicateError(err) {
			if id, ok := s.findService(ctx, p.SystemName); ok {
				s.resultSkip("service:" + p.SystemName)
				return id, nil
			}
		}
		return 0, err
	}
	return resp.Service.ID, nil
}

func backendVersion(authMode string) string {
	switch authMode {
	case "app_id":
		return "2"
	case "oidc":
		return "oidc"
	default:
		return "1"
	}
}

func (s *Seeder) updateServiceBackendVersion(ctx context.Context, serviceID int, authMode string) error {
	if s.opts.DryRun {
		return nil
	}
	form := url.Values{"backend_version": {backendVersion(authMode)}}
	return s.client.PutForm(ctx, fmt.Sprintf("/services/%d", serviceID), form, nil)
}

func (s *Seeder) configureAuth(ctx context.Context, serviceID int, p ProductFixture) error {
	if s.opts.DryRun {
		return nil
	}
	if err := s.updateServiceBackendVersion(ctx, serviceID, p.AuthMode); err != nil {
		return err
	}

	path := fmt.Sprintf("/services/%d/proxy", serviceID)
	form := url.Values{"credentials_location": {"headers"}}
	switch p.AuthMode {
	case "app_id":
		form.Set("auth_app_id", "true")
		form.Set("auth_user_key", "false")
	case "oidc":
		if p.OIDC == nil {
			return fmt.Errorf("oidc fixture required for product %q", p.SystemName)
		}
		form.Set("oidc_issuer_type", "keycloak")
		form.Set("oidc_issuer_endpoint", p.OIDC.IssuerEndpoint())
	case "api_key":
		form.Set("auth_user_key", "true")
		form.Set("auth_app_id", "false")
	default:
		form.Set("auth_user_key", "true")
		form.Set("auth_app_id", "false")
	}
	if err := s.client.PutForm(ctx, path, form, nil); err != nil {
		return err
	}

	if p.AuthMode == "oidc" && p.OIDC != nil {
		oidcPath := fmt.Sprintf("/services/%d/proxy/oidc_configuration", serviceID)
		oidcForm := url.Values{
			"oidc_configuration[issuer_type]": {"keycloak"},
		}
		if p.OIDC.StandardFlow {
			oidcForm.Set("oidc_configuration[standard_flow_enabled]", "true")
		}
		if p.OIDC.ServiceAccounts {
			oidcForm.Set("oidc_configuration[service_accounts_enabled]", "true")
		}
		if err := s.client.PutForm(ctx, oidcPath, oidcForm, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *Seeder) ensureMappingRule(ctx context.Context, serviceID int) error {
	if s.opts.DryRun {
		return nil
	}
	metricID, err := s.hitsMetricID(ctx, serviceID)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/services/%d/proxy/mapping_rules", serviceID)
	form := url.Values{
		"http_method": {"GET"},
		"pattern":     {"/seed-health"},
		"metric_id":   {strconv.Itoa(metricID)},
		"delta":       {"1"},
	}
	if err := s.client.PostForm(ctx, path, form, nil); err != nil && !isDuplicateError(err) {
		return err
	}
	return nil
}

func (s *Seeder) linkBackends(ctx context.Context, serviceID int, p ProductFixture, result *Result) error {
	if s.opts.DryRun {
		return nil
	}
	for i, ref := range p.BackendRefs {
		backendID, ok := result.Backends[ref]
		if !ok {
			return fmt.Errorf("backend %q not created", ref)
		}
		path := fmt.Sprintf("/services/%d/backend_usages", serviceID)
		form := url.Values{
			"backend_api_id": {strconv.Itoa(backendID)},
			"path":           {backendPath(i)},
		}
		if err := s.client.PostForm(ctx, path, form, nil); err != nil {
			if isDuplicateError(err) {
				continue
			}
			return err
		}
	}
	return nil
}

func backendPath(index int) string {
	if index == 0 {
		return "/"
	}
	return fmt.Sprintf("/backend-%d", index+1)
}

func (s *Seeder) configurePolicies(ctx context.Context, serviceID int, p ProductFixture) error {
	if s.opts.DryRun || len(p.PolicyNames) == 0 {
		return nil
	}
	chain, err := buildPolicyChain(p.PolicyNames)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/services/%d/proxy/policies", serviceID)
	form := url.Values{"policies_config": {chain}}
	return s.client.PutForm(ctx, path, form, nil)
}

func (s *Seeder) deployProxy(ctx context.Context, serviceID int) error {
	if s.opts.DryRun {
		return nil
	}
	path := fmt.Sprintf("/services/%d/proxy/deploy", serviceID)
	return s.client.PostForm(ctx, path, url.Values{"environment": {"sandbox"}}, nil)
}

func (s *Seeder) ensurePlan(ctx context.Context, serviceID int, plan PlanFixture) (int, error) {
	if s.opts.DryRun {
		return 0, nil
	}
	path := fmt.Sprintf("/services/%d/application_plans", serviceID)
	form := url.Values{
		"name":        {plan.Name},
		"system_name": {plan.SystemName},
		"state_event": {"publish"},
	}
	var resp struct {
		Plan struct {
			ID int `json:"id"`
		} `json:"application_plan"`
	}
	if err := s.client.PostForm(ctx, path, form, &resp); err != nil {
		if isDuplicateError(err) {
			if id, ok := s.findPlanID(ctx, serviceID, plan.SystemName); ok {
				return id, nil
			}
			return 0, nil
		}
		return 0, err
	}
	return resp.Plan.ID, nil
}

func (s *Seeder) createLimit(ctx context.Context, planID, metricID, value int) error {
	if s.opts.DryRun {
		return nil
	}
	path := fmt.Sprintf("/application_plans/%d/metrics/%d/limits", planID, metricID)
	form := url.Values{
		"limit[period]": {"minute"},
		"limit[value]":  {strconv.Itoa(value)},
	}
	if err := s.client.PostForm(ctx, path, form, nil); err != nil && !isDuplicateError(err) {
		return err
	}
	return nil
}

func (s *Seeder) createPricingRule(ctx context.Context, planID, metricID, index int) error {
	if s.opts.DryRun {
		return nil
	}
	path := fmt.Sprintf("/application_plans/%d/metrics/%d/pricing_rules", planID, metricID)
	from := (index-1)*100 + 1
	to := index * 100
	form := url.Values{
		"pricing_rule[from]":           {strconv.Itoa(from)},
		"pricing_rule[to]":             {strconv.Itoa(to)},
		"pricing_rule[price_per_unit]": {"0.001"},
	}
	if err := s.client.PostForm(ctx, path, form, nil); err != nil && !isDuplicateError(err) {
		return err
	}
	return nil
}

func (s *Seeder) ensureApplication(ctx context.Context, accountID, planID int, app ApplicationFixture, authMode string, result *Result) error {
	if s.opts.DryRun {
		return nil
	}
	path := fmt.Sprintf("/accounts/%d/applications", accountID)
	form := url.Values{
		"plan_id":              {strconv.Itoa(planID)},
		"name":                 {app.Name},
		"application[plan_id]": {strconv.Itoa(planID)},
		"application[name]":    {app.Name},
	}
	if authMode == "oidc" && app.RedirectURL != "" {
		form.Set("application[redirect_url]", app.RedirectURL)
	}
	var resp struct {
		Application struct {
			ID           int    `json:"id"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
			UserKey      string `json:"user_key"`
		} `json:"application"`
	}
	if err := s.client.PostForm(ctx, path, form, &resp); err != nil {
		if isDuplicateError(err) || strings.Contains(strings.ToLower(err.Error()), "404") {
			result.Applications = append(result.Applications, app.Name)
			return nil
		}
		return err
	}
	if authMode == "oidc" && resp.Application.ClientID == "" && resp.Application.UserKey != "" {
		return fmt.Errorf("application %q created with user_key instead of OIDC client credentials", app.Name)
	}
	result.Applications = append(result.Applications, app.Name)
	return nil
}

func (s *Seeder) refreshOIDCApplications(ctx context.Context, accountID, serviceID int) error {
	if s.opts.DryRun {
		return nil
	}
	var resp struct {
		Applications []struct {
			Application struct {
				ID        int    `json:"id"`
				Name      string `json:"name"`
				UserKey   string `json:"user_key"`
				ClientID  string `json:"client_id"`
				ServiceID int    `json:"service_id"`
			} `json:"application"`
		} `json:"applications"`
	}
	path := fmt.Sprintf("/accounts/%d/applications", accountID)
	if err := s.client.Get(ctx, path, &resp); err != nil {
		return err
	}
	for _, item := range resp.Applications {
		app := item.Application
		if app.ServiceID != serviceID {
			continue
		}
		delPath := fmt.Sprintf("/accounts/%d/applications/%d", accountID, app.ID)
		if err := s.client.Delete(ctx, delPath); err != nil {
			fmt.Fprintf(os.Stderr, "warn: delete app %q (%d): %v\n", app.Name, app.ID, err)
		}
	}
	return nil
}

func (s *Seeder) hitsMetricID(ctx context.Context, serviceID int) (int, error) {
	var resp struct {
		Metrics []struct {
			Metric struct {
				ID         int    `json:"id"`
				SystemName string `json:"system_name"`
			} `json:"metric"`
		} `json:"metrics"`
	}
	path := fmt.Sprintf("/services/%d/metrics", serviceID)
	if err := s.client.Get(ctx, path, &resp); err != nil {
		return 0, err
	}
	for _, item := range resp.Metrics {
		if item.Metric.SystemName == "hits" {
			return item.Metric.ID, nil
		}
	}
	if len(resp.Metrics) > 0 {
		return resp.Metrics[0].Metric.ID, nil
	}
	return 0, fmt.Errorf("no metrics found for service %d", serviceID)
}

func (s *Seeder) findService(ctx context.Context, systemName string) (int, bool) {
	var resp struct {
		Services []struct {
			Service struct {
				ID         int    `json:"id"`
				SystemName string `json:"system_name"`
			} `json:"service"`
		} `json:"services"`
	}
	if err := s.client.Get(ctx, "/services", &resp); err != nil {
		return 0, false
	}
	for _, entry := range resp.Services {
		if entry.Service.SystemName == systemName {
			return entry.Service.ID, true
		}
	}
	return 0, false
}

func (s *Seeder) findBackend(ctx context.Context, systemName string) (int, bool) {
	var resp struct {
		BackendAPIs []struct {
			BackendAPI struct {
				ID         int    `json:"id"`
				SystemName string `json:"system_name"`
			} `json:"backend_api"`
		} `json:"backend_apis"`
	}
	if err := s.client.Get(ctx, "/backend_apis", &resp); err != nil {
		return 0, false
	}
	for _, entry := range resp.BackendAPIs {
		if entry.BackendAPI.SystemName == systemName {
			return entry.BackendAPI.ID, true
		}
	}
	return 0, false
}

func (s *Seeder) findAccountByUsername(ctx context.Context, username string) (int, bool) {
	var resp struct {
		Accounts []struct {
			Account struct {
				ID       int    `json:"id"`
				Username string `json:"username"`
				OrgName  string `json:"org_name"`
			} `json:"account"`
		} `json:"accounts"`
	}
	if err := s.client.Get(ctx, "/accounts", &resp); err != nil {
		return 0, false
	}
	for _, item := range resp.Accounts {
		if strings.EqualFold(item.Account.Username, username) {
			return item.Account.ID, true
		}
		if strings.EqualFold(item.Account.OrgName, "Seed Demo Organization") {
			return item.Account.ID, true
		}
	}
	return 0, false
}

func (s *Seeder) findBySystemName(ctx context.Context, path, rootKey, itemKey, systemName string) (int, bool) {
	raw := map[string]json.RawMessage{}
	if err := s.client.Get(ctx, path, &raw); err != nil {
		return 0, false
	}
	itemsRaw, ok := raw[rootKey]
	if !ok {
		return 0, false
	}
	var items []json.RawMessage
	if err := json.Unmarshal(itemsRaw, &items); err != nil {
		return 0, false
	}
	for _, itemRaw := range items {
		var wrap map[string]json.RawMessage
		if err := json.Unmarshal(itemRaw, &wrap); err != nil {
			continue
		}
		entityRaw, ok := wrap[itemKey]
		if !ok {
			continue
		}
		var entity struct {
			ID         int    `json:"id"`
			SystemName string `json:"system_name"`
		}
		if err := json.Unmarshal(entityRaw, &entity); err != nil {
			continue
		}
		if entity.SystemName == systemName {
			return entity.ID, true
		}
	}
	return 0, false
}

func (s *Seeder) resultSkip(label string) {
	s.skipped = append(s.skipped, label)
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already been taken") ||
		strings.Contains(msg, "422") ||
		strings.Contains(msg, "has already been taken")
}

func (s *Seeder) findPlanID(ctx context.Context, serviceID int, systemName string) (int, bool) {
	var resp struct {
		Plans []struct {
			Plan struct {
				ID         int    `json:"id"`
				SystemName string `json:"system_name"`
			} `json:"application_plan"`
		} `json:"plans"`
	}
	path := fmt.Sprintf("/services/%d/application_plans", serviceID)
	if err := s.client.Get(ctx, path, &resp); err != nil {
		return 0, false
	}
	for _, entry := range resp.Plans {
		if entry.Plan.SystemName == systemName {
			return entry.Plan.ID, true
		}
	}
	return 0, false
}
