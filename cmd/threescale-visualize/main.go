package main

import (
	"fmt"
	"os"

	"github.com/Everything-is-Code/3scaleextract/internal/version"
	"github.com/Everything-is-Code/3scaleextract/internal/visualize"
	"github.com/spf13/cobra"
)

func main() {
	os.Exit(run())
}

func run() int {
	return executeVisualize(nil)
}

func newVisualizeCommand() *cobra.Command {
	outputDir := "./report"

	cmd := &cobra.Command{
		Use:   "threescale-visualize [export-dir]",
		Short: "Generate Markdown report from a threescale-export directory",
		Long: `Reads an export directory produced by threescale-export (schema v1.0)
and writes a multi-file Markdown bundle for migration review.

See docs/VISUALIZE.md for usage and report layout.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			tenant, err := visualize.LoadExport(args[0])
			if err != nil {
				return err
			}
			return visualize.WriteReport(tenant, outputDir)
		},
	}
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./report", "report output directory")
	cmd.Version = version.Version
	return cmd
}

func executeVisualize(args []string) int {
	cmd := newVisualizeCommand()
	if args != nil {
		cmd.SetArgs(args)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}
