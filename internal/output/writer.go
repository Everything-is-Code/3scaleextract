package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const SchemaVersion = "1.0"

type Manifest struct {
	SchemaVersion       string `json:"schema_version"`
	ExportedAt          string `json:"exported_at"`
	AdminURL            string `json:"admin_url"`
	ProductCount        int    `json:"product_count"`
	BackendCount        int    `json:"backend_count"`
	ApplicationCount    int    `json:"application_count,omitempty"`
	IncludeApplications bool   `json:"include_applications"`
	Incomplete          bool   `json:"incomplete"`
}

type Writer struct {
	root string
}

func NewWriter(root string) (*Writer, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("output directory is required")
	}
	return &Writer{root: root}, nil
}

func (w *Writer) Root() string {
	return w.root
}

func (w *Writer) EnsureLayout() error {
	for _, dir := range []string{
		"products",
		"backends",
		"applications",
		"accounts",
		"policies",
	} {
		if err := os.MkdirAll(filepath.Join(w.root, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) WriteManifest(m *Manifest) error {
	if m.SchemaVersion == "" {
		m.SchemaVersion = SchemaVersion
	}
	return w.WriteJSON("manifest.json", m)
}

func (w *Writer) WriteJSON(relPath string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return w.writeAtomic(relPath, data)
}

func (w *Writer) WriteBytes(relPath string, data []byte) error {
	return w.writeAtomic(relPath, data)
}

func (w *Writer) WriteRawJSON(relPath string, raw json.RawMessage) error {
	if len(raw) == 0 {
		return w.WriteJSON(relPath, map[string]any{})
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return w.WriteBytes(relPath, raw)
	}
	return w.WriteJSON(relPath, v)
}

func (w *Writer) writeAtomic(relPath string, data []byte) error {
	path := filepath.Join(w.root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".write-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
