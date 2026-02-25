package server

// AnnotationGuide holds actionable fix information for a single annotation mapping issue.
type AnnotationGuide struct {
	What        string // what the annotation does
	Fix         string // specific steps to configure equivalent in target
	Example     string // YAML/command example (optional)
	DocsLink    string // upstream docs link (optional)
	Consequence string // what happens if this is not migrated (optional)
	IssueUrl    string // upstream GitHub issue URL for unsupported features (optional)
}

// traefikGuides maps annotation key → actionable fix guide for Traefik target.
var traefikGuides = map[string]AnnotationGuide{
	// --- PARTIAL (workaround needed) ---
	"affinity-mode": {
		What:        "Controls how sticky sessions are re-balanced when pod replicas change. 'balanced' re-assigns cookies when pods scale up/down; 'persistent' keeps the same cookie mapped to the same backend indefinitely.",
		Fix:         "Traefik sticky sessions always use persistent mode — the cookie always maps to the same backend. Balanced mode (re-assignment on scaling) is not available.\nTo reduce impact when pods restart, enable Traefik health checks so unhealthy pods are removed quickly.",
		Example:     "# Health check on Service (minimizes impact on pod restarts):\nannotations:\n  traefik.ingress.kubernetes.io/service.healthcheck.path: /healthz\n  traefik.ingress.kubernetes.io/service.healthcheck.interval: 10s\n  traefik.ingress.kubernetes.io/service.healthcheck.timeout: 3s",
		DocsLink:    "https://doc.traefik.io/traefik/routing/services/#sticky-sessions",
		Consequence: "When pods scale up, Traefik won't re-balance existing sticky sessions to new pods. Load may be uneven until existing sessions expire or pods restart.",
	},
	"proxy-http-version": {
		What:    "Forces the proxy to use a specific HTTP version when communicating with the backend (e.g., 1.1 for WebSocket, 2.0 for gRPC/h2c).",
		Fix:     "For HTTP/2 (h2c) backends (gRPC): configure ServersTransport and set the service scheme to h2c.\nFor HTTP/1.1 (WebSocket): Traefik supports WebSocket natively — no config needed.\nHTTP/1.0 backends are not supported.",
		Example: "# For h2c (HTTP/2 cleartext) backend — gRPC:\n# Service annotation:\ntraefik.ingress.kubernetes.io/service.serversscheme: h2c\n\n# For WebSocket (HTTP/1.1 upgrade): no config needed",
		DocsLink: "https://doc.traefik.io/traefik/routing/services/#servers-transport",
	},
	"session-cookie-path": {
		What:    "Restricts session affinity cookie to a specific path.",
		Fix:     "Traefik sticky cookies support limited path config. Set the cookie path via the service annotation traefik.ingress.kubernetes.io/affinity-cookie-path on your Service resource.",
		Example: "# On the Service:\nannotations:\n  traefik.ingress.kubernetes.io/affinity-cookie-path: \"/app\"",
	},
	"auth-method": {
		What:    "Sets the HTTP method ForwardAuth should use when calling the auth URL.",
		Fix:     "Traefik ForwardAuth always uses GET. If your auth server requires POST, add a proxy adapter in front of it, or switch to a GET-compatible auth endpoint.",
		Example: "# ForwardAuth always calls auth-url with GET — no config needed",
	},
	"auth-request-redirect": {
		What:    "Sets the redirect URL passed to the auth service on failure.",
		Fix:     "Add redirectUntrusted: true and configure the redirect in your auth service instead. The ForwardAuth middleware passes the original URL via X-Forwarded-Uri header.",
		Example: "apiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: auth-mw\nspec:\n  forwardAuth:\n    address: https://auth.example.com/verify\n    redirectUntrusted: true\n    trustForwardHeader: true",
	},
	"auth-type": {
		What:    "Enables HTTP Basic authentication using an htpasswd secret.",
		Fix:     "Create a Kubernetes secret with htpasswd users, then use BasicAuth Middleware. The secret format differs from NGINX — values must be bcrypt hashed.",
		Example: "# 1. Create secret:\nkubectl create secret generic basic-auth \\\n  --from-literal=users=$(htpasswd -nb user 'mypassword') \\\n  -n <namespace>\n\n# 2. Create Middleware:\napiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: basic-auth-mw\nspec:\n  basicAuth:\n    secret: basic-auth",
		DocsLink: "https://doc.traefik.io/traefik/middlewares/http/basicauth/",
	},
	"auth-secret": {
		What:    "References the Kubernetes secret containing htpasswd credentials.",
		Fix:     "Traefik BasicAuth Middleware references a secret directly but expects bcrypt-hashed values (not MD5). Regenerate the secret with: htpasswd -nB user password",
		Example: "# Regenerate with bcrypt:\nhtpasswd -nB admin mypassword\n# Output: admin:$2y$...\nkubectl create secret generic basic-auth --from-literal=users='admin:$2y$...' -n <ns>",
	},
	"custom-headers": {
		What:    "Adds custom request/response headers via a ConfigMap reference.",
		Fix:     "Traefik Headers Middleware doesn't support ConfigMap refs. Copy the key-value pairs inline into the Headers Middleware YAML. This is already done in the generated middleware file — verify the values are correct.",
		Example: "apiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: custom-headers-mw\nspec:\n  headers:\n    customRequestHeaders:\n      X-App-Version: \"v2\"\n      X-Custom-Header: \"value\"",
	},
	"canary-by-header": {
		What:    "Routes a percentage of traffic to canary based on a request header presence.",
		Fix:     "Add a router rule matching the header. The generated Ingress uses traefik.ingress.kubernetes.io/router.rule to match the header. Verify the rule syntax.",
		Example: "# Traefik router rule for header-based routing:\nannotations:\n  traefik.ingress.kubernetes.io/router.rule: \"PathPrefix(`/`) && Headers(`X-Canary`, `always`)\"\n  traefik.ingress.kubernetes.io/router.priority: \"10\"",
	},
	"canary-by-header-value": {
		What:    "Routes traffic to canary when the header matches a specific value.",
		Fix:     "Update the Traefik router rule to match the exact header value.",
		Example: "annotations:\n  traefik.ingress.kubernetes.io/router.rule: \"PathPrefix(`/`) && Headers(`X-Version`, `v2`)\"",
	},
	"canary-by-cookie": {
		What:    "Routes traffic to canary based on cookie presence.",
		Fix:     "Add a Traefik router rule matching the cookie. Cookie matching uses HeadersRegexp in Traefik router rules.",
		Example: "annotations:\n  traefik.ingress.kubernetes.io/router.rule: \"PathPrefix(`/`) && HeadersRegexp(`Cookie`, `canary=always`)\"",
	},
	"proxy-read-timeout": {
		What:    "Sets the timeout for reading the response from the backend.",
		Fix:     "Create a ServersTransport CRD and reference it from the Service annotation. The generated files include a ServersTransport — verify the timeout values.",
		Example: "apiVersion: traefik.io/v1alpha1\nkind: ServersTransport\nmetadata:\n  name: custom-transport\nspec:\n  forwardingTimeouts:\n    responseHeaderTimeout: \"60s\"\n    readIdleTimeout: \"90s\"\n---\n# Reference it on the Service:\nannotations:\n  traefik.ingress.kubernetes.io/service.serversscheme: https\n  traefik.ingress.kubernetes.io/service.serverstransport: <namespace>-custom-transport@kubernetescrd",
		DocsLink: "https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/#kind-serversTransport",
	},
	"proxy-send-timeout": {
		What:    "Sets the timeout for transmitting a request to the backend.",
		Fix:     "Use ServersTransport with dialTimeout for the initial connection timeout.",
		Example: "spec:\n  forwardingTimeouts:\n    dialTimeout: \"30s\"",
	},
	"proxy-connect-timeout": {
		What:    "Sets the timeout for establishing the connection to the backend.",
		Fix:     "Use ServersTransport dialTimeout field.",
		Example: "spec:\n  forwardingTimeouts:\n    dialTimeout: \"10s\"",
	},
	"ssl-passthrough": {
		What:    "Passes TLS traffic directly to the backend without terminating SSL.",
		Fix:     "Configure a Traefik TCP router with a dedicated entrypoint and passthrough TLS. This cannot be done via Ingress — use IngressRouteTCP CRD instead.",
		Example: "apiVersion: traefik.io/v1alpha1\nkind: IngressRouteTCP\nmetadata:\n  name: myapp-passthrough\nspec:\n  entryPoints:\n    - websecure\n  routes:\n    - match: HostSNI(`myapp.example.com`)\n      services:\n        - name: myapp\n          port: 443\n  tls:\n    passthrough: true",
		DocsLink: "https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/#kind-ingressroutetcp",
	},
	"backend-protocol": {
		What:    "Sets the backend communication protocol (HTTPS, GRPC, GRPCS, AJP, FCGI).",
		Fix:     "For HTTPS backends: add ServersTransport with rootCAs. For gRPC: configure h2c in ServersTransport. See example below.",
		Example: "# For gRPC (h2c):\napiVersion: traefik.io/v1alpha1\nkind: ServersTransport\nmetadata:\n  name: grpc-transport\nspec:\n  disableHTTP2: false\n# Then add on Service:\nannotations:\n  traefik.ingress.kubernetes.io/service.serversscheme: h2c",
	},
	"grpc-backend": {
		What:    "Marks a backend as gRPC, enabling HTTP/2 and gRPC-specific routing.",
		Fix:     "Configure h2c (HTTP/2 cleartext) via ServersTransport and set the service scheme annotation.",
		Example: "# Service annotation:\ntraefik.ingress.kubernetes.io/service.serversscheme: h2c\n\n# ServersTransport (optional, for custom timeouts):\napiVersion: traefik.io/v1alpha1\nkind: ServersTransport\nmetadata:\n  name: grpc-transport\nspec:\n  forwardingTimeouts:\n    responseHeaderTimeout: \"0s\"  # no timeout for streaming",
	},

	// --- PARTIAL (workaround needed) ---
	"proxy-body-size": {
		What:        "Limits the maximum client request body size (e.g., for file uploads). Requests larger than this limit are rejected with 413 Request Entity Too Large.",
		Fix:         "Create a Buffering Middleware with maxRequestBodyBytes:\n1. Apply the generated middleware YAML (02-middlewares/body-size-mw.yaml)\n2. The middleware is auto-attached to the Ingress via traefik.ingress.kubernetes.io/router.middlewares annotation",
		Example:     "apiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: body-size-limit\n  namespace: default\nspec:\n  buffering:\n    maxRequestBodyBytes: 5242880  # 5 MiB (convert from nginx value)\n    retryExpression: IsNetworkError() && Attempts() <= 2",
		DocsLink:    "https://doc.traefik.io/traefik/middlewares/http/buffering/",
		Consequence: "Without this limit, Traefik will accept request bodies of any size to your backend, which could exhaust backend memory.",
	},
	"proxy-request-buffering": {
		What:        "Controls whether the request body is fully buffered before forwarding to the backend. 'off' enables streaming (request body forwarded as it arrives).",
		Fix:         "Traefik does not buffer requests by default — streaming is the default. If proxy-request-buffering is 'off', no action needed. If you need buffering enabled, add a Buffering Middleware.",
		Example:     "# proxy-request-buffering: off → no action needed (Traefik streams by default)\n\n# If you need buffering on:\napiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: buffering-mw\nspec:\n  buffering:\n    maxRequestBodyBytes: 0  # 0 = unlimited buffering",
		DocsLink:    "https://doc.traefik.io/traefik/middlewares/http/buffering/",
	},
	"session-cookie-expires": {
		What:        "Sets a hard expiry (in seconds) for the session affinity cookie. After this time the browser discards the cookie and the user is re-assigned to a backend.",
		Fix:         "Add the service.sticky.cookie.maxage annotation to the Service resource. Convert seconds to the same integer value.",
		Example:     "# On the Service resource:\nannotations:\n  traefik.ingress.kubernetes.io/service.sticky.cookie: \"true\"\n  traefik.ingress.kubernetes.io/service.sticky.cookie.name: SERVERID\n  traefik.ingress.kubernetes.io/service.sticky.cookie.maxage: \"172800\"  # same value in seconds",
		DocsLink:    "https://doc.traefik.io/traefik/routing/services/#sticky-sessions",
		Consequence: "Without expiry, the sticky session cookie persists for the browser session only (deleted when the tab is closed).",
	},
	"session-cookie-max-age": {
		What:        "Sets the Max-Age attribute on the session affinity cookie (seconds until expiry).",
		Fix:         "Set service.sticky.cookie.maxage on the Service. This maps directly to the cookie Max-Age attribute.",
		Example:     "annotations:\n  traefik.ingress.kubernetes.io/service.sticky.cookie.maxage: \"3600\"  # 1 hour",
		DocsLink:    "https://doc.traefik.io/traefik/routing/services/#sticky-sessions",
	},
	"service-upstream": {
		What:        "Routes requests directly to the Service ClusterIP instead of individual pod endpoints, bypassing kube-proxy endpoint slicing.",
		Fix:         "Set the nativelb annotation on the Service to enable direct Service IP routing in Traefik.",
		Example:     "# On the Service resource:\nannotations:\n  traefik.ingress.kubernetes.io/service.nativelb: \"true\"",
		DocsLink:    "https://doc.traefik.io/traefik/providers/kubernetes-ingress/#on-service-annotations",
	},
	"from-to-www-redirect": {
		What:        "Automatically redirects the bare domain (example.com) to the www subdomain (www.example.com) or vice versa.",
		Fix:         "Create a RedirectRegex Middleware and an additional Ingress rule for the source domain:\n1. Apply the generated middleware YAML\n2. The generated Ingress includes the redirect rule for the source domain",
		Example:     "apiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: www-redirect\nspec:\n  redirectRegex:\n    regex: \"^https?://example\\.com/(.*)\"\n    replacement: \"https://www.example.com/${1}\"\n    permanent: true\n---\n# Separate Ingress for the bare domain:\napiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n  name: www-redirect-ingress\n  annotations:\n    traefik.ingress.kubernetes.io/router.middlewares: default-www-redirect@kubernetescrd\nspec:\n  rules:\n    - host: example.com\n      http:\n        paths:\n          - path: /\n            pathType: Prefix\n            backend:\n              service:\n                name: placeholder-svc\n                port:\n                  number: 80",
		DocsLink:    "https://doc.traefik.io/traefik/middlewares/http/redirectregex/",
	},
	"upstream-vhost": {
		What:        "Overrides the Host header sent to the backend service (useful when the backend expects a different hostname than the public URL).",
		Fix:         "1. Create a Headers Middleware that sets the custom Host header\n2. Set passhostheader: false on the Ingress service to stop the original host from being forwarded",
		Example:     "apiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: vhost-headers\nspec:\n  headers:\n    customRequestHeaders:\n      Host: \"backend.internal.example.com\"\n---\n# On the Ingress:\nannotations:\n  traefik.ingress.kubernetes.io/router.middlewares: default-vhost-headers@kubernetescrd\n  traefik.ingress.kubernetes.io/service.passhostheader: \"false\"",
		DocsLink:    "https://doc.traefik.io/traefik/routing/services/#pass-host-header",
	},
	"secure-verify-ca-secret": {
		What:        "Verifies the backend's TLS certificate against a custom CA (enables mTLS or one-way TLS to the backend).",
		Fix:         "Create a ServersTransport CRD with rootcas referencing the CA secret, then reference it from the Service:\n1. Apply the generated ServersTransport YAML\n2. Add the service.serverstransport annotation to the Service",
		Example:     "apiVersion: traefik.io/v1alpha1\nkind: ServersTransport\nmetadata:\n  name: verify-backend-ca\nspec:\n  rootcas:\n    - secret: my-ca-secret  # same secret name as in nginx annotation\n  serverName: backend.internal\n---\n# On the Service:\nannotations:\n  traefik.ingress.kubernetes.io/service.serversscheme: https\n  traefik.ingress.kubernetes.io/service.serverstransport: default-verify-backend-ca@kubernetescrd",
		DocsLink:    "https://doc.traefik.io/traefik/routing/providers/kubernetes-crd/#kind-serversTransport",
	},

	// --- UNSUPPORTED (manual intervention required) ---
	"session-cookie-conditional-samesite-none": {
		What:        "Sets SameSite=None on the sticky session cookie only for browsers that correctly support the SameSite attribute (detects incompatible browsers via User-Agent header).",
		Fix:         "Traefik sets SameSite statically — no User-Agent conditional logic is available. Options:\n1. Set SameSite=None unconditionally (affects < 1% of users on iOS 12 / Chrome 51-66)\n2. Add a Traefik plugin that performs UA-sniffing logic\n3. If your users are predominantly on modern browsers, set SameSite: none unconditionally",
		Example:     "# Set SameSite unconditionally (recommended for modern traffic):\nannotations:\n  traefik.ingress.kubernetes.io/service.sticky.cookie.samesite: none\n  traefik.ingress.kubernetes.io/service.sticky.cookie.secure: \"true\"",
		DocsLink:    "https://doc.traefik.io/traefik/routing/services/#sticky-sessions",
		Consequence: "~0.5–1% of users on iOS 12 or Chrome 51-66 may have sticky sessions broken when SameSite=None is set unconditionally.",
		IssueUrl:    "https://github.com/traefik/traefik/issues/6962",
	},
	"session-cookie-change-on-failure": {
		What:        "Generates a new sticky session cookie when the assigned backend fails, transparently re-routing the user to a healthy backend without a visible error.",
		Fix:         "No direct Traefik equivalent. To minimize user impact on backend failure:\n1. Enable Traefik health checks to quickly remove unhealthy backends\n2. Add a Retry Middleware to handle transient failures\n3. Users may see one 502 on backend failure before Traefik re-routes them",
		Example:     "# Health checks on the Service:\nannotations:\n  traefik.ingress.kubernetes.io/service.healthcheck.path: /healthz\n  traefik.ingress.kubernetes.io/service.healthcheck.interval: 10s\n---\n# Retry Middleware:\napiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: retry-mw\nspec:\n  retry:\n    attempts: 3\n    initialInterval: 100ms",
		DocsLink:    "https://doc.traefik.io/traefik/middlewares/http/retry/",
		Consequence: "On backend failure, a user with a sticky session may receive one error response before Traefik detects the failure and re-routes them to a healthy backend.",
		IssueUrl:    "https://github.com/traefik/traefik/issues/1299",
	},
	"client-body-buffer-size": {
		What:        "Sets the size of the buffer for reading the client request body before proxying.",
		Fix:         "No Traefik equivalent. Configure buffering at the application server level. If you need request body buffering, consider a sidecar proxy or the Buffering Middleware (which controls the full body, not just the buffer size).",
		Example:     "# Not directly configurable in Traefik.\n# Handle at application level, or use Buffering Middleware to limit full body size.",
		Consequence: "Requests may be forwarded to the backend without being fully buffered first (streaming behavior).",
	},
	"configuration-snippet": {
		What:        "Injects arbitrary NGINX config into the server block (e.g., custom headers, rewrite rules, custom log formats).",
		Fix:         "Identify what each directive in the snippet does and replace with native Traefik Middleware CRDs:\n- Custom headers → Headers Middleware\n- Rewrites → ReplacePath / ReplacePathRegex Middleware\n- Redirects → RedirectScheme / RedirectRegex Middleware\n- Auth logic → ForwardAuth / BasicAuth Middleware\n- Rate limiting → RateLimit Middleware",
		Example:     "# Replace a custom header snippet:\n# NGINX snippet: add_header X-Frame-Options DENY;\n# Traefik equivalent:\napiVersion: traefik.io/v1alpha1\nkind: Middleware\nmetadata:\n  name: security-headers\nspec:\n  headers:\n    customResponseHeaders:\n      X-Frame-Options: \"DENY\"\n      X-Content-Type-Options: \"nosniff\"",
		Consequence: "Any custom NGINX directives in the snippet will NOT be applied. Features they implement (custom headers, rewrites, etc.) must be manually replaced with Middleware CRDs.",
	},
	"server-snippet": {
		What:        "Injects arbitrary NGINX config into the server-level block.",
		Fix:         "Same approach as configuration-snippet — identify each directive and replace with native Traefik Middleware CRDs or switch to IngressRoute CRD for full control.",
		Example:     "# Convert per-feature snippets to typed Middleware CRDs.\n# Use IngressRoute CRD for full control over routing.",
		Consequence: "Server-level NGINX directives will NOT be applied. Review the snippet and replace each directive with its Traefik equivalent.",
	},
	"proxy-buffering": {
		What:        "Enables/disables NGINX proxy response buffering.",
		Fix:         "Traefik doesn't expose proxy response buffering via Ingress annotations. For streaming APIs (SSE, chunked transfer), buffering is automatically disabled. For large responses, configure at the application level.",
		Example:     "# Traefik handles streaming natively.\n# No config needed for SSE or chunked responses.",
		Consequence: "If proxy-buffering was disabled for streaming (SSE/WebSocket), Traefik handles this natively — no impact. If it was enabled for performance, behavior may differ slightly.",
	},
	"upstream-hash-by": {
		What:        "Enables consistent hash load balancing based on a request variable (URL, header, cookie) — ensures the same client always reaches the same backend.",
		Fix:         "Traefik Ingress uses round-robin only. For session affinity, use cookie-based sticky sessions instead. For true consistent hashing, switch to Traefik IngressRoute CRD.",
		Example:     "# Use sticky sessions instead:\nannotations:\n  traefik.ingress.kubernetes.io/affinity: \"true\"\n  traefik.ingress.kubernetes.io/affinity-cookie-name: \"SERVERID\"",
		Consequence: "Consistent hash-based routing will be replaced with round-robin. If your application relies on hash-based affinity (e.g., for caching), use sticky sessions or IngressRoute CRD.",
	},
	"load-balance": {
		What:        "Selects the load balancing algorithm for upstream selection (e.g., ewma, round_robin, ip_hash).",
		Fix:         "Traefik Ingress uses round-robin. For advanced load balancing switch to Traefik IngressRoute CRD with a Weighted service, or use an external load balancer.",
		Example:     "# For weighted backends via IngressRoute:\nspec:\n  routes:\n    - kind: Rule\n      match: Host(`example.com`)\n      services:\n        - name: app-v1\n          port: 80\n          weight: 80\n        - name: app-v2\n          port: 80\n          weight: 20",
		Consequence: "Custom load balancing algorithm will be replaced with round-robin. Impact depends on your use case.",
	},
}

