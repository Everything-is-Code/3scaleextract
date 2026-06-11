package visualize

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WriteReport generates the Markdown report bundle for a tenant export.
func WriteReport(t *Tenant, outDir string) error {
	if t == nil {
		return fmt.Errorf("tenant is required")
	}
	outDir = strings.TrimSpace(outDir)
	if outDir == "" {
		return fmt.Errorf("output directory is required")
	}

	if err := os.MkdirAll(filepath.Join(outDir, "products"), 0o755); err != nil {
		return err
	}

	if err := writeFile(filepath.Join(outDir, "index.md"), renderIndex(t, outDir)); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(outDir, "backends.md"), renderBackends(t)); err != nil {
		return err
	}
	for _, product := range t.Products {
		path := filepath.Join(outDir, "products", product.SystemName+".md")
		if err := writeFile(path, renderProduct(t, &product)); err != nil {
			return err
		}
	}
	if t.Manifest.IncludeApplications {
		if err := writeFile(filepath.Join(outDir, "applications.md"), renderApplications(t)); err != nil {
			return err
		}
	}
	return nil
}

func writeFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func renderIndex(t *Tenant, outDir string) string {
	var b strings.Builder
	b.WriteString("# 3scale Tenant Report\n\n")

	if t.Manifest.Incomplete {
		b.WriteString("> **Warning:** Export marked incomplete — some data may be missing.\n\n")
	}
	if len(t.Manifest.Warnings) > 0 {
		b.WriteString("## Export warnings\n\n")
		for _, warning := range t.Manifest.Warnings {
			b.WriteString(fmt.Sprintf("- %s\n", mdCell(warning)))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Overview\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| Admin URL | %s |\n", mdCell(t.Manifest.AdminURL)))
	b.WriteString(fmt.Sprintf("| Exported at | %s |\n", mdCell(t.Manifest.ExportedAt)))
	b.WriteString(fmt.Sprintf("| Products | %d |\n", t.Manifest.ProductCount))
	b.WriteString(fmt.Sprintf("| Backends | %d |\n", t.Manifest.BackendCount))
	if t.Manifest.IncludeApplications {
		b.WriteString(fmt.Sprintf("| Applications | %d |\n", t.Manifest.ApplicationCount))
	}
	b.WriteString("\n")

	b.WriteString("## Navigation\n\n")
	b.WriteString("- [Backends](backends.md)\n")
	if t.Manifest.IncludeApplications {
		b.WriteString("- [Applications](applications.md)\n")
	}
	for _, product := range t.Products {
		b.WriteString(fmt.Sprintf("- [%s](products/%s.md)\n", mdCell(product.DisplayName), product.SystemName))
	}
	b.WriteString("\n")

	b.WriteString("## Auth Matrix\n\n")
	b.WriteString("| Product | Auth | OIDC Issuer | Staging | Production |\n")
	b.WriteString("|---------|------|-------------|---------|------------|\n")
	for _, product := range t.Products {
		issuer := "—"
		if product.OIDC != nil && product.OIDC.IssuerEndpoint != "" {
			issuer = product.OIDC.IssuerEndpoint
		} else if product.AuthType == "oidc" && len(product.MissingFiles) > 0 {
			issuer = "_unavailable_"
		}
		b.WriteString(fmt.Sprintf("| [%s](products/%s.md) | %s | %s | %s | %s |\n",
			mdCell(product.DisplayName),
			product.SystemName,
			mdCell(authLabel(product.AuthType)),
			mdCell(issuer),
			mdCell(product.StagingEndpoint),
			mdCell(product.ProductionEndpoint),
		))
	}
	b.WriteString("\n")

	b.WriteString("## Product ↔ Backend Topology\n\n")
	b.WriteString("```mermaid\nflowchart LR\n")
	nodes := make(map[string]struct{})
	for _, product := range t.Products {
		pid := mermaidID("product", product.SystemName)
		nodes[pid] = struct{}{}
		b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", pid, mermaidLabel(product.SystemName)))
	}
	for _, backend := range t.Backends {
		bid := mermaidID("backend", backend.SystemName)
		nodes[bid] = struct{}{}
		b.WriteString(fmt.Sprintf("  %s[(\"%s\")]\n", bid, mermaidLabel(backend.SystemName)))
	}
	for _, product := range t.Products {
		pid := mermaidID("product", product.SystemName)
		for _, usage := range product.BackendUsages {
			if usage.Backend == nil {
				continue
			}
			bid := mermaidID("backend", usage.Backend.SystemName)
			label := mermaidLabel(usage.Path)
			if label == "" {
				label = "/"
			}
			b.WriteString(fmt.Sprintf("  %s -->|\"%s\"| %s\n", pid, label, bid))
		}
	}
	b.WriteString("```\n\n")
	_ = outDir
	return b.String()
}

func renderBackends(t *Tenant) string {
	var b strings.Builder
	b.WriteString("# Backends\n\n")
	b.WriteString("[← Index](index.md)\n\n")
	b.WriteString("| System Name | Name | Private Endpoint | Referenced By |\n")
	b.WriteString("|-------------|------|------------------|---------------|\n")
	for _, backend := range t.Backends {
		refs := "—"
		if len(backend.ReferencedBy) > 0 {
			links := make([]string, len(backend.ReferencedBy))
			for i, name := range backend.ReferencedBy {
				links[i] = fmt.Sprintf("[%s](products/%s.md)", name, name)
			}
			refs = strings.Join(links, ", ")
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			mdCell(backend.SystemName),
			mdCell(backend.Name),
			mdCell(backend.PrivateEndpoint),
			refs,
		))
	}
	b.WriteString("\n")
	return b.String()
}

func renderProduct(t *Tenant, product *Product) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", mdCell(product.DisplayName)))
	b.WriteString("[← Index](../index.md)\n\n")

	b.WriteString("## Summary\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| System name | `%s` |\n", product.SystemName))
	b.WriteString(fmt.Sprintf("| Service ID | %d |\n", product.ServiceID))
	b.WriteString(fmt.Sprintf("| Auth | %s |\n", mdCell(authLabel(product.AuthType))))
	b.WriteString("\n")

	b.WriteString("## Authentication\n\n")
	if product.AuthType == "oidc" {
		if product.OIDC != nil {
			b.WriteString("| Field | Value |\n")
			b.WriteString("|-------|-------|\n")
			b.WriteString(fmt.Sprintf("| Issuer type | %s |\n", mdCell(product.OIDC.IssuerType)))
			b.WriteString(fmt.Sprintf("| Issuer endpoint | %s |\n", mdCell(product.OIDC.IssuerEndpoint)))
			b.WriteString("\n")
		} else {
			b.WriteString("_OIDC configuration unavailable")
			if len(product.MissingFiles) > 0 {
				b.WriteString(fmt.Sprintf(" (`%s` missing)", strings.Join(product.MissingFiles, "`, `")))
			}
			b.WriteString("._\n\n")
		}
	} else {
		b.WriteString(fmt.Sprintf("Auth mode: **%s**\n\n", authLabel(product.AuthType)))
	}

	b.WriteString("## Policy Chain\n\n")
	if len(product.Policies) == 0 {
		b.WriteString("_No policies exported._\n\n")
	} else {
		for i, policy := range product.Policies {
			b.WriteString(fmt.Sprintf("%d. `%s`\n", i+1, policy.Name))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Application Plans\n\n")
	if len(product.Plans) == 0 {
		b.WriteString("_No plans exported._\n\n")
	} else {
		b.WriteString("| Name | System Name | State |\n")
		b.WriteString("|------|-------------|-------|\n")
		for _, plan := range product.Plans {
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
				mdCell(plan.Name),
				mdCell(plan.SystemName),
				mdCell(plan.State),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString("## Backend Usages\n\n")
	if len(product.BackendUsages) == 0 {
		b.WriteString("_No backend usages exported._\n\n")
	} else {
		b.WriteString("| Backend | Path | Private Endpoint |\n")
		b.WriteString("|---------|------|------------------|\n")
		for _, usage := range product.BackendUsages {
			backendName := fmt.Sprintf("id:%d", usage.BackendID)
			endpoint := "—"
			if usage.Backend != nil {
				backendName = usage.Backend.SystemName
				endpoint = usage.Backend.PrivateEndpoint
			}
			b.WriteString(fmt.Sprintf("| [%s](../backends.md#%s) | %s | %s |\n",
				mdCell(backendName),
				anchor(backendName),
				mdCell(usage.Path),
				mdCell(endpoint),
			))
		}
		b.WriteString("\n")
	}

	if t.Manifest.IncludeApplications {
		apps := productApplications(t, product.SystemName)
		if len(apps) > 0 {
			b.WriteString("## Applications\n\n")
			b.WriteString("| App | Plan | State | Account |\n")
			b.WriteString("|-----|------|-------|---------|\n")
			for _, app := range apps {
				b.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n",
					mdCell(app.Name),
					mdCell(app.PlanName),
					mdCell(app.State),
					app.AccountID,
				))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderApplications(t *Tenant) string {
	var b strings.Builder
	b.WriteString("# Applications\n\n")
	b.WriteString("[← Index](index.md)\n\n")
	b.WriteString("| Application | Product | Plan | State | Account |\n")
	b.WriteString("|-------------|---------|------|-------|---------|\n")

	apps := append([]Application(nil), t.Applications...)
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].ProductName == apps[j].ProductName {
			return apps[i].Name < apps[j].Name
		}
		return apps[i].ProductName < apps[j].ProductName
	})

	for _, app := range apps {
		productLink := mdCell(app.ProductName)
		if app.ProductName != "" {
			productLink = fmt.Sprintf("[%s](products/%s.md)", app.ProductName, app.ProductName)
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
			mdCell(app.Name),
			productLink,
			mdCell(app.PlanName),
			mdCell(app.State),
			app.AccountID,
		))
	}
	b.WriteString("\n")
	return b.String()
}

func productApplications(t *Tenant, systemName string) []Application {
	var out []Application
	for _, app := range t.Applications {
		if app.ProductName == systemName {
			out = append(out, app)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func authLabel(authType string) string {
	switch authType {
	case "api_key", "api_key_and_app_id":
		return "API Key"
	case "app_key", "app_id_and_app_key":
		return "App ID + App Key"
	case "oidc":
		return "OIDC"
	case "":
		return "unknown"
	default:
		return authType
	}
}

func mdCell(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func mermaidID(prefix, name string) string {
	id := prefix + "_" + strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, name)
	return id
}

func mermaidLabel(value string) string {
	return strings.ReplaceAll(value, `"`, `'`)
}

func anchor(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}
