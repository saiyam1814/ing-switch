package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/migrator/gatewayapi"
	"github.com/saiyam1814/ing-switch/pkg/migrator/traefik"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	diffTarget string
	diffName   string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show before/after diff of original resources vs generated migration output",
	Long: `Generates migration output and displays it alongside the original resource
for easy comparison. Shows what changes for each Ingress or IngressRoute.

Examples:
  # Diff all ingresses against Gateway API
  ing-switch diff --target gateway-api

  # Diff a specific ingress
  ing-switch diff --target gateway-api-traefik --name my-app

  # Diff against Traefik
  ing-switch diff --target traefik --name api-cors`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDiff()
	},
}

func init() {
	diffCmd.Flags().StringVar(&diffTarget, "target", "", "Target controller: traefik|gateway-api|gateway-api-traefik (required)")
	diffCmd.MarkFlagRequired("target")
	diffCmd.Flags().StringVar(&diffName, "name", "", "Filter to a specific ingress/ingressroute name")
	rootCmd.AddCommand(diffCmd)
}

func runDiff() error {
	switch diffTarget {
	case "traefik", "gateway-api", "gateway-api-traefik":
	default:
		return fmt.Errorf("unknown target %q — use 'traefik', 'gateway-api', or 'gateway-api-traefik'", diffTarget)
	}

	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		return fmt.Errorf("connecting to cluster: %w", err)
	}

	scanResult, err := s.Scan(namespace)
	if err != nil {
		return fmt.Errorf("scanning cluster: %w", err)
	}

	if len(scanResult.Ingresses) == 0 {
		fmt.Println("  No resources found to diff.")
		return nil
	}

	a := analyzer.NewAnalyzer(diffTarget)
	report := a.Analyze(scanResult)

	var files []generator.GeneratedFile
	switch diffTarget {
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
		return fmt.Errorf("generating migration: %w", err)
	}

	// Build a map of generated files by ingress name
	filesByIngress := make(map[string][]generator.GeneratedFile)
	for _, f := range files {
		for _, ing := range scanResult.Ingresses {
			key := ing.Namespace + "-" + ing.Name
			if strings.Contains(f.RelPath, key) {
				filesByIngress[key] = append(filesByIngress[key], f)
			}
		}
	}

	// Print diffs
	for _, ing := range scanResult.Ingresses {
		if diffName != "" && ing.Name != diffName {
			continue
		}

		key := ing.Namespace + "-" + ing.Name
		sourceType := "Ingress"
		if ing.SourceType == scanner.SourceTraefikIngressRoute {
			sourceType = "IngressRoute"
		}

		fmt.Printf("\n%s\n", strings.Repeat("=", 80))
		fmt.Printf("  %s/%s (%s) → %s\n", ing.Namespace, ing.Name, sourceType, diffTarget)
		fmt.Printf("%s\n", strings.Repeat("=", 80))

		// BEFORE: Show original resource summary
		fmt.Printf("\n%s BEFORE %s\n", colorDim("───"), colorDim(strings.Repeat("─", 60)))
		fmt.Printf("%s  Source: %s %s/%s\n", colorRed("-"), sourceType, ing.Namespace, ing.Name)
		if len(ing.Hosts) > 0 {
			fmt.Printf("%s  Hosts:  %s\n", colorRed("-"), strings.Join(ing.Hosts, ", "))
		}
		fmt.Printf("%s  Paths:  %d path(s)\n", colorRed("-"), len(ing.Paths))
		for _, p := range ing.Paths {
			fmt.Printf("%s    %s %s → %s:%d\n", colorRed("-"), p.Host, p.Path, p.ServiceName, p.ServicePort)
		}
		if ing.TLSEnabled {
			fmt.Printf("%s  TLS:    enabled", colorRed("-"))
			if len(ing.TLSSecrets) > 0 {
				fmt.Printf(" (%s)", strings.Join(ing.TLSSecrets, ", "))
			}
			fmt.Printf("\n")
		}
		if len(ing.NginxAnnotations) > 0 {
			fmt.Printf("%s  Features (%d):\n", colorRed("-"), len(ing.NginxAnnotations))
			for k, v := range ing.NginxAnnotations {
				display := v
				if len(display) > 50 {
					display = display[:50] + "..."
				}
				fmt.Printf("%s    %s: %s\n", colorRed("-"), k, display)
			}
		}
		if len(ing.Middlewares) > 0 {
			fmt.Printf("%s  Middlewares: %s\n", colorRed("-"), strings.Join(ing.Middlewares, ", "))
		}

		// AFTER: Show generated files
		genFiles := filesByIngress[key]
		if len(genFiles) == 0 {
			fmt.Printf("\n%s AFTER %s\n", colorDim("───"), colorDim(strings.Repeat("─", 61)))
			fmt.Printf("  (no specific files generated for this resource)\n")
			continue
		}

		// Also find the analysis for this ingress
		for _, ir := range report.IngressReports {
			if ir.Namespace == ing.Namespace && ir.Name == ing.Name {
				supported := 0
				partial := 0
				unsup := 0
				for _, m := range ir.Mappings {
					switch m.Status {
					case analyzer.StatusSupported:
						supported++
					case analyzer.StatusPartial:
						partial++
					case analyzer.StatusUnsupported:
						unsup++
					}
				}
				fmt.Printf("\n  Compatibility: %s%d supported%s  %s%d partial%s  %s%d unsupported%s\n",
					"\033[32m", supported, "\033[0m",
					"\033[33m", partial, "\033[0m",
					"\033[31m", unsup, "\033[0m")
				break
			}
		}

		fmt.Printf("\n%s AFTER %s\n", colorDim("───"), colorDim(strings.Repeat("─", 61)))
		for _, f := range genFiles {
			fmt.Printf("%s  File: %s\n", colorGreen("+"), f.RelPath)
			fmt.Printf("%s  Desc: %s\n", colorGreen("+"), f.Description)
			// Show first 30 lines of content
			lines := strings.Split(f.Content, "\n")
			maxLines := 30
			if len(lines) < maxLines {
				maxLines = len(lines)
			}
			for i := 0; i < maxLines; i++ {
				fmt.Printf("%s  %s\n", colorGreen("+"), lines[i])
			}
			if len(lines) > 30 {
				fmt.Printf("%s  ... (%d more lines)\n", colorDim(" "), len(lines)-30)
			}
			fmt.Println()
		}
	}

	// Summary
	fmt.Printf("%s\n", strings.Repeat("=", 80))
	fmt.Printf("  Total: %d resource(s) diffed → %d file(s) generated for %s\n",
		len(scanResult.Ingresses), len(files), diffTarget)
	fmt.Printf("%s\n\n", strings.Repeat("=", 80))

	return nil
}

// Color helpers — use ANSI codes, with NO_COLOR support
func colorRed(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}

func colorGreen(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

func colorDim(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}
