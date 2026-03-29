package analyzer

// MappingStatus represents how well an annotation maps to the target controller.
type MappingStatus string

const (
	StatusSupported   MappingStatus = "supported"
	StatusPartial     MappingStatus = "partial"
	StatusUnsupported MappingStatus = "unsupported"
)

// AnnotationMapping describes how a specific annotation translates to the target.
type AnnotationMapping struct {
	OriginalKey   string        `json:"originalKey"`
	OriginalValue string        `json:"originalValue"`
	Status        MappingStatus `json:"status"`
	TargetResource string       `json:"targetResource"` // e.g., "Middleware/ssl-redirect"
	GeneratedYAML string        `json:"generatedYaml,omitempty"`
	Note          string        `json:"note"`
}

// traefikMappings defines how each nginx annotation maps to Traefik.
var traefikMappings = map[string]struct {
	Status         MappingStatus
	TargetResource string
	Note           string
}{
	"ssl-redirect":             {StatusSupported, "Middleware (RedirectScheme)", "Generates RedirectScheme middleware"},
	"force-ssl-redirect":       {StatusSupported, "Middleware (RedirectScheme)", "Permanent redirect to HTTPS"},
	"enable-cors":              {StatusSupported, "Middleware (Headers)", "Generates CORS Headers middleware"},
	"cors-allow-origin":        {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"cors-allow-methods":       {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"cors-allow-headers":       {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"cors-expose-headers":      {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"cors-allow-credentials":   {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"cors-max-age":             {StatusSupported, "Middleware (Headers)", "Part of Headers CORS middleware"},
	"affinity":                 {StatusSupported, "Service sticky annotation", "Traefik sticky cookie on Service"},
	"session-cookie-name":      {StatusSupported, "Service sticky annotation", "Cookie name for session affinity"},
	"session-cookie-path":      {StatusPartial, "Service sticky annotation", "Limited path support"},
	"session-cookie-samesite":  {StatusSupported, "Service sticky annotation", "SameSite attribute"},
	"session-cookie-secure":    {StatusSupported, "Service sticky annotation", "Secure flag"},
	"limit-rps":                {StatusSupported, "Middleware (RateLimit)", "Generates RateLimit middleware"},
	"limit-rpm":                {StatusSupported, "Middleware (RateLimit)", "Converted to average rate"},
	"limit-connections":        {StatusSupported, "Middleware (InFlightReq)", "Max concurrent requests"},
	"limit-burst-multiplier":   {StatusSupported, "Middleware (RateLimit)", "Burst size multiplier"},
	"limit-whitelist":          {StatusSupported, "Middleware (RateLimit)", "Exempt IPs from rate limiting"},
	"auth-url":                 {StatusSupported, "Middleware (ForwardAuth)", "Generates ForwardAuth middleware"},
	"auth-method":              {StatusPartial, "Middleware (ForwardAuth)", "Only GET/POST supported"},
	"auth-response-headers":    {StatusSupported, "Middleware (ForwardAuth)", "Headers passed after auth"},
	"auth-request-redirect":    {StatusPartial, "Middleware (ForwardAuth)", "Redirect URL for auth failure"},
	"auth-type":                {StatusPartial, "Middleware (BasicAuth)", "Basic auth only; digest not supported"},
	"auth-secret":              {StatusPartial, "Middleware (BasicAuth)", "Secret format differs from NGINX"},
	"auth-realm":               {StatusSupported, "Middleware (BasicAuth)", "Auth realm"},
	"whitelist-source-range":   {StatusSupported, "Middleware (IPAllowList)", "Generates IPAllowList middleware"},
	"denylist-source-range":    {StatusSupported, "Middleware (IPDenyList)", "Generates IPDenyList middleware"},
	"custom-headers":           {StatusPartial, "Middleware (Headers)", "ConfigMap ref not supported; inline headers needed"},
	"rewrite-target":           {StatusSupported, "Middleware (ReplacePath/AddPrefix)", "URL rewrite middleware"},
	"use-regex":                {StatusSupported, "Router (native)", "Traefik supports regex routing natively"},
	"app-root":                 {StatusSupported, "Router + Middleware", "Redirect root path"},
	"permanent-redirect":       {StatusSupported, "Middleware (RedirectRegex)", "Permanent redirect"},
	"temporal-redirect":        {StatusSupported, "Middleware (RedirectRegex)", "Temporary redirect"},
	"canary":                   {StatusSupported, "Weighted Services", "Weighted backend traffic splitting"},
	"canary-weight":            {StatusSupported, "Weighted Services", "Traffic weight percentage"},
	"canary-by-header":         {StatusPartial, "Router rules", "Header matching in router rules"},
	"canary-by-header-value":   {StatusPartial, "Router rules", "Header value matching"},
	"canary-by-cookie":         {StatusPartial, "Router rules", "Cookie-based routing via rules"},
	"proxy-read-timeout":       {StatusPartial, "ServersTransport CRD", "Requires ServersTransport resource"},
	"proxy-send-timeout":       {StatusPartial, "ServersTransport CRD", "Requires ServersTransport resource"},
	"proxy-connect-timeout":    {StatusPartial, "ServersTransport CRD", "Requires ServersTransport resource"},
	"proxy-buffering":                          {StatusUnsupported, "", "Impact: NONE. Controls whether NGINX buffers backend responses — Traefik streams responses by default which works for all use cases"},
	"proxy-body-size":                          {StatusPartial, "Middleware (Buffering)", "Traefik Buffering middleware with maxRequestBodyBytes"},
	"proxy-request-buffering":                  {StatusPartial, "Native (off by default)", "Off is default behavior; enabling request buffering requires Buffering middleware"},
	"client-body-buffer-size":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal buffer tuning for request body — Traefik handles request body buffering automatically"},
	"configuration-snippet":                    {StatusUnsupported, "", "Impact: VARIES. Raw NGINX config injection — inherently non-portable. Review snippet content to find Traefik equivalents per feature"},
	"server-snippet":                           {StatusUnsupported, "", "Impact: VARIES. Raw NGINX server block injection — inherently non-portable. Review snippet content to find Traefik equivalents per feature"},
	"ssl-passthrough":                          {StatusPartial, "Traefik TCP router", "Requires TCP entrypoint config"},
	"backend-protocol":                         {StatusPartial, "Service annotation", "HTTPS/GRPC backends need ServersTransport"},
	"websocket-services":                       {StatusSupported, "Native", "Traefik supports WebSocket natively"},
	"grpc-backend":                             {StatusPartial, "ServersTransport + h2c", "gRPC requires h2c configuration"},
	"upstream-hash-by":                         {StatusUnsupported, "", "Impact: MEDIUM. Consistent hash-based load balancing — Traefik only supports round-robin and weighted-round-robin. Use session affinity (sticky cookies) as an alternative for session-pinning use cases"},
	"load-balance":                             {StatusUnsupported, "", "Impact: LOW. Traefik uses round-robin by default — only matters if you specifically need least-connections or random algorithms. Round-robin works well for most workloads"},
	"affinity-mode":                             {StatusPartial, "Service (sticky cookie)", "Traefik always uses persistent affinity; balanced re-balancing is not available"},
	"canary-weight-total":                       {StatusSupported, "Weighted Services", "Traefik uses relative weights; total is implicit"},
	"proxy-http-version":                        {StatusPartial, "ServersTransport CRD", "HTTP/2 via ServersTransport; HTTP/1.0 not supported"},
	"session-cookie-expires":                   {StatusPartial, "Service (sticky cookie maxage)", "Convert seconds to service.sticky.cookie.maxage annotation on Service"},
	"session-cookie-max-age":                   {StatusSupported, "Service (sticky cookie maxage)", "Maps to service.sticky.cookie.maxage annotation on Service"},
	"session-cookie-conditional-samesite-none": {StatusUnsupported, "", "Impact: LOW. Sends SameSite=None only for compatible browsers — Traefik sets SameSite statically. Modern browsers all support SameSite=None so conditional logic is rarely needed"},
	"session-cookie-change-on-failure":         {StatusUnsupported, "", "Impact: LOW. Re-issues session cookie when backend fails — Traefik sticky cookies persist across failures. Only matters for specific failover patterns"},
	"service-upstream":                         {StatusSupported, "Service annotation (nativelb)", "Set traefik.ingress.kubernetes.io/service.nativelb: 'true' on Service"},
	"from-to-www-redirect":                     {StatusPartial, "Middleware (RedirectRegex)", "Requires RedirectRegex middleware + separate Ingress entry for www"},
	"upstream-vhost":                           {StatusPartial, "Middleware (Headers) + passhostheader", "Headers middleware sets Host header; disable passhostheader on Service"},
	"secure-verify-ca-secret":                  {StatusPartial, "ServersTransport (rootCAs)", "ServersTransport CRD referencing the CA secret"},

	// ── New annotations ──────────────────────────────────────────────────
	// Client certificate auth
	"auth-tls-secret":                          {StatusPartial, "TLSOption CRD", "Traefik TLSOption with clientAuth.secretNames; requires IngressRoute or Gateway API TLS config"},
	"auth-tls-verify-client":                   {StatusPartial, "TLSOption CRD", "TLSOption clientAuth.clientAuthType maps to on/off/optional/optional_no_ca"},
	"auth-tls-verify-depth":                    {StatusUnsupported, "", "Impact: LOW. Traefik does not expose certificate chain depth — uses full chain by default, which is safe for most setups"},
	"auth-tls-error-page":                      {StatusUnsupported, "", "Impact: LOW. No Traefik equivalent — clients get a raw TLS error instead of a redirect page. Only cosmetic; auth still works"},
	"auth-tls-pass-certificate-to-upstream":    {StatusPartial, "Middleware (PassTLSClientCert)", "PassTLSClientCert middleware forwards cert info in headers; field mapping differs from NGINX"},
	"auth-tls-match-cn":                        {StatusUnsupported, "", "Impact: MEDIUM. Traefik has no CN regex matching — use cert-manager to issue certs with the right CN or validate in your app"},

	// External auth extras
	"auth-secret-type":                         {StatusPartial, "Middleware (BasicAuth)", "BasicAuth uses htpasswd format; auth-map format requires manual conversion"},
	"auth-cache-key":                           {StatusUnsupported, "", "Impact: LOW. Traefik ForwardAuth does not cache responses — every request hits the auth service. Use auth service-side caching instead"},
	"auth-cache-duration":                      {StatusUnsupported, "", "Impact: LOW. No auth caching in Traefik — adds latency per request but auth behavior is correct"},
	"auth-keepalive":                           {StatusUnsupported, "", "Impact: NONE. NGINX-internal optimization — Traefik manages its own connection pooling automatically"},
	"auth-keepalive-share-vars":                {StatusUnsupported, "", "Impact: NONE. NGINX-internal variable sharing — not applicable to Traefik architecture"},
	"auth-keepalive-requests":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal optimization — Traefik handles connection reuse automatically"},
	"auth-keepalive-timeout":                   {StatusUnsupported, "", "Impact: NONE. NGINX-internal timeout — Traefik manages connection lifecycle automatically"},
	"auth-proxy-set-headers":                   {StatusPartial, "Middleware (ForwardAuth)", "ForwardAuth supports authRequestHeaders but reads from middleware config, not ConfigMap"},
	"auth-snippet":                             {StatusUnsupported, "", "Impact: VARIES. NGINX-specific config snippet — review the snippet content to determine if Traefik has an equivalent feature"},
	"auth-always-set-cookie":                   {StatusUnsupported, "", "Impact: LOW. Traefik ForwardAuth always forwards response headers including Set-Cookie — this is default behavior"},
	"auth-signin":                              {StatusPartial, "Middleware (ForwardAuth)", "ForwardAuth can handle redirects but auth-signin-specific behavior requires custom auth service logic"},
	"auth-signin-redirect-param":               {StatusUnsupported, "", "Impact: LOW. Controls the query param name for redirect URL — configure this in your auth service instead"},
	"enable-global-auth":                       {StatusPartial, "Middleware (ForwardAuth)", "Traefik uses middleware chains — add/remove ForwardAuth from the chain to enable/disable per-ingress"},

	// Redirect code customization
	"permanent-redirect-code":                  {StatusPartial, "Middleware (RedirectRegex)", "RedirectRegex supports custom status codes; default 301"},
	"temporal-redirect-code":                   {StatusPartial, "Middleware (RedirectRegex)", "RedirectRegex supports custom status codes; default 302"},
	"preserve-trailing-slash":                  {StatusUnsupported, "", "Impact: LOW. Traefik preserves trailing slashes by default in most redirect scenarios"},

	// Session affinity extras
	"session-cookie-domain":                    {StatusUnsupported, "", "Impact: MEDIUM. Traefik sticky cookies don't support custom Domain — cookie is scoped to request host by default"},
	"affinity-canary-behavior":                 {StatusUnsupported, "", "Impact: LOW. Controls NGINX canary+affinity interaction — Traefik weighted services handle this differently but produce similar results"},

	// Rate limiting extras
	"limit-rate":                               {StatusUnsupported, "", "Impact: LOW. Response-rate limiting (KB/s) — Traefik rate limiting is request-based, not response-bandwidth based. Rarely needed outside file downloads"},
	"limit-rate-after":                         {StatusUnsupported, "", "Impact: LOW. Threshold before response rate limiting — same as limit-rate, not applicable to Traefik's request-based model"},

	// Proxy SSL / Backend TLS
	"proxy-ssl-ciphers":                        {StatusPartial, "ServersTransport CRD", "ServersTransport supports TLS cipher config for backend connections"},
	"proxy-ssl-name":                           {StatusPartial, "ServersTransport CRD", "ServersTransport serverName field for SNI to backend"},
	"proxy-ssl-protocols":                      {StatusPartial, "ServersTransport CRD", "ServersTransport does not expose per-connection protocol selection — uses Go default TLS stack"},
	"proxy-ssl-verify":                         {StatusPartial, "ServersTransport CRD", "ServersTransport insecureSkipVerify=false enables backend cert verification"},
	"proxy-ssl-verify-depth":                   {StatusUnsupported, "", "Impact: LOW. Traefik uses full chain verification — no depth limit needed in most setups"},
	"proxy-ssl-server-name":                    {StatusPartial, "ServersTransport CRD", "SNI is sent automatically when serverName is configured in ServersTransport"},

	// Proxy cookie rewriting
	"proxy-cookie-domain":                      {StatusUnsupported, "", "Impact: MEDIUM. No Traefik equivalent for Set-Cookie domain rewriting — handle in your application or use Headers middleware to strip/replace"},
	"proxy-cookie-path":                        {StatusUnsupported, "", "Impact: MEDIUM. No Traefik equivalent for Set-Cookie path rewriting — handle in your application"},

	// Proxy redirect rewriting
	"proxy-redirect-from":                      {StatusUnsupported, "", "Impact: LOW. NGINX proxy_redirect rewrites Location headers from upstream — Traefik does not rewrite backend response headers. Usually not needed if backends are configured correctly"},
	"proxy-redirect-to":                        {StatusUnsupported, "", "Impact: LOW. Part of proxy_redirect — same as proxy-redirect-from"},

	// Proxy upstream retry
	"proxy-next-upstream":                      {StatusPartial, "Service (retry)", "Traefik retry middleware can retry on network errors; condition-based retry (e.g., on 502) is limited"},
	"proxy-next-upstream-timeout":              {StatusUnsupported, "", "Impact: LOW. Traefik retry has attempt-based limit, not timeout-based — use retry.attempts instead"},
	"proxy-next-upstream-tries":                {StatusPartial, "Middleware (Retry)", "Retry middleware attempts field maps directly"},

	// Proxy buffer extras
	"proxy-buffers-number":                     {StatusUnsupported, "", "Impact: NONE. NGINX-internal buffer tuning — Traefik handles buffering automatically with no user-facing knobs"},
	"proxy-busy-buffers-size":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal buffer tuning — not applicable to Traefik architecture"},
	"proxy-buffer-size":                        {StatusUnsupported, "", "Impact: NONE. NGINX-internal buffer tuning — Traefik handles response buffering automatically"},

	// Canary extras
	"canary-by-header-pattern":                 {StatusPartial, "Router rules (regex)", "Traefik router rules support HeaderRegexp matcher for regex header matching"},

	// Error handling
	"custom-http-errors":                       {StatusPartial, "Middleware (Errors)", "Traefik Errors middleware can intercept status codes and serve from a different service"},
	"default-backend":                          {StatusPartial, "Traefik default rule", "Traefik default routing rule or catch-all router; not a per-ingress annotation"},

	// Observability
	"enable-access-log":                        {StatusPartial, "Traefik static config", "Traefik access logging is global (static config), not per-route — enabled by default"},
	"enable-opentelemetry":                     {StatusPartial, "Traefik static config", "OpenTelemetry tracing is global in Traefik static config; cannot toggle per-ingress"},

	// Request mirroring
	"mirror-target":                            {StatusPartial, "Middleware (Mirroring)", "Traefik Mirroring middleware supports traffic mirroring to additional services"},
	"mirror-request-body":                      {StatusPartial, "Middleware (Mirroring)", "Mirroring middleware includes body by default; no toggle"},

	// Misc
	"server-alias":                             {StatusPartial, "Ingress spec.rules", "Add additional hosts in Ingress spec.rules[].host — not an annotation concept in Traefik"},
	"satisfy":                                  {StatusUnsupported, "", "Impact: LOW. NGINX any/all auth satisfaction — Traefik middleware chains always require all to pass (AND logic). Rarely used with 'any'"},
	"enable-modsecurity":                       {StatusUnsupported, "", "Impact: MEDIUM. No built-in WAF in Traefik — use Traefik plugin ecosystem (e.g., traefik-modsecurity-plugin) or external WAF"},
	"modsecurity-snippet":                      {StatusUnsupported, "", "Impact: MEDIUM. Requires WAF plugin — see enable-modsecurity note"},
	"modsecurity-transaction-id":               {StatusUnsupported, "", "Impact: LOW. WAF transaction ID — only relevant if using ModSecurity plugin"},
	"x-forwarded-prefix":                       {StatusSupported, "Middleware (Headers)", "Headers middleware can set X-Forwarded-Prefix; Traefik also auto-sets this with StripPrefix"},
	"connection-proxy-header":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal Connection header override — Traefik handles WebSocket upgrade and connection headers automatically"},
	"enable-owasp-modsecurity-crs":             {StatusUnsupported, "", "Impact: MEDIUM. Requires WAF plugin — see enable-modsecurity note"},
	"ssl-ciphers":                              {StatusPartial, "TLSOption CRD", "TLSOption CRD supports cipher suite configuration"},
}

// gatewayAPIMappings defines how each nginx annotation maps to Gateway API.
var gatewayAPIMappings = map[string]struct {
	Status         MappingStatus
	TargetResource string
	Note           string
}{
	"ssl-redirect":           {StatusSupported, "HTTPRoute (RequestRedirect filter)", "RequestRedirect filter with scheme=https"},
	"force-ssl-redirect":     {StatusSupported, "HTTPRoute (RequestRedirect filter)", "301 redirect to HTTPS"},
	"rewrite-target":         {StatusSupported, "HTTPRoute (URLRewrite filter)", "Path rewrite via URLRewrite filter"},
	"custom-headers":         {StatusSupported, "HTTPRoute (ResponseHeaderModifier)", "Response header manipulation filter"},
	"canary":                 {StatusSupported, "HTTPRoute (weighted backendRefs)", "Traffic split via backendRefs weights"},
	"canary-weight":          {StatusSupported, "HTTPRoute (weighted backendRefs)", "Weight value in backendRefs"},
	"enable-cors":            {StatusSupported, "HTTPRoute (CORS filter)", "Native CORS filter (GA in Gateway API v1.5)"},
	"cors-allow-origin":      {StatusSupported, "HTTPRoute (CORS filter)", "allowOrigins in CORS filter"},
	"cors-allow-methods":     {StatusSupported, "HTTPRoute (CORS filter)", "allowMethods in CORS filter"},
	"cors-allow-headers":     {StatusSupported, "HTTPRoute (CORS filter)", "allowHeaders in CORS filter"},
	"cors-allow-credentials": {StatusSupported, "HTTPRoute (CORS filter)", "allowCredentials in CORS filter"},
	"auth-url":               {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "Envoy Gateway SecurityPolicy or experimental externalAuth HTTPRoute filter (Gateway API v1.4)"},
	"auth-response-headers":  {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "Part of SecurityPolicy ext-auth or externalAuth filter config"},
	"limit-rps":              {StatusPartial, "BackendTrafficPolicy (RateLimit)", "Envoy Gateway BackendTrafficPolicy"},
	"limit-rpm":              {StatusPartial, "BackendTrafficPolicy (RateLimit)", "Converted to rate limit"},
	"limit-connections":      {StatusPartial, "BackendTrafficPolicy (CircuitBreaker)", "Circuit breaker policy"},
	"whitelist-source-range": {StatusPartial, "HTTPRoute (source IP match)", "HTTPRouteMatch with client IP — limited support"},
	"denylist-source-range":  {StatusPartial, "SecurityPolicy (IPFilter)", "Envoy Gateway SecurityPolicy IP filter"},
	"affinity":               {StatusPartial, "BackendLBPolicy (SessionPersistence)", "Gateway API v1.1 SessionPersistence"},
	"session-cookie-name":    {StatusPartial, "BackendLBPolicy", "Cookie name in SessionPersistence"},
	"proxy-read-timeout":     {StatusPartial, "HTTPRoute (timeouts)", "HTTPRoute spec.rules[].timeouts.backendRequest"},
	"proxy-connect-timeout":  {StatusPartial, "HTTPRoute (timeouts)", "HTTPRoute spec.rules[].timeouts.request"},
	"canary-by-header":       {StatusSupported, "HTTPRoute (header match)", "Match header in HTTPRouteMatch"},
	"canary-by-header-value": {StatusSupported, "HTTPRoute (header match)", "Exact header value match"},
	"use-regex":              {StatusSupported, "HTTPRoute (PathMatch RegularExpression)", "Native regex path matching"},
	"permanent-redirect":     {StatusSupported, "HTTPRoute (RequestRedirect)", "301 redirect filter"},
	"temporal-redirect":      {StatusSupported, "HTTPRoute (RequestRedirect)", "302 redirect filter"},
	"backend-protocol":       {StatusPartial, "Gateway TLS config", "TLS backend via Gateway listener config"},
	"websocket-services":     {StatusSupported, "Native", "Gateway API supports WebSocket natively"},
	"grpc-backend":           {StatusSupported, "GRPCRoute", "Dedicated GRPCRoute resource"},
	"proxy-body-size":                          {StatusPartial, "BackendTrafficPolicy (requestBuffer)", "Envoy Gateway BackendTrafficPolicy with requestBuffer.limit"},
	"proxy-request-buffering":                  {StatusSupported, "Native", "Envoy Gateway streams requests by default (off is the default)"},
	"configuration-snippet":                    {StatusUnsupported, "", "Impact: VARIES. Raw NGINX config injection — inherently non-portable. Review snippet content to find Gateway API equivalents per feature"},
	"server-snippet":                           {StatusUnsupported, "", "Impact: VARIES. Raw NGINX server block injection — inherently non-portable. Review snippet content to find Gateway API equivalents per feature"},
	"auth-type":                                {StatusUnsupported, "", "Impact: MEDIUM. Basic/digest auth — not in core Gateway API. Use externalAuth filter (experimental v1.4) pointing to an auth service that handles basic auth"},
	"auth-secret":                              {StatusUnsupported, "", "Impact: MEDIUM. Credential secret for basic auth — not in core Gateway API. Move credentials to an external auth service"},
	"ssl-passthrough":                          {StatusSupported, "TLSRoute", "TLS passthrough via TLSRoute (GA in Gateway API v1.5)"},
	"load-balance":                             {StatusUnsupported, "", "Impact: LOW. Gateway API uses implementation-default LB (usually round-robin) — only matters if you specifically need least-connections or random. Round-robin works well for most workloads"},
	"upstream-hash-by":                         {StatusUnsupported, "", "Impact: MEDIUM. Consistent hash-based LB — not in core Gateway API. Use BackendLBPolicy SessionPersistence as an alternative for session-pinning"},
	"affinity-mode":                             {StatusPartial, "BackendLBPolicy (SessionPersistence)", "Cookie persistence in BackendLBPolicy; balanced re-balancing unavailable in spec"},
	"canary-weight-total":                       {StatusSupported, "HTTPRoute (weighted backendRefs)", "Gateway API backendRefs weights are relative; total is implicit"},
	"proxy-http-version":                        {StatusSupported, "Native", "Envoy Gateway handles HTTP/2 and HTTP/1.1 natively"},
	"session-cookie-expires":                   {StatusPartial, "BackendLBPolicy (absoluteTimeout)", "BackendLBPolicy cookieConfig.lifetimeType: Permanent + absoluteTimeout"},
	"session-cookie-max-age":                   {StatusPartial, "BackendLBPolicy (absoluteTimeout)", "BackendLBPolicy cookieConfig.absoluteTimeout field"},
	"service-upstream":                         {StatusSupported, "Native", "Gateway API routes to pod IPs natively (no kube-proxy overhead)"},
	"from-to-www-redirect":                     {StatusSupported, "HTTPRoute (RequestRedirect)", "HTTPRoute RequestRedirect filter with hostname replacement"},
	"upstream-vhost":                           {StatusSupported, "HTTPRoute (RequestHeaderModifier)", "RequestHeaderModifier filter sets Host header"},
	"secure-verify-ca-secret":                  {StatusSupported, "BackendTLSPolicy", "BackendTLSPolicy with caCertificateRefs (GA in Gateway API v1.4)"},
	// Session cookie fields not covered by BackendLBPolicy
	"session-cookie-samesite": {StatusUnsupported, "", "Impact: LOW. SameSite not configurable in BackendLBPolicy — most implementations default to Lax which is correct for modern browsers"},
	"session-cookie-path":     {StatusUnsupported, "", "Impact: LOW. Cookie path scoping not in BackendLBPolicy — cookie scoped to / by default which works for most apps"},
	"session-cookie-secure":   {StatusUnsupported, "", "Impact: LOW. Secure flag not configurable in BackendLBPolicy — most implementations set Secure=true by default when using HTTPS"},
	"session-cookie-conditional-samesite-none": {StatusUnsupported, "", "Impact: LOW. UA-conditional SameSite — modern browsers all support SameSite=None, so conditional logic is rarely needed"},
	"session-cookie-change-on-failure":         {StatusUnsupported, "", "Impact: LOW. Re-issue cookie on failure — not in Gateway API. Only matters for specific failover patterns"},
	// Proxy / buffering
	"proxy-send-timeout": {StatusUnsupported, "", "Impact: LOW. Gateway API only has backendRequest timeout (from proxy-read-timeout) — send timeout is rarely a bottleneck in practice"},
	"proxy-buffering":    {StatusUnsupported, "", "Impact: NONE. Implementation-internal buffering — Gateway API abstracts this away. No user impact"},
	// CORS extra fields — native CORS filter in Gateway API v1.5
	"cors-expose-headers": {StatusSupported, "HTTPRoute (CORS filter)", "exposeHeaders in CORS filter"},
	"cors-max-age":        {StatusSupported, "HTTPRoute (CORS filter)", "maxAge in CORS filter"},
	// Rate limiting extras
	"limit-burst-multiplier": {StatusPartial, "BackendTrafficPolicy (RateLimit)", "Burst is configurable in BackendTrafficPolicy but uses tokens, not a multiplier"},
	"limit-whitelist":        {StatusUnsupported, "", "Impact: LOW. Per-IP rate limit exemption — not in BackendTrafficPolicy. Use SecurityPolicy IP filters to allow specific IPs as a workaround"},
	// Canary cookie routing
	"canary-by-cookie": {StatusUnsupported, "", "Impact: MEDIUM. Cookie-based canary routing not in core Gateway API — use header-based canary (canary-by-header) or implementation-specific ExtensionRef"},
	// App root redirect
	"app-root": {StatusPartial, "HTTPRoute (URLRewrite)", "URLRewrite on root path can redirect / to the app root; limited to exact path"},

	// ── New annotations ──────────────────────────────────────────────────
	// Client certificate auth
	"auth-tls-secret":                          {StatusSupported, "Gateway TLS (Client Certificate)", "Gateway spec.listeners[].tls.options with client cert validation (GA in v1.5)"},
	"auth-tls-verify-client":                   {StatusSupported, "Gateway TLS (Client Certificate)", "Gateway Client Certificate validation (GA in Gateway API v1.5)"},
	"auth-tls-verify-depth":                    {StatusUnsupported, "", "Impact: LOW. Gateway API does not expose chain depth — full chain verification is the default and safe for most setups"},
	"auth-tls-error-page":                      {StatusUnsupported, "", "Impact: LOW. No Gateway API equivalent — clients see TLS error directly. Cosmetic only; auth still enforced"},
	"auth-tls-pass-certificate-to-upstream":    {StatusUnsupported, "", "Impact: MEDIUM. No standard way to forward client cert to backend in Gateway API — use implementation-specific extensions or handle in your service mesh"},
	"auth-tls-match-cn":                        {StatusUnsupported, "", "Impact: MEDIUM. No CN regex matching in Gateway API — validate CN in your application or use cert-manager policies"},

	// External auth extras
	"auth-secret-type":                         {StatusUnsupported, "", "Impact: NONE. No basic auth in core Gateway API — auth-secret-type is irrelevant since auth-secret is also unsupported"},
	"auth-method":                              {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "Auth method configurable in SecurityPolicy or externalAuth filter"},
	"auth-request-redirect":                    {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "Redirect on auth failure configurable in externalAuth filter (experimental v1.4)"},
	"auth-cache-key":                           {StatusUnsupported, "", "Impact: LOW. No auth response caching in Gateway API — add caching in your auth service instead"},
	"auth-cache-duration":                      {StatusUnsupported, "", "Impact: LOW. No auth caching — every request hits auth service. Adds latency but auth is correct"},
	"auth-keepalive":                           {StatusUnsupported, "", "Impact: NONE. NGINX-internal connection pooling — Gateway API implementations manage connections automatically"},
	"auth-keepalive-share-vars":                {StatusUnsupported, "", "Impact: NONE. NGINX-internal variable sharing — not applicable to Gateway API architecture"},
	"auth-keepalive-requests":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal optimization — not applicable"},
	"auth-keepalive-timeout":                   {StatusUnsupported, "", "Impact: NONE. NGINX-internal timeout — not applicable"},
	"auth-proxy-set-headers":                   {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "externalAuth filter supports headersToBackend for forwarding custom headers"},
	"auth-snippet":                             {StatusUnsupported, "", "Impact: VARIES. NGINX-specific snippet — review content to determine if Gateway API has equivalent features"},
	"auth-always-set-cookie":                   {StatusUnsupported, "", "Impact: LOW. Behavior depends on implementation — most Gateway API implementations forward auth response headers including Set-Cookie"},
	"auth-signin":                              {StatusPartial, "SecurityPolicy / HTTPRoute externalAuth", "Auth redirect configurable via externalAuth filter redirectURL (experimental)"},
	"auth-signin-redirect-param":               {StatusUnsupported, "", "Impact: LOW. Configure redirect param name in your auth service instead"},
	"enable-global-auth":                       {StatusUnsupported, "", "Impact: LOW. Gateway API has no global auth concept — apply SecurityPolicy per-route or use a mesh-level policy"},
	"auth-realm":                               {StatusUnsupported, "", "Impact: NONE. No basic auth in core Gateway API — realm is irrelevant"},

	// Redirect code customization
	"permanent-redirect-code":                  {StatusSupported, "HTTPRoute (RequestRedirect)", "RequestRedirect filter statusCode field supports custom codes"},
	"temporal-redirect-code":                   {StatusSupported, "HTTPRoute (RequestRedirect)", "RequestRedirect filter statusCode field supports custom codes"},
	"preserve-trailing-slash":                  {StatusUnsupported, "", "Impact: LOW. Gateway API preserves URL structure by default — trailing slash handling depends on implementation"},

	// Session affinity extras
	"session-cookie-domain":                    {StatusUnsupported, "", "Impact: MEDIUM. BackendLBPolicy cookieConfig does not support Domain attribute — cookie scoped to request host by default"},
	"affinity-canary-behavior":                 {StatusUnsupported, "", "Impact: LOW. Gateway API weighted backendRefs don't have affinity interaction — traffic splitting is stateless by default"},

	// Rate limiting extras
	"limit-rate":                               {StatusUnsupported, "", "Impact: LOW. Response bandwidth limiting (KB/s) — Gateway API rate limiting is request-count based. Rarely needed outside large file downloads"},
	"limit-rate-after":                         {StatusUnsupported, "", "Impact: LOW. Threshold before response rate limiting — same as limit-rate, not applicable"},

	// Proxy SSL / Backend TLS
	"proxy-ssl-secret":                         {StatusSupported, "BackendTLSPolicy", "BackendTLSPolicy with client certificate for mTLS to backend (GA in v1.4)"},
	"proxy-ssl-ciphers":                        {StatusUnsupported, "", "Impact: LOW. Gateway API does not expose per-backend cipher selection — implementation uses default TLS stack which covers standard ciphers"},
	"proxy-ssl-name":                           {StatusSupported, "BackendTLSPolicy", "BackendTLSPolicy hostname field for backend SNI verification"},
	"proxy-ssl-protocols":                      {StatusUnsupported, "", "Impact: LOW. No per-backend protocol version selection in Gateway API — TLS 1.2+ is default and covers all standard use cases"},
	"proxy-ssl-verify":                         {StatusSupported, "BackendTLSPolicy", "BackendTLSPolicy enables backend TLS verification with caCertificateRefs (GA in v1.4)"},
	"proxy-ssl-verify-depth":                   {StatusUnsupported, "", "Impact: LOW. Full chain verification is the default — no depth limit needed in most setups"},
	"proxy-ssl-server-name":                    {StatusSupported, "BackendTLSPolicy", "SNI automatically sent when BackendTLSPolicy hostname is configured"},

	// Proxy cookie rewriting
	"proxy-cookie-domain":                      {StatusUnsupported, "", "Impact: MEDIUM. No Set-Cookie domain rewriting in Gateway API — handle in your application or use a response header filter to strip/replace"},
	"proxy-cookie-path":                        {StatusUnsupported, "", "Impact: MEDIUM. No Set-Cookie path rewriting in Gateway API — handle in your application"},

	// Proxy redirect rewriting
	"proxy-redirect-from":                      {StatusUnsupported, "", "Impact: LOW. No proxy_redirect equivalent — Gateway API does not rewrite backend Location headers. Usually not needed if backends return correct URLs"},
	"proxy-redirect-to":                        {StatusUnsupported, "", "Impact: LOW. Part of proxy_redirect — same as proxy-redirect-from"},

	// Proxy upstream retry
	"proxy-next-upstream":                      {StatusUnsupported, "", "Impact: MEDIUM. Gateway API retry budgets are experimental (v1.3) — not yet widely implemented. Most implementations have default retry behavior"},
	"proxy-next-upstream-timeout":              {StatusUnsupported, "", "Impact: LOW. Retry timeouts not yet in Gateway API spec — implementations use their own defaults"},
	"proxy-next-upstream-tries":                {StatusUnsupported, "", "Impact: LOW. Retry attempt counts not yet in standard Gateway API — experimental retry budgets in v1.3"},

	// Proxy buffer extras
	"proxy-buffers-number":                     {StatusUnsupported, "", "Impact: NONE. Implementation-internal buffer tuning — Gateway API abstracts this away. No user impact"},
	"proxy-busy-buffers-size":                  {StatusUnsupported, "", "Impact: NONE. Implementation-internal buffer tuning — not applicable"},
	"proxy-buffer-size":                        {StatusUnsupported, "", "Impact: NONE. Implementation-internal buffer tuning — not applicable"},
	"client-body-buffer-size":                  {StatusUnsupported, "", "Impact: NONE. Implementation-internal buffer tuning — not applicable"},

	// Canary extras
	"canary-by-header-pattern":                 {StatusUnsupported, "", "Impact: MEDIUM. Gateway API HTTPRouteMatch only supports exact header matching — regex header matching requires implementation-specific extensions"},

	// Error handling
	"custom-http-errors":                       {StatusUnsupported, "", "Impact: LOW. No standard error interception in Gateway API — implement custom error pages in your backend or use implementation-specific extensions"},
	"default-backend":                          {StatusPartial, "HTTPRoute (catch-all)", "Create a catch-all HTTPRoute with / path and low priority — functions as default backend"},

	// Observability
	"enable-access-log":                        {StatusUnsupported, "", "Impact: NONE. Access logging is implementation-specific (Envoy Gateway has observability policies) — not a Gateway API spec concern"},
	"enable-opentelemetry":                     {StatusPartial, "Envoy Gateway ObservabilityPolicy", "Envoy Gateway supports OTLP via BackendTrafficPolicy; not in core Gateway API spec"},

	// Request mirroring
	"mirror-target":                            {StatusSupported, "HTTPRoute (RequestMirror filter)", "RequestMirror filter is Standard in Gateway API v1.3+ — mirrors traffic to a backendRef"},
	"mirror-request-body":                      {StatusSupported, "HTTPRoute (RequestMirror filter)", "Request body is included in mirror by default; percentage-based mirroring also available (v1.3)"},

	// Misc
	"server-alias":                             {StatusPartial, "HTTPRoute hostnames", "Add additional hostnames in HTTPRoute spec.hostnames array"},
	"satisfy":                                  {StatusUnsupported, "", "Impact: LOW. NGINX any/all auth satisfaction logic — Gateway API has no equivalent; all policies apply (AND logic). 'any' mode is rarely used"},
	"enable-modsecurity":                       {StatusUnsupported, "", "Impact: MEDIUM. No built-in WAF in Gateway API — use implementation-specific extensions or an external WAF"},
	"modsecurity-snippet":                      {StatusUnsupported, "", "Impact: MEDIUM. Requires WAF support — see enable-modsecurity note"},
	"modsecurity-transaction-id":               {StatusUnsupported, "", "Impact: LOW. WAF transaction ID — only relevant if using a WAF extension"},
	"x-forwarded-prefix":                       {StatusSupported, "HTTPRoute (RequestHeaderModifier)", "RequestHeaderModifier filter can set X-Forwarded-Prefix header"},
	"connection-proxy-header":                  {StatusUnsupported, "", "Impact: NONE. NGINX-internal Connection header override — Gateway API implementations handle WebSocket upgrade and connection headers automatically"},
	"enable-owasp-modsecurity-crs":             {StatusUnsupported, "", "Impact: MEDIUM. Requires WAF support — see enable-modsecurity note"},
	"ssl-ciphers":                              {StatusUnsupported, "", "Impact: LOW. Gateway API does not expose listener cipher configuration — implementations use secure defaults. Only matters for legacy client compatibility"},
}

// MapAnnotation returns the mapping for a given annotation key and target.
func MapAnnotation(key, value, target string) AnnotationMapping {
	var mappings map[string]struct {
		Status         MappingStatus
		TargetResource string
		Note           string
	}

	switch target {
	case "traefik":
		mappings = traefikMappings
	case "gateway-api", "gateway-api-traefik":
		mappings = gatewayAPIMappings
	default:
		return AnnotationMapping{
			OriginalKey:   key,
			OriginalValue: value,
			Status:        StatusUnsupported,
			Note:          "Unknown target",
		}
	}

	if m, ok := mappings[key]; ok {
		return AnnotationMapping{
			OriginalKey:    key,
			OriginalValue:  value,
			Status:         m.Status,
			TargetResource: m.TargetResource,
			Note:           m.Note,
		}
	}

	return AnnotationMapping{
		OriginalKey:   key,
		OriginalValue: value,
		Status:        StatusUnsupported,
		Note:          "Unknown annotation — manual review required",
	}
}
