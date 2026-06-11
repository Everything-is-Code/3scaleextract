package seed

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
)

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
