//go:build integration

package export_test

import (
	"context"
	"os"
	"testing"

	"github.com/fmeneses/3scaleextract/internal/admin"
	"github.com/fmeneses/3scaleextract/internal/export"
)

func TestIntegrationExport(t *testing.T) {
	adminURL := os.Getenv("THREESCALE_ADMIN_URL")
	token := os.Getenv("THREESCALE_ACCESS_TOKEN")
	outDir := os.Getenv("THREESCALE_OUTPUT_DIR")
	if adminURL == "" || token == "" || outDir == "" {
		t.Skip("set THREESCALE_ADMIN_URL, THREESCALE_ACCESS_TOKEN, THREESCALE_OUTPUT_DIR")
	}

	client, err := admin.NewClient(admin.Options{BaseURL: adminURL, Token: token})
	if err != nil {
		t.Fatal(err)
	}
	tb, err := export.NewToolbox(export.ToolboxOptions{NativeBinary: "3scale"})
	if err != nil {
		t.Fatal(err)
	}
	svc := export.NewService(client, tb)
	_, err = svc.Export(context.Background(), export.Options{
		AdminURL: adminURL,
		Token:    token,
		OutDir:   outDir,
		PerPage:  500,
	})
	if err != nil {
		t.Fatal(err)
	}
}
