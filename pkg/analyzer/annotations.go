package analyzer

// AnnotationDef describes a known nginx.ingress.kubernetes.io annotation.
type AnnotationDef struct {
	Key         string
	Category    string // "tls", "auth", "routing", "traffic", "cors", "headers", "canary"
	Description string
}

// KnownAnnotations is the master list of supported nginx.ingress.kubernetes.io annotations.
var KnownAnnotations = []AnnotationDef{
	// TLS / Redirect
	{Key: "ssl-redirect", Category: "tls", Description: "Redirect HTTP to HTTPS"},
	{Key: "force-ssl-redirect", Category: "tls", Description: "Force HTTPS redirect (ignore x-forwarded-proto)"},
	{Key: "ssl-passthrough", Category: "tls", Description: "Pass SSL directly to backend"},
	{Key: "ssl-ciphers", Category: "tls", Description: "Custom SSL cipher list"},
	{Key: "auth-tls-secret", Category: "tls", Description: "Client certificate authentication"},
	{Key: "auth-tls-verify-client", Category: "tls", Description: "Client certificate verification mode"},

	// Authentication
	{Key: "auth-url", Category: "auth", Description: "External authentication service URL"},
	{Key: "auth-method", Category: "auth", Description: "HTTP method for auth request"},
	{Key: "auth-response-headers", Category: "auth", Description: "Headers to pass from auth response"},
	{Key: "auth-request-redirect", Category: "auth", Description: "Redirect for auth failures"},
	{Key: "auth-type", Category: "auth", Description: "Authentication type (basic/digest)"},
	{Key: "auth-secret", Category: "auth", Description: "Secret containing credentials"},
	{Key: "auth-realm", Category: "auth", Description: "Authentication realm"},

	// Routing
	{Key: "rewrite-target", Category: "routing", Description: "Rewrite the request URL"},
	{Key: "use-regex", Category: "routing", Description: "Enable regex path matching"},
	{Key: "app-root", Category: "routing", Description: "Redirect root path requests"},
	{Key: "permanent-redirect", Category: "routing", Description: "Permanent redirect URL"},
	{Key: "temporal-redirect", Category: "routing", Description: "Temporary redirect URL"},
	{Key: "server-snippet", Category: "routing", Description: "NGINX server block snippet (UNSUPPORTED)"},
	{Key: "configuration-snippet", Category: "routing", Description: "NGINX configuration snippet (UNSUPPORTED)"},

	// Session / Affinity
	{Key: "affinity", Category: "affinity", Description: "Session affinity type (cookie)"},
	{Key: "affinity-mode", Category: "affinity", Description: "Affinity mode (balanced/persistent)"},
	{Key: "session-cookie-name", Category: "affinity", Description: "Session cookie name"},
	{Key: "session-cookie-path", Category: "affinity", Description: "Session cookie path"},
	{Key: "session-cookie-samesite", Category: "affinity", Description: "SameSite cookie attribute"},
	{Key: "session-cookie-conditional-samesite-none", Category: "affinity", Description: "Conditional SameSite None"},
	{Key: "session-cookie-expires", Category: "affinity", Description: "Session cookie expiry"},
	{Key: "session-cookie-max-age", Category: "affinity", Description: "Session cookie max age"},
	{Key: "session-cookie-secure", Category: "affinity", Description: "Secure flag for session cookie"},

	// Rate Limiting
	{Key: "limit-rps", Category: "ratelimit", Description: "Rate limit: requests per second"},
	{Key: "limit-rpm", Category: "ratelimit", Description: "Rate limit: requests per minute"},
	{Key: "limit-connections", Category: "ratelimit", Description: "Max concurrent connections"},
	{Key: "limit-burst-multiplier", Category: "ratelimit", Description: "Burst multiplier for rate limit"},
	{Key: "limit-whitelist", Category: "ratelimit", Description: "IPs exempt from rate limiting"},

	// Proxy / Timeouts
	{Key: "proxy-body-size", Category: "proxy", Description: "Max request body size (UNSUPPORTED in Traefik)"},
	{Key: "proxy-read-timeout", Category: "proxy", Description: "Backend read timeout"},
	{Key: "proxy-send-timeout", Category: "proxy", Description: "Backend send timeout"},
	{Key: "proxy-connect-timeout", Category: "proxy", Description: "Backend connection timeout"},
	{Key: "proxy-buffering", Category: "proxy", Description: "Enable/disable proxy buffering"},
	{Key: "proxy-buffer-size", Category: "proxy", Description: "Proxy buffer size"},
	{Key: "proxy-next-upstream", Category: "proxy", Description: "Next upstream retry conditions"},
	{Key: "proxy-http-version", Category: "proxy", Description: "HTTP version for backend"},
	{Key: "proxy-ssl-secret", Category: "proxy", Description: "TLS secret for backend connections"},
	{Key: "client-body-buffer-size", Category: "proxy", Description: "Client body buffer size (UNSUPPORTED)"},

	// CORS
	{Key: "enable-cors", Category: "cors", Description: "Enable CORS"},
	{Key: "cors-allow-origin", Category: "cors", Description: "Allowed origins"},
	{Key: "cors-allow-methods", Category: "cors", Description: "Allowed HTTP methods"},
	{Key: "cors-allow-headers", Category: "cors", Description: "Allowed request headers"},
	{Key: "cors-expose-headers", Category: "cors", Description: "Exposed response headers"},
	{Key: "cors-allow-credentials", Category: "cors", Description: "Allow credentials"},
	{Key: "cors-max-age", Category: "cors", Description: "Preflight cache duration"},

	// Headers
	{Key: "configuration-snippet", Category: "headers", Description: "Custom NGINX config (UNSUPPORTED)"},
	{Key: "custom-headers", Category: "headers", Description: "Custom response headers from ConfigMap"},
	{Key: "whitelist-source-range", Category: "access", Description: "Allowed IP/CIDR ranges"},
	{Key: "denylist-source-range", Category: "access", Description: "Blocked IP/CIDR ranges"},

	// Canary
	{Key: "canary", Category: "canary", Description: "Enable canary deployment"},
	{Key: "canary-weight", Category: "canary", Description: "Canary traffic weight (0-100)"},
	{Key: "canary-weight-total", Category: "canary", Description: "Total weight denominator"},
	{Key: "canary-by-header", Category: "canary", Description: "Header-based canary routing"},
	{Key: "canary-by-header-value", Category: "canary", Description: "Header value for canary"},
	{Key: "canary-by-cookie", Category: "canary", Description: "Cookie-based canary routing"},

	// WebSocket / gRPC
	{Key: "websocket-services", Category: "protocol", Description: "Services that use WebSocket"},
	{Key: "grpc-backend", Category: "protocol", Description: "Enable gRPC passthrough"},
	{Key: "backend-protocol", Category: "protocol", Description: "Backend protocol (GRPC/GRPCS/HTTPS/AJP)"},
	{Key: "upstream-hash-by", Category: "lb", Description: "Hash-based load balancing key"},
	{Key: "load-balance", Category: "lb", Description: "Load balancing algorithm"},
}

// AnnotationsByKey provides O(1) lookup.
var AnnotationsByKey map[string]AnnotationDef

func init() {
	AnnotationsByKey = make(map[string]AnnotationDef, len(KnownAnnotations))
	for _, a := range KnownAnnotations {
		AnnotationsByKey[a.Key] = a
	}
}
