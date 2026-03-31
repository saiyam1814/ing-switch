package scanner

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// IngressRoute GVRs — try traefik.io first, fall back to traefik.containo.us
var ingressRouteGVRs = []schema.GroupVersionResource{
	{Group: "traefik.io", Version: "v1alpha1", Resource: "ingressroutes"},
	{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes"},
}

var middlewareGVRs = []schema.GroupVersionResource{
	{Group: "traefik.io", Version: "v1alpha1", Resource: "middlewares"},
	{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "middlewares"},
}

// ScanIngressRoutes discovers Traefik IngressRoute CRDs and converts them
// to IngressInfo with pseudo-annotations that the analyzer can process.
func (s *Scanner) ScanIngressRoutes(namespace string, restConfig *rest.Config) ([]IngressInfo, error) {
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	// Find which GVR is available
	irGVR, err := findAvailableGVR(dynClient, ingressRouteGVRs, namespace)
	if err != nil {
		return nil, nil // No IngressRoute CRDs installed — not an error
	}

	// List all IngressRoutes
	var irList *unstructured.UnstructuredList
	if namespace != "" {
		irList, err = dynClient.Resource(irGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	} else {
		irList, err = dynClient.Resource(irGVR).Namespace("").List(context.Background(), metav1.ListOptions{})
	}
	if err != nil {
		return nil, nil // CRDs exist but listing failed — skip
	}

	if len(irList.Items) == 0 {
		return nil, nil
	}

	// Load middlewares for resolution
	mwMap := loadMiddlewares(dynClient, namespace)

	var infos []IngressInfo
	for _, ir := range irList.Items {
		info := parseIngressRoute(ir, mwMap)
		if info != nil {
			infos = append(infos, *info)
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Namespace != infos[j].Namespace {
			return infos[i].Namespace < infos[j].Namespace
		}
		return infos[i].Name < infos[j].Name
	})

	return infos, nil
}

// findAvailableGVR tries each GVR and returns the first one that works.
func findAvailableGVR(dynClient dynamic.Interface, gvrs []schema.GroupVersionResource, namespace string) (schema.GroupVersionResource, error) {
	for _, gvr := range gvrs {
		var err error
		if namespace != "" {
			_, err = dynClient.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{Limit: 1})
		} else {
			_, err = dynClient.Resource(gvr).Namespace("").List(context.Background(), metav1.ListOptions{Limit: 1})
		}
		if err == nil {
			return gvr, nil
		}
	}
	return schema.GroupVersionResource{}, fmt.Errorf("no IngressRoute CRD found")
}

// loadMiddlewares loads all Traefik Middleware CRDs into a map for reference lookup.
// Key: "namespace/name", Value: parsed middleware spec.
func loadMiddlewares(dynClient dynamic.Interface, namespace string) map[string]map[string]interface{} {
	mwMap := make(map[string]map[string]interface{})

	for _, gvr := range middlewareGVRs {
		var list *unstructured.UnstructuredList
		var err error
		if namespace != "" {
			list, err = dynClient.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
		} else {
			list, err = dynClient.Resource(gvr).Namespace("").List(context.Background(), metav1.ListOptions{})
		}
		if err != nil {
			continue
		}
		for _, mw := range list.Items {
			ns := mw.GetNamespace()
			name := mw.GetName()
			spec, ok := mw.Object["spec"].(map[string]interface{})
			if ok {
				mwMap[ns+"/"+name] = spec
			}
		}
		if len(mwMap) > 0 {
			break // Found middlewares with this GVR
		}
	}

	return mwMap
}

// parseIngressRoute converts a Traefik IngressRoute to an IngressInfo.
func parseIngressRoute(ir unstructured.Unstructured, mwMap map[string]map[string]interface{}) *IngressInfo {
	spec, ok := ir.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	info := &IngressInfo{
		Namespace:        ir.GetNamespace(),
		Name:             ir.GetName(),
		SourceType:       SourceTraefikIngressRoute,
		Annotations:      ir.GetAnnotations(),
		NginxAnnotations: make(map[string]string),
	}

	// EntryPoints
	if eps, ok := spec["entryPoints"].([]interface{}); ok {
		for _, ep := range eps {
			if epStr, ok := ep.(string); ok {
				if epStr == "websecure" || epStr == "https" {
					info.TLSEnabled = true
				}
			}
		}
	}

	// TLS
	if tls, ok := spec["tls"].(map[string]interface{}); ok {
		info.TLSEnabled = true
		if secretName, ok := tls["secretName"].(string); ok && secretName != "" {
			info.TLSSecrets = append(info.TLSSecrets, secretName)
		}
		if passthrough, ok := tls["passthrough"].(bool); ok && passthrough {
			info.NginxAnnotations["ssl-passthrough"] = "true"
		}
	}

	// Routes
	routes, _ := spec["routes"].([]interface{})
	hostSet := make(map[string]bool)
	serviceSet := make(map[string]ServiceRef)

	for _, r := range routes {
		route, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		matchStr, _ := route["match"].(string)
		hosts, paths := parseTraefikMatch(matchStr)

		for _, h := range hosts {
			hostSet[h] = true
		}

		// Services
		services, _ := route["services"].([]interface{})
		for _, svc := range services {
			svcMap, ok := svc.(map[string]interface{})
			if !ok {
				continue
			}
			svcName, _ := svcMap["name"].(string)
			svcPort := int32(0)
			if p, ok := svcMap["port"].(int64); ok {
				svcPort = int32(p)
			} else if p, ok := svcMap["port"].(float64); ok {
				svcPort = int32(p)
			}

			// Weighted services
			if weight, ok := svcMap["weight"].(int64); ok && weight > 0 {
				info.NginxAnnotations["canary"] = "true"
				info.NginxAnnotations["canary-weight"] = fmt.Sprintf("%d", weight)
			} else if weight, ok := svcMap["weight"].(float64); ok && weight > 0 {
				info.NginxAnnotations["canary"] = "true"
				info.NginxAnnotations["canary-weight"] = fmt.Sprintf("%d", int(weight))
			}

			// Sticky sessions
			if sticky, ok := svcMap["sticky"].(map[string]interface{}); ok {
				if cookie, ok := sticky["cookie"].(map[string]interface{}); ok {
					info.NginxAnnotations["affinity"] = "cookie"
					if name, ok := cookie["name"].(string); ok {
						info.NginxAnnotations["session-cookie-name"] = name
					}
					if sameSite, ok := cookie["sameSite"].(string); ok {
						info.NginxAnnotations["session-cookie-samesite"] = sameSite
					}
					if secure, ok := cookie["secure"].(bool); ok && secure {
						info.NginxAnnotations["session-cookie-secure"] = "true"
					}
				}
			}

			for _, host := range hosts {
				for _, pathStr := range paths {
					if pathStr == "" {
						pathStr = "/"
					}
					pi := PathInfo{
						Host:        host,
						Path:        pathStr,
						PathType:    "Prefix",
						ServiceName: svcName,
						ServicePort: svcPort,
					}
					info.Paths = append(info.Paths, pi)
				}
			}

			if svcName != "" {
				key := ir.GetNamespace() + "/" + svcName
				serviceSet[key] = ServiceRef{
					Namespace: ir.GetNamespace(),
					Name:      svcName,
					Port:      svcPort,
				}
			}
		}

		// Middlewares
		middlewares, _ := route["middlewares"].([]interface{})
		for _, mw := range middlewares {
			mwRef, ok := mw.(map[string]interface{})
			if !ok {
				continue
			}
			mwName, _ := mwRef["name"].(string)
			mwNs, _ := mwRef["namespace"].(string)
			if mwNs == "" {
				mwNs = ir.GetNamespace()
			}
			info.Middlewares = append(info.Middlewares, mwNs+"/"+mwName)

			// Resolve middleware spec and generate pseudo-annotations
			resolveMiddleware(info, mwNs+"/"+mwName, mwMap)
		}
	}

	for h := range hostSet {
		info.Hosts = append(info.Hosts, h)
	}
	sort.Strings(info.Hosts)

	for _, svc := range serviceSet {
		info.Services = append(info.Services, svc)
	}

	// If no paths were found but we have hosts, add a default / path
	if len(info.Paths) == 0 && len(info.Hosts) > 0 {
		for _, h := range info.Hosts {
			info.Paths = append(info.Paths, PathInfo{
				Host:     h,
				Path:     "/",
				PathType: "Prefix",
			})
		}
	}

	info.Complexity = classifyTraefikComplexity(info)
	return info
}

// resolveMiddleware looks up a middleware spec and adds pseudo-annotations.
func resolveMiddleware(info *IngressInfo, key string, mwMap map[string]map[string]interface{}) {
	spec, ok := mwMap[key]
	if !ok {
		return
	}

	// RateLimit
	if rl, ok := spec["rateLimit"].(map[string]interface{}); ok {
		if avg, ok := rl["average"].(float64); ok {
			info.NginxAnnotations["limit-rps"] = fmt.Sprintf("%d", int(avg))
		} else if avg, ok := rl["average"].(int64); ok {
			info.NginxAnnotations["limit-rps"] = fmt.Sprintf("%d", avg)
		}
		if burst, ok := rl["burst"].(float64); ok {
			info.NginxAnnotations["limit-burst-multiplier"] = fmt.Sprintf("%d", int(burst))
		} else if burst, ok := rl["burst"].(int64); ok {
			info.NginxAnnotations["limit-burst-multiplier"] = fmt.Sprintf("%d", burst)
		}
	}

	// ForwardAuth
	if fa, ok := spec["forwardAuth"].(map[string]interface{}); ok {
		if addr, ok := fa["address"].(string); ok {
			info.NginxAnnotations["auth-url"] = addr
		}
		if headers, ok := fa["authResponseHeaders"].([]interface{}); ok {
			var hList []string
			for _, h := range headers {
				if hs, ok := h.(string); ok {
					hList = append(hList, hs)
				}
			}
			if len(hList) > 0 {
				info.NginxAnnotations["auth-response-headers"] = strings.Join(hList, ",")
			}
		}
	}

	// BasicAuth
	if _, ok := spec["basicAuth"].(map[string]interface{}); ok {
		info.NginxAnnotations["auth-type"] = "basic"
	}

	// IPAllowList / IPWhiteList
	if ipAL, ok := spec["ipAllowList"].(map[string]interface{}); ok {
		if sr, ok := ipAL["sourceRange"].([]interface{}); ok {
			var ranges []string
			for _, r := range sr {
				if rs, ok := r.(string); ok {
					ranges = append(ranges, rs)
				}
			}
			info.NginxAnnotations["whitelist-source-range"] = strings.Join(ranges, ",")
		}
	}
	if ipWL, ok := spec["ipWhiteList"].(map[string]interface{}); ok {
		if sr, ok := ipWL["sourceRange"].([]interface{}); ok {
			var ranges []string
			for _, r := range sr {
				if rs, ok := r.(string); ok {
					ranges = append(ranges, rs)
				}
			}
			info.NginxAnnotations["whitelist-source-range"] = strings.Join(ranges, ",")
		}
	}

	// Headers (CORS)
	if hdrs, ok := spec["headers"].(map[string]interface{}); ok {
		if origins, ok := hdrs["accessControlAllowOriginList"].([]interface{}); ok && len(origins) > 0 {
			info.NginxAnnotations["enable-cors"] = "true"
			var originList []string
			for _, o := range origins {
				if os, ok := o.(string); ok {
					originList = append(originList, os)
				}
			}
			info.NginxAnnotations["cors-allow-origin"] = strings.Join(originList, ",")
		}
		if methods, ok := hdrs["accessControlAllowMethods"].([]interface{}); ok {
			var mList []string
			for _, m := range methods {
				if ms, ok := m.(string); ok {
					mList = append(mList, ms)
				}
			}
			info.NginxAnnotations["cors-allow-methods"] = strings.Join(mList, ",")
		}
		if headers, ok := hdrs["accessControlAllowHeaders"].([]interface{}); ok {
			var hList []string
			for _, h := range headers {
				if hs, ok := h.(string); ok {
					hList = append(hList, hs)
				}
			}
			info.NginxAnnotations["cors-allow-headers"] = strings.Join(hList, ",")
		}
		if creds, ok := hdrs["accessControlAllowCredentials"].(bool); ok && creds {
			info.NginxAnnotations["cors-allow-credentials"] = "true"
		}
		if maxAge, ok := hdrs["accessControlMaxAge"].(float64); ok {
			info.NginxAnnotations["cors-max-age"] = fmt.Sprintf("%d", int(maxAge))
		} else if maxAge, ok := hdrs["accessControlMaxAge"].(int64); ok {
			info.NginxAnnotations["cors-max-age"] = fmt.Sprintf("%d", maxAge)
		}
		if sslRedirect, ok := hdrs["sslRedirect"].(bool); ok && sslRedirect {
			info.NginxAnnotations["ssl-redirect"] = "true"
		}
		if forceSSL, ok := hdrs["forceSTSHeader"].(bool); ok && forceSSL {
			info.NginxAnnotations["force-ssl-redirect"] = "true"
		}
	}

	// StripPrefix
	if sp, ok := spec["stripPrefix"].(map[string]interface{}); ok {
		if prefixes, ok := sp["prefixes"].([]interface{}); ok && len(prefixes) > 0 {
			if p, ok := prefixes[0].(string); ok {
				info.NginxAnnotations["rewrite-target"] = p
			}
		}
	}

	// ReplacePath
	if rp, ok := spec["replacePath"].(map[string]interface{}); ok {
		if path, ok := rp["path"].(string); ok {
			info.NginxAnnotations["rewrite-target"] = path
		}
	}

	// RedirectScheme
	if rs, ok := spec["redirectScheme"].(map[string]interface{}); ok {
		if scheme, ok := rs["scheme"].(string); ok && scheme == "https" {
			info.NginxAnnotations["ssl-redirect"] = "true"
		}
	}

	// Buffering
	if buf, ok := spec["buffering"].(map[string]interface{}); ok {
		if maxReq, ok := buf["maxRequestBodyBytes"].(float64); ok {
			info.NginxAnnotations["proxy-body-size"] = fmt.Sprintf("%dm", int(maxReq/1024/1024))
		} else if maxReq, ok := buf["maxRequestBodyBytes"].(int64); ok {
			info.NginxAnnotations["proxy-body-size"] = fmt.Sprintf("%dm", maxReq/1024/1024)
		}
	}

	// InFlightReq
	if ifr, ok := spec["inFlightReq"].(map[string]interface{}); ok {
		if amount, ok := ifr["amount"].(float64); ok {
			info.NginxAnnotations["limit-connections"] = fmt.Sprintf("%d", int(amount))
		} else if amount, ok := ifr["amount"].(int64); ok {
			info.NginxAnnotations["limit-connections"] = fmt.Sprintf("%d", amount)
		}
	}

	// Retry
	if retry, ok := spec["retry"].(map[string]interface{}); ok {
		if attempts, ok := retry["attempts"].(float64); ok {
			info.NginxAnnotations["proxy-next-upstream-tries"] = fmt.Sprintf("%d", int(attempts))
		} else if attempts, ok := retry["attempts"].(int64); ok {
			info.NginxAnnotations["proxy-next-upstream-tries"] = fmt.Sprintf("%d", attempts)
		}
	}
}

// parseTraefikMatch parses a Traefik match expression and extracts hosts and paths.
// Examples:
//
//	"Host(`example.com`) && PathPrefix(`/api`)"
//	"Host(`a.com`) || Host(`b.com`)"
//	"HostRegexp(`{subdomain:.+}.example.com`)"
var (
	hostPattern       = regexp.MustCompile(`Host\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	hostRegexpPattern = regexp.MustCompile(`HostRegexp\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	pathPrefixPattern = regexp.MustCompile(`PathPrefix\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
	pathPattern       = regexp.MustCompile(`Path\(` + "`" + `([^` + "`" + `]+)` + "`" + `\)`)
)

func parseTraefikMatch(match string) (hosts []string, paths []string) {
	// Extract hosts
	for _, m := range hostPattern.FindAllStringSubmatch(match, -1) {
		if len(m) > 1 {
			hosts = append(hosts, m[1])
		}
	}
	for _, m := range hostRegexpPattern.FindAllStringSubmatch(match, -1) {
		if len(m) > 1 {
			// Convert Traefik host regexp to a simple hostname if possible
			host := m[1]
			// Strip {name:pattern} capture groups
			host = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(host, "*")
			hosts = append(hosts, host)
		}
	}

	// Extract paths
	for _, m := range pathPrefixPattern.FindAllStringSubmatch(match, -1) {
		if len(m) > 1 {
			paths = append(paths, m[1])
		}
	}
	for _, m := range pathPattern.FindAllStringSubmatch(match, -1) {
		if len(m) > 1 {
			paths = append(paths, m[1])
		}
	}

	if len(paths) == 0 {
		paths = []string{"/"}
	}

	return hosts, paths
}

// classifyTraefikComplexity assigns a complexity tier for Traefik IngressRoutes.
func classifyTraefikComplexity(info *IngressInfo) string {
	if len(info.NginxAnnotations) == 0 && len(info.Middlewares) == 0 {
		return "simple"
	}

	complexKeys := map[string]bool{
		"auth-url": true, "canary": true, "limit-rps": true,
		"whitelist-source-range": true, "rewrite-target": true,
		"enable-cors": true, "ssl-passthrough": true,
	}

	for k := range info.NginxAnnotations {
		if complexKeys[k] {
			return "complex"
		}
	}

	if len(info.Middlewares) > 2 {
		return "complex"
	}

	return "simple"
}
