package traefik

import (
	"fmt"
	"strings"

	"github.com/saiyam1814/ing-switch/pkg/scanner"
)

// MiddlewareSpec describes a Traefik Middleware to generate.
type MiddlewareSpec struct {
	Name      string
	Namespace string
	YAML      string
}

// generateMiddlewares produces all necessary Traefik Middleware CRDs for an Ingress.
func generateMiddlewares(ing scanner.IngressInfo) []MiddlewareSpec {
	var middlewares []MiddlewareSpec
	annotations := ing.NginxAnnotations

	// SSL Redirect
	if v, ok := annotations["ssl-redirect"]; ok && v == "true" {
		mw := generateSSLRedirect(ing.Name, ing.Namespace, false)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}
	if v, ok := annotations["force-ssl-redirect"]; ok && v == "true" {
		mw := generateSSLRedirect(ing.Name, ing.Namespace, true)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// CORS
	if v, ok := annotations["enable-cors"]; ok && v == "true" {
		mw := generateCORSMiddleware(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// ForwardAuth
	if authURL, ok := annotations["auth-url"]; ok && authURL != "" {
		mw := generateForwardAuth(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// BasicAuth
	if authType, ok := annotations["auth-type"]; ok && authType == "basic" {
		mw := generateBasicAuth(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// RateLimit
	if _, hasRPS := annotations["limit-rps"]; hasRPS {
		mw := generateRateLimit(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	} else if _, hasRPM := annotations["limit-rpm"]; hasRPM {
		mw := generateRateLimit(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// InFlightReq (connection limit)
	if _, hasConn := annotations["limit-connections"]; hasConn {
		mw := generateInFlightReq(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// IPAllowList
	if cidr, ok := annotations["whitelist-source-range"]; ok && cidr != "" {
		mw := generateIPAllowList(ing.Name, ing.Namespace, cidr)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// IPDenyList
	if cidr, ok := annotations["denylist-source-range"]; ok && cidr != "" {
		mw := generateIPDenyList(ing.Name, ing.Namespace, cidr)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// ReplacePath / URL rewrite
	if target, ok := annotations["rewrite-target"]; ok && target != "" {
		mw := generateRewriteMiddleware(ing.Name, ing.Namespace, target, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	// Custom headers
	if _, hasCustom := annotations["custom-headers"]; hasCustom {
		mw := generateHeadersMiddleware(ing.Name, ing.Namespace, annotations)
		if mw != nil {
			middlewares = append(middlewares, *mw)
		}
	}

	return middlewares
}

func generateSSLRedirect(ingName, ns string, permanent bool) *MiddlewareSpec {
	name := ingName + "-ssl-redirect"
	permanentStr := "false"
	if permanent {
		permanentStr = "true"
		name = ingName + "-force-ssl-redirect"
	}
	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  redirectScheme:
    scheme: https
    permanent: %s
`, name, ns, permanentStr),
	}
}

func generateCORSMiddleware(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-cors"
	origin := getAnnotation(annotations, "cors-allow-origin", "*")
	methods := getAnnotation(annotations, "cors-allow-methods", "GET, PUT, POST, DELETE, PATCH, OPTIONS")
	headers := getAnnotation(annotations, "cors-allow-headers", "DNT,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,Authorization")
	credentials := getAnnotation(annotations, "cors-allow-credentials", "true")
	maxAge := getAnnotation(annotations, "cors-max-age", "1728000")
	exposeHeaders := getAnnotation(annotations, "cors-expose-headers", "")

	exposeSection := ""
	if exposeHeaders != "" {
		exposeSection = fmt.Sprintf("\n    exposedHeaders:\n      - \"%s\"", exposeHeaders)
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  headers:
    accessControlAllowOriginList:
      - "%s"
    accessControlAllowMethods:
      - "%s"
    accessControlAllowHeaders:
      - "%s"
    accessControlAllowCredentials: %s
    accessControlMaxAge: %s%s
`, name, ns, origin, methods, headers, credentials, maxAge, exposeSection),
	}
}

func generateForwardAuth(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-auth"
	authURL := annotations["auth-url"]

	responseHeaders := ""
	if rh, ok := annotations["auth-response-headers"]; ok && rh != "" {
		headers := strings.Split(rh, ",")
		var lines []string
		for _, h := range headers {
			lines = append(lines, fmt.Sprintf("      - \"%s\"", strings.TrimSpace(h)))
		}
		responseHeaders = fmt.Sprintf("\n    authResponseHeaders:\n%s", strings.Join(lines, "\n"))
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  forwardAuth:
    address: "%s"
    trustForwardHeader: true%s
`, name, ns, authURL, responseHeaders),
	}
}

func generateBasicAuth(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-basicauth"
	secret := getAnnotation(annotations, "auth-secret", ingName+"-basic-auth")
	realm := getAnnotation(annotations, "auth-realm", "traefik")

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  basicAuth:
    secret: %s
    realm: "%s"
# NOTE: Create the secret with htpasswd-encoded credentials:
# kubectl create secret generic %s --from-literal=users="$(htpasswd -nb user password)" -n %s
`, name, ns, secret, realm, secret, ns),
	}
}

func generateRateLimit(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-ratelimit"

	average := "100"
	if rps, ok := annotations["limit-rps"]; ok && rps != "" {
		average = rps
	} else if rpm, ok := annotations["limit-rpm"]; ok && rpm != "" {
		// Convert RPM to average per second (approximate)
		average = fmt.Sprintf("%s # converted from %s RPM", rpm, rpm)
	}

	burst := "200"
	if bm, ok := annotations["limit-burst-multiplier"]; ok && bm != "" {
		burst = fmt.Sprintf("%s # burst-multiplier applied", bm)
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  rateLimit:
    average: %s
    burst: %s
    period: 1s
`, name, ns, average, burst),
	}
}

func generateInFlightReq(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-inflightreq"
	amount := getAnnotation(annotations, "limit-connections", "10")

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  inFlightReq:
    amount: %s
`, name, ns, amount),
	}
}

func generateIPAllowList(ingName, ns, cidr string) *MiddlewareSpec {
	name := ingName + "-ipallowlist"
	cidrs := strings.Split(cidr, ",")
	var cidrLines []string
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c != "" {
			cidrLines = append(cidrLines, fmt.Sprintf("      - \"%s\"", c))
		}
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  ipAllowList:
    sourceRange:
%s
`, name, ns, strings.Join(cidrLines, "\n")),
	}
}

func generateIPDenyList(ingName, ns, cidr string) *MiddlewareSpec {
	name := ingName + "-ipdenylist"
	cidrs := strings.Split(cidr, ",")
	var cidrLines []string
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c != "" {
			cidrLines = append(cidrLines, fmt.Sprintf("      - \"%s\"", c))
		}
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  ipDenyList:
    sourceRange:
%s
`, name, ns, strings.Join(cidrLines, "\n")),
	}
}

func generateRewriteMiddleware(ingName, ns, target string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-rewrite"

	// If regex is used, use ReplacePathRegex
	if _, useRegex := annotations["use-regex"]; useRegex {
		// Find a source pattern — for now use a general pattern
		return &MiddlewareSpec{
			Name:      name,
			Namespace: ns,
			YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  replacePathRegex:
    regex: "^/[^/]*(.*)"
    replacement: "%s"
# NOTE: Adjust regex to match your path pattern from the Ingress spec
`, name, ns, target),
		}
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  replacePath:
    path: "%s"
`, name, ns, target),
	}
}

func generateHeadersMiddleware(ingName, ns string, annotations map[string]string) *MiddlewareSpec {
	name := ingName + "-headers"
	customHeaders := getAnnotation(annotations, "custom-headers", "")

	note := ""
	if customHeaders != "" {
		note = fmt.Sprintf(`
# NOTE: Original annotation referenced ConfigMap: %s
# Inline the headers below from that ConfigMap`, customHeaders)
	}

	return &MiddlewareSpec{
		Name:      name,
		Namespace: ns,
		YAML: fmt.Sprintf(`apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: %s
  namespace: %s
spec:
  headers:
    customResponseHeaders:
      X-Custom-Header: "value"  # Replace with your actual headers%s
`, name, ns, note),
	}
}

func getAnnotation(annotations map[string]string, key, defaultVal string) string {
	if v, ok := annotations[key]; ok && v != "" {
		return v
	}
	return defaultVal
}
