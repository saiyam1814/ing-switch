package scanner

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// Istio VirtualService GVRs
var virtualServiceGVRs = []schema.GroupVersionResource{
	{Group: "networking.istio.io", Version: "v1", Resource: "virtualservices"},
	{Group: "networking.istio.io", Version: "v1beta1", Resource: "virtualservices"},
	{Group: "networking.istio.io", Version: "v1alpha3", Resource: "virtualservices"},
}

// ScanIstioVirtualServices discovers Istio VirtualService CRDs and converts them
// to IngressInfo with pseudo-annotations that the analyzer can process.
func (s *Scanner) ScanIstioVirtualServices(namespace string, restConfig *rest.Config) ([]IngressInfo, error) {
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	vsGVR, err := findAvailableGVR(dynClient, virtualServiceGVRs, namespace)
	if err != nil {
		return nil, nil // No VirtualService CRDs installed
	}

	var vsList *unstructured.UnstructuredList
	if namespace != "" {
		vsList, err = dynClient.Resource(vsGVR).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	} else {
		vsList, err = dynClient.Resource(vsGVR).Namespace("").List(context.Background(), metav1.ListOptions{})
	}
	if err != nil {
		return nil, nil
	}

	if len(vsList.Items) == 0 {
		return nil, nil
	}

	var infos []IngressInfo
	for _, vs := range vsList.Items {
		info := parseVirtualService(vs)
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

// parseVirtualService converts an Istio VirtualService into an IngressInfo.
func parseVirtualService(vs unstructured.Unstructured) *IngressInfo {
	spec, ok := vs.Object["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	info := &IngressInfo{
		Namespace:        vs.GetNamespace(),
		Name:             vs.GetName(),
		SourceType:       SourceIstioVirtualService,
		Annotations:      vs.GetAnnotations(),
		NginxAnnotations: make(map[string]string),
	}

	// Hosts
	if hosts, ok := spec["hosts"].([]interface{}); ok {
		for _, h := range hosts {
			if hs, ok := h.(string); ok {
				info.Hosts = append(info.Hosts, hs)
			}
		}
	}
	sort.Strings(info.Hosts)

	// Gateways — if specified, this VS is exposed via a gateway (external traffic)
	if gateways, ok := spec["gateways"].([]interface{}); ok {
		for _, g := range gateways {
			if gs, ok := g.(string); ok {
				if gs != "mesh" {
					// External gateway means TLS is likely configured at the gateway level
					info.TLSEnabled = true
				}
			}
		}
	}

	// TLS routes
	if tlsRoutes, ok := spec["tls"].([]interface{}); ok {
		for _, tr := range tlsRoutes {
			trMap, ok := tr.(map[string]interface{})
			if !ok {
				continue
			}
			// TLS passthrough mode
			if matchList, ok := trMap["match"].([]interface{}); ok {
				for _, m := range matchList {
					mMap, ok := m.(map[string]interface{})
					if !ok {
						continue
					}
					if sniHosts, ok := mMap["sniHosts"].([]interface{}); ok && len(sniHosts) > 0 {
						info.NginxAnnotations["ssl-passthrough"] = "true"
					}
				}
			}
		}
	}

	// HTTP routes
	httpRoutes, _ := spec["http"].([]interface{})
	serviceSet := make(map[string]ServiceRef)

	for _, hr := range httpRoutes {
		httpRoute, ok := hr.(map[string]interface{})
		if !ok {
			continue
		}

		// Match conditions
		paths := parseIstioHTTPMatch(httpRoute, info)

		// Route destinations
		routeList, _ := httpRoute["route"].([]interface{})
		for _, r := range routeList {
			rMap, ok := r.(map[string]interface{})
			if !ok {
				continue
			}

			dest, _ := rMap["destination"].(map[string]interface{})
			if dest == nil {
				continue
			}

			svcHost, _ := dest["host"].(string)
			svcPort := int32(0)
			if port, ok := dest["port"].(map[string]interface{}); ok {
				if num, ok := port["number"].(float64); ok {
					svcPort = int32(num)
				} else if num, ok := port["number"].(int64); ok {
					svcPort = int32(num)
				}
			}

			// Short service name (strip .namespace.svc.cluster.local)
			svcName := svcHost
			if idx := strings.Index(svcName, "."); idx > 0 {
				svcName = svcName[:idx]
			}

			// Weighted routing (canary)
			if weight, ok := rMap["weight"].(float64); ok && weight > 0 && weight < 100 {
				info.NginxAnnotations["canary"] = "true"
				info.NginxAnnotations["canary-weight"] = fmt.Sprintf("%d", int(weight))
			} else if weight, ok := rMap["weight"].(int64); ok && weight > 0 && weight < 100 {
				info.NginxAnnotations["canary"] = "true"
				info.NginxAnnotations["canary-weight"] = fmt.Sprintf("%d", weight)
			}

			// Add header modifications
			if headers, ok := rMap["headers"].(map[string]interface{}); ok {
				resolveIstioHeaders(info, headers)
			}

			for _, p := range paths {
				pi := PathInfo{
					Host:        p.host,
					Path:        p.path,
					PathType:    p.pathType,
					ServiceName: svcName,
					ServicePort: svcPort,
				}
				info.Paths = append(info.Paths, pi)
			}

			key := vs.GetNamespace() + "/" + svcName
			serviceSet[key] = ServiceRef{
				Namespace: vs.GetNamespace(),
				Name:      svcName,
				Port:      svcPort,
			}
		}

		// Redirects
		if redirect, ok := httpRoute["redirect"].(map[string]interface{}); ok {
			resolveIstioRedirect(info, redirect)
		}

		// Rewrites
		if rewrite, ok := httpRoute["rewrite"].(map[string]interface{}); ok {
			if uri, ok := rewrite["uri"].(string); ok {
				info.NginxAnnotations["rewrite-target"] = uri
			}
		}

		// Timeout
		if timeout, ok := httpRoute["timeout"].(string); ok && timeout != "" {
			info.NginxAnnotations["proxy-read-timeout"] = timeout
		}

		// Retries
		if retries, ok := httpRoute["retries"].(map[string]interface{}); ok {
			if attempts, ok := retries["attempts"].(float64); ok {
				info.NginxAnnotations["proxy-next-upstream-tries"] = fmt.Sprintf("%d", int(attempts))
			} else if attempts, ok := retries["attempts"].(int64); ok {
				info.NginxAnnotations["proxy-next-upstream-tries"] = fmt.Sprintf("%d", attempts)
			}
			if perTryTimeout, ok := retries["perTryTimeout"].(string); ok {
				info.NginxAnnotations["proxy-connect-timeout"] = perTryTimeout
			}
		}

		// Fault injection
		if fault, ok := httpRoute["fault"].(map[string]interface{}); ok {
			resolveIstioFault(info, fault)
		}

		// CORS policy
		if corsPolicy, ok := httpRoute["corsPolicy"].(map[string]interface{}); ok {
			resolveIstioCORS(info, corsPolicy)
		}

		// Mirror (traffic shadowing)
		if mirror, ok := httpRoute["mirror"].(map[string]interface{}); ok {
			if mirrorHost, ok := mirror["host"].(string); ok {
				info.NginxAnnotations["mirror-target"] = mirrorHost
			}
		}

		// Headers at route level
		if headers, ok := httpRoute["headers"].(map[string]interface{}); ok {
			resolveIstioHeaders(info, headers)
		}
	}

	// Default path if none found
	if len(info.Paths) == 0 && len(info.Hosts) > 0 {
		for _, h := range info.Hosts {
			info.Paths = append(info.Paths, PathInfo{
				Host:     h,
				Path:     "/",
				PathType: "Prefix",
			})
		}
	}

	for _, svc := range serviceSet {
		info.Services = append(info.Services, svc)
	}

	info.Complexity = classifyIstioComplexity(info)
	return info
}

type istioPath struct {
	host     string
	path     string
	pathType string
}

// parseIstioHTTPMatch extracts paths and header/query matching from Istio HTTPMatch rules.
func parseIstioHTTPMatch(httpRoute map[string]interface{}, info *IngressInfo) []istioPath {
	matchList, _ := httpRoute["match"].([]interface{})
	if len(matchList) == 0 {
		// No match = match everything
		if len(info.Hosts) > 0 {
			var paths []istioPath
			for _, h := range info.Hosts {
				paths = append(paths, istioPath{host: h, path: "/", pathType: "Prefix"})
			}
			return paths
		}
		return []istioPath{{path: "/", pathType: "Prefix"}}
	}

	var paths []istioPath
	for _, m := range matchList {
		mMap, ok := m.(map[string]interface{})
		if !ok {
			continue
		}

		path := "/"
		pathType := "Prefix"

		if uri, ok := mMap["uri"].(map[string]interface{}); ok {
			if exact, ok := uri["exact"].(string); ok {
				path = exact
				pathType = "Exact"
			} else if prefix, ok := uri["prefix"].(string); ok {
				path = prefix
				pathType = "Prefix"
			} else if regex, ok := uri["regex"].(string); ok {
				path = regex
				pathType = "ImplementationSpecific"
				info.NginxAnnotations["use-regex"] = "true"
			}
		}

		// Header-based routing
		if headers, ok := mMap["headers"].(map[string]interface{}); ok {
			for hk, hv := range headers {
				if hvMap, ok := hv.(map[string]interface{}); ok {
					if exact, ok := hvMap["exact"].(string); ok {
						info.NginxAnnotations["canary-by-header"] = hk
						info.NginxAnnotations["canary-by-header-value"] = exact
					}
				}
			}
		}

		// Add paths for each host
		if len(info.Hosts) > 0 {
			for _, h := range info.Hosts {
				paths = append(paths, istioPath{host: h, path: path, pathType: pathType})
			}
		} else {
			paths = append(paths, istioPath{path: path, pathType: pathType})
		}
	}

	return paths
}

// resolveIstioRedirect converts Istio HTTPRedirect to pseudo-annotations.
func resolveIstioRedirect(info *IngressInfo, redirect map[string]interface{}) {
	code := 301
	if rc, ok := redirect["redirectCode"].(float64); ok {
		code = int(rc)
	} else if rc, ok := redirect["redirectCode"].(int64); ok {
		code = int(rc)
	}

	if uri, ok := redirect["uri"].(string); ok {
		if code == 301 {
			info.NginxAnnotations["permanent-redirect"] = uri
		} else {
			info.NginxAnnotations["temporal-redirect"] = uri
		}
	}

	if scheme, ok := redirect["scheme"].(string); ok && scheme == "https" {
		info.NginxAnnotations["ssl-redirect"] = "true"
	}
}

// resolveIstioCORS converts Istio CorsPolicy to pseudo-annotations.
func resolveIstioCORS(info *IngressInfo, cors map[string]interface{}) {
	info.NginxAnnotations["enable-cors"] = "true"

	if origins, ok := cors["allowOrigins"].([]interface{}); ok && len(origins) > 0 {
		var originList []string
		for _, o := range origins {
			if oMap, ok := o.(map[string]interface{}); ok {
				if exact, ok := oMap["exact"].(string); ok {
					originList = append(originList, exact)
				} else if prefix, ok := oMap["prefix"].(string); ok {
					originList = append(originList, prefix+"*")
				} else if regex, ok := oMap["regex"].(string); ok {
					originList = append(originList, regex)
				}
			}
		}
		if len(originList) > 0 {
			info.NginxAnnotations["cors-allow-origin"] = strings.Join(originList, ",")
		}
	}
	// Legacy allowOrigin field (string list)
	if origins, ok := cors["allowOrigin"].([]interface{}); ok && len(origins) > 0 {
		var originList []string
		for _, o := range origins {
			if os, ok := o.(string); ok {
				originList = append(originList, os)
			}
		}
		if len(originList) > 0 {
			info.NginxAnnotations["cors-allow-origin"] = strings.Join(originList, ",")
		}
	}

	if methods, ok := cors["allowMethods"].([]interface{}); ok {
		var mList []string
		for _, m := range methods {
			if ms, ok := m.(string); ok {
				mList = append(mList, ms)
			}
		}
		info.NginxAnnotations["cors-allow-methods"] = strings.Join(mList, ",")
	}

	if headers, ok := cors["allowHeaders"].([]interface{}); ok {
		var hList []string
		for _, h := range headers {
			if hs, ok := h.(string); ok {
				hList = append(hList, hs)
			}
		}
		info.NginxAnnotations["cors-allow-headers"] = strings.Join(hList, ",")
	}

	if exposed, ok := cors["exposeHeaders"].([]interface{}); ok {
		var eList []string
		for _, e := range exposed {
			if es, ok := e.(string); ok {
				eList = append(eList, es)
			}
		}
		info.NginxAnnotations["cors-expose-headers"] = strings.Join(eList, ",")
	}

	if creds, ok := cors["allowCredentials"].(bool); ok && creds {
		info.NginxAnnotations["cors-allow-credentials"] = "true"
	}

	if maxAge, ok := cors["maxAge"].(string); ok && maxAge != "" {
		info.NginxAnnotations["cors-max-age"] = maxAge
	}
}

// resolveIstioFault converts Istio fault injection config to pseudo-annotations.
func resolveIstioFault(info *IngressInfo, fault map[string]interface{}) {
	// Delay injection
	if delay, ok := fault["delay"].(map[string]interface{}); ok {
		if fixedDelay, ok := delay["fixedDelay"].(string); ok {
			info.NginxAnnotations["fault-delay"] = fixedDelay
		}
		if pct, ok := delay["percentage"].(map[string]interface{}); ok {
			if val, ok := pct["value"].(float64); ok {
				info.NginxAnnotations["fault-delay-percentage"] = fmt.Sprintf("%g", val)
			}
		}
	}

	// Abort injection
	if abort, ok := fault["abort"].(map[string]interface{}); ok {
		if httpStatus, ok := abort["httpStatus"].(float64); ok {
			info.NginxAnnotations["fault-abort-code"] = fmt.Sprintf("%d", int(httpStatus))
		} else if httpStatus, ok := abort["httpStatus"].(int64); ok {
			info.NginxAnnotations["fault-abort-code"] = fmt.Sprintf("%d", httpStatus)
		}
		if pct, ok := abort["percentage"].(map[string]interface{}); ok {
			if val, ok := pct["value"].(float64); ok {
				info.NginxAnnotations["fault-abort-percentage"] = fmt.Sprintf("%g", val)
			}
		}
	}
}

// resolveIstioHeaders converts Istio header manipulation to pseudo-annotations.
func resolveIstioHeaders(info *IngressInfo, headers map[string]interface{}) {
	if request, ok := headers["request"].(map[string]interface{}); ok {
		if add, ok := request["add"].(map[string]interface{}); ok {
			var parts []string
			for k, v := range add {
				if vs, ok := v.(string); ok {
					parts = append(parts, k+":"+vs)
				}
			}
			if len(parts) > 0 {
				info.NginxAnnotations["custom-headers"] = strings.Join(parts, ",")
			}
		}
		if set, ok := request["set"].(map[string]interface{}); ok {
			for k, v := range set {
				if vs, ok := v.(string); ok {
					if k == "Host" {
						info.NginxAnnotations["upstream-vhost"] = vs
					}
				}
			}
		}
	}
}

// classifyIstioComplexity assigns complexity for Istio VirtualServices.
func classifyIstioComplexity(info *IngressInfo) string {
	if len(info.NginxAnnotations) == 0 {
		return "simple"
	}

	// Istio-specific features that have no direct equivalent
	unsupportedKeys := map[string]bool{
		"fault-delay":            true,
		"fault-delay-percentage": true,
		"fault-abort-code":       true,
		"fault-abort-percentage": true,
		"mirror-target":          true,
	}

	complexKeys := map[string]bool{
		"auth-url": true, "canary": true, "limit-rps": true,
		"whitelist-source-range": true, "rewrite-target": true,
		"enable-cors": true, "ssl-passthrough": true,
		"canary-by-header": true, "use-regex": true,
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
