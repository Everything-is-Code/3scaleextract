package seed

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

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
