package server

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// RichValidationResult is the enriched validation output returned by the API.
type RichValidationResult struct {
	Target    string            `json:"target"`
	Phase     string            `json:"phase"`     // "pre-migration" | "migrating" | "post-migration"
	PhaseDesc string            `json:"phaseDesc"` // human-readable phase description
	Checks    []ValidationCheck `json:"checks"`
	Overall   string            `json:"overall"` // "pass" | "warn" | "fail"
	NextSteps []string          `json:"nextSteps"`
}

func runRichValidation(kubeconfig, kubecontext, target, ns string) (*RichValidationResult, error) {
	result := &RichValidationResult{Target: target}

	// Build k8s clients
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	overrides := &clientcmd.ConfigOverrides{}
	if kubecontext != "" {
		overrides.CurrentContext = kubecontext
	}
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	restCfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("cannot build kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	dynClient, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// --- Gather facts ---

	// 1. Scan cluster for ingresses + current controller
	s, scanErr := scanner.NewScanner(kubeconfig, kubecontext)
	var ingressCount int
	nginxPresent := false
	nginxNamespace := ""
	if scanErr == nil {
		sr, err := s.Scan(ns)
		if err == nil {
			ingressCount = len(sr.Ingresses)
			nginxPresent = sr.Controller.Detected && sr.Controller.Type == "ingress-nginx"
			nginxNamespace = sr.Controller.Namespace
		}
	}

	// 2. Detect target controller pods
	targetRunning, targetNamespace, targetVersion := detectTargetController(ctx, client, target)

	// 3. Discover API groups to check CRDs
	traefikCRDsInstalled := false
	gatewayAPICRDsInstalled := false
	apiGroups, err := client.Discovery().ServerGroups()
	if err == nil {
		for _, g := range apiGroups.Groups {
			if strings.Contains(g.Name, "traefik.io") || strings.Contains(g.Name, "traefik.containo.us") {
				traefikCRDsInstalled = true
			}
			if strings.Contains(g.Name, "gateway.networking.k8s.io") {
				gatewayAPICRDsInstalled = true
			}
		}
	}

	// 4. Count migration resources in cluster
	middlewareCount := countResource(ctx, dynClient, traefikCRDsInstalled,
		[]schema.GroupVersionResource{
			{Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares"},
			{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "middlewares"},
		})

	httprouteCount := countResource(ctx, dynClient, gatewayAPICRDsInstalled,
		[]schema.GroupVersionResource{
			{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"},
			{Group: "gateway.networking.k8s.io", Version: "v1beta1", Resource: "httproutes"},
		})

	// 5. Check ingresses for traefik middleware annotations (ingress update applied?)
	ingressesWithTraefikAnnotations := 0
	if scanErr == nil {
		sr, err := s.Scan(ns)
		if err == nil {
			for _, ing := range sr.Ingresses {
				for k := range ing.Annotations {
					if strings.Contains(k, "traefik.ingress.kubernetes.io") {
						ingressesWithTraefikAnnotations++
						break
					}
				}
			}
		}
	}

	// --- Build checks ---
	var checks []ValidationCheck

	// Check: Ingress resources
	if ingressCount > 0 {
		checks = append(checks, ValidationCheck{
			Name:    fmt.Sprintf("Ingress resources detected (%d total)", ingressCount),
			Status:  "pass",
			Message: fmt.Sprintf("%d Ingress objects found across the cluster", ingressCount),
		})
	} else {
		checks = append(checks, ValidationCheck{
			Name:    "Ingress resources",
			Status:  "warn",
			Message: "No Ingress resources found. Run a cluster scan first.",
		})
	}

	// Check: NGINX controller status
	if nginxPresent {
		checks = append(checks, ValidationCheck{
			Name:    "Ingress NGINX running",
			Status:  "warn",
			Message: fmt.Sprintf("Ingress NGINX running in '%s'. Still handling production traffic — this is expected during parallel migration.", nginxNamespace),
		})
	} else {
		checks = append(checks, ValidationCheck{
			Name:    "Ingress NGINX",
			Status:  "pass",
			Message: "Ingress NGINX not detected. Either not installed or successfully removed.",
		})
	}

	// Target-specific checks
	switch target {
	case "traefik":
		appendTraefikChecks(&checks, targetRunning, targetNamespace, targetVersion,
			traefikCRDsInstalled, middlewareCount, ingressesWithTraefikAnnotations, ingressCount)
	case "gateway-api":
		appendGatewayAPIChecks(&checks, targetRunning, targetNamespace, targetVersion,
			gatewayAPICRDsInstalled, httprouteCount, ingressCount)
	}

	result.Checks = checks

	// --- Determine migration phase ---
	switch {
	case targetRunning && !nginxPresent:
		result.Phase = "post-migration"
		result.PhaseDesc = "Target controller is active. NGINX has been removed. Migration complete."
	case targetRunning && nginxPresent:
		result.Phase = "migrating"
		result.PhaseDesc = "Both NGINX and target controller are running. Zero-downtime parallel migration in progress."
	default:
		result.Phase = "pre-migration"
		result.PhaseDesc = "NGINX is handling all traffic. Target controller not yet installed."
	}

	// --- Overall status ---
	result.Overall = "pass"
	for _, c := range result.Checks {
		if c.Status == "fail" {
			result.Overall = "fail"
			break
		}
		if c.Status == "warn" && result.Overall != "fail" {
			result.Overall = "warn"
		}
	}

	// --- Next steps ---
	result.NextSteps = buildNextSteps(target, result.Phase,
		targetRunning, traefikCRDsInstalled, gatewayAPICRDsInstalled,
		middlewareCount, httprouteCount, ingressesWithTraefikAnnotations, ingressCount)

	return result, nil
}

func detectTargetController(ctx context.Context, client kubernetes.Interface, target string) (running bool, namespace, version string) {
	var selectors []string
	var namespaces []string

	switch target {
	case "traefik":
		selectors = []string{"app.kubernetes.io/name=traefik", "app=traefik"}
		namespaces = []string{"traefik", "kube-system", "default", "ingress"}
	case "gateway-api":
		selectors = []string{"app.kubernetes.io/name=envoy-gateway", "app=envoy-gateway", "control-plane=envoy-gateway"}
		namespaces = []string{"envoy-gateway-system", "kube-system", "default"}
	}

	for _, ns := range namespaces {
		for _, sel := range selectors {
			pods, err := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: sel, Limit: 1})
			if err != nil || len(pods.Items) == 0 {
				continue
			}
			pod := pods.Items[0]
			if pod.Status.Phase == "Running" {
				running = true
				namespace = ns
				for _, c := range pod.Spec.Containers {
					parts := strings.Split(c.Image, ":")
					if len(parts) >= 2 {
						version = parts[len(parts)-1]
					}
				}
				return
			}
		}
	}
	return
}

func countResource(ctx context.Context, dynClient dynamic.Interface, installed bool, gvrs []schema.GroupVersionResource) int {
	if !installed {
		return 0
	}
	for _, gvr := range gvrs {
		list, err := dynClient.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
		if err == nil {
			return len(list.Items)
		}
	}
	return 0
}

func appendTraefikChecks(checks *[]ValidationCheck, targetRunning bool, namespace, version string,
	crdInstalled bool, middlewareCount, ingressesUpdated, ingressTotal int) {

	if targetRunning {
		*checks = append(*checks, ValidationCheck{
			Name:    fmt.Sprintf("Traefik controller running (%s)", version),
			Status:  "pass",
			Message: fmt.Sprintf("Traefik %s is running in namespace '%s' and ready to serve traffic", version, namespace),
		})
	} else {
		*checks = append(*checks, ValidationCheck{
			Name:    "Traefik controller",
			Status:  "fail",
			Message: "Traefik not detected. Install: helm install traefik traefik/traefik -n traefik --create-namespace --version 32.x",
		})
	}

	if crdInstalled {
		*checks = append(*checks, ValidationCheck{
			Name:    "Traefik CRDs installed",
			Status:  "pass",
			Message: "traefik.io API group present — Middleware, IngressRoute, ServersTransport CRDs available",
		})
	} else {
		*checks = append(*checks, ValidationCheck{
			Name:    "Traefik CRDs",
			Status:  "fail",
			Message: "Traefik CRDs not found. Install Traefik via Helm to register CRDs automatically.",
		})
	}

	if middlewareCount > 0 {
		*checks = append(*checks, ValidationCheck{
			Name:    fmt.Sprintf("Traefik Middlewares applied (%d found)", middlewareCount),
			Status:  "pass",
			Message: fmt.Sprintf("%d Middleware CRDs present (rate-limit, auth, CORS, headers, etc.)", middlewareCount),
		})
	} else if crdInstalled {
		*checks = append(*checks, ValidationCheck{
			Name:    "Traefik Middlewares",
			Status:  "warn",
			Message: "No Middleware resources found. Apply generated 02-middlewares/ files to create routing rules.",
		})
	}

	if ingressTotal > 0 {
		if ingressesUpdated == ingressTotal {
			*checks = append(*checks, ValidationCheck{
				Name:    fmt.Sprintf("Ingresses updated for Traefik (%d/%d)", ingressesUpdated, ingressTotal),
				Status:  "pass",
				Message: "All Ingresses have traefik.ingress.kubernetes.io/router.middlewares annotations applied.",
			})
		} else if ingressesUpdated > 0 {
			*checks = append(*checks, ValidationCheck{
				Name:    fmt.Sprintf("Ingresses updated for Traefik (%d/%d)", ingressesUpdated, ingressTotal),
				Status:  "warn",
				Message: fmt.Sprintf("%d of %d Ingresses updated. Apply remaining files from 03-ingresses/.", ingressesUpdated, ingressTotal),
			})
		} else {
			*checks = append(*checks, ValidationCheck{
				Name:    "Ingresses updated for Traefik",
				Status:  "warn",
				Message: "No Ingresses have Traefik annotations yet. Apply files from 03-ingresses/ directory.",
			})
		}
	}
}

func appendGatewayAPIChecks(checks *[]ValidationCheck, targetRunning bool, namespace, version string,
	crdInstalled bool, httprouteCount, ingressTotal int) {

	if targetRunning {
		*checks = append(*checks, ValidationCheck{
			Name:    fmt.Sprintf("Envoy Gateway running (%s)", version),
			Status:  "pass",
			Message: fmt.Sprintf("Envoy Gateway %s is running in namespace '%s'", version, namespace),
		})
	} else {
		*checks = append(*checks, ValidationCheck{
			Name:    "Envoy Gateway",
			Status:  "fail",
			Message: "Envoy Gateway not detected. Install: helm install eg oci://docker.io/envoyproxy/gateway-helm --version v1.3.0 -n envoy-gateway-system --create-namespace",
		})
	}

	if crdInstalled {
		*checks = append(*checks, ValidationCheck{
			Name:    "Gateway API CRDs installed (v1.2+)",
			Status:  "pass",
			Message: "gateway.networking.k8s.io CRDs present — HTTPRoute, GatewayClass, Gateway, ReferenceGrant available",
		})
	} else {
		*checks = append(*checks, ValidationCheck{
			Name:    "Gateway API CRDs",
			Status:  "fail",
			Message: "Gateway API CRDs not found. Install: kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml",
		})
	}

	if httprouteCount > 0 {
		*checks = append(*checks, ValidationCheck{
			Name:    fmt.Sprintf("HTTPRoutes applied (%d found)", httprouteCount),
			Status:  "pass",
			Message: fmt.Sprintf("%d HTTPRoute resources present in cluster", httprouteCount),
		})
	} else if crdInstalled {
		*checks = append(*checks, ValidationCheck{
			Name:    "HTTPRoutes",
			Status:  "warn",
			Message: "No HTTPRoute resources found. Apply generated files from 04-httproutes/ directory.",
		})
	}

	if ingressTotal > 0 && httprouteCount > 0 {
		coverage := (httprouteCount * 100) / ingressTotal
		status := "pass"
		msg := fmt.Sprintf("%d HTTPRoutes cover %d original Ingresses (%d%% coverage)", httprouteCount, ingressTotal, coverage)
		if coverage < 80 {
			status = "warn"
			msg += " — some Ingresses may not have corresponding HTTPRoutes yet"
		}
		*checks = append(*checks, ValidationCheck{
			Name:    "HTTPRoute coverage",
			Status:  status,
			Message: msg,
		})
	}
}

func buildNextSteps(target, phase string, targetRunning, traefikCRDs, gatewayAPICRDs bool,
	middlewareCount, httprouteCount, ingressesUpdated, ingressTotal int) []string {

	switch phase {
	case "pre-migration":
		if target == "traefik" {
			return []string{
				"Generate migration files on the Migrate tab",
				"Review 00-migration-report.md to understand all changes",
				"Install Traefik alongside NGINX (safe — won't affect production traffic)",
			}
		}
		return []string{
			"Generate migration files on the Migrate tab",
			"Review 00-migration-report.md for a full migration overview",
			"Install Gateway API CRDs (standard-install.yaml)",
			"Install Envoy Gateway alongside NGINX (safe — won't affect production traffic)",
		}
	case "migrating":
		var steps []string
		if target == "traefik" {
			if middlewareCount == 0 {
				steps = append(steps, "Apply Middleware CRDs (02-middlewares/) — creates routing rules")
			}
			if ingressesUpdated < ingressTotal {
				steps = append(steps, "Apply updated Ingresses (03-ingresses/) — attaches middlewares")
			}
			steps = append(steps, "Run 04-verify.sh to test Traefik handles traffic correctly")
			steps = append(steps, "Update DNS to Traefik's LoadBalancer IP once verified")
			steps = append(steps, "After DNS propagation: run 06-cleanup/ to remove NGINX")
		} else {
			if httprouteCount == 0 {
				steps = append(steps, "Apply GatewayClass + Gateway (03-gateway/)")
				steps = append(steps, "Apply HTTPRoutes (04-httproutes/)")
				steps = append(steps, "Apply Policies if any (05-policies/)")
			}
			steps = append(steps, "Run 06-verify.sh to test Envoy Gateway")
			steps = append(steps, "Update DNS to Envoy Gateway's LoadBalancer IP once verified")
			steps = append(steps, "After DNS propagation: run 07-cleanup/ to remove NGINX")
		}
		return steps
	case "post-migration":
		return []string{
			"Monitor application logs and metrics for 24+ hours",
			"Verify all TLS certificates renew correctly (cert-manager check)",
			"Keep NGINX Helm values.yaml as backup for emergency rollback",
			"Update any CI/CD pipelines to deploy new resource types",
		}
	}
	return nil
}
