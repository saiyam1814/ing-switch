package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	analyzeTarget string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze annotation compatibility for a target ingress controller",
	Long: `Analyzes every Ingress resource in the cluster and maps each
nginx.ingress.kubernetes.io/* annotation to its equivalent in the target controller.

Status indicators:
  supported   - Full equivalent exists, will work identically
  partial     - Equivalent exists but with behavioral differences
  unsupported - No equivalent; manual intervention required

Supported targets:
  traefik      Traefik v3.x (lowest migration friction)
  gateway-api  Kubernetes Gateway API via Envoy Gateway`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnalyze(cmd)
	},
}

func init() {
	analyzeCmd.Flags().StringVar(&analyzeTarget, "target", "", "Target controller: traefik|gateway-api (required)")
	analyzeCmd.MarkFlagRequired("target")
	analyzeCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json")
	rootCmd.AddCommand(analyzeCmd)
}

func runAnalyze(_ *cobra.Command) error {
	switch analyzeTarget {
	case "traefik", "gateway-api":
	default:
		return fmt.Errorf("unknown target %q — use 'traefik' or 'gateway-api'", analyzeTarget)
	}

	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	scanResult, err := s.Scan(namespace)
	if err != nil {
		return fmt.Errorf("scanning cluster: %w", err)
	}

	a := analyzer.NewAnalyzer(analyzeTarget)
	report := a.Analyze(scanResult)

	switch outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	default:
		printAnalysisReport(report)
	}
	return nil
}

func printAnalysisReport(report *analyzer.AnalysisReport) {
	fmt.Printf("\n  ing-switch — Compatibility Analysis\n")
	fmt.Printf("  Target: %s\n", report.Target)
	fmt.Println()

	fmt.Printf("  Summary\n")
	fmt.Printf("  -------\n")
	fmt.Printf("  Total ingresses:      %d\n", report.Summary.Total)
	fmt.Printf("  Fully compatible:     %d\n", report.Summary.FullyCompatible)
	fmt.Printf("  Needs workarounds:    %d\n", report.Summary.NeedsWorkaround)
	fmt.Printf("  Has unsupported:      %d\n", report.Summary.HasUnsupported)
	fmt.Println()

	for _, ir := range report.IngressReports {
		fmt.Printf("  %s/%s\n", ir.Namespace, ir.Name)
		fmt.Printf("  %s\n", repeatChar("-", len(ir.Namespace)+len(ir.Name)+1))

		if len(ir.Mappings) == 0 {
			fmt.Printf("  No nginx annotations — ready to migrate as-is\n\n")
			continue
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintf(w, "  ANNOTATION\tSTATUS\tTARGET RESOURCE\tNOTES\n")
		fmt.Fprintf(w, "  ----------\t------\t---------------\t-----\n")
		for _, m := range ir.Mappings {
			statusIcon := statusToIcon(string(m.Status))
			fmt.Fprintf(w, "  %s\t%s\t%s\t%s\n", m.OriginalKey, statusIcon, m.TargetResource, m.Note)
		}
		w.Flush()
		fmt.Println()
	}

	fmt.Printf("  Run 'ing-switch migrate --target %s' to generate migration files\n\n", report.Target)
}

func statusToIcon(s string) string {
	switch s {
	case "supported":
		return "[supported]"
	case "partial":
		return "[partial]  "
	case "unsupported":
		return "[UNSUPPORTED]"
	default:
		return s
	}
}

func repeatChar(c string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += c
	}
	return result
}
