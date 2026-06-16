package visualize

import (
	"sort"
	"strings"
)

// TopologyData is the compact payload for canvas, HTML, and catalog views.
type TopologyData struct {
	Manifest  map[string]any    `json:"m"`
	Cat       map[string]string `json:"cat"`
	CatCounts map[string]int    `json:"catCounts"`
	Backends  []string          `json:"backends"`
	Products  []TopologyProduct `json:"products"`
	Shared    []TopologyShared  `json:"shared"`
}

type TopologyProduct struct {
	Name        string        `json:"n"`
	Category    string        `json:"c"`
	Auth        string        `json:"a"`
	Edges       [][2]any      `json:"e"`
	Apps        [][3]string   `json:"p,omitempty"`
	PolicyNames []string      `json:"pol,omitempty"`
}

type TopologyShared struct {
	Backend  string   `json:"b"`
	Count    int      `json:"n"`
	Products []string `json:"p"`
}

var topologyCategoryLabels = map[string]string{
	"I": "Integration (-IS)",
	"B": "Business API",
	"S": "SAP",
	"P": "Platform / misc",
}

// BuildTopologyData converts a loaded tenant export into topology JSON.
func BuildTopologyData(tenant *Tenant) TopologyData {
	backendNames := make([]string, 0, len(tenant.Backends))
	backendIndex := map[string]int{}
	for _, b := range tenant.Backends {
		backendIndex[b.SystemName] = len(backendNames)
		backendNames = append(backendNames, b.SystemName)
	}

	refs := map[string]map[string]struct{}{}
	appsByProduct := map[string][][3]string{}
	for _, a := range tenant.Applications {
		if strings.TrimSpace(a.ProductName) == "" {
			continue
		}
		appsByProduct[a.ProductName] = append(appsByProduct[a.ProductName], [3]string{
			a.Name,
			a.PlanName,
			a.State,
		})
	}
	for name := range appsByProduct {
		sort.Slice(appsByProduct[name], func(i, j int) bool {
			return appsByProduct[name][i][0] < appsByProduct[name][j][0]
		})
	}

	products := make([]TopologyProduct, 0, len(tenant.Products))
	for _, p := range tenant.Products {
		edges := make([][2]any, 0, len(p.BackendUsages))
		for _, u := range p.BackendUsages {
			if u.Backend == nil {
				continue
			}
			idx := backendIndex[u.Backend.SystemName]
			edges = append(edges, [2]any{idx, u.Path})
			if refs[u.Backend.SystemName] == nil {
				refs[u.Backend.SystemName] = map[string]struct{}{}
			}
			refs[u.Backend.SystemName][p.SystemName] = struct{}{}
		}

		policyNames := make([]string, 0, len(p.Policies))
		for _, policy := range p.Policies {
			policyNames = append(policyNames, policy.Name)
		}

		product := TopologyProduct{
			Name:        p.SystemName,
			Category:    topologyCategoryKey(p.SystemName),
			Auth:        authLabel(p.AuthType),
			Edges:       edges,
			PolicyNames: policyNames,
		}
		if apps := appsByProduct[p.SystemName]; len(apps) > 0 {
			product.Apps = apps
		}
		products = append(products, product)
	}

	var shared []TopologyShared
	for backend, ps := range refs {
		if len(ps) < 2 {
			continue
		}
		names := make([]string, 0, len(ps))
		for name := range ps {
			names = append(names, name)
		}
		sort.Strings(names)
		limit := len(names)
		if limit > 6 {
			limit = 6
		}
		shared = append(shared, TopologyShared{
			Backend:  backend,
			Count:    len(names),
			Products: names[:limit],
		})
	}
	sort.Slice(shared, func(i, j int) bool {
		if shared[i].Count != shared[j].Count {
			return shared[i].Count > shared[j].Count
		}
		return shared[i].Backend < shared[j].Backend
	})
	if len(shared) > 12 {
		shared = shared[:12]
	}

	catCounts := map[string]int{"I": 0, "B": 0, "S": 0, "P": 0}
	for _, p := range products {
		catCounts[p.Category]++
	}

	manifest := map[string]any{
		"schema_version":       tenant.Manifest.SchemaVersion,
		"exported_at":          tenant.Manifest.ExportedAt,
		"admin_url":            tenant.Manifest.AdminURL,
		"product_count":        tenant.Manifest.ProductCount,
		"backend_count":        tenant.Manifest.BackendCount,
		"application_count":    tenant.Manifest.ApplicationCount,
		"include_applications": tenant.Manifest.IncludeApplications,
		"incomplete":           tenant.Manifest.Incomplete,
	}

	return TopologyData{
		Manifest:  manifest,
		Cat:       topologyCategoryLabels,
		CatCounts: catCounts,
		Backends:  backendNames,
		Products:  products,
		Shared:    shared,
	}
}

func topologyCategoryKey(name string) string {
	switch {
	case strings.HasSuffix(name, "-IS") || name == "Industrial-IS-int":
		return "I"
	case strings.Contains(name, "SAP") || strings.HasPrefix(name, "SalidaSap"):
		return "S"
	case name == "dummy-for-alerts" || name == "llm" || name == "ddrr":
		return "P"
	default:
		return "B"
	}
}

func formatPolicyChain(names []string) string {
	if len(names) == 0 {
		return "—"
	}
	return strings.Join(names, " → ")
}