// gatewayAPIGuides maps annotation key → actionable fix guide for Gateway API target.
var gatewayAPIGuides = map[string]AnnotationGuide{
	// --- PARTIAL (workaround needed) ---
	"enable-cors": {
		What:    "Enables CORS with configured allowed origins, methods, and headers.",
		Fix:     "Gateway API v1 has no native CORS filter. Add ResponseHeaderModifier filters to your HTTPRoute rules to set CORS headers manually. For preflight (OPTIONS) requests add a separate HTTPRoute rule.",
		Example: "spec:\n  rules:\n    - matches:\n        - method: OPTIONS\n      filters:\n        - type: ResponseHeaderModifier\n          responseHeaderModifier:\n            set:\n              - name: Access-Control-Allow-Origin\n                value: \"https://app.example.com\"\n              - name: Access-Control-Allow-Methods\n                value: \"GET,POST,PUT,DELETE,OPTIONS\"\n              - name: Access-Control-Allow-Headers\n                value: \"Authorization,Content-Type\"\n              - name: Access-Control-Max-Age\n                value: \"86400\"",
	},
	"cors-allow-origin": {
		What:    "Sets the Access-Control-Allow-Origin CORS header.",
		Fix:     "Add ResponseHeaderModifier filter to HTTPRoute with the Access-Control-Allow-Origin header.",
		Example: "filters:\n  - type: ResponseHeaderModifier\n    responseHeaderModifier:\n      set:\n        - name: Access-Control-Allow-Origin\n          value: \"https://app.example.com\"",
	},
	"cors-allow-methods": {
		What:    "Sets the Access-Control-Allow-Methods CORS header.",
		Fix:     "Include Access-Control-Allow-Methods in the ResponseHeaderModifier filter.",
		Example: "- name: Access-Control-Allow-Methods\n  value: \"GET,POST,DELETE,OPTIONS\"",
	},
	"cors-allow-headers": {
		What:    "Sets the Access-Control-Allow-Headers CORS header.",
		Fix:     "Include Access-Control-Allow-Headers in the ResponseHeaderModifier filter.",
		Example: "- name: Access-Control-Allow-Headers\n  value: \"Authorization,Content-Type,X-Request-ID\"",
	},
	"cors-allow-credentials": {
		What:    "Sets Access-Control-Allow-Credentials: true on CORS responses.",
		Fix:     "Include Access-Control-Allow-Credentials in the ResponseHeaderModifier filter.",
		Example: "- name: Access-Control-Allow-Credentials\n  value: \"true\"",
	},
	"auth-url": {
		What:    "Forwards every request to an external URL for authentication before proxying.",
		Fix:     "Create an Envoy Gateway SecurityPolicy with ext_authz or basic_auth. The generated files include a SecurityPolicy — verify the ext-auth service address.",
		Example: "apiVersion: gateway.envoyproxy.io/v1alpha1\nkind: SecurityPolicy\nmetadata:\n  name: ext-auth-policy\nspec:\n  targetRef:\n    group: gateway.networking.k8s.io\n    kind: HTTPRoute\n    name: myapp-route\n  extAuth:\n    http:\n      backendRef:\n        name: oauth2-proxy\n        port: 4180\n      headersToBackend:\n        - Authorization\n        - Cookie",
		DocsLink: "https://gateway.envoyproxy.io/docs/tasks/security/ext-auth/",
	},
	"auth-response-headers": {
		What:    "Sets which headers from the auth response should be passed to the upstream.",
		Fix:     "Configure headersToBackend in the SecurityPolicy ext-auth block.",
		Example: "extAuth:\n  http:\n    headersToBackend:\n      - Authorization\n      - X-User-Id\n      - X-Email",
	},
	"limit-rps": {
		What:    "Limits requests per second from a single IP to prevent abuse.",
		Fix:     "Create an Envoy Gateway BackendTrafficPolicy with a rateLimit rule. The generated files include this — verify the namespace and targetRef.",
		Example: "apiVersion: gateway.envoyproxy.io/v1alpha1\nkind: BackendTrafficPolicy\nmetadata:\n  name: rate-limit-policy\nspec:\n  targetRef:\n    group: gateway.networking.k8s.io\n    kind: HTTPRoute\n    name: myapp-route\n  rateLimit:\n    type: Global\n    global:\n      rules:\n        - clientSelectors:\n            - sourceIP: 0.0.0.0/0\n          limit:\n            requests: 100\n            unit: Second",
		DocsLink: "https://gateway.envoyproxy.io/docs/tasks/traffic/global-rate-limit/",
	},
	"limit-rpm": {
		What:    "Limits requests per minute from a single IP.",
		Fix:     "Same as limit-rps but set unit: Minute in BackendTrafficPolicy.",
		Example: "limit:\n  requests: 300\n  unit: Minute",
	},
	"limit-connections": {
		What:    "Limits the number of concurrent connections from a single IP.",
		Fix:     "Envoy Gateway doesn't have per-IP connection limits directly. Use circuit breaker via BackendTrafficPolicy or configure at the load balancer level.",
		Example: "# For circuit breaking:\nspec:\n  circuitBreaker:\n    consecutiveErrors: 5\n    interval: 30s\n    baseEjectionTime: 30s",
	},
	"affinity": {
		What:    "Enables session affinity (sticky sessions) using a cookie.",
		Fix:     "Create a BackendLBPolicy with sessionPersistence (Gateway API v1.1). Envoy Gateway supports this from v1.2.",
		Example: "apiVersion: gateway.networking.k8s.io/v1alpha2\nkind: BackendLBPolicy\nmetadata:\n  name: sticky-sessions\nspec:\n  targetRef:\n    group: \"\"\n    kind: Service\n    name: myapp\n  sessionPersistence:\n    sessionName: SERVERID\n    type: Cookie",
		DocsLink: "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io%2fv1alpha2.BackendLBPolicy",
	},
	"session-cookie-name": {
		What:    "Sets the name of the session affinity cookie.",
		Fix:     "Set sessionPersistence.sessionName in the BackendLBPolicy.",
		Example: "sessionPersistence:\n  sessionName: MY_SESSION_COOKIE",
	},
	"proxy-read-timeout": {
		What:    "Timeout for reading the full response from the backend.",
		Fix:     "Add a timeouts block to the HTTPRoute rule. Gateway API supports per-rule timeouts natively in v1.",
		Example: "spec:\n  rules:\n    - matches:\n        - path:\n            value: /\n      timeouts:\n        backendRequest: 60s  # backend response timeout\n        request: 90s         # total request timeout",
		DocsLink: "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.HTTPRouteTimeouts",
	},
	"proxy-connect-timeout": {
		What:    "Timeout for establishing the TCP connection to the backend.",
		Fix:     "Set timeouts.request in the HTTPRoute rule. Connection timeouts are included in the overall request timeout.",
		Example: "timeouts:\n  request: 10s",
	},
	"whitelist-source-range": {
		What:    "Allows traffic only from the specified IP CIDR ranges.",
		Fix:     "Create a SecurityPolicy with ipFilter action Allow. The generated files include this — verify the IP ranges are correct.",
		Example: "apiVersion: gateway.envoyproxy.io/v1alpha1\nkind: SecurityPolicy\nmetadata:\n  name: ip-allowlist\nspec:\n  targetRef:\n    kind: HTTPRoute\n    name: myapp-route\n  ipFilter:\n    action: Allow\n    cidrs:\n      - ip: 10.0.0.0\n        mask: 8\n      - ip: 192.168.1.0\n        mask: 24",
		DocsLink: "https://gateway.envoyproxy.io/docs/tasks/security/restrict-ip-access/",
	},
	"denylist-source-range": {
		What:    "Blocks traffic from specified IP CIDR ranges.",
		Fix:     "Create a SecurityPolicy with ipFilter action Deny.",
		Example: "spec:\n  ipFilter:\n    action: Deny\n    cidrs:\n      - ip: 1.2.3.0\n        mask: 24",
	},
	"ssl-passthrough": {
		What:    "Passes TLS traffic to the backend without terminating SSL at the gateway.",
		Fix:     "Use a TLSRoute with passthrough mode. The Gateway listener must have tls.mode: Passthrough.",
		Example: "# Gateway listener:\nlisteners:\n  - name: tls-passthrough\n    port: 443\n    protocol: TLS\n    tls:\n      mode: Passthrough\n---\napiVersion: gateway.networking.k8s.io/v1alpha2\nkind: TLSRoute\nmetadata:\n  name: myapp-tls\nspec:\n  parentRefs:\n    - name: my-gateway\n  hostnames:\n    - myapp.example.com\n  rules:\n    - backendRefs:\n        - name: myapp\n          port: 443",
	},
	"backend-protocol": {
		What:    "Sets backend protocol (HTTPS, GRPC, GRPCS, AJP).",
		Fix:     "For HTTPS backends: configure TLS on the Gateway listener. For gRPC use GRPCRoute (see below).",
		Example: "# For HTTPS backend, set parentRef with TLS listener and use httpsRoute\n# For gRPC backend, switch to GRPCRoute instead of HTTPRoute",
	},

	// --- PARTIAL ---
	"affinity-mode": {
		What:        "Controls how sticky sessions are re-balanced when pod replicas change. 'balanced' re-distributes sessions on scaling; 'persistent' keeps cookies mapped to the same backend.",
		Fix:         "BackendLBPolicy always uses persistent cookie affinity. Balanced mode is not in the Gateway API spec. Enable backend health checks in BackendTrafficPolicy to handle pod failures gracefully.",
		Example:     "apiVersion: gateway.envoyproxy.io/v1alpha1\nkind: BackendTrafficPolicy\nmetadata:\n  name: health-checks\nspec:\n  targetRef:\n    group: gateway.networking.k8s.io\n    kind: HTTPRoute\n    name: myapp-route\n  healthCheck:\n    active:\n      type: HTTP\n      http:\n        path: /healthz\n      interval: 10s\n      timeout: 3s",
		DocsLink:    "https://gateway.envoyproxy.io/docs/api/extension_types/#backendtrafficpolicy",
		Consequence: "When pods scale up, existing sticky sessions are not re-balanced to new pods. Load distribution may be uneven until sessions expire.",
	},
	"proxy-body-size": {
		What:        "Limits the maximum client request body size. Requests exceeding this limit are rejected with 413 Request Entity Too Large.",
		Fix:         "Use Envoy Gateway BackendTrafficPolicy with the requestBuffer.limit field:\n1. Ensure Envoy Gateway v1.3+ is installed\n2. Apply the generated BackendTrafficPolicy YAML",
		Example:     "apiVersion: gateway.envoyproxy.io/v1alpha1\nkind: BackendTrafficPolicy\nmetadata:\n  name: body-size-policy\n  namespace: default\nspec:\n  targetRef:\n    group: gateway.networking.k8s.io\n    kind: HTTPRoute\n    name: myapp-route\n  requestBuffer:\n    limit: 10Mi  # convert from nginx value (e.g., 10m → 10Mi)",
		DocsLink:    "https://gateway.envoyproxy.io/docs/api/extension_types/#backendtrafficpolicy",
		Consequence: "Without this limit, Envoy Gateway will accept request bodies of any size to your backend.",
	},
	"proxy-request-buffering": {
		What:    "Controls whether the request body is fully buffered before forwarding. 'off' enables streaming.",
		Fix:     "Envoy Gateway streams requests by default (equivalent to off). No configuration needed.",
		Example: "# No configuration needed — streaming is the default in Envoy Gateway.",
	},
	"proxy-http-version": {
		What:    "Forces the proxy to use a specific HTTP version when talking to the backend (1.1 for WebSocket, 2.0 for gRPC/h2).",
		Fix:     "Envoy Gateway handles HTTP/2 and HTTP/1.1 natively. For gRPC backends use GRPCRoute. For WebSocket, HTTPRoute with no special config is sufficient.",
		Example: "# For gRPC backends — use GRPCRoute instead of HTTPRoute:\napiVersion: gateway.networking.k8s.io/v1\nkind: GRPCRoute\nmetadata:\n  name: myapp-grpc\nspec:\n  parentRefs:\n    - name: my-gateway\n  hostnames:\n    - grpc.example.com\n  rules:\n    - backendRefs:\n        - name: myapp\n          port: 9090",
		DocsLink: "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.GRPCRoute",
	},
	"session-cookie-expires": {
		What:        "Sets a hard expiry (in seconds) for the session affinity cookie.",
		Fix:         "Configure BackendLBPolicy with cookieConfig.lifetimeType: Permanent and absoluteTimeout.\nNote: Requires Gateway API v1.1+ and Envoy Gateway v1.2+.",
		Example:     "apiVersion: gateway.networking.k8s.io/v1alpha2\nkind: BackendLBPolicy\nmetadata:\n  name: sticky-sessions\nspec:\n  targetRef:\n    group: \"\"\n    kind: Service\n    name: myapp\n  sessionPersistence:\n    sessionName: SERVERID\n    type: Cookie\n    cookieConfig:\n      lifetimeType: Permanent\n      absoluteTimeout: 48h  # convert: 172800s = 48h",
		DocsLink:    "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io%2fv1alpha2.BackendLBPolicy",
		Consequence: "Without expiry, the sticky session cookie persists for the browser session only.",
	},
	"session-cookie-max-age": {
		What:        "Sets the Max-Age attribute on the session affinity cookie.",
		Fix:         "Configure BackendLBPolicy with cookieConfig.absoluteTimeout. Convert seconds to a duration string.",
		Example:     "sessionPersistence:\n  cookieConfig:\n    lifetimeType: Permanent\n    absoluteTimeout: 1h  # convert from seconds",
		DocsLink:    "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io%2fv1alpha2.BackendLBPolicy",
	},
	"service-upstream": {
		What:        "Routes requests to the Service ClusterIP instead of individual pod endpoints.",
		Fix:         "Gateway API routes to pod IPs by default (same behavior as service-upstream: true). No configuration needed.",
		Example:     "# No configuration needed — Gateway API uses pod endpoint IPs by default.",
	},
	"from-to-www-redirect": {
		What:        "Automatically redirects the bare domain to the www subdomain (or vice versa).",
		Fix:         "Add a separate HTTPRoute for the source domain with a RequestRedirect filter that replaces the hostname.",
		Example:     "# HTTPRoute to redirect example.com → www.example.com:\napiVersion: gateway.networking.k8s.io/v1\nkind: HTTPRoute\nmetadata:\n  name: www-redirect\nspec:\n  parentRefs:\n    - name: my-gateway\n  hostnames:\n    - example.com\n  rules:\n    - filters:\n        - type: RequestRedirect\n          requestRedirect:\n            hostname: www.example.com\n            statusCode: 301",
		DocsLink:    "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.HTTPRequestRedirectFilter",
	},
	"upstream-vhost": {
		What:        "Overrides the Host header sent to the backend service.",
		Fix:         "Add a RequestHeaderModifier filter to the HTTPRoute rule that sets the Host header.",
		Example:     "filters:\n  - type: RequestHeaderModifier\n    requestHeaderModifier:\n      set:\n        - name: Host\n          value: backend.internal.example.com",
		DocsLink:    "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1.HTTPRequestHeaderFilter",
	},
	"secure-verify-ca-secret": {
		What:        "Verifies the backend's TLS certificate against a custom CA (mTLS or one-way TLS to the backend).",
		Fix:         "Create a BackendTLSPolicy with caCertificateRefs pointing to a ConfigMap containing the CA:\n1. Export CA cert from the Kubernetes secret\n2. Create a ConfigMap with the CA\n3. Apply the generated BackendTLSPolicy YAML",
		Example:     "apiVersion: gateway.networking.k8s.io/v1alpha3\nkind: BackendTLSPolicy\nmetadata:\n  name: backend-tls-verify\nspec:\n  targetRefs:\n    - group: \"\"\n      kind: Service\n      name: myapp\n      sectionName: https\n  validation:\n    caCertificateRefs:\n      - group: \"\"\n        kind: ConfigMap\n        name: backend-ca-cert\n    hostname: backend.internal.example.com",
		DocsLink:    "https://gateway-api.sigs.k8s.io/reference/spec/#gateway.networking.k8s.io/v1alpha3.BackendTLSPolicy",
	},

	// --- UNSUPPORTED ---
	"session-cookie-conditional-samesite-none": {
		What:        "Sets SameSite=None on the sticky session cookie only for browsers that correctly support the SameSite attribute (detects incompatible browsers via User-Agent).",
		Fix:         "Gateway API / Envoy Gateway does not support conditional SameSite logic. Options:\n1. Set SameSite=None unconditionally (affects < 1% of users on iOS 12 / Chrome 51-66)\n2. Deploy a sidecar or Envoy plugin that performs UA-sniffing",
		Consequence: "~0.5–1% of users on iOS 12 or Chrome 51-66 may have sticky sessions broken when SameSite=None is set unconditionally.",
		IssueUrl:    "https://github.com/envoyproxy/envoy/issues/15555",
	},
	"session-cookie-change-on-failure": {
		What:        "Regenerates the sticky session cookie when the backend fails, transparently re-routing the user to a healthy backend.",
		Fix:         "Not supported in Gateway API. Configure health checks and circuit breaking in BackendTrafficPolicy to handle backend failures gracefully.",
		Example:     "spec:\n  circuitBreaker:\n    consecutiveErrors: 5\n    interval: 30s\n    baseEjectionTime: 30s",
		DocsLink:    "https://gateway.envoyproxy.io/docs/api/extension_types/#backendtrafficpolicy",
		Consequence: "On backend failure, a user with a sticky session may receive one error response before Envoy Gateway detects the failure and re-routes them.",
	},
	"configuration-snippet": {
		What:        "Injects raw NGINX configuration directly into the server block.",
		Fix:         "Identify what the snippet does and replace with native Gateway API resources:\n- HTTP headers → HTTPRoute ResponseHeaderModifier/RequestHeaderModifier filter\n- Redirects → HTTPRoute RequestRedirect filter\n- Rate limiting → BackendTrafficPolicy\n- Auth → SecurityPolicy\n- Custom Envoy config → EnvoyPatchPolicy (xDS patch)",
		Example:     "# Example: snippet that adds security headers\n# Replace with HTTPRoute filter:\nfilters:\n  - type: ResponseHeaderModifier\n    responseHeaderModifier:\n      set:\n        - name: X-Frame-Options\n          value: DENY\n        - name: X-Content-Type-Options\n          value: nosniff\n        - name: X-XSS-Protection\n          value: \"1; mode=block\"",
		Consequence: "Custom NGINX directives will NOT be applied. Each feature in the snippet must be manually replaced with Gateway API resources.",
	},
	"server-snippet": {
		What:        "Injects raw NGINX configuration at the server level.",
		Fix:         "Same as configuration-snippet. Identify each directive and map to Gateway API resources. Use EnvoyPatchPolicy for advanced Envoy-specific config.",
		Example:     "# For advanced Envoy configuration use EnvoyPatchPolicy:\napiVersion: gateway.envoyproxy.io/v1alpha1\nkind: EnvoyPatchPolicy\nmetadata:\n  name: custom-config\nspec:\n  targetRef:\n    group: gateway.networking.k8s.io\n    kind: Gateway\n    name: my-gateway\n  type: JSONPatch\n  jsonPatches:\n    - type: type.googleapis.com/envoy.config.listener.v3.Listener\n      name: <listener-name>\n      operation:\n        op: add\n        path: \"/...\"\n        value: {}",
		DocsLink:    "https://gateway.envoyproxy.io/docs/api/extension_types/#envoypatchpolicy",
		Consequence: "Server-level NGINX directives will NOT be applied. Review the snippet and replace each directive.",
	},
	"auth-type": {
		What:        "Enables HTTP Basic authentication (type: basic).",
		Fix:         "Core Gateway API has no basic auth. Use SecurityPolicy with BasicAuth (Envoy Gateway v1.1+) or deploy oauth2-proxy as an auth sidecar with ext-auth.",
		Example:     "# Envoy Gateway BasicAuth (v1.1+):\napiVersion: gateway.envoyproxy.io/v1alpha1\nkind: SecurityPolicy\nmetadata:\n  name: basic-auth-policy\nspec:\n  targetRef:\n    kind: HTTPRoute\n    name: myapp-route\n  basicAuth:\n    users:\n      name: basic-auth-users  # Secret with .htpasswd key",
		DocsLink:    "https://gateway.envoyproxy.io/docs/tasks/security/basic-auth/",
		Consequence: "The endpoint will have no basic auth protection until a SecurityPolicy is applied. Do NOT expose this to the internet without auth.",
	},
	"auth-secret": {
		What:        "References the Kubernetes secret containing htpasswd credentials for basic auth.",
		Fix:         "Create a secret with key 'users' containing htpasswd content and reference in SecurityPolicy basicAuth.",
		Example:     "kubectl create secret generic basic-auth-users \\\n  --from-literal=users=$(htpasswd -nB admin 'password') \\\n  -n <namespace>",
		Consequence: "Basic auth will be disabled until the SecurityPolicy is created with the correct secret reference.",
	},
	"load-balance": {
		What:        "Selects the load balancing algorithm for upstream selection.",
		Fix:         "Use BackendLBPolicy (Gateway API v1.1) for LeastRequest or RoundRobin. For consistent hashing use Envoy Gateway's BackendLBPolicy with ConsistentHashType.",
		Example:     "apiVersion: gateway.networking.k8s.io/v1alpha2\nkind: BackendLBPolicy\nmetadata:\n  name: lb-policy\nspec:\n  targetRef:\n    group: \"\"\n    kind: Service\n    name: myapp\n  # For load balance algorithm: use EnvoyPatchPolicy for advanced LB config",
		Consequence: "Envoy Gateway defaults to round-robin. Custom algorithm will not be applied until a BackendLBPolicy or EnvoyPatchPolicy is created.",
	},
	"upstream-hash-by": {
		What:        "Enables consistent hash load balancing using a request variable.",
		Fix:         "Use BackendLBPolicy with ConsistentHash (Envoy Gateway extension) or configure via EnvoyPatchPolicy.",
		Example:     "# Via EnvoyPatchPolicy for consistent hashing:\n# Patch the cluster config to use RING_HASH or MAGLEV",
		Consequence: "Consistent hash-based routing will be replaced with round-robin. If your application relies on hash affinity, configure BackendLBPolicy.",
	},
}

// GetAnnotationGuide returns the fix guide for a given target and annotation key.
// Returns an empty guide if no specific guide exists (falls back to the note from the mapping).
func GetAnnotationGuide(target, annotationKey string) AnnotationGuide {
	var guides map[string]AnnotationGuide
	switch target {
	case "traefik":
		guides = traefikGuides
	case "gateway-api":
		guides = gatewayAPIGuides
	default:
		return AnnotationGuide{}
	}
	return guides[annotationKey]
}
