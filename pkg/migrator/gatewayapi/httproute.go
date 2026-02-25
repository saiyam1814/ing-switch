package gatewayapi

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// generateHTTPRoute converts a single Ingress to Gateway API HTTPRoute YAML.
//
// When the ingress has ssl-redirect/force-ssl-redirect, two HTTPRoute resources
// are generated in a single YAML file (separated by ---):
//
//  1. A "-redirect" HTTPRoute attached to the HTTP listener (sectionName: http)
//     containing ONLY RequestRedirect rules — no backendRefs.
//  2. A backend HTTPRoute attached to the matching HTTPS listener
//     (sectionName: https-N looked up via hostnameToSection) containing
//     ONLY backend rules — no RequestRedirect.
//
// This follows the correct Gateway API pattern: redirect and backend rules must
// live in separate routes attached to separate listeners to avoid redirect loops.
// Without sectionName the same route attaches to both HTTP and HTTPS listeners
// and RequestRedirect fires on HTTPS requests too (creating an infinite loop).
func generateHTTPRoute(ing scanner.IngressInfo, gatewayName, gatewayNamespace string, hostnameToSection map[string]string) string {
	annotations := ing.NginxAnnotations
	hasSSLRedirect := annotations["force-ssl-redirect"] == "true" || annotations["ssl-redirect"] == "true"

	if hasSSLRedirect {
		return generateSplitHTTPRoutes(ing, gatewayName, gatewayNamespace, hostnameToSection)
	}
	return generateSingleHTTPRoute(ing, gatewayName, gatewayNamespace, "")
}

// generateSplitHTTPRoutes creates two HTTPRoute docs in one YAML file:
// a redirect route (HTTP listener) and a backend route (HTTPS listener).
func generateSplitHTTPRoutes(ing scanner.IngressInfo, gatewayName, gatewayNamespace string, hostnameToSection map[string]string) string {
	annotations := ing.NginxAnnotations

	// Determine HTTPS listener sectionName from the ingress's primary hostname.
	httpsSectionName := ""
	if len(ing.Hosts) > 0 {
		httpsSectionName = hostnameToSection[ing.Hosts[0]]
	}

	statusCode := 301
	if annotations["ssl-redirect"] == "true" && annotations["force-ssl-redirect"] != "true" {
		statusCode = 302
	}

	// Build hostname section (shared between both routes)
	hostnameSection := buildHostnameSection(ing.Hosts)

	// ── Redirect route ────────────────────────────────────────────────────────
	// Attached to HTTP listener only via sectionName: http
	redirectParentRef := fmt.Sprintf(`  - name: %s
    namespace: %s
    sectionName: http`, gatewayName, gatewayNamespace)

	redirectRules := buildRedirectOnlyRules(ing, statusCode)

	redirectRoute := fmt.Sprintf(`# HTTP→HTTPS redirect route (attached to HTTP listener only)
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s-redirect
  namespace: %s
spec:
  parentRefs:
%s
%s  rules:
%s`, ing.Name, ing.Namespace, redirectParentRef, hostnameSection, redirectRules)

	// ── Backend route ─────────────────────────────────────────────────────────
	// Attached to HTTPS listener via sectionName: https-N (no redirect filter).
	backendRoute := generateSingleHTTPRoute(ing, gatewayName, gatewayNamespace, httpsSectionName)

	return redirectRoute + "---\n" + backendRoute
}

// generateSingleHTTPRoute creates one HTTPRoute doc with no redirect filter.
// sectionName is added to parentRef when non-empty.
func generateSingleHTTPRoute(ing scanner.IngressInfo, gatewayName, gatewayNamespace, sectionName string) string {
	annotations := ing.NginxAnnotations

	parentRef := fmt.Sprintf("  - name: %s\n    namespace: %s", gatewayName, gatewayNamespace)
	if sectionName != "" {
		parentRef += fmt.Sprintf("\n    sectionName: %s", sectionName)
	}

	hostnameSection := buildHostnameSection(ing.Hosts)
	rules := buildBackendOnlyRules(ing, annotations)

	return fmt.Sprintf(`apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: %s
  namespace: %s
spec:
  parentRefs:
%s
%s  rules:
%s`, ing.Name, ing.Namespace, parentRef, hostnameSection, rules)
}

