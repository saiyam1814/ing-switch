package scanner

import (
	"sort"
	"strings"
)

const (
	haproxyAnnotationPrefix    = "haproxy-ingress.github.io/"
	haproxyOrgAnnotationPrefix = "haproxy.org/"
)

// ScanHAProxyIngresses finds standard Ingress resources that use HAProxy's
// ingress class and converts HAProxy-specific annotations to pseudo-annotations.
func (s *Scanner) ScanHAProxyIngresses(namespace string) ([]IngressInfo, error) {
	allIngresses, err := s.listIngresses(namespace)
	if err != nil {
		return nil, err
	}

	var haproxyIngresses []IngressInfo
	for _, ing := range allIngresses {
		if isHAProxyIngress(ing) {
			haproxyIngresses = append(haproxyIngresses, ing)
		}
	}

	if len(haproxyIngresses) == 0 {
		return nil, nil
	}

	var result []IngressInfo
	for _, ing := range haproxyIngresses {
		processed := processHAProxyIngress(ing)
		result = append(result, processed)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Namespace != result[j].Namespace {
			return result[i].Namespace < result[j].Namespace
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// isHAProxyIngress determines if an Ingress resource belongs to HAProxy.
func isHAProxyIngress(ing IngressInfo) bool {
	if ing.IngressClass == "haproxy" {
		return true
	}
	for k := range ing.Annotations {
		if strings.HasPrefix(k, haproxyAnnotationPrefix) || strings.HasPrefix(k, haproxyOrgAnnotationPrefix) {
			return true
		}
	}
	return false
}

// processHAProxyIngress converts an HAProxy Ingress into an IngressInfo with pseudo-annotations.
func processHAProxyIngress(ing IngressInfo) IngressInfo {
	ing.SourceType = SourceHAProxyIngress
	ing.NginxAnnotations = make(map[string]string)

	for k, v := range ing.Annotations {
		if strings.HasPrefix(k, haproxyAnnotationPrefix) {
			shortKey := strings.TrimPrefix(k, haproxyAnnotationPrefix)
			mapHAProxyAnnotation(&ing, shortKey, v)
		}
		if strings.HasPrefix(k, haproxyOrgAnnotationPrefix) {
			shortKey := strings.TrimPrefix(k, haproxyOrgAnnotationPrefix)
			mapHAProxyAnnotation(&ing, shortKey, v)
		}
	}

	ing.Complexity = classifyHAProxyComplexity(&ing)
	return ing
}

// mapHAProxyAnnotation translates a single HAProxy annotation to a pseudo-annotation.
func mapHAProxyAnnotation(info *IngressInfo, key, value string) {
	switch key {
	// --- SSL / TLS ---
	case "ssl-redirect":
		info.NginxAnnotations["ssl-redirect"] = value
	case "ssl-passthrough":
		if value == "true" {
			info.NginxAnnotations["ssl-passthrough"] = "true"
		}

	// --- Rewrites ---
	case "rewrite-target":
		info.NginxAnnotations["rewrite-target"] = value
	case "app-root":
		info.NginxAnnotations["app-root"] = value

	// --- Auth ---
	case "auth-type":
		info.NginxAnnotations["auth-type"] = value
	case "auth-secret":
		info.NginxAnnotations["auth-secret"] = value
	case "auth-realm":
		info.NginxAnnotations["auth-realm"] = value
	case "auth-url":
		info.NginxAnnotations["auth-url"] = value

	// --- CORS ---
	case "cors-enable":
		if value == "true" {
			info.NginxAnnotations["enable-cors"] = "true"
		}
	case "cors-allow-origin":
		info.NginxAnnotations["cors-allow-origin"] = value
	case "cors-allow-methods":
		info.NginxAnnotations["cors-allow-methods"] = value
	case "cors-allow-headers":
		info.NginxAnnotations["cors-allow-headers"] = value
	case "cors-allow-credentials":
		info.NginxAnnotations["cors-allow-credentials"] = value
	case "cors-max-age":
		info.NginxAnnotations["cors-max-age"] = value
	case "cors-expose-headers":
		info.NginxAnnotations["cors-expose-headers"] = value

	// --- Rate Limiting ---
	case "limit-rps":
		info.NginxAnnotations["limit-rps"] = value
	case "limit-connections":
		info.NginxAnnotations["limit-connections"] = value

	// --- IP Allow / Deny ---
	case "allowlist-source-range", "whitelist-source-range":
		info.NginxAnnotations["whitelist-source-range"] = value
	case "denylist-source-range":
		info.NginxAnnotations["denylist-source-range"] = value

	// --- Timeouts ---
	case "timeout-connect":
		info.NginxAnnotations["proxy-connect-timeout"] = value
	case "timeout-server":
		info.NginxAnnotations["proxy-read-timeout"] = value
	case "timeout-client":
		info.NginxAnnotations["proxy-send-timeout"] = value
	case "timeout-tunnel":
		info.NginxAnnotations["proxy-read-timeout"] = value

	// --- Session Affinity ---
	case "affinity":
		info.NginxAnnotations["affinity"] = value
	case "session-cookie-name":
		info.NginxAnnotations["session-cookie-name"] = value
	case "cookie-key":
		info.NginxAnnotations["session-cookie-name"] = value
	case "session-cookie-strategy":
		// HAProxy uses "insert"/"prefix"/"rewrite" — map to affinity mode
		info.NginxAnnotations["affinity-mode"] = value

	// --- Backend Protocol ---
	case "backend-protocol":
		info.NginxAnnotations["backend-protocol"] = value
	case "secure-backends":
		if value == "true" {
			info.NginxAnnotations["backend-protocol"] = "HTTPS"
		}

	// --- Proxy / Buffering ---
	case "proxy-body-size":
		info.NginxAnnotations["proxy-body-size"] = value
	case "maxconn-server":
		info.NginxAnnotations["limit-connections"] = value

	// --- Headers ---
	case "headers":
		info.NginxAnnotations["custom-headers"] = value
	case "response-set-header", "set-header":
		info.NginxAnnotations["custom-headers"] = value

	// --- WAF / Modsecurity ---
	case "waf":
		if value == "modsecurity" {
			info.NginxAnnotations["enable-modsecurity"] = "true"
		}

	// --- Load balancing ---
	case "balance-algorithm":
		info.NginxAnnotations["load-balance"] = value
	case "load-balance":
		info.NginxAnnotations["load-balance"] = value

	// --- Redirect ---
	case "redirect", "http-redirect":
		info.NginxAnnotations["permanent-redirect"] = value
	case "redirect-code":
		if value == "301" {
			info.NginxAnnotations["permanent-redirect"] = "true"
		} else {
			info.NginxAnnotations["temporal-redirect"] = "true"
		}

	// --- gRPC ---
	case "grpc-backend":
		info.NginxAnnotations["grpc-backend"] = "true"
		info.NginxAnnotations["backend-protocol"] = "GRPC"

	// --- Configuration snippets (non-portable) ---
	case "config-backend", "config-frontend", "config-global":
		info.NginxAnnotations["configuration-snippet"] = value
	}
}

// classifyHAProxyComplexity assigns complexity for HAProxy Ingresses.
func classifyHAProxyComplexity(info *IngressInfo) string {
	if len(info.NginxAnnotations) == 0 {
		return "simple"
	}

	unsupportedKeys := map[string]bool{
		"configuration-snippet": true,
		"enable-modsecurity":    true,
	}

	complexKeys := map[string]bool{
		"auth-type": true, "auth-url": true,
		"limit-rps": true, "limit-connections": true,
		"whitelist-source-range": true, "denylist-source-range": true,
		"enable-cors": true, "rewrite-target": true,
		"ssl-passthrough": true, "grpc-backend": true,
		"affinity": true, "backend-protocol": true,
	}

	for k := range info.NginxAnnotations {
		if unsupportedKeys[k] {
			return "unsupported"
		}
	}

	for k := range info.NginxAnnotations {
		if complexKeys[k] {
			return "complex"
		}
	}

	return "simple"
}
