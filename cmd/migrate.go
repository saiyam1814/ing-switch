package cmd

import (
	"fmt"
	"os"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/migrator/gatewayapi"
	"github.com/saiyam1814/ing-switch/pkg/migrator/traefik"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	migrateTarget    string
	migrateOutputDir string
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Generate migration manifests for the target ingress controller",
	Long: `Generates a complete set of migration files in the output directory.

For Traefik:
  - Helm install script + values.yaml
  - Traefik Middleware CRDs (one per annotation group)
  - Updated Ingress manifests (annotations translated)
  - Verification script
  - DNS migration guide
  - Cleanup script (preserve IngressClass, remove NGINX)

For Gateway API (Envoy Gateway):
  - Gateway API CRD install script
  - Envoy Gateway Helm install
  - GatewayClass + Gateway resources
  - HTTPRoute manifests (one per Ingress)
  - BackendTrafficPolicy / SecurityPolicy for advanced features
  - Verification and cleanup scripts

All generated files are valid YAML you can review before applying.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMigrate(cmd)
	},
}

func init() {
	migrateCmd.Flags().StringVar(&migrateTarget, "target", "", "Target controller: traefik|gateway-api (required)")
	migrateCmd.MarkFlagRequired("target")
	migrateCmd.Flags().StringVar(&migrateOutputDir, "output-dir", "./migration", "Directory to write generated files")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command) error {
	switch migrateTarget {
	case "traefik", "gateway-api":
	default:
		return fmt.Errorf("unknown target %q — use 'traefik' or 'gateway-api'", migrateTarget)
	}

	fmt.Printf("\n  ing-switch — Generating Migration Files\n")
	fmt.Printf("  Target:     %s\n", migrateTarget)
	fmt.Printf("  Output dir: %s\n\n", migrateOutputDir)

	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	scanResult, err := s.Scan(namespace)
	if err != nil {
		return fmt.Errorf("scanning cluster: %w", err)
	}

	a := analyzer.NewAnalyzer(migrateTarget)
	report := a.Analyze(scanResult)

	var files []generator.GeneratedFile

	switch migrateTarget {
	case "traefik":
		m := traefik.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api":
		m := gatewayapi.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	}
	if err != nil {
		return fmt.Errorf("generating migration files: %w", err)
	}

	gen := generator.NewOutputGenerator(migrateOutputDir)
	if err := gen.Write(files, report); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("  Generated %d files in %s/\n\n", len(files), migrateOutputDir)

	fmt.Printf("  Next steps:\n")
	switch migrateTarget {
	case "traefik":
		fmt.Printf("  1. Review %s/00-migration-report.md\n", migrateOutputDir)
		fmt.Printf("  2. Run %s/01-install-traefik/helm-install.sh\n", migrateOutputDir)
		fmt.Printf("  3. Apply %s/02-middlewares/\n", migrateOutputDir)
		fmt.Printf("  4. Apply %s/03-ingresses/\n", migrateOutputDir)
		fmt.Printf("  5. Run %s/04-verify.sh to test both controllers\n", migrateOutputDir)
		fmt.Printf("  6. Follow %s/05-dns-migration.md\n", migrateOutputDir)
		fmt.Printf("  7. Run %s/06-cleanup/ when ready\n", migrateOutputDir)
	case "gateway-api":
		fmt.Printf("  1. Review %s/00-migration-report.md\n", migrateOutputDir)
		fmt.Printf("  2. Run %s/01-install-gateway-api-crds/install.sh\n", migrateOutputDir)
		fmt.Printf("  3. Run %s/02-install-envoy-gateway/helm-install.sh\n", migrateOutputDir)
		fmt.Printf("  4. Apply %s/03-gateway/\n", migrateOutputDir)
		fmt.Printf("  5. Apply %s/04-httproutes/\n", migrateOutputDir)
		fmt.Printf("  6. Apply %s/05-policies/ (if applicable)\n", migrateOutputDir)
		fmt.Printf("  7. Run %s/06-verify.sh\n", migrateOutputDir)
	}

	fmt.Printf("\n  Run 'ing-switch ui' to open the visual migration dashboard\n\n")

	_ = os.Stdout
	return nil
}