func buildHostnameSection(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}
	var lines []string
	for _, h := range hosts {
		lines = append(lines, fmt.Sprintf("  - %q", h))
	}
	return fmt.Sprintf("  hostnames:\n%s\n", strings.Join(lines, "\n"))
}

// buildRedirectOnlyRules generates one redirect rule per path (no backendRefs).
func buildRedirectOnlyRules(ing scanner.IngressInfo, statusCode int) string {
	annotations := ing.NginxAnnotations

	hostOrder := []string{}
	hostPaths := make(map[string][]scanner.PathInfo)
	for _, p := range ing.Paths {
		if _, exists := hostPaths[p.Host]; !exists {
			hostOrder = append(hostOrder, p.Host)
		}
		hostPaths[p.Host] = append(hostPaths[p.Host], p)
	}

	var rules []string
	for _, host := range hostOrder {
		for _, p := range hostPaths[host] {
			pathMatch := buildPathMatch(p, annotations)
			headerMatches := buildHeaderMatches(annotations)

			match := fmt.Sprintf("  - matches:\n    - %s", pathMatch)
			if headerMatches != "" {
				match += headerMatches
			}
			match += "\n"

			rule := match + fmt.Sprintf(`    filters:
    - type: RequestRedirect
      requestRedirect:
        scheme: https
        statusCode: %d
`, statusCode)
			rules = append(rules, rule)
		}
	}
	return strings.Join(rules, "")
}

// buildBackendOnlyRules generates one backend rule per path (no RequestRedirect).
// URLRewrite, CORS, custom headers, backendRefs, and timeouts are included here.
func buildBackendOnlyRules(ing scanner.IngressInfo, annotations map[string]string) string {
	hostOrder := []string{}
	hostPaths := make(map[string][]scanner.PathInfo)
	for _, p := range ing.Paths {
		if _, exists := hostPaths[p.Host]; !exists {
			hostOrder = append(hostOrder, p.Host)
		}
		hostPaths[p.Host] = append(hostPaths[p.Host], p)
	}

	isCanary := annotations["canary"] == "true"
	canaryWeight := annotations["canary-weight"]

	var rules []string
	for _, host := range hostOrder {
		for _, p := range hostPaths[host] {
			pathMatch := buildPathMatch(p, annotations)
			headerMatches := buildHeaderMatches(annotations)

			match := fmt.Sprintf("  - matches:\n    - %s", pathMatch)
			if headerMatches != "" {
				match += headerMatches
			}
			match += "\n"

			filters := buildBackendFilters(annotations)
			filterSection := ""
			if len(filters) > 0 {
				filterSection = "    filters:\n" + strings.Join(filters, "")
			}

			backendSection := buildBackendRefs(p, isCanary, canaryWeight)
			timeoutSection := buildTimeouts(annotations)

			rules = append(rules, match+filterSection+backendSection+timeoutSection)
		}
	}
	return strings.Join(rules, "")
}

func buildPathMatch(path scanner.PathInfo, annotations map[string]string) string {
	pathValue := path.Path
	if pathValue == "" {
		pathValue = "/"
	}

	// Use RegularExpression when use-regex annotation is set OR when the path
	// contains characters that are invalid for PathPrefix/Exact types.
	_, useRegex := annotations["use-regex"]
	if useRegex || pathHasRegexChars(pathValue) {
		return fmt.Sprintf(`path:
        type: RegularExpression
        value: "%s"`, pathValue)
	}

	pathType := "PathPrefix"
	if path.PathType == "Exact" {
		pathType = "Exact"
	}

	return fmt.Sprintf(`path:
        type: %s
        value: "%s"`, pathType, pathValue)
}

// pathHasRegexChars returns true when the path contains characters that are
// only valid in the RegularExpression path type (e.g. from nginx use-regex paths).
func pathHasRegexChars(path string) bool {
	for _, ch := range []string{"(", ")", "|", "[", "]", "{", "}"} {
		if strings.Contains(path, ch) {
			return true
		}
	}
	return false
}

