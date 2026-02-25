package traefik

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// Migrator generates Traefik migration files from an NGINX ingress setup.
type Migrator struct{}

// NewMigrator creates a new Traefik Migrator.
func NewMigrator() *Migrator {
	return &Migrator{}
}

// Migrate generates all files needed to migrate from NGINX to Traefik.
func (m *Migrator) Migrate(scan *scanner.ScanResult, report *analyzer.AnalysisReport) ([]generator.GeneratedFile, error) {
	var files []generator.GeneratedFile

	// 1. Helm install
	files = append(files, generateHelmInstall())
	files = append(files, generateHelmValues())

	// 2. Middlewares — one file per ingress containing all its middlewares
	middlewareNames := make(map[string][]string) // ingress key → middleware names
	for _, ing := range scan.Ingresses {
		mws := generateMiddlewares(ing)
		var mwYAMLs []string
		var names []string
		for _, mw := range mws {
			mwYAMLs = append(mwYAMLs, mw.YAML)
			names = append(names, fmt.Sprintf("%s-%s@kubernetescrd", ing.Namespace, mw.Name))
		}
		if len(mwYAMLs) > 0 {
			key := ing.Namespace + "-" + ing.Name
			middlewareNames[key] = names
			files = append(files, generator.GeneratedFile{
				RelPath:     fmt.Sprintf("02-middlewares/%s-%s-middlewares.yaml", ing.Namespace, ing.Name),
				Content:     strings.Join(mwYAMLs, "---\n"),
				Description: fmt.Sprintf("Traefik Middlewares for %s/%s", ing.Namespace, ing.Name),
				Category:    "middleware",
			})
		}
	}

	// 3. Updated Ingress manifests (same format, updated annotations to attach middlewares)
	for _, ing := range scan.Ingresses {
		key := ing.Namespace + "-" + ing.Name
		mwNames := middlewareNames[key]
		ingressYAML := generateUpdatedIngress(ing, mwNames)
		files = append(files, generator.GeneratedFile{
			RelPath:     fmt.Sprintf("03-ingresses/%s-%s.yaml", ing.Namespace, ing.Name),
			Content:     ingressYAML,
			Description: fmt.Sprintf("Updated Ingress manifest for %s/%s with Traefik annotations", ing.Namespace, ing.Name),
			Category:    "ingress",
		})
	}

	// 4. Verify script
	files = append(files, generateVerifyScript(scan))

	// 5. DNS migration guide
	files = append(files, generateDNSGuide())

	// 6. Cleanup scripts
	files = append(files, generatePreserveIngressClass())
	files = append(files, generateCleanupScript())

	return files, nil
}

func generateHelmInstall() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "01-install-traefik/helm-install.sh",
		Content: `#!/bin/bash
# Install Traefik alongside existing NGINX Ingress Controller
# Both will run in parallel — zero downtime migration
set -e

echo "Adding Traefik Helm repository..."
helm repo add traefik https://traefik.github.io/charts
helm repo update

echo "Installing Traefik with Kubernetes Ingress NGINX provider..."
helm upgrade --install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace \
  --values values.yaml \
  --version ">=3.6.2"

echo "Waiting for Traefik to be ready..."
kubectl rollout status deployment/traefik -n traefik --timeout=120s

echo ""
echo "Traefik installed successfully!"
echo ""
echo "Get Traefik LoadBalancer IP:"
kubectl get svc -n traefik traefik
echo ""
echo "Test that Traefik can serve your Ingress resources:"
echo "  bash ../04-verify.sh"
`,
		Description: "Helm install script for Traefik",
		Category:    "install",
	}
}

