package seed

import (
	"context"
	"net/url"
	"strings"
)

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
