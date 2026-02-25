package gatewayapi

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/analyzer"
	"github.com/saiyam1814/ing-switch/pkg/generator"
	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// Migrator generates Gateway API migration files.
type Migrator struct{}

// NewMigrator creates a new Gateway API Migrator.
func NewMigrator() *Migrator {
	return &Migrator{}
}

// Migrate generates all files for Gateway API (Envoy Gateway) migration.
func (m *Migrator) Migrate(scan *scanner.ScanResult, report *analyzer.AnalysisReport) ([]generator.GeneratedFile, error) {
	var files []generator.GeneratedFile

	// 1. Install Gateway API CRDs
	files = append(files, generateCRDInstall())

	// 2. Install Envoy Gateway
	files = append(files, generateEnvoyGatewayInstall())
	files = append(files, generateEnvoyGatewayValues())

	// 3. GatewayClass + Gateway
	files = append(files, generator.GeneratedFile{
		RelPath:     "03-gateway/gatewayclass.yaml",
		Content:     generateGatewayClass(),
		Description: "GatewayClass using Envoy Gateway controller",
		Category:    "gateway",
	})
	files = append(files, generator.GeneratedFile{
		RelPath:     "03-gateway/gateway.yaml",
		Content:     generateGateway(scan),
		Description: "Gateway with HTTP and HTTPS listeners",
		Category:    "gateway",
	})

	// 4. HTTPRoutes — one per Ingress
	// Build hostname→sectionName map to enable sectionName-based listener binding
	// for ssl-redirect routes (redirect on HTTP listener, backend on HTTPS listener).
	hostnameToSection := buildHostnameToSection(scan)
	for _, ing := range scan.Ingresses {
		httpRouteYAML := generateHTTPRoute(ing, defaultGatewayName, defaultGatewayNamespace, hostnameToSection)
		files = append(files, generator.GeneratedFile{
			RelPath:     fmt.Sprintf("04-httproutes/%s-%s.yaml", ing.Namespace, ing.Name),
			Content:     httpRouteYAML,
			Description: fmt.Sprintf("HTTPRoute for %s/%s", ing.Namespace, ing.Name),
			Category:    "httproute",
		})
	}

	// 5. Extension Policies
	policies := generatePolicies(scan)
	for _, p := range policies {
		files = append(files, generator.GeneratedFile{
			RelPath:     fmt.Sprintf("05-policies/%s.yaml", p.name),
			Content:     p.yaml,
			Description: fmt.Sprintf("Envoy Gateway policy: %s", p.name),
			Category:    "policy",
		})
	}

	// 6. Verify script
	files = append(files, generateGatewayVerifyScript(scan))

	// 7. Cleanup
	files = append(files, generateGatewayCleanup())

	return files, nil
}

// buildHostnameToSection maps each TLS-enabled ingress's primary hostname to
// its corresponding HTTPS listener sectionName (https-0, https-1, …).
// The index order matches the TLS listener generation in generateGateway().
func buildHostnameToSection(scan *scanner.ScanResult) map[string]string {
	m := make(map[string]string)
	idx := 0
	for _, ing := range scan.Ingresses {
		if ing.TLSEnabled && len(ing.TLSSecrets) > 0 {
			for range ing.TLSSecrets {
				if len(ing.Hosts) > 0 {
					m[ing.Hosts[0]] = fmt.Sprintf("https-%d", idx)
				}
				idx++
			}
		}
	}
	return m
}

func generateCRDInstall() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "01-install-gateway-api-crds/install.sh",
		Content: `#!/bin/bash
# Install Kubernetes Gateway API CRDs (Standard channel)
set -e

GATEWAY_API_VERSION="v1.2.0"

echo "Installing Gateway API CRDs (version ${GATEWAY_API_VERSION})..."
kubectl apply -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

echo ""
echo "Verifying CRDs..."
kubectl get crd gateways.gateway.networking.k8s.io
kubectl get crd httproutes.gateway.networking.k8s.io
kubectl get crd gatewayclasses.gateway.networking.k8s.io

echo ""
echo "Gateway API CRDs installed successfully!"
`,
		Description: "Install Gateway API CRDs",
		Category:    "install",
	}
}