func generateHelmValues() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "01-install-traefik/values.yaml",
		Content: `# Traefik Helm values for NGINX Ingress migration
# Requires Traefik v3.6.2+

providers:
  # Enable the Kubernetes Ingress NGINX compatibility provider
  # This makes Traefik watch Ingress resources with ingressClassName: nginx
  # and automatically translates common nginx.ingress.kubernetes.io annotations
  kubernetesIngressNginx:
    enabled: true

  kubernetesIngress:
    enabled: true
    # Watch all namespaces by default
    allowCrossNamespace: false

# High availability — spread across nodes
deployment:
  replicas: 2

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app.kubernetes.io/name: traefik
            app.kubernetes.io/instance: traefik
        topologyKey: kubernetes.io/hostname

# Ensure at least one pod is always available
podDisruptionBudget:
  enabled: true
  minAvailable: 1

# Service configuration — Traefik gets its own LoadBalancer IP
service:
  enabled: true
  type: LoadBalancer

# Entrypoints
ports:
  web:
    port: 8000
    expose:
      default: true
    exposedPort: 80
  websecure:
    port: 8443
    expose:
      default: true
    exposedPort: 443
    tls:
      enabled: true

# Optional: Enable dashboard (access via kubectl port-forward)
# api:
#   dashboard: true
#   insecure: true

logs:
  general:
    level: INFO
  access:
    enabled: true
`,
		Description: "Traefik Helm values file",
		Category:    "install",
	}
}

