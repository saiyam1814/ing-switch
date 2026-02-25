package gatewayapi

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

const (
	defaultGatewayName      = "ing-switch-gateway"
	defaultGatewayNamespace = "default"
)

// generateGatewayClass creates the GatewayClass resource for Envoy Gateway.
func generateGatewayClass() string {
	return `apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: eg
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
`
}

// generateGateway creates the Gateway resource with HTTP and HTTPS listeners.
func generateGateway(scan *scanner.ScanResult) string {
	// Collect all TLS secrets referenced by Ingresses
	type tlsEntry struct {
		hosts      []string
		secretName string
		namespace  string
	}
	var tlsEntries []tlsEntry

	for _, ing := range scan.Ingresses {
		if ing.TLSEnabled && len(ing.TLSSecrets) > 0 {
			for _, secret := range ing.TLSSecrets {
				tlsEntries = append(tlsEntries, tlsEntry{
					hosts:      ing.Hosts,
					secretName: secret,
					namespace:  ing.Namespace,
				})
			}
		}
	}

	// Build TLS listeners
	tlsListeners := ""
	for i, tls := range tlsEntries {
		hosts := buildHostnameList(tls.hosts)
		tlsListeners += fmt.Sprintf(`  - name: https-%d
    protocol: HTTPS
    port: 443
%s    tls:
      mode: Terminate
      certificateRefs:
      - name: %s
        namespace: %s
    allowedRoutes:
      namespaces:
        from: All
`, i, hosts, tls.secretName, tls.namespace)
	}

	if tlsListeners == "" {
		// Add a default HTTPS listener placeholder
		tlsListeners = `  - name: https
    protocol: HTTPS
    port: 443
    # Add TLS configuration:
    # tls:
    #   mode: Terminate
    #   certificateRefs:
    #   - name: your-tls-secret
    #     namespace: your-namespace
`
	}

	return fmt.Sprintf(`apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: %s
  namespace: %s
spec:
  gatewayClassName: eg
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    allowedRoutes:
      namespaces:
        from: All
%s`, defaultGatewayName, defaultGatewayNamespace, tlsListeners)
}

func buildHostnameList(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}
	// Use only the first host as a listener hostname
	return fmt.Sprintf("    hostname: \"%s\"\n", hosts[0])
}

// generatePolicies creates Envoy Gateway extension policies for advanced features.
func generatePolicies(scan *scanner.ScanResult) []policyFile {
	var policies []policyFile

	for _, ing := range scan.Ingresses {
		annotations := ing.NginxAnnotations

		// Rate limiting via BackendTrafficPolicy
		if _, hasRPS := annotations["limit-rps"]; hasRPS {
			policies = append(policies, generateRateLimitPolicy(ing))
		}

		// External auth via SecurityPolicy
		if authURL, ok := annotations["auth-url"]; ok && authURL != "" {
			policies = append(policies, generateSecurityPolicy(ing))
		}

		// IP filter via SecurityPolicy
		if denyList, ok := annotations["denylist-source-range"]; ok && denyList != "" {
			policies = append(policies, generateIPFilterPolicy(ing, denyList, "Deny"))
		}
		if allowList, ok := annotations["whitelist-source-range"]; ok && allowList != "" {
			policies = append(policies, generateIPFilterPolicy(ing, allowList, "Allow"))
		}
	}

	return policies
}

type policyFile struct {
	name string
	yaml string
}

func generateRateLimitPolicy(ing scanner.IngressInfo) policyFile {
	rps := ing.NginxAnnotations["limit-rps"]
	if rps == "" {
		rps = "100"
	}

	name := fmt.Sprintf("%s-%s-ratelimit", ing.Namespace, ing.Name)
	yaml := fmt.Sprintf(`apiVersion: gateway.envoyproxy.io/v1alpha1
kind: BackendTrafficPolicy
metadata:
  name: %s
  namespace: %s
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: %s
  rateLimit:
    type: Global
    global:
      rules:
      - clientSelectors:
        - sourceCIDR:
            type: Distinct
            value: "0.0.0.0/0"
        limit:
          requests: %s
          unit: Second
`, name, ing.Namespace, ing.Name, rps)

	return policyFile{name: name, yaml: yaml}
}

func generateSecurityPolicy(ing scanner.IngressInfo) policyFile {
	authURL := ing.NginxAnnotations["auth-url"]
	name := fmt.Sprintf("%s-%s-extauth", ing.Namespace, ing.Name)

	responseHeaders := "[]"
	if rh, ok := ing.NginxAnnotations["auth-response-headers"]; ok && rh != "" {
		headers := strings.Split(rh, ",")
		var lines []string
		for _, h := range headers {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", strings.TrimSpace(h)))
		}
		responseHeaders = "\n" + strings.Join(lines, "\n")
	}

	yaml := fmt.Sprintf(`apiVersion: gateway.envoyproxy.io/v1alpha1
kind: SecurityPolicy
metadata:
  name: %s
  namespace: %s
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: %s
  extAuth:
    http:
      backendRef:
        name: auth-service   # Replace with your auth service name
        port: 9001           # Replace with your auth service port
      # Original auth-url: %s
      # The auth service URL above should match your auth-url service
      headersToBackend:%s
`, name, ing.Namespace, ing.Name, authURL, responseHeaders)

	return policyFile{name: name, yaml: yaml}
}

func generateIPFilterPolicy(ing scanner.IngressInfo, cidr, action string) policyFile {
	name := fmt.Sprintf("%s-%s-ipfilter", ing.Namespace, ing.Name)
	cidrs := strings.Split(cidr, ",")
	var cidrLines []string
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c != "" {
			cidrLines = append(cidrLines, fmt.Sprintf("      - cidrRange: \"%s\"", c))
		}
	}

	yaml := fmt.Sprintf(`apiVersion: gateway.envoyproxy.io/v1alpha1
kind: SecurityPolicy
metadata:
  name: %s
  namespace: %s
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: %s
  authorization:
    defaultAction: %s
    rules:
    - action: %s
      principal:
        clientCIDRs:
%s
`, name, ing.Namespace, ing.Name,
		map[string]string{"Allow": "Deny", "Deny": "Allow"}[action], // default is opposite
		action, strings.Join(cidrLines, "\n"))

	return policyFile{name: name, yaml: yaml}
}
