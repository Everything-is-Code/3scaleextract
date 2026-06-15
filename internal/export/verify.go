package export

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Everything-is-Code/3scaleextract/internal/output"
)

var ErrStrictSidecar = errors.New("strict export: missing product sidecar")

// VerifyExport checks manifest fields and on-disk layout after an export completes.
func VerifyExport(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		return errors.New("export directory is required")
	}

	manifestPath := filepath.Join(root, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest.json: %w", err)
	}

	var manifest output.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse manifest.json: %w", err)
	}
	if manifest.SchemaVersion != output.SchemaVersion {
		return fmt.Errorf("manifest schema_version = %q, want %q", manifest.SchemaVersion, output.SchemaVersion)
	}
	if strings.TrimSpace(manifest.AdminURL) == "" {
		return errors.New("manifest admin_url is required")
	}
	if strings.TrimSpace(manifest.ExportedAt) == "" {
		return errors.New("manifest exported_at is required")
	}

	for _, path := range []string{"products", "backends", "policies"} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			return fmt.Errorf("missing directory %s: %w", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "policies", "catalog.json")); err != nil {
		return fmt.Errorf("missing policies/catalog.json: %w", err)
	}

	productYAMLs, err := filepath.Glob(filepath.Join(root, "products", "*.yaml"))
	if err != nil {
		return err
	}
	if manifest.ProductCount > 0 && len(productYAMLs) == 0 {
		return errors.New("expected at least one products/*.yaml")
	}
	if manifest.ProductCount != len(productYAMLs) {
		return fmt.Errorf("manifest product_count = %d, found %d products/*.yaml", manifest.ProductCount, len(productYAMLs))
	}

	backendJSONs, err := filepath.Glob(filepath.Join(root, "backends", "*.json"))
	if err != nil {
		return err
	}
	if manifest.BackendCount != len(backendJSONs) {
		return fmt.Errorf("manifest backend_count = %d, found %d backends/*.json", manifest.BackendCount, len(backendJSONs))
	}

	if manifest.IncludeApplications {
		appPages, err := filepath.Glob(filepath.Join(root, "applications", "page-*.json"))
		if err != nil {
			return err
		}
		if manifest.ApplicationCount > 0 && len(appPages) == 0 {
			return errors.New("expected applications/page-*.json when application_count > 0")
		}
	}

	return nil
}

// ListExportPaths returns sorted relative file paths under an export root (directories excluded).
func ListExportPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// VerifyNoCleartextSecrets scans JSON and YAML artifacts under root for residual
// sensitive values aligned with the redaction contract. Returns a path-qualified
// error when cleartext is detected.
func VerifyNoCleartextSecrets(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json", ".yaml", ".yml":
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if ContainsCleartextSecret(data) {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				return fmt.Errorf("cleartext secret in %s", filepath.ToSlash(rel))
			}
		}
		return nil
	})
}
