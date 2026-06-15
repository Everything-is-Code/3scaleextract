package export

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sensitiveJSONKeys = map[string]struct{}{
	"access_token":              {},
	"client_secret":               {},
	"secret":                      {},
	"api_key":                     {},
	"user_key":                    {},
	"app_key":                     {},
	"provider_key":                {},
	"provider_verification_key":   {},
	"client_id":                   {},
	"app_id":                      {},
}

var issuerJSONKeys = map[string]struct{}{
	"issuer_endpoint":      {},
	"oidc_issuer_endpoint": {},
}

var yamlSecretPattern = regexp.MustCompile(`(?m)^(\s*(?:access_token|client_secret|secret|api_key|user_key|app_key|provider_key|provider_verification_key|client_id|app_id)\s*:\s*).+$`)

var yamlIssuerPattern = regexp.MustCompile(`(?m)^(\s*(?:issuer_endpoint|oidc_issuer_endpoint)\s*:\s*)(.+)$`)

const redactedValue = `"***REDACTED***"`

func RedactDirectory(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			return redactJSONFile(path)
		case ".yaml", ".yml":
			return redactYAMLFile(path)
		default:
			return nil
		}
	})
}

func redactJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	redactJSONValue(root)
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func redactYAMLFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	out := redactYAMLContent(string(data))
	return os.WriteFile(path, []byte(out), 0o644)
}

func redactYAMLContent(content string) string {
	content = yamlSecretPattern.ReplaceAllString(content, `${1}`+redactedValue)
	return yamlIssuerPattern.ReplaceAllStringFunc(content, func(match string) string {
		sub := yamlIssuerPattern.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		return sub[1] + stripURLUserinfo(strings.TrimSpace(sub[2]))
	})
}

func redactJSONValue(v any) {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			lowerK := strings.ToLower(k)
			if _, ok := sensitiveJSONKeys[lowerK]; ok {
				node[k] = "***REDACTED***"
				continue
			}
			if _, ok := issuerJSONKeys[lowerK]; ok {
				if s, ok := val.(string); ok {
					node[k] = stripURLUserinfo(s)
				}
				continue
			}
			redactJSONValue(val)
		}
	case []any:
		for _, item := range node {
			redactJSONValue(item)
		}
	}
}

func stripURLUserinfo(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
}

func containsIssuerUserinfo(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.User != nil
}

func ContainsCleartextSecret(data []byte) bool {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return yamlHasCleartextSecret(data)
	}
	return jsonHasCleartextSecret(root)
}

func yamlHasCleartextSecret(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		if sub := yamlSecretPattern.FindStringSubmatch(line); len(sub) == 2 {
			val := yamlValueFromLine(sub[0])
			if val != "" && val != "***REDACTED***" {
				return true
			}
		}
		if sub := yamlIssuerPattern.FindStringSubmatch(line); len(sub) == 3 {
			val := strings.TrimSpace(sub[2])
			val = strings.Trim(val, `"'`)
			if containsIssuerUserinfo(val) {
				return true
			}
		}
	}
	return false
}

func yamlValueFromLine(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.Trim(strings.TrimSpace(parts[1]), `"'`)
}

func jsonHasCleartextSecret(v any) bool {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			lowerK := strings.ToLower(k)
			if _, ok := sensitiveJSONKeys[lowerK]; ok {
				if s, ok := val.(string); ok && s != "" && s != "***REDACTED***" {
					return true
				}
			}
			if _, ok := issuerJSONKeys[lowerK]; ok {
				if s, ok := val.(string); ok && containsIssuerUserinfo(s) {
					return true
				}
			}
			if jsonHasCleartextSecret(val) {
				return true
			}
		}
	case []any:
		for _, item := range node {
			if jsonHasCleartextSecret(item) {
				return true
			}
		}
	}
	return false
}

func RedactBytes(ext string, data []byte) ([]byte, error) {
	switch strings.ToLower(ext) {
	case ".json":
		var root any
		if err := json.Unmarshal(data, &root); err != nil {
			return nil, err
		}
		redactJSONValue(root)
		out, err := json.MarshalIndent(root, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(out, '\n'), nil
	case ".yaml", ".yml":
		return []byte(redactYAMLContent(string(data))), nil
	default:
		return data, fmt.Errorf("unsupported extension %q", ext)
	}
}
