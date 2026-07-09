package visualize

import (
	"encoding/json"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func resolveProductAuthType(product *Product, yamlPath string) {
	if product == nil {
		return
	}

	if fromYAML, err := readAuthTypeFromYAML(yamlPath); err == nil && fromYAML != "" {
		product.AuthType = fromYAML
		return
	}

	authType := normalizeAuthType(product.AuthType)
	if authType == "" && product.OIDC != nil && strings.TrimSpace(product.OIDC.IssuerEndpoint) != "" {
		authType = "oidc"
	}
	product.AuthType = authType
}

func readAuthTypeFromYAML(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	spec, ok := productSpecFromYAML(stripUTF8BOM(data))
	if !ok {
		return "", nil
	}
	return authTypeFromProductSpec(spec), nil
}

func productSpecFromYAML(data []byte) (map[string]any, bool) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, false
	}

	kind, _ := root["kind"].(string)
	switch kind {
	case "Product":
		spec, _ := root["spec"].(map[string]any)
		return spec, spec != nil
	case "List":
		items, _ := root["items"].([]any)
		for _, item := range items {
			entry, _ := item.(map[string]any)
			if entry == nil {
				continue
			}
			if entryKind, _ := entry["kind"].(string); entryKind != "Product" {
				continue
			}
			spec, _ := entry["spec"].(map[string]any)
			if spec != nil {
				return spec, true
			}
		}
	}
	return nil, false
}

func authTypeFromProductSpec(spec map[string]any) string {
	deployment, _ := spec["deployment"].(map[string]any)
	if deployment == nil {
		return ""
	}
	apicast, _ := deployment["apicastHosted"].(map[string]any)
	if apicast == nil {
		return ""
	}
	authentication, _ := apicast["authentication"].(map[string]any)
	if authentication == nil {
		return ""
	}
	if _, ok := authentication["oidc"]; ok {
		return "oidc"
	}
	if _, ok := authentication["userkey"]; ok {
		return "api_key"
	}
	if _, ok := authentication["userKey"]; ok {
		return "api_key"
	}
	if _, ok := authentication["appKeyAppID"]; ok {
		return "app_id_and_app_key"
	}
	return ""
}

func inferAuthTypeFromProxy(authType, userKey, appID, appKey, oidcIssuerType, oidcIssuerEndpoint string, policiesConfig json.RawMessage) string {
	if t := normalizeAuthType(authType); t != "" {
		return t
	}
	if strings.TrimSpace(oidcIssuerEndpoint) != "" {
		return "oidc"
	}
	if fromPolicy := authTypeFromPoliciesConfig(policiesConfig); fromPolicy != "" {
		return fromPolicy
	}

	userEnabled := proxyAuthModeEnabled(userKey)
	appIDEnabled := proxyAuthModeEnabled(appID)
	appKeyEnabled := proxyAuthModeEnabled(appKey)

	if strings.EqualFold(strings.TrimSpace(userKey), "true") && strings.EqualFold(strings.TrimSpace(appID), "false") {
		return "api_key"
	}
	if strings.EqualFold(strings.TrimSpace(appID), "true") && strings.EqualFold(strings.TrimSpace(userKey), "false") {
		return "app_id_and_app_key"
	}
	if userEnabled && !appIDEnabled && !appKeyEnabled {
		return "api_key"
	}
	if appIDEnabled && appKeyEnabled {
		return "app_id_and_app_key"
	}
	if userEnabled {
		return "api_key"
	}
	return ""
}

func authTypeFromPoliciesConfig(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var policies []struct {
		Name          string `json:"name"`
		Enabled       bool   `json:"enabled"`
		Configuration struct {
			AuthType string `json:"auth_type"`
		} `json:"configuration"`
	}
	if err := json.Unmarshal(raw, &policies); err != nil {
		return ""
	}
	for _, policy := range policies {
		if policy.Name != "default_credentials" {
			continue
		}
		if t := normalizeAuthType(policy.Configuration.AuthType); t != "" {
			if policy.Enabled {
				return t
			}
		}
	}
	for _, policy := range policies {
		if policy.Name != "default_credentials" {
			continue
		}
		if t := normalizeAuthType(policy.Configuration.AuthType); t != "" {
			return t
		}
	}
	return ""
}

func proxyAuthModeEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "false":
		return false
	default:
		return true
	}
}

func normalizeAuthType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return ""
	case "api_key", "user_key", "apikey", "userkey":
		return "api_key"
	case "app_key", "app_id", "app_id_and_app_key", "app_key_and_app_id":
		return "app_id_and_app_key"
	case "oidc", "openid_connect":
		return "oidc"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
