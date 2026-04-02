package cmd

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Quick health check — shows migration readiness at a glance",
	Long: `Performs a single-command health check of your cluster's ingress migration status.

Shows:
  - Controller detection (NGINX, Traefik, Kong, HAProxy, Istio)
  - Resource counts (Ingress, IngressRoute, VirtualService)
  - Annotation complexity breakdown
  - Migration readiness per target
  - Recommended next steps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor() error {
	fmt.Printf("\n  ing-switch doctor\n")
	fmt.Printf("  =================\n\n")

	// Connect
	s, err := scanner.NewScanner(kubeconfig, kubecontext)
	if err != nil {
		fmt.Printf("  [FAIL] Cannot connect to cluster: %v\n", err)
		fmt.Printf("\n  Tip: Use --kubeconfig or set KUBECONFIG\n\n")
		return err
	}
	fmt.Printf("  [OK]   Connected to cluster\n")

	// Scan
	result, err := s.Scan(namespace)
	if err != nil {
		fmt.Printf("  [FAIL] Scan failed: %v\n", err)
		return err
	}

	// Controller
	if result.Controller.Detected {
		icon := "[OK]  "
		if result.Controller.Type == "ingress-nginx" {
			icon = "[WARN]"
		}
		fmt.Printf("  %s Controller: %s", icon, result.Controller.Type)
		if result.Controller.Version != "" && result.Controller.Version != "unknown" {
			fmt.Printf(" (%s)", result.Controller.Version)
		}
		fmt.Printf(" in %s\n", result.Controller.Namespace)
		if result.Controller.Type == "ingress-nginx" {
			fmt.Printf("         ingress-nginx reached END OF LIFE on March 31, 2026\n")
		}
	} else {
		fmt.Printf("  [WARN] No ingress controller detected\n")
	}

	if len(result.Ingresses) == 0 {
		fmt.Printf("  [OK]   No Ingress or IngressRoute resources found\n")
		fmt.Printf("\n  Nothing to migrate!\n\n")
		return nil
	}

	// Count by source type
	nginxCount := 0
	irCount := 0
	kongCount := 0
	haproxyCount := 0
	istioCount := 0
	for _, ing := range result.Ingresses {
		switch ing.SourceType {
		case scanner.SourceTraefikIngressRoute:
			irCount++
		case scanner.SourceKongIngress:
			kongCount++
		case scanner.SourceHAProxyIngress:
			haproxyCount++
		case scanner.SourceIstioVirtualService:
			istioCount++
		default:
			nginxCount++
		}
	}

	fmt.Printf("  [OK]   Found %d resource(s)", len(result.Ingresses))
	var parts []string
	if nginxCount > 0 {
		parts = append(parts, fmt.Sprintf("%d Ingress", nginxCount))
	}
	if irCount > 0 {
		parts = append(parts, fmt.Sprintf("%d IngressRoute", irCount))
	}
	if kongCount > 0 {
		parts = append(parts, fmt.Sprintf("%d Kong", kongCount))
	}
	if haproxyCount > 0 {
		parts = append(parts, fmt.Sprintf("%d HAProxy", haproxyCount))
	}
	if istioCount > 0 {
		parts = append(parts, fmt.Sprintf("%d Istio VS", istioCount))
	}
	if len(parts) > 0 {
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Printf(" across %d namespace(s)\n", len(result.Namespaces))

	// Complexity breakdown
	simple, complex, unsupported := 0, 0, 0
	totalAnnotations := 0
	for _, ing := range result.Ingresses {
		switch ing.Complexity {
		case "simple":
			simple++
		case "complex":
			complex++
		case "unsupported":
			unsupported++
		}
		totalAnnotations += len(ing.NginxAnnotations)
	}

	fmt.Printf("\n  Resource Complexity\n")
	fmt.Printf("  -------------------\n")
	fmt.Printf("  Simple:      %d  (basic routing, quick migration)\n", simple)
	fmt.Printf("  Complex:     %d  (annotations need mapping)\n", complex)
	fmt.Printf("  Unsupported: %d  (manual review required)\n", unsupported)
	fmt.Printf("  Total annotations/features: %d\n", totalAnnotations)

	// Migration readiness per target
	targets := []string{"traefik", "gateway-api", "gateway-api-traefik"}
	targetLabels := map[string]string{
		"traefik":              "Traefik v3",
		"gateway-api":          "Gateway API (Envoy)",
		"gateway-api-traefik":  "Gateway API (Traefik)",
	}

	fmt.Printf("\n  Migration Readiness\n")
	fmt.Printf("  -------------------\n")

	bestTarget := ""
	bestScore := -1

	for _, target := range targets {
		// Skip traefik target if source is already traefik IngressRoute
		if target == "traefik" && irCount > 0 && nginxCount == 0 {
			continue
		}

		a := analyzer.NewAnalyzer(target)
		report := a.Analyze(result)

		ready := report.Summary.FullyCompatible
		workaround := report.Summary.NeedsWorkaround
		breaking := report.Summary.HasUnsupported
		total := report.Summary.Total

		score := 0
		if total > 0 {
			score = ((ready * 100) + (workaround * 70)) / total
		}

		bar := renderBar(score)
		fmt.Printf("  %s  %s %d%%\n", targetLabels[target], bar, score)
		fmt.Printf("    %d ready | %d workaround | %d breaking\n", ready, workaround, breaking)

		if score > bestScore {
			bestScore = score
			bestTarget = target
		}
	}

	// Recommendation
	fmt.Printf("\n  Recommendation\n")
	fmt.Printf("  ---------------\n")

	if unsupported > 0 {
		fmt.Printf("  %d resource(s) have unsupported features — review these first:\n", unsupported)
		for _, ing := range result.Ingresses {
			if ing.Complexity == "unsupported" {
				src := "Ingress"
				switch ing.SourceType {
				case scanner.SourceTraefikIngressRoute:
					src = "IngressRoute"
				case scanner.SourceKongIngress:
					src = "Kong"
				case scanner.SourceHAProxyIngress:
					src = "HAProxy"
				case scanner.SourceIstioVirtualService:
					src = "Istio VS"
				}
				fmt.Printf("    %s/%s (%s)\n", ing.Namespace, ing.Name, src)
			}
		}
		fmt.Printf("\n")
	}

	if bestTarget != "" {
		fmt.Printf("  Best target: %s (%d%% readiness)\n", targetLabels[bestTarget], bestScore)
	}

	fmt.Printf("\n  Next steps:\n")
	fmt.Printf("  1. ing-switch analyze --target %s     (see full annotation mapping)\n", bestTarget)
	fmt.Printf("  2. ing-switch migrate --target %s     (generate migration files)\n", bestTarget)
	fmt.Printf("  3. ing-switch ui                              (visual dashboard)\n")
	fmt.Printf("\n")

	return nil
}

func renderBar(percent int) string {
	filled := percent / 5
	if filled > 20 {
		filled = 20
	}
	empty := 20 - filled
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", empty) + "]"
}
