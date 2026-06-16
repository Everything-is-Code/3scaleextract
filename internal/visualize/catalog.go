package visualize

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// WriteProductsCatalog writes products-catalog.md under outDir.
func WriteProductsCatalog(t *Tenant, outDir string) error {
	if t == nil {
		return fmt.Errorf("tenant is required")
	}
	data := BuildTopologyData(t)
	content := renderProductsCatalog(data)
	return writeFile(filepath.Join(outDir, "products-catalog.md"), content)
}

func renderProductsCatalog(data TopologyData) string {
	products := append([]TopologyProduct(nil), data.Products...)
	sort.Slice(products, func(i, j int) bool {
		return products[i].Name < products[j].Name
	})

	var b strings.Builder
	b.WriteString("# Product catalog\n\n")
	b.WriteString("[← Index](index.md)\n\n")
	b.WriteString("Consolidated view of API products with auth, backend usage, and policy chains.\n\n")
	b.WriteString("| Product | Category | Auth | Backends | Apps | Policies | Policy names |\n")
	b.WriteString("|---------|----------|------|----------|------|----------|-------------|\n")

	for _, p := range products {
		category := data.Cat[p.Category]
		if category == "" {
			category = p.Category
		}
		b.WriteString(fmt.Sprintf(
			"| [%s](products/%s.md) | %s | %s | %d | %d | %d | %s |\n",
			mdCell(p.Name),
			p.Name,
			mdCell(category),
			mdCell(p.Auth),
			len(p.Edges),
			len(p.Apps),
			len(p.PolicyNames),
			mdCell(formatPolicyChain(p.PolicyNames)),
		))
	}
	b.WriteString("\n")
	return b.String()
}