func buildHeaderMatches(annotations map[string]string) string {
	header := annotations["canary-by-header"]
	headerValue := annotations["canary-by-header-value"]
	if header == "" {
		return ""
	}
	if headerValue != "" {
		return fmt.Sprintf(`
      headers:
      - name: "%s"
        value: "%s"`, header, headerValue)
	}
	return fmt.Sprintf(`
      headers:
      - name: "%s"
        type: Present`, header)
}

// buildBackendFilters builds filters for backend rules (no RequestRedirect).
// URLRewrite is safe here since it never appears alongside RequestRedirect.
func buildBackendFilters(annotations map[string]string) []string {
	var filters []string

	// URL rewrite
	if target, ok := annotations["rewrite-target"]; ok && target != "" {
		_, useRegex := annotations["use-regex"]
		if useRegex {
			filters = append(filters, fmt.Sprintf(`    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplaceFullPath
          replaceFullPath: "%s"
# NOTE: For regex captures like $1, manual conversion to ReplacePrefixMatch may be needed
`, target))
		} else {
			filters = append(filters, fmt.Sprintf(`    - type: URLRewrite
      urlRewrite:
        path:
          type: ReplaceFullPath
          replaceFullPath: "%s"
`, target))
		}
	}

	// CORS
	if annotations["enable-cors"] == "true" {
		filters = append(filters, buildCORSFilter(annotations))
	}

	// Custom response headers
	if _, hasCustom := annotations["custom-headers"]; hasCustom {
		filters = append(filters, `    - type: ResponseHeaderModifier
      responseHeaderModifier:
        add:
          - name: "X-Custom-Header"
            value: "value"
# NOTE: Populate headers from your ConfigMap reference in nginx annotation
`)
	}

	return filters
}

func buildCORSFilter(annotations map[string]string) string {
	origin := getAnnotation(annotations, "cors-allow-origin", "*")
	methods := getAnnotation(annotations, "cors-allow-methods", "GET, PUT, POST, DELETE, PATCH, OPTIONS")
	headers := getAnnotation(annotations, "cors-allow-headers", "Content-Type, Authorization")
	credentials := getAnnotation(annotations, "cors-allow-credentials", "true")

	return fmt.Sprintf(`    - type: ResponseHeaderModifier
      responseHeaderModifier:
        add:
          - name: "Access-Control-Allow-Origin"
            value: "%s"
          - name: "Access-Control-Allow-Methods"
            value: "%s"
          - name: "Access-Control-Allow-Headers"
            value: "%s"
          - name: "Access-Control-Allow-Credentials"
            value: "%s"
`, origin, methods, headers, credentials)
}

func buildBackendRefs(path scanner.PathInfo, isCanary bool, canaryWeight string) string {
	port := path.ServicePort
	if port == 0 {
		port = 80
	}

	if isCanary && canaryWeight != "" {
		return fmt.Sprintf(`    backendRefs:
    - name: %s
      port: %d
      weight: %s
    # NOTE: Add your stable backend below with weight = (100 - %s):
    # - name: stable-service
    #   port: %d
    #   weight:
`, path.ServiceName, port, canaryWeight, canaryWeight, port)
	}

	return fmt.Sprintf(`    backendRefs:
    - name: %s
      port: %d
`, path.ServiceName, port)
}

func buildTimeouts(annotations map[string]string) string {
	readTimeout := annotations["proxy-read-timeout"]
	if readTimeout == "" {
		return ""
	}
	// proxy-read-timeout → backendRequest only.
	// proxy-connect-timeout is omitted: setting both would require backendRequest ≤ request,
	// and the typical nginx config (read=300s, connect=5s) violates that constraint.
	return fmt.Sprintf("    timeouts:\n      backendRequest: %ss\n", readTimeout)
}

func getAnnotation(annotations map[string]string, key, defaultVal string) string {
	if v, ok := annotations[key]; ok && v != "" {
		return v
	}
	return defaultVal
}
