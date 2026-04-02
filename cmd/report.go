package cmd

import (
	"fmt"
	"os"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	reportTarget string
	reportOutput string
)

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a self-contained HTML migration report",
	Long: `Generates a shareable HTML file containing the full migration analysis,
including annotation compatibility matrix, per-ingress status, and readiness score.

The HTML file is self-contained (no external dependencies) and can be shared
with non-technical stakeholders, attached to tickets, or hosted on an intranet.

Examples:
  # Generate report for traefik migration
  ing-switch report --target traefik

  # Custom output file
  ing-switch report --target gateway-api -o migration-report.html`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReport()
	},
}

func init() {
	reportCmd.Flags().StringVar(&reportTarget, "target", "", "Target controller: traefik|gateway-api|gateway-api-traefik (required)")
	reportCmd.MarkFlagRequired("target")
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "migration-report.html", "Output HTML file path")
	rootCmd.AddCommand(reportCmd)
}

func runReport() error {
	switch reportTarget {
	case "traefik", "gateway-api", "gateway-api-traefik":
	default:
		return fmt.Errorf("unknown target %q — use 'traefik', 'gateway-api', or 'gateway-api-traefik'", reportTarget)
	}

	fmt.Printf("\n  ing-switch — Generating HTML Report\n")
	fmt.Printf("  Target: %s\n\n", reportTarget)

	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	scanResult, err := s.Scan(namespace)
	if err != nil {
		return fmt.Errorf("scanning cluster: %w", err)
	}

	a := analyzer.NewAnalyzer(reportTarget)
	report := a.Analyze(scanResult)

	htmlContent := generator.GenerateHTMLReport(scanResult, report)

	if err := os.WriteFile(reportOutput, []byte(htmlContent), 0644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Printf("  Written to %s\n", reportOutput)
	fmt.Printf("  Open in a browser to view the report.\n\n")

	return nil
}
