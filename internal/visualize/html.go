package visualize

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed html/topology.html.tmpl
var topologyHTMLTemplate string

const htmlDataPlaceholder = "__TOPOLOGY_DATA_JSON__"

// WriteTopologyHTML renders a self-contained browser dashboard from tenant export data.
func WriteTopologyHTML(tenant *Tenant, outPath string) error {
	if tenant == nil {
		return fmt.Errorf("tenant is nil")
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("html output path is empty")
	}

	data := BuildTopologyData(tenant)
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal topology data: %w", err)
	}

	content := strings.Replace(
		topologyHTMLTemplate,
		htmlDataPlaceholder,
		string(payload),
		1,
	)
	if strings.Contains(content, htmlDataPlaceholder) {
		return fmt.Errorf("html template placeholder was not replaced")
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create html directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write html: %w", err)
	}
	return nil
}
