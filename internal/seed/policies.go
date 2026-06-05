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
	},
	"ip_check": {
		"allow_ip_addrs": []string{"10.0.0.0/8", "192.168.0.0/16"},
	},
	"jwt_claim_check": {
		"rules": []map[string]any{
			{
				"operations": []map[string]any{
					{
						"jwt_claim":  "azp",
						"value_type": "plain",
						"value":      "client_id",
					},
				},
			},
		},
	},
	"edge_limit": {
		"limits": []map[string]any{
			{
				"metric": "hits",
				"period": "minute",
				"value":  100,
			},
		},
	},
	"url_rewriting": {
		"rules": []map[string]any{
			{
				"match":  "/seed-public",
				"replace": "/seed-private",
			},
		},
	},
}