func generateUpdatedIngress(ing scanner.IngressInfo, middlewareNames []string) string {
	annotations := copyAnnotations(ing.Annotations)

	// Remove nginx.ingress annotations that Traefik handles differently
	toRemove := []string{
		"nginx.ingress.kubernetes.io/ssl-redirect",
		"nginx.ingress.kubernetes.io/force-ssl-redirect",
		"nginx.ingress.kubernetes.io/enable-cors",
		"nginx.ingress.kubernetes.io/cors-allow-origin",
		"nginx.ingress.kubernetes.io/cors-allow-methods",
		"nginx.ingress.kubernetes.io/cors-allow-headers",
		"nginx.ingress.kubernetes.io/cors-expose-headers",
		"nginx.ingress.kubernetes.io/cors-allow-credentials",
		"nginx.ingress.kubernetes.io/cors-max-age",
		"nginx.ingress.kubernetes.io/auth-url",
		"nginx.ingress.kubernetes.io/auth-response-headers",
		"nginx.ingress.kubernetes.io/auth-method",
		"nginx.ingress.kubernetes.io/limit-rps",
		"nginx.ingress.kubernetes.io/limit-rpm",
		"nginx.ingress.kubernetes.io/limit-connections",
		"nginx.ingress.kubernetes.io/whitelist-source-range",
		"nginx.ingress.kubernetes.io/denylist-source-range",
		"nginx.ingress.kubernetes.io/rewrite-target",
		"nginx.ingress.kubernetes.io/use-regex",
	}
	for _, k := range toRemove {
		delete(annotations, k)
	}

	// Add Traefik middleware annotation
	if len(middlewareNames) > 0 {
		annotations["traefik.ingress.kubernetes.io/router.middlewares"] = strings.Join(middlewareNames, ",")
	}

	// Build annotations YAML
	var annotationLines []string
	for k, v := range annotations {
		annotationLines = append(annotationLines, fmt.Sprintf("    %s: %q", k, v))
	}

	// Build TLS section
	tlsSection := ""
	for _, secret := range ing.TLSSecrets {
		hosts := hostListForTLS(ing)
		tlsSection += fmt.Sprintf(`  - hosts:
%s
    secretName: %s
`, hosts, secret)
	}

	// Build rules
	rulesSection := buildRulesSection(ing)

	ingressClass := ing.IngressClass
	if ingressClass == "" {
		ingressClass = "nginx" // Keep the nginx class — Traefik watches it
	}

	yaml := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  annotations:
%s
spec:
  ingressClassName: %s
  rules:
%s`, ing.Name, ing.Namespace,
		strings.Join(annotationLines, "\n"),
		ingressClass,
		rulesSection)

	if tlsSection != "" {
		yaml += fmt.Sprintf("  tls:\n%s", tlsSection)
	}

	yaml += "\n"
	return yaml
}

func buildRulesSection(ing scanner.IngressInfo) string {
	// Group paths by host
	hostPaths := make(map[string][]scanner.PathInfo)
	for _, p := range ing.Paths {
		hostPaths[p.Host] = append(hostPaths[p.Host], p)
	}

	var rules []string
	for host, paths := range hostPaths {
		hostLine := ""
		if host != "" {
			hostLine = fmt.Sprintf("  - host: %s\n", host)
		} else {
			hostLine = "  - {}\n"
		}

		httpLines := "    http:\n      paths:\n"
		for _, p := range paths {
			pathType := p.PathType
			if pathType == "" {
				pathType = "Prefix"
			}
			path := p.Path
			if path == "" {
				path = "/"
			}
			httpLines += fmt.Sprintf(`      - path: %s
        pathType: %s
        backend:
          service:
            name: %s
            port:
              number: %d
`, path, pathType, p.ServiceName, p.ServicePort)
		}
		rules = append(rules, hostLine+httpLines)
	}

	return strings.Join(rules, "")
}

func hostListForTLS(ing scanner.IngressInfo) string {
	var lines []string
	for _, h := range ing.Hosts {
		lines = append(lines, fmt.Sprintf("    - %s", h))
	}
	return strings.Join(lines, "\n")
}

func copyAnnotations(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func generateVerifyScript(scan *scanner.ScanResult) generator.GeneratedFile {
	var testLines []string
	for _, ing := range scan.Ingresses {
		for _, host := range ing.Hosts {
			testLines = append(testLines, fmt.Sprintf(`echo "Testing %s/%s → %s"
TRAEFIK_IP=$(kubectl get svc -n traefik traefik -o go-template='{{ $ing := index .status.loadBalancer.ingress 0 }}{{ if $ing.ip }}{{ $ing.ip }}{{ else }}{{ $ing.hostname }}{{ end }}')
curl -s --connect-to "%s:80:${TRAEFIK_IP}:80" "http://%s" -o /dev/null -w "HTTP %%{http_code}\n" || true
`, ing.Namespace, ing.Name, host, host, host))
		}
	}

	return generator.GeneratedFile{
		RelPath: "04-verify.sh",
		Content: fmt.Sprintf(`#!/bin/bash
# Verify Traefik is handling your Ingress resources correctly
# Run this BEFORE cutting over DNS to Traefik
set -e

echo "=== ing-switch Verification Script ==="
echo ""

TRAEFIK_IP=$(kubectl get svc -n traefik traefik -o go-template='{{ $ing := index .status.loadBalancer.ingress 0 }}{{ if $ing.ip }}{{ $ing.ip }}{{ else }}{{ $ing.hostname }}{{ end }}' 2>/dev/null || echo "")

if [ -z "$TRAEFIK_IP" ]; then
  echo "ERROR: Traefik LoadBalancer IP not assigned yet. Wait for it:"
  kubectl get svc -n traefik traefik
  exit 1
fi

echo "Traefik LoadBalancer: $TRAEFIK_IP"
echo ""
echo "Testing Ingress resources via Traefik..."
echo ""

%s

echo ""
echo "=== Traefik Dashboard ==="
echo "kubectl port-forward -n traefik svc/traefik 9000:9000"
echo "Open http://localhost:9000/dashboard/"
echo ""
echo "If all tests pass, proceed to 05-dns-migration.md"
`, strings.Join(testLines, "\n")),
		Description: "Verification script to test Traefik before DNS cutover",
		Category:    "verify",
	}
}

func generateDNSGuide() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "05-dns-migration.md",
		Content: `# DNS Migration Guide

## Overview

At this point:
- NGINX is still running and handling all production traffic
- Traefik is running with its own LoadBalancer IP
- Both controllers are watching the same Ingress resources
- You have verified Traefik works correctly (verify.sh passed)

## Step 1: Get both LoadBalancer IPs

` + "```bash" + `
NGINX_IP=$(kubectl get svc -n ingress-nginx ingress-nginx-controller \
  -o go-template='{{ $ing := index .status.loadBalancer.ingress 0 }}{{ if $ing.ip }}{{ $ing.ip }}{{ else }}{{ $ing.hostname }}{{ end }}')

TRAEFIK_IP=$(kubectl get svc -n traefik traefik \
  -o go-template='{{ $ing := index .status.loadBalancer.ingress 0 }}{{ if $ing.ip }}{{ $ing.ip }}{{ else }}{{ $ing.hostname }}{{ end }}')

echo "NGINX:   $NGINX_IP"
echo "Traefik: $TRAEFIK_IP"
` + "```" + `

## Step 2: Add Traefik to DNS (parallel traffic)

In your DNS provider, add the Traefik IP alongside the NGINX IP for your domains.
Both will receive traffic via round-robin.

**Set a low TTL (e.g., 60s) before making changes** so rollback is fast.

## Step 3: Monitor

Watch traffic on both controllers:
` + "```bash" + `
# NGINX access logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx -f

# Traefik access logs
kubectl logs -n traefik -l app.kubernetes.io/name=traefik -f
` + "```" + `

## Step 4: Remove NGINX from DNS

Once you're confident Traefik is handling traffic correctly:
1. Remove the NGINX LoadBalancer IP from your DNS records
2. Wait 24-48 hours for DNS propagation (some ISPs ignore TTL)
3. Keep NGINX running during this period

## Step 5: Cleanup

Once traffic is fully on Traefik, proceed to **06-cleanup/**.

## Rollback

To roll back at any point:
1. Remove Traefik from DNS records
2. Ensure NGINX IP is the only record
3. Traffic will return to NGINX within TTL seconds
`,
		Description: "Step-by-step DNS migration guide",
		Category:    "guide",
	}
}

func generatePreserveIngressClass() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "06-cleanup/01-preserve-ingressclass.yaml",
		Content: `# Apply this BEFORE uninstalling NGINX to preserve the IngressClass.
# Traefik needs this to continue discovering your Ingress resources.
#
# If NGINX was installed via Helm, annotate the IngressClass to survive uninstall:
#   kubectl annotate ingressclass nginx helm.sh/resource-policy=keep
#
# This file creates a standalone IngressClass as a fallback:

apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx
  annotations:
    ingressclass.kubernetes.io/is-default-class: "false"
    # Prevent Helm from deleting this when NGINX is uninstalled
    helm.sh/resource-policy: keep
spec:
  controller: k8s.io/ingress-nginx
  # Note: Traefik watches this class via its kubernetesIngressNginx provider
`,
		Description: "Preserve nginx IngressClass before removing NGINX",
		Category:    "cleanup",
	}
}

func generateCleanupScript() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "06-cleanup/02-remove-nginx.sh",
		Content: `#!/bin/bash
# Remove Ingress NGINX Controller after migration is complete.
#
# PREREQUISITES:
# 1. All traffic is routing through Traefik
# 2. NGINX has been removed from DNS for 24-48 hours
# 3. You have applied 01-preserve-ingressclass.yaml
#
set -e

echo "=== Removing Ingress NGINX Controller ==="
echo ""

# Step 1: Preserve IngressClass
echo "Step 1: Preserving nginx IngressClass..."
kubectl apply -f 01-preserve-ingressclass.yaml
kubectl annotate ingressclass nginx helm.sh/resource-policy=keep --overwrite

# Step 2: Remove admission webhooks
echo "Step 2: Removing NGINX admission webhooks..."
kubectl delete validatingwebhookconfiguration ingress-nginx-admission --ignore-not-found
kubectl delete mutatingwebhookconfiguration ingress-nginx-admission --ignore-not-found

# Step 3: Uninstall NGINX Helm release
echo "Step 3: Uninstalling NGINX Helm release..."
if helm list -n ingress-nginx | grep -q ingress-nginx; then
  helm uninstall ingress-nginx -n ingress-nginx
else
  echo "  NGINX Helm release not found — may have been installed differently"
  echo "  Manual cleanup may be required"
fi

# Step 4: Verify IngressClass still exists
echo ""
echo "Step 4: Verifying IngressClass still exists..."
kubectl get ingressclass nginx

# Step 5: Clean up namespace
echo "Step 5: Cleaning up ingress-nginx namespace..."
kubectl delete namespace ingress-nginx --ignore-not-found

echo ""
echo "=== Migration Complete ==="
echo ""
echo "Verify Traefik is still serving your Ingresses:"
kubectl get ingress --all-namespaces
`,
		Description: "Remove NGINX after migration is verified complete",
		Category:    "cleanup",
	}
}
