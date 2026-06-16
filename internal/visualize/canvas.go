package visualize

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed canvas/topology.canvas.tsx.tmpl
var topologyCanvasTemplate string

const canvasDataPlaceholder = "__CANVAS_DATA_JSON__"

// WriteCanvasTSX renders a Cursor IDE topology canvas from tenant export data.
func WriteCanvasTSX(tenant *Tenant, outPath string) error {
	if tenant == nil {
		return fmt.Errorf("tenant is nil")
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("canvas output path is empty")
	}

	data := BuildTopologyData(tenant)
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal canvas data: %w", err)
	}

	content := strings.Replace(
		topologyCanvasTemplate,
		canvasDataPlaceholder,
		string(payload),
		1,
	)
	if !strings.Contains(content, `"products"`) {
		return fmt.Errorf("canvas template placeholder was not replaced")
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create canvas directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write canvas: %w", err)
	}
	return nil
}
