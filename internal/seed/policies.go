package seed

import "encoding/json"

type policyEntry struct {
	Name          string         `json:"name"`
	Version       string         `json:"version"`
	Enabled       bool           `json:"enabled"`
	Configuration map[string]any `json:"configuration"`
}

func buildPolicyChain(names []string) (string, error) {
	chain := make([]policyEntry, 0, len(names)+1)
	for _, name := range names {
		cfg, ok := policyConfigurations[name]
		if !ok {
			cfg = map[string]any{}
		}
		chain = append(chain, policyEntry{
			Name:          name,
			Version:       "builtin",
			Enabled:       true,
			Configuration: cfg,
		})
	}
	chain = append(chain, policyEntry{
		Name:          "apicast",
		Version:       "builtin",
		Enabled:       true,
		Configuration: map[string]any{},
	})
	raw, err := json.Marshal(chain)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

var policyConfigurations = map[string]map[string]any{
	"cors": {
		"allow_credentials": true,
		"allow_origin":      "*",
		"allow_methods":     "GET, POST, OPTIONS",
		"allow_headers":     "Authorization, Content-Type",
		"max_age":           600,
	},
	"ip_check": {
		"check_type":        "whitelist",
		"client_ip_sources": []string{"X-Forwarded-For", "Remote Address"},
		"ips":               []string{"10.0.0.0/8", "192.168.0.0/16"},
		// legacy keys some tenants still use
		"allow_ip_addrs": []string{"10.0.0.0/8", "192.168.0.0/16"},
	},
	"jwt_claim_check": {
		"rules": []map[string]any{
			{
				"operations": []map[string]any{
					{
						"op":             "==",
						"jwt_claim":      "azp",
						"jwt_claim_type": "plain",
						"value":          "rhcl-seed-client",
						"value_type":     "plain",
					},
				},
			},
		},
	},
	// Legacy name used by older seed fixtures
	"edge_limit": {
		"limits": []map[string]any{
			{
				"metric": "hits",
				"period": "minute",
				"value":  100,
			},
		},
	},
	// Canonical 3scale / RHCL conversion policy name
	"edge_limiting": {
		"limits": []map[string]any{
			{
				"count":       100,
				"period":      60,
				"period_unit": "seconds",
			},
		},
	},
	// RHCL ConversionService expects "commands" (PCRE → Lua), not legacy "rules".
	"url_rewriting": {
		"commands": []map[string]any{
			{
				"op":      "sub",
				"regex":   "^/seed-public",
				"replace": "/seed-private",
			},
		},
	},
	"headers": {
		"request": []map[string]any{
			{"op": "set", "header": "X-Rhcl-Seed", "value": "headers"},
		},
		"response": []map[string]any{
			{"op": "set", "header": "X-Rhcl-Seed-Resp", "value": "ok"},
		},
	},
	"header_modification": {
		"request": []map[string]any{
			{"op": "set", "header": "X-Rhcl-Seed", "value": "header-modification"},
		},
		"response": []map[string]any{
			{"op": "set", "header": "X-Rhcl-Seed-Resp", "value": "ok"},
		},
	},
	"token_introspection": {
		"auth_type":         "client_id+client_secret",
		"client_id":         "rhcl-seed-client",
		"client_secret":     "rhcl-seed-secret",
		"introspection_url": "https://sso.example.com/auth/realms/seed/protocol/openid-connect/token/introspect",
	},
	"logging": {
		"enable_access_logs": true,
		"enable_json_logs":   true,
		"json_object_config": []map[string]any{
			{"key": "host", "value": "{{host}}", "value_type": "liquid"},
			{"key": "method", "value": "{{method}}", "value_type": "liquid"},
			{"key": "path", "value": "{{path}}", "value_type": "liquid"},
		},
	},
	"default_credentials": {
		"auth_type": "user_key",
		"user_key":  "rhcl-seed-anonymous-user-key",
	},
	// APIcast system name for "3scale Auth Caching"
	"caching": {
		"caching_type": "strict",
	},
	// Alias some tenants / docs still use
	"3scale_auth_caching": {
		"caching_type": "strict",
	},
	"upstream_connection": {
		"connect_timeout": 5,
		"send_timeout":    10,
		"read_timeout":    30,
	},
	// APIcast system name for "Response/Request Content Limits"
	"payload_limits": {
		"request":  1048576,
		"response": 2097152,
	},
	// Alias used by RHCL CompatibilityService / older docs
	"content_limits": {
		"request":  1048576,
		"response": 2097152,
	},
	"retry": {
		"retries": 3,
	},
	"keycloak_role_check": {
		"type": "whitelist",
		"scopes": []map[string]any{
			{
				"realm_roles": []map[string]any{
					{"name": "rhcl-demo-user", "name_type": "plain"},
				},
				"client_roles": []map[string]any{
					{
						"name": "rhcl-seed-client",
						"roles": []map[string]any{
							{"name": "api-access", "name_type": "plain"},
						},
					},
				},
				"methods": []string{"ANY"},
			},
		},
	},
}
