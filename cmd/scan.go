package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan your cluster for Ingress and IngressRoute resources",
	Long: `Scans the Kubernetes cluster for all Ingress resources and Traefik IngressRoute
CRDs across namespaces, detects which ingress controller is running, and
summarizes the annotation complexity of each resource.

Sources auto-detected:
  Kubernetes Ingress   - Standard Ingress with nginx.ingress.kubernetes.io annotations
  Traefik IngressRoute - IngressRoute CRDs with referenced Middleware CRDs

Each resource is classified as:
  simple     - Only basic routing, no complex annotations/middlewares
  complex    - Uses features that require migration work
  unsupported - Uses features with no equivalent in target controllers`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if runScan(cmd) {
			return fmt.Errorf("scan failed")
		}
		return nil
	},
}

func init() {
	scanCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json")
	rootCmd.AddCommand(scanCmd)
}

func runScan(_ *cobra.Command) bool {
	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to cluster: %v\n", err)
		fmt.Fprintln(os.Stderr, "\nTip: Use --kubeconfig or set KUBECONFIG environment variable")
		return true
	}

	result, err := s.Scan(namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning cluster: %v\n", err)
		return true
	}

	switch outputFormat {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	default:
		printScanResult(result)
	}
	return false
}

func printScanResult(result *scanner.ScanResult) {
	fmt.Printf("\n  ing-switch — Cluster Scan Results\n")
	fmt.Printf("  Cluster: %s\n", result.ClusterName)
	fmt.Println()

	// Controller info
	if result.Controller.Detected {
		fmt.Printf("  Ingress Controller Detected\n")
		fmt.Printf("  Type:      %s\n", result.Controller.Type)
		fmt.Printf("  Version:   %s\n", result.Controller.Version)
		fmt.Printf("  Namespace: %s\n", result.Controller.Namespace)
	} else {
		fmt.Printf("  No ingress controller detected (or insufficient permissions)\n")
	}
	fmt.Println()

	if len(result.Ingresses) == 0 {
		fmt.Println("  No Ingress resources found.")
		return
	}

	// Count by source type
	nginxCount := 0
	irCount := 0
	for _, ing := range result.Ingresses {
		if ing.SourceType == scanner.SourceTraefikIngressRoute {
			irCount++
		} else {
			nginxCount++
		}
	}

	fmt.Printf("  Found %d resource(s)", len(result.Ingresses))
	if irCount > 0 && nginxCount > 0 {
		fmt.Printf(" (%d Ingress, %d IngressRoute)", nginxCount, irCount)
	} else if irCount > 0 {
		fmt.Printf(" (%d IngressRoute)", irCount)
	}
	fmt.Printf("\n\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "  NAMESPACE\tNAME\tTYPE\tHOSTS\tANNOTATIONS\tTLS\tCOMPLEXITY\n")
	fmt.Fprintf(w, "  ---------\t----\t----\t-----\t-----------\t---\t----------\n")

	for _, ing := range result.Ingresses {
		hosts := ""
		if len(ing.Hosts) > 0 {
			hosts = ing.Hosts[0]
			if len(ing.Hosts) > 1 {
				hosts += fmt.Sprintf(" +%d", len(ing.Hosts)-1)
			}
		}
		tls := "no"
		if ing.TLSEnabled {
			tls = "yes"
		}
		complexity := complexityIcon(ing.Complexity)
		sourceLabel := "Ingress"
		if ing.SourceType == scanner.SourceTraefikIngressRoute {
			sourceLabel = "IngressRoute"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			ing.Namespace, ing.Name, sourceLabel, hosts, len(ing.NginxAnnotations), tls, complexity)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("  Complexity: [simple] [complex] [unsupported]\n")
	fmt.Printf("  Run 'ing-switch analyze --target traefik' for detailed annotation mapping\n\n")
}

func complexityIcon(c string) string {
	switch c {
	case "simple":
		return "simple"
	case "complex":
		return "complex"
	case "unsupported":
		return "unsupported"
	default:
		return c
	}
}
