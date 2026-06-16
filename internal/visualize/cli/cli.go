package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Everything-is-Code/3scaleextract/internal/version"
	"github.com/Everything-is-Code/3scaleextract/internal/visualize"
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	outputDir := "./report"
	canvasPath := ""
	html := false

	cmd := &cobra.Command{
		Use:   "threescale-visualize [export-dir]",
		Short: "Generate Markdown report from a threescale-export directory",
		Long: `Reads an export directory produced by threescale-export (schema v1.0)
and writes a multi-file Markdown bundle for migration review.

Optionally writes a self-contained HTML topology dashboard or a Cursor IDE
topology canvas (.canvas.tsx) for interactive exploration. See docs/VISUALIZE.md
for usage and report layout.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return RunVisualize(args[0], outputDir, canvasPath, html)
		},
	}
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./report", "report output directory")
	cmd.Flags().StringVar(&canvasPath, "canvas", "", "write Cursor IDE topology canvas (.canvas.tsx)")
	cmd.Flags().BoolVar(&html, "html", false, "write self-contained topology dashboard (topology.html)")
	cmd.Version = version.Version
	return cmd
}

func RunVisualize(exportDir, outputDir, canvasPath string, html bool) error {
	tenant, err := visualize.LoadExport(exportDir)
	if err != nil {
		return err
	}
	if canvasPath != "" {
		if err := visualize.WriteCanvasTSX(tenant, canvasPath); err != nil {
			return fmt.Errorf("write canvas: %w", err)
		}
	}
	if html {
		htmlPath := filepath.Join(outputDir, "topology.html")
		if err := visualize.WriteTopologyHTML(tenant, htmlPath); err != nil {
			return fmt.Errorf("write html: %w", err)
		}
	}
	return visualize.WriteReport(tenant, outputDir, html)
}

func Execute() int {
	return execute(nil)
}

func execute(args []string) int {
	cmd := NewRoot()
	if args != nil {
		cmd.SetArgs(args)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
