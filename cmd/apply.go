package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/migrator/gatewayapi"
	"github.com/saiyam1814/ing-switch/pkg/migrator/traefik"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	applyTarget   string
	applyCategory string
	applyDryRun   bool
)

// applyableCategories are file categories whose YAML can be kubectl-applied directly.
var applyableCategories = map[string]bool{
	"middleware": true,
	"ingress":    true,
	"gateway":    true,
	"httproute":  true,
	"policy":     true,
}

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply generated migration manifests to the cluster",
	Long: `Generates migration files and applies them directly to the cluster using kubectl.

Use --category to apply a specific step of the migration:
  middleware  - Traefik Middleware CRDs
  ingress    - Updated Ingress manifests
  gateway    - GatewayClass + Gateway resources
  httproute  - HTTPRoute manifests
  policy     - BackendTrafficPolicy / SecurityPolicy / Middleware CRDs

Use --dry-run to preview what would be applied (uses kubectl --dry-run=server).

Examples:
  # Dry-run all middleware resources for traefik
  ing-switch apply --target traefik --category middleware --dry-run

  # Apply gateway resources
  ing-switch apply --target gateway-api --category gateway

  # Apply HTTPRoutes
  ing-switch apply --target gateway-api --category httproute

  # Apply everything for a target (all applyable categories)
  ing-switch apply --target traefik`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runApply()
	},
}

func init() {
	applyCmd.Flags().StringVar(&applyTarget, "target", "", "Target controller: traefik|gateway-api|gateway-api-traefik (required)")
	applyCmd.MarkFlagRequired("target")
	applyCmd.Flags().StringVar(&applyCategory, "category", "", "File category to apply (middleware|ingress|gateway|httproute|policy); omit to apply all")
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Preview changes using kubectl --dry-run=server")
	rootCmd.AddCommand(applyCmd)
}

func runApply() error {
	switch applyTarget {
	case "traefik", "gateway-api", "gateway-api-traefik":
	default:
		return fmt.Errorf("unknown target %q — use 'traefik', 'gateway-api', or 'gateway-api-traefik'", applyTarget)
	}

	if applyCategory != "" && !applyableCategories[applyCategory] {
		valid := []string{}
		for k := range applyableCategories {
			valid = append(valid, k)
		}
		return fmt.Errorf("category %q is not kubectl-applyable — use one of: %s\nFor install/verify/cleanup, download the files via 'ing-switch migrate' and run manually",
			applyCategory, strings.Join(valid, ", "))
	}

	mode := "APPLY"
	if applyDryRun {
		mode = "DRY-RUN"
	}
	fmt.Printf("\n  ing-switch apply [%s]\n", mode)
	fmt.Printf("  Target:   %s\n", applyTarget)
	if applyCategory != "" {
		fmt.Printf("  Category: %s\n", applyCategory)
	} else {
		fmt.Printf("  Category: all\n")
	}
	fmt.Println()

	// Scan
	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	scanResult, err := s.Scan(namespace)
	if err != nil {
		return fmt.Errorf("scanning cluster: %w", err)
	}

	if len(scanResult.Ingresses) == 0 {
		fmt.Println("  No ingress resources found — nothing to apply.")
		return nil
	}

	// Analyze
	a := analyzer.NewAnalyzer(applyTarget)
	report := a.Analyze(scanResult)

	// Generate migration files
	var files []generator.GeneratedFile
	switch applyTarget {
	case "traefik":
		m := traefik.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api":
		m := gatewayapi.NewMigrator()
		files, err = m.Migrate(scanResult, report)
	case "gateway-api-traefik":
		m := gatewayapi.NewTraefikGatewayMigrator()
		files, err = m.Migrate(scanResult, report)
	}
	if err != nil {
		return fmt.Errorf("generating migration files: %w", err)
	}

	// Filter to requested category (or all applyable categories)
	var yamlFiles []generator.GeneratedFile
	for _, f := range files {
		if !strings.HasSuffix(f.RelPath, ".yaml") && !strings.HasSuffix(f.RelPath, ".yml") {
			continue
		}
		if applyCategory != "" {
			if f.Category == applyCategory {
				yamlFiles = append(yamlFiles, f)
			}
		} else {
			if applyableCategories[f.Category] {
				yamlFiles = append(yamlFiles, f)
			}
		}
	}

	if len(yamlFiles) == 0 {
		cat := applyCategory
		if cat == "" {
			cat = "all applyable categories"
		}
		fmt.Printf("  No YAML files found for %s.\n", cat)
		fmt.Printf("  Use 'ing-switch migrate --target %s' to generate and review files first.\n\n", applyTarget)
		return nil
	}

	// Write to temp dir
	tmpDir, err := os.MkdirTemp("", "ing-switch-apply-*")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	var applied []string
	for _, f := range yamlFiles {
		fname := filepath.Base(f.RelPath)
		dest := filepath.Join(tmpDir, fname)
		if err := os.WriteFile(dest, []byte(f.Content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not write %s: %v\n", fname, err)
			continue
		}
		applied = append(applied, f.RelPath)
	}

	// Build kubectl command
	args := []string{}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	if kubecontext != "" {
		args = append(args, "--context", kubecontext)
	}
	args = append(args, "apply", "-f", tmpDir)
	if applyDryRun {
		args = append(args, "--dry-run=server")
	}

	fmt.Printf("  Applying %d file(s):\n", len(applied))
	for _, f := range applied {
		fmt.Printf("    + %s\n", f)
	}
	fmt.Println()

	cmd := exec.Command("kubectl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	fmt.Println()
	if applyDryRun {
		fmt.Printf("  Dry-run complete. Run without --dry-run to apply for real.\n\n")
	} else {
		fmt.Printf("  Applied successfully.\n")
		fmt.Printf("  Run 'ing-switch doctor' to verify cluster state.\n\n")
	}

	return nil
}