func generateEnvoyGatewayInstall() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "02-install-envoy-gateway/helm-install.sh",
		Content: `#!/bin/bash
# Install Envoy Gateway
set -e

echo "Adding Envoy Gateway Helm repository..."
helm repo add eg https://charts.gateway.envoyproxy.io
helm repo update

echo "Installing Envoy Gateway..."
helm upgrade --install eg eg/gateway-helm \
  --namespace envoy-gateway-system \
  --create-namespace \
  --version v1.2.0 \
  --values values.yaml

echo "Waiting for Envoy Gateway to be ready..."
kubectl rollout status deployment/envoy-gateway -n envoy-gateway-system --timeout=120s

echo ""
echo "Envoy Gateway installed successfully!"
echo ""
echo "Next: Apply the GatewayClass and Gateway resources"
echo "  kubectl apply -f ../03-gateway/"
`,
		Description: "Helm install script for Envoy Gateway",
		Category:    "install",
	}
}

func generateEnvoyGatewayValues() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "02-install-envoy-gateway/values.yaml",
		Content: `# Envoy Gateway Helm values

# Configuration
config:
  envoyGateway:
    gateway:
      controllerName: gateway.envoyproxy.io/gatewayclass-controller
    provider:
      type: Kubernetes
    logging:
      level:
        default: info

# High availability
deployment:
  replicas: 2
`,
		Description: "Envoy Gateway Helm values",
		Category:    "install",
	}
}

func generateGatewayVerifyScript(scan *scanner.ScanResult) generator.GeneratedFile {
	var testLines []string
	for _, ing := range scan.Ingresses {
		for _, host := range ing.Hosts {
			testLines = append(testLines, fmt.Sprintf(`echo "Testing %s/%s → %s"
GATEWAY_IP=$(kubectl get gateway %s -n %s -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || echo "")
if [ -n "$GATEWAY_IP" ]; then
  curl -s --connect-to "%s:80:${GATEWAY_IP}:80" "http://%s" -o /dev/null -w "HTTP %%{http_code}\n" || true
else
  echo "  Gateway IP not yet assigned"
fi
`, ing.Namespace, ing.Name, host, defaultGatewayName, defaultGatewayNamespace, host, host))
		}
	}

	return generator.GeneratedFile{
		RelPath: "06-verify.sh",
		Content: fmt.Sprintf(`#!/bin/bash
# Verify Gateway API (Envoy Gateway) is handling your routes
set -e

echo "=== ing-switch Gateway API Verification ==="
echo ""

# Check gateway status
echo "Gateway status:"
kubectl get gateway %s -n %s
echo ""

# Check HTTPRoutes
echo "HTTPRoute status:"
kubectl get httproutes --all-namespaces
echo ""

echo "Testing routes..."
echo ""

%s

echo ""
echo "If all tests pass, proceed to DNS migration:"
echo "  Update DNS to point to Gateway's LoadBalancer IP"
echo ""
echo "Get Gateway LoadBalancer IP:"
kubectl get gateway %s -n %s -o jsonpath='{.status.addresses[0].value}'
echo ""
`, defaultGatewayName, defaultGatewayNamespace,
			strings.Join(testLines, "\n"),
			defaultGatewayName, defaultGatewayNamespace),
		Description: "Verification script for Gateway API routes",
		Category:    "verify",
	}
}

func generateGatewayCleanup() generator.GeneratedFile {
	return generator.GeneratedFile{
		RelPath: "07-cleanup/remove-nginx.sh",
		Content: `#!/bin/bash
# Remove Ingress NGINX after Gateway API migration is complete
set -e

echo "=== Removing Ingress NGINX Controller ==="

# Remove admission webhooks
kubectl delete validatingwebhookconfiguration ingress-nginx-admission --ignore-not-found
kubectl delete mutatingwebhookconfiguration ingress-nginx-admission --ignore-not-found

# Uninstall NGINX
if helm list -n ingress-nginx | grep -q ingress-nginx; then
  helm uninstall ingress-nginx -n ingress-nginx
fi

# Remove namespace
kubectl delete namespace ingress-nginx --ignore-not-found

echo ""
echo "NGINX removed. Your Gateway API routes are now the only ingress path."
echo ""
echo "Verify HTTPRoutes are all healthy:"
kubectl get httproutes --all-namespaces
`,
		Description: "Remove NGINX after Gateway API migration",
		Category:    "cleanup",
	}
}
