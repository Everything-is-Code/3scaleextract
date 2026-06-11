package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Everything-is-Code/3scaleextract/internal/admin"
	"github.com/Everything-is-Code/3scaleextract/internal/config"
	"github.com/Everything-is-Code/3scaleextract/internal/seed"
	"github.com/Everything-is-Code/3scaleextract/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	os.Exit(run())
}

func run() int {
	return executeSeed(nil)
}

func newSeedCommand() *cobra.Command {
	cfg := config.LoadAuthFromEnv()
	skipExisting := true
	dryRun := false
	listFixtures := false

	cmd := &cobra.Command{
		Use:   "threescale-seed",
		Short: "Load dummy 3scale tenant data for export testing",
		Long: `Optional lab/demo tool — loads sample backends, products, plans, policies,
and applications into a 3scale tenant via Admin API.

See docs/SEED.md for fixtures, OIDC setup, and the demo seed-and-export script.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listFixtures {
				return printFixtures()
			}
			return execute(cmd.Context(), cfg, skipExisting, dryRun)
		},
	}
	config.BindAuthFlags(cmd.Flags(), &cfg)
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", true, "skip resources that already exist (by system_name)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate fixture plan without calling Admin API")
	cmd.Flags().BoolVar(&listFixtures, "list-fixtures", false, "show fixture coverage matrix and exit")
	cmd.Version = version.Version
	return cmd
}

func executeSeed(args []string) int {
	cmd := newSeedCommand()
	if args != nil {
		cmd.SetArgs(args)
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func execute(ctx context.Context, cfg config.AuthConfig, skipExisting, dryRun bool) error {
	if dryRun {
		if err := printFixtures(); err != nil {
			return err
		}
		fmt.Println("\n(dry-run: no Admin API calls)")
	}

	if err := cfg.ValidateAuth(); err != nil {
		return err
	}
	adminURL, err := config.NormalizeAdminURL(cfg.AdminURL)
	if err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 90 * time.Second}
	if cfg.InsecureTLS {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}

	client, err := admin.NewClient(admin.Options{
		BaseURL:    adminURL,
		Token:      cfg.Token,
		HTTPClient: httpClient,
	})
	if err != nil {
		return err
	}

	result, err := seed.NewSeeder(client, seed.Options{
		SkipExisting: skipExisting,
		DryRun:       dryRun,
	}).Run(ctx)
	if err != nil {
		return err
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
	return nil
}

func printFixtures() error {
	backends, account, products := seed.DefaultFixtures()
	fmt.Println("=== Seed fixtures ===")
	fmt.Printf("Account: %s (%s)\n\n", account.Username, account.Email)
	fmt.Printf("Backends (%d):\n", len(backends))
	for _, b := range backends {
		fmt.Printf("  - %s\n", b.SystemName)
	}
	fmt.Printf("\nProducts (%d):\n", len(products))
	for _, p := range products {
		fmt.Printf("  - %s [%s]\n", p.SystemName, p.AuthMode)
		if cov, ok := seed.CoverageMatrix[p.SystemName]; ok {
			for _, item := range cov {
				fmt.Printf("      • %s\n", item)
			}
		}
	}
	return nil
}
