package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// sensitiveRedactKeys lists field names that become ***REDACTED*** under --redact-secrets.
// Matching uses normalizeRedactKey (lower-case, underscores removed) so api_key and apikey,
// issuerEndpoint and issuer_endpoint, stagingPublicBaseURL and staging_public_base_url align.
//
// Covers the consulting sanitize contract (secrets + infra URLs) including the BPBA key list.
var sensitiveRedactKeys = map[string]struct{}{
	// Original secret keys
	"accesstoken":             {},
	"clientsecret":            {},
	"secret":                  {},
	"apikey":                  {},
	"userkey":                 {},
	"appkey":                  {},
	"providerkey":             {},
	"providerverificationkey": {},
	"clientid":                {},
	"appid":                   {},
	// Expanded secrets (client findings)
	"password":        {},
	"secrettoken":     {},
	"certificate":     {},
	"certificatekey":  {},
	"cacertificates":  {},
	"username":        {},
	// Infra / URL fields (client findings) — fully redacted for sanitize delivery
	"href":                    {},
	"privateendpoint":         {},
	"tokenurl":                {},
	"apibackend":              {},
	"endpoint":                {},
	"oidcissuerendpoint":      {},
	"issuerendpoint":          {},
	"sandboxendpoint":         {},
	"stagingendpoint":         {},
	"stagingpublicbaseurl":    {},
	"productionpublicbaseurl": {},
	"privatebaseurl":          {},
	"adminurl":                {},
}

// yamlSensitiveKeyPattern matches a YAML mapping key that should be fully redacted.
// Captures indent+key+colon so the value (same line and/or indented block) can be replaced.
var yamlSensitiveKeyPattern = regexp.MustCompile(`(?i)^([ \t]*)(` +
	`access_token|client_secret|secret_token|secret|api_key|apikey|user_key|app_key|` +
	`provider_key|provider_verification_key|client_id|app_id|password|username|` +
	`certificate_key|certificate|ca_certificates|` +
	`href|private_endpoint|token_url|api_backend|endpoint|` +
	`oidc_issuer_endpoint|issuer_endpoint|issuerEndpoint|sandbox_endpoint|staging_endpoint|` +
	`stagingPublicBaseURL|productionPublicBaseURL|privateBaseURL|admin_url` +
	`)[ \t]*:[ \t]*(.*)$`)

const redactedMarker = "***REDACTED***"
const redactedValue = `"` + redactedMarker + `"`

func normalizeRedactKey(k string) string {
	var b strings.Builder
	b.Grow(len(k))
	for _, r := range k {
		if r == '_' || r == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func isSensitiveRedactKey(k string) bool {
	_, ok := sensitiveRedactKeys[normalizeRedactKey(k)]
	return ok
}

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
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		line := lines[i]
		sub := yamlSensitiveKeyPattern.FindStringSubmatch(line)
		if sub == nil {
			out = append(out, line)
			i++
			continue
		}
		indent, key, rest := sub[1], sub[2], strings.TrimSpace(sub[3])
		out = append(out, indent+key+": "+redactedValue)
		i++
		// Drop folded/literal block body (PEM certificates, multi-line secrets).
		if rest == "|" || rest == "|-" || rest == "|+" || rest == ">" || rest == ">-" || rest == ">+" || rest == "" {
			keyIndent := len(indent)
			for i < len(lines) {
				next := lines[i]
				if strings.TrimSpace(next) == "" {
					// Blank line inside a block: keep skipping while still in block context.
					if i+1 < len(lines) && leadingWSLen(lines[i+1]) > keyIndent {
						i++
						continue
					}
					break
				}
				if leadingWSLen(next) > keyIndent {
					i++
					continue
				}
				break
			}
		}
	}
	return strings.Join(out, "\n")
}

func leadingWSLen(s string) int {
	n := 0
	for _, r := range s {
		if r == ' ' || r == '\t' {
			n++
			continue
		}
		break
	}
	return n
}

func redactJSONValue(v any) {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if !isSensitiveRedactKey(k) {
				redactJSONValue(val)
				continue
			}
			switch typed := val.(type) {
			case string:
				node[k] = redactedMarker
			case []any:
				redactSensitiveArray(typed)
				node[k] = typed
			case map[string]any:
				// Schema/object under a sensitive key name (e.g. policy catalog): recurse, do not wipe structure.
				redactJSONValue(typed)
			default:
				if typed != nil {
					redactJSONValue(typed)
				}
			}
		}
	case []any:
		for _, item := range node {
			redactJSONValue(item)
		}
	}
}

func redactSensitiveArray(items []any) {
	for i, item := range items {
		switch typed := item.(type) {
		case string:
			items[i] = redactedMarker
		default:
			redactJSONValue(typed)
		}
	}
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
		sub := yamlSensitiveKeyPattern.FindStringSubmatch(line)
		if sub == nil {
			continue
		}
		rest := strings.TrimSpace(sub[3])
		rest = strings.Trim(rest, `"'`)
		if rest == "" || rest == "|" || rest == "|-" || rest == "|+" || rest == ">" || rest == ">-" || rest == ">+" {
			// Block scalar or empty value: residual PEM lines are detected below.
			continue
		}
		if rest != redactedMarker {
			return true
		}
	}
	// Residual PEM / private-key material left after incomplete redaction.
	lower := strings.ToLower(string(data))
	if strings.Contains(lower, "-----begin ") && (strings.Contains(lower, " private key-----") || strings.Contains(lower, " certificate-----")) {
		return true
	}
	return false
}

func jsonHasCleartextSecret(v any) bool {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if isSensitiveRedactKey(k) {
				switch typed := val.(type) {
				case string:
					if typed != "" && typed != redactedMarker {
						return true
					}
				case []any:
					for _, item := range typed {
						if s, ok := item.(string); ok && s != "" && s != redactedMarker {
							return true
						}
						if jsonHasCleartextSecret(item) {
							return true
						}
					}
				default:
					if jsonHasCleartextSecret(val) {
						return true
					}
				}
				continue
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
