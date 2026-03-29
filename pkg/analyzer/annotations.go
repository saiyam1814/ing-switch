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
	{Key: "auth-tls-verify-depth", Category: "tls", Description: "Client certificate chain verification depth"},
	{Key: "auth-tls-error-page", Category: "tls", Description: "Redirect URL on client cert auth failure"},
	{Key: "auth-tls-pass-certificate-to-upstream", Category: "tls", Description: "Pass client certificate to upstream in header"},
	{Key: "auth-tls-match-cn", Category: "tls", Description: "Validate client cert Common Name against string/regex"},

	// Authentication
	{Key: "auth-url", Category: "auth", Description: "External authentication service URL"},
	{Key: "auth-method", Category: "auth", Description: "HTTP method for auth request"},
	{Key: "auth-response-headers", Category: "auth", Description: "Headers to pass from auth response"},
	{Key: "auth-request-redirect", Category: "auth", Description: "Redirect for auth failures"},
	{Key: "auth-type", Category: "auth", Description: "Authentication type (basic/digest)"},
	{Key: "auth-secret", Category: "auth", Description: "Secret containing credentials"},
	{Key: "auth-secret-type", Category: "auth", Description: "Format of auth secret (auth-file or auth-map)"},
	{Key: "auth-realm", Category: "auth", Description: "Authentication realm"},
	{Key: "auth-cache-key", Category: "auth", Description: "Cache key for external auth responses"},
	{Key: "auth-cache-duration", Category: "auth", Description: "Duration to cache external auth responses"},
	{Key: "auth-keepalive", Category: "auth", Description: "Max keepalive connections to external auth"},
	{Key: "auth-keepalive-share-vars", Category: "auth", Description: "Share NGINX vars between main and auth subrequest"},
	{Key: "auth-keepalive-requests", Category: "auth", Description: "Max requests per keepalive to auth service"},
	{Key: "auth-keepalive-timeout", Category: "auth", Description: "Idle timeout for keepalive to auth service"},
	{Key: "auth-proxy-set-headers", Category: "auth", Description: "ConfigMap of extra headers to send to auth service"},
	{Key: "auth-snippet", Category: "auth", Description: "Custom NGINX snippet for auth location block"},
	{Key: "auth-always-set-cookie", Category: "auth", Description: "Always set cookies from auth service, even on deny"},
	{Key: "auth-signin", Category: "auth", Description: "URL to redirect to on 401 from auth service"},
	{Key: "auth-signin-redirect-param", Category: "auth", Description: "URL param name for signin redirect"},
	{Key: "enable-global-auth", Category: "auth", Description: "Enable/disable global external auth for this ingress"},

	// Routing
	{Key: "rewrite-target", Category: "routing", Description: "Rewrite the request URL"},
	{Key: "use-regex", Category: "routing", Description: "Enable regex path matching"},
	{Key: "app-root", Category: "routing", Description: "Redirect root path requests"},
	{Key: "permanent-redirect", Category: "routing", Description: "Permanent redirect URL"},
	{Key: "permanent-redirect-code", Category: "routing", Description: "Custom status code for permanent redirect (default 301)"},
	{Key: "temporal-redirect", Category: "routing", Description: "Temporary redirect URL"},
	{Key: "temporal-redirect-code", Category: "routing", Description: "Custom status code for temporal redirect (default 302)"},
	{Key: "preserve-trailing-slash", Category: "routing", Description: "Preserve trailing slash on SSL redirect"},
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
	{Key: "session-cookie-domain", Category: "affinity", Description: "Domain attribute for session cookie"},
	{Key: "affinity-canary-behavior", Category: "affinity", Description: "Canary behavior with session affinity (sticky/legacy)"},

	// Rate Limiting
	{Key: "limit-rps", Category: "ratelimit", Description: "Rate limit: requests per second"},
	{Key: "limit-rpm", Category: "ratelimit", Description: "Rate limit: requests per minute"},
	{Key: "limit-connections", Category: "ratelimit", Description: "Max concurrent connections"},
	{Key: "limit-burst-multiplier", Category: "ratelimit", Description: "Burst multiplier for rate limit"},
	{Key: "limit-whitelist", Category: "ratelimit", Description: "IPs exempt from rate limiting"},
	{Key: "limit-rate", Category: "ratelimit", Description: "Response rate limit in KB/s"},
	{Key: "limit-rate-after", Category: "ratelimit", Description: "KB threshold before rate limiting applies"},

	// Proxy / Timeouts
	{Key: "proxy-body-size", Category: "proxy", Description: "Max request body size (UNSUPPORTED in Traefik)"},
	{Key: "proxy-read-timeout", Category: "proxy", Description: "Backend read timeout"},
	{Key: "proxy-send-timeout", Category: "proxy", Description: "Backend send timeout"},
	{Key: "proxy-connect-timeout", Category: "proxy", Description: "Backend connection timeout"},
	{Key: "proxy-buffering", Category: "proxy", Description: "Enable/disable proxy buffering"},
	{Key: "proxy-buffer-size", Category: "proxy", Description: "Proxy buffer size"},
	{Key: "proxy-next-upstream", Category: "proxy", Description: "Next upstream retry conditions"},
	{Key: "proxy-next-upstream-timeout", Category: "proxy", Description: "Timeout for trying next upstream (0=disabled)"},
	{Key: "proxy-next-upstream-tries", Category: "proxy", Description: "Max retries against upstream servers"},
	{Key: "proxy-http-version", Category: "proxy", Description: "HTTP version for backend"},
	{Key: "proxy-ssl-secret", Category: "proxy", Description: "TLS secret for backend connections"},
	{Key: "proxy-ssl-ciphers", Category: "proxy", Description: "SSL ciphers for backend connections"},
	{Key: "proxy-ssl-name", Category: "proxy", Description: "Server name for backend SSL verification"},
	{Key: "proxy-ssl-protocols", Category: "proxy", Description: "SSL/TLS protocols for backend connections"},
	{Key: "proxy-ssl-verify", Category: "proxy", Description: "Enable backend TLS certificate verification"},
	{Key: "proxy-ssl-verify-depth", Category: "proxy", Description: "Backend certificate chain verification depth"},
	{Key: "proxy-ssl-server-name", Category: "proxy", Description: "Send SNI server name to backend via TLS"},
	{Key: "proxy-cookie-domain", Category: "proxy", Description: "Rewrite Domain in upstream Set-Cookie headers"},
	{Key: "proxy-cookie-path", Category: "proxy", Description: "Rewrite Path in upstream Set-Cookie headers"},
	{Key: "proxy-redirect-from", Category: "proxy", Description: "Source for proxy_redirect (rewrite Location headers)"},
	{Key: "proxy-redirect-to", Category: "proxy", Description: "Target for proxy_redirect (rewrite Location headers)"},
	{Key: "proxy-buffers-number", Category: "proxy", Description: "Number of buffers for reading upstream response"},
	{Key: "proxy-busy-buffers-size", Category: "proxy", Description: "Max size of busy buffers for upstream response"},
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
	{Key: "canary-by-header-pattern", Category: "canary", Description: "Regex pattern for canary header value match"},

	// Custom Error Handling
	{Key: "custom-http-errors", Category: "errors", Description: "HTTP error codes to intercept via default backend"},
	{Key: "default-backend", Category: "errors", Description: "Custom default backend service (namespace/name)"},

	// Miscellaneous
	{Key: "enable-access-log", Category: "misc", Description: "Enable/disable access logging for this ingress"},
	{Key: "enable-opentelemetry", Category: "misc", Description: "Enable OpenTelemetry tracing for this ingress"},
	{Key: "mirror-target", Category: "misc", Description: "Mirror request traffic to another service"},
	{Key: "mirror-request-body", Category: "misc", Description: "Include request body in mirrored requests"},
	{Key: "server-alias", Category: "misc", Description: "Additional hostname alias for the server"},
	{Key: "satisfy", Category: "misc", Description: "Auth satisfaction logic (any/all)"},
	{Key: "enable-modsecurity", Category: "misc", Description: "Enable ModSecurity WAF for this ingress"},
	{Key: "modsecurity-snippet", Category: "misc", Description: "Custom ModSecurity configuration rules"},
	{Key: "modsecurity-transaction-id", Category: "misc", Description: "Custom transaction ID for ModSecurity"},
	{Key: "x-forwarded-prefix", Category: "misc", Description: "Override X-Forwarded-Prefix header value"},
	{Key: "connection-proxy-header", Category: "misc", Description: "Custom Connection header for proxy"},
	{Key: "enable-owasp-modsecurity-crs", Category: "misc", Description: "Enable OWASP Core Rule Set for ModSecurity"},

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
