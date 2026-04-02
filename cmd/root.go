package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	kubeconfig string
	kubecontext string
	namespace   string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "ing-switch",
	Short: "Migrate between Kubernetes ingress controllers — scan, analyze, diff, and migrate",
	Long: `ing-switch scans your cluster for Ingress resources and Traefik IngressRoute CRDs,
analyzes compatibility, generates migration manifests for Traefik or Gateway API,
and provides a local UI to visualize the migration plan.

Examples:
  # Quick health check
  ing-switch doctor

  # Scan cluster (auto-detects Ingress + IngressRoute)
  ing-switch scan

  # Analyze compatibility with Gateway API
  ing-switch analyze --target gateway-api-traefik

  # Visual diff — before/after for each resource
  ing-switch diff --target gateway-api

  # Generate migration files
  ing-switch migrate --target gateway-api --output-dir ./migration

  # Apply migration manifests (dry-run first)
  ing-switch apply --target traefik --dry-run
  ing-switch apply --target traefik --category middleware

  # Open local UI
  ing-switch ui`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file (default: $KUBECONFIG or ~/.kube/config)")
	rootCmd.PersistentFlags().StringVar(&kubecontext, "context", "", "Kubernetes context to use")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scan (default: all namespaces)")
}
