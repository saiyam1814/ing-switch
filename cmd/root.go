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
	Short: "Kubernetes Ingress migration tool — scan, analyze, and migrate from Ingress NGINX",
	Long: `ing-switch is a single-stop solution for migrating Kubernetes Ingress resources
away from the retiring Ingress NGINX controller.

It scans your cluster, analyzes annotation compatibility, generates migration
manifests for Traefik or Gateway API, and provides a local UI to visualize the
migration plan.

Examples:
  # Scan cluster for all ingresses
  ing-switch scan

  # Analyze compatibility with Traefik
  ing-switch analyze --target traefik

  # Generate migration files for Gateway API
  ing-switch migrate --target gateway-api --output-dir ./migration

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
