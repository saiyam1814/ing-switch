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
	"proxy-buffering":                          {StatusUnsupported, "", "No direct Traefik equivalent"},
	"proxy-body-size":                          {StatusPartial, "Middleware (Buffering)", "Traefik Buffering middleware with maxRequestBodyBytes"},
	"proxy-request-buffering":                  {StatusPartial, "Native (off by default)", "Off is default behavior; enabling request buffering requires Buffering middleware"},
	"client-body-buffer-size":                  {StatusUnsupported, "", "No Traefik equivalent"},
	"configuration-snippet":                    {StatusUnsupported, "", "NGINX-specific; intentionally not supported"},
	"server-snippet":                           {StatusUnsupported, "", "NGINX-specific; intentionally not supported"},
	"ssl-passthrough":                          {StatusPartial, "Traefik TCP router", "Requires TCP entrypoint config"},
	"backend-protocol":                         {StatusPartial, "Service annotation", "HTTPS/GRPC backends need ServersTransport"},
	"websocket-services":                       {StatusSupported, "Native", "Traefik supports WebSocket natively"},
	"grpc-backend":                             {StatusPartial, "ServersTransport + h2c", "gRPC requires h2c configuration"},
	"upstream-hash-by":                         {StatusUnsupported, "", "Hash-based LB not supported in Traefik Ingress"},
	"load-balance":                             {StatusUnsupported, "", "Traefik uses round-robin; custom LB not via Ingress"},
	"affinity-mode":                             {StatusPartial, "Service (sticky cookie)", "Traefik always uses persistent affinity; balanced re-balancing is not available"},
	"canary-weight-total":                       {StatusSupported, "Weighted Services", "Traefik uses relative weights; total is implicit"},
	"proxy-http-version":                        {StatusPartial, "ServersTransport CRD", "HTTP/2 via ServersTransport; HTTP/1.0 not supported"},
	"session-cookie-expires":                   {StatusPartial, "Service (sticky cookie maxage)", "Convert seconds to service.sticky.cookie.maxage annotation on Service"},
	"session-cookie-max-age":                   {StatusSupported, "Service (sticky cookie maxage)", "Maps to service.sticky.cookie.maxage annotation on Service"},
	"session-cookie-conditional-samesite-none": {StatusUnsupported, "", "Traefik sets SameSite statically — no UA-conditional logic"},
	"session-cookie-change-on-failure":         {StatusUnsupported, "", "No Traefik equivalent (traefik/traefik#1299)"},
	"service-upstream":                         {StatusSupported, "Service annotation (nativelb)", "Set traefik.ingress.kubernetes.io/service.nativelb: 'true' on Service"},
	"from-to-www-redirect":                     {StatusPartial, "Middleware (RedirectRegex)", "Requires RedirectRegex middleware + separate Ingress entry for www"},
	"upstream-vhost":                           {StatusPartial, "Middleware (Headers) + passhostheader", "Headers middleware sets Host header; disable passhostheader on Service"},
	"secure-verify-ca-secret":                  {StatusPartial, "ServersTransport (rootCAs)", "ServersTransport CRD referencing the CA secret"},
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
	"enable-cors":            {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Manual CORS headers; no native CORS filter in v1"},
	"cors-allow-origin":      {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Allow-Origin header"},
	"cors-allow-methods":     {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Allow-Methods header"},
	"cors-allow-headers":     {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Allow-Headers header"},
	"cors-allow-credentials": {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Allow-Credentials header"},
	"auth-url":               {StatusPartial, "SecurityPolicy (ExtensionRef)", "Envoy Gateway SecurityPolicy with ext-auth"},
	"auth-response-headers":  {StatusPartial, "SecurityPolicy (ExtensionRef)", "Part of SecurityPolicy ext-auth config"},
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
	"configuration-snippet":                    {StatusUnsupported, "", "NGINX-specific; no equivalent"},
	"server-snippet":                           {StatusUnsupported, "", "NGINX-specific; no equivalent"},
	"auth-type":                                {StatusUnsupported, "", "No basic auth in core Gateway API"},
	"auth-secret":                              {StatusUnsupported, "", "No basic auth in core Gateway API"},
	"ssl-passthrough":                          {StatusPartial, "TLSRoute", "TLS passthrough via TLSRoute"},
	"load-balance":                             {StatusUnsupported, "", "Not configurable via Gateway API"},
	"upstream-hash-by":                         {StatusUnsupported, "", "Not in core Gateway API"},
	"affinity-mode":                             {StatusPartial, "BackendLBPolicy (SessionPersistence)", "Cookie persistence in BackendLBPolicy; balanced re-balancing unavailable in spec"},
	"canary-weight-total":                       {StatusSupported, "HTTPRoute (weighted backendRefs)", "Gateway API backendRefs weights are relative; total is implicit"},
	"proxy-http-version":                        {StatusSupported, "Native", "Envoy Gateway handles HTTP/2 and HTTP/1.1 natively"},
	"session-cookie-expires":                   {StatusPartial, "BackendLBPolicy (absoluteTimeout)", "BackendLBPolicy cookieConfig.lifetimeType: Permanent + absoluteTimeout"},
	"session-cookie-max-age":                   {StatusPartial, "BackendLBPolicy (absoluteTimeout)", "BackendLBPolicy cookieConfig.absoluteTimeout field"},
	"session-cookie-conditional-samesite-none": {StatusUnsupported, "", "No Gateway API or Envoy Gateway equivalent"},
	"session-cookie-change-on-failure":         {StatusUnsupported, "", "No Gateway API equivalent"},
	"service-upstream":                         {StatusSupported, "Native", "Gateway API routes to pod IPs natively (no kube-proxy overhead)"},
	"from-to-www-redirect":                     {StatusSupported, "HTTPRoute (RequestRedirect)", "HTTPRoute RequestRedirect filter with hostname replacement"},
	"upstream-vhost":                           {StatusSupported, "HTTPRoute (RequestHeaderModifier)", "RequestHeaderModifier filter sets Host header"},
	"secure-verify-ca-secret":                  {StatusPartial, "BackendTLSPolicy", "BackendTLSPolicy with caCertificateRefs for backend TLS verification"},
	// Session cookie fields not covered by BackendLBPolicy
	"session-cookie-samesite": {StatusUnsupported, "", "SameSite is not configurable in BackendLBPolicy; no Gateway API equivalent"},
	"session-cookie-path":     {StatusUnsupported, "", "Cookie path scoping is not in BackendLBPolicy spec"},
	"session-cookie-secure":   {StatusUnsupported, "", "Secure flag is not configurable in BackendLBPolicy spec"},
	// Proxy / buffering
	"proxy-send-timeout": {StatusUnsupported, "", "No Gateway API equivalent; only backendRequest timeout (from proxy-read-timeout) is supported"},
	"proxy-buffering":    {StatusUnsupported, "", "No Gateway API or Envoy Gateway equivalent"},
	// CORS extra fields — can be set as raw response headers via ResponseHeaderModifier
	"cors-expose-headers": {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Expose-Headers via ResponseHeaderModifier"},
	"cors-max-age":        {StatusPartial, "HTTPRoute (ResponseHeaderModifier)", "Set Access-Control-Max-Age via ResponseHeaderModifier"},
	// Rate limiting extras
	"limit-burst-multiplier": {StatusPartial, "BackendTrafficPolicy (RateLimit)", "Burst is configurable in BackendTrafficPolicy but uses tokens, not a multiplier"},
	"limit-whitelist":        {StatusUnsupported, "", "Per-IP rate limit exemption list is not in BackendTrafficPolicy spec"},
	// Canary cookie routing
	"canary-by-cookie": {StatusUnsupported, "", "Cookie-based canary routing requires ExtensionRef; not in core Gateway API"},
	// App root redirect
	"app-root": {StatusPartial, "HTTPRoute (URLRewrite)", "URLRewrite on root path can redirect / to the app root; limited to exact path"},
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
	case "gateway-api":
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
