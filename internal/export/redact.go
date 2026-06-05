package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sensitiveJSONKeys = map[string]struct{}{
	"access_token":  {},
	"client_secret": {},
	"secret":        {},
	"api_key":       {},
	"user_key":      {},
	"app_key":       {},
	"provider_key":  {},
}

var yamlSecretPattern = regexp.MustCompile(`(?m)^(\s*(?:access_token|client_secret|secret|api_key|user_key|app_key|provider_key)\s*:\s*).+$`)

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
	out := yamlSecretPattern.ReplaceAllString(string(data), `${1}`+redactedValue)
	return os.WriteFile(path, []byte(out), 0o644)
}

func redactJSONValue(v any) {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if _, ok := sensitiveJSONKeys[strings.ToLower(k)]; ok {
				node[k] = "***REDACTED***"
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

func ContainsCleartextSecret(data []byte) bool {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return yamlSecretPattern.Match(data)
	}
	return jsonHasCleartextSecret(root)
}

func jsonHasCleartextSecret(v any) bool {
	switch node := v.(type) {
	case map[string]any:
		for k, val := range node {
			if _, ok := sensitiveJSONKeys[strings.ToLower(k)]; ok {
				if s, ok := val.(string); ok && s != "" && s != "***REDACTED***" {
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
		return []byte(yamlSecretPattern.ReplaceAllString(string(data), `${1}`+redactedValue)), nil
	default:
		return data, fmt.Errorf("unsupported extension %q", ext)
	}
}
