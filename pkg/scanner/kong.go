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

const (
	kongAnnotationPrefix = "konghq.com/"
	kongConfigPrefix     = "configuration.konghq.com/"
)

// KongPlugin GVRs
var kongPluginGVRs = []schema.GroupVersionResource{
	{Group: "configuration.konghq.com", Version: "v1", Resource: "kongplugins"},
}

var kongClusterPluginGVRs = []schema.GroupVersionResource{
	{Group: "configuration.konghq.com", Version: "v1", Resource: "kongclusterplugins"},
}

// ScanKongIngresses finds standard Ingress resources that use Kong's ingress class
// and resolves referenced KongPlugin CRDs into pseudo-annotations.
func (s *Scanner) ScanKongIngresses(namespace string, restConfig *rest.Config) ([]IngressInfo, error) {
	// List all Ingresses first
	allIngresses, err := s.listIngresses(namespace)
	if err != nil {
		return nil, err
	}

	// Filter to Kong ingresses (ingressClass = "kong" or has konghq.com annotations)
	var kongIngresses []IngressInfo
	for _, ing := range allIngresses {
		if isKongIngress(ing) {
			kongIngresses = append(kongIngresses, ing)
		}
	}

	if len(kongIngresses) == 0 {
		return nil, nil
	}

	// Load KongPlugin CRDs for reference resolution
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return kongIngresses, nil // Return what we have without plugin resolution
	}

	pluginMap := loadKongPlugins(dynClient, namespace)
	clusterPluginMap := loadKongClusterPlugins(dynClient)

	// Process each Kong ingress
	var result []IngressInfo
	for _, ing := range kongIngresses {
		processed := processKongIngress(ing, pluginMap, clusterPluginMap)
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

// isKongIngress determines if an Ingress resource belongs to Kong.
func isKongIngress(ing IngressInfo) bool {
	if ing.IngressClass == "kong" {
		return true
	}
	for k := range ing.Annotations {
		if strings.HasPrefix(k, kongAnnotationPrefix) || strings.HasPrefix(k, kongConfigPrefix) {
			return true
		}
	}
	return false
}

// loadKongPlugins loads all KongPlugin CRDs into a map.
// Key: "namespace/name", Value: plugin spec
func loadKongPlugins(dynClient dynamic.Interface, namespace string) map[string]kongPluginSpec {
	plugins := make(map[string]kongPluginSpec)

	for _, gvr := range kongPluginGVRs {
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
		for _, p := range list.Items {
			spec := parseKongPlugin(p)
			plugins[p.GetNamespace()+"/"+p.GetName()] = spec
		}
		if len(plugins) > 0 {
			break
		}
	}

	return plugins
}

// loadKongClusterPlugins loads all KongClusterPlugin CRDs.
func loadKongClusterPlugins(dynClient dynamic.Interface) map[string]kongPluginSpec {
	plugins := make(map[string]kongPluginSpec)

	for _, gvr := range kongClusterPluginGVRs {
		list, err := dynClient.Resource(gvr).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			continue
		}
		for _, p := range list.Items {
			spec := parseKongPlugin(p)
			plugins[p.GetName()] = spec
		}
		if len(plugins) > 0 {
			break
		}
	}

	return plugins
}

type kongPluginSpec struct {
	PluginName string
	Config     map[string]interface{}
	Disabled   bool
}

func parseKongPlugin(obj unstructured.Unstructured) kongPluginSpec {
	spec := kongPluginSpec{}
	spec.PluginName, _, _ = unstructured.NestedString(obj.Object, "plugin")
	if spec.PluginName == "" {
		// Fallback: some versions use "spec.plugin"
		spec.PluginName, _, _ = unstructured.NestedString(obj.Object, "spec", "plugin")
	}

	config, exists, _ := unstructured.NestedMap(obj.Object, "config")
	if exists {
		spec.Config = config
	} else {
		config, exists, _ = unstructured.NestedMap(obj.Object, "spec", "config")
		if exists {
			spec.Config = config
		}
	}

	disabled, _, _ := unstructured.NestedBool(obj.Object, "disabled")
	spec.Disabled = disabled

	return spec
}

// processKongIngress converts a Kong Ingress into an IngressInfo with pseudo-annotations.
func processKongIngress(ing IngressInfo, pluginMap map[string]kongPluginSpec, clusterPluginMap map[string]kongPluginSpec) IngressInfo {
	ing.SourceType = SourceKongIngress
	ing.NginxAnnotations = make(map[string]string)

	// Extract Kong-specific annotations
	for k, v := range ing.Annotations {
		if strings.HasPrefix(k, kongAnnotationPrefix) {
			shortKey := strings.TrimPrefix(k, kongAnnotationPrefix)
			processKongAnnotation(&ing, shortKey, v)
		}
		if strings.HasPrefix(k, kongConfigPrefix) {
			shortKey := strings.TrimPrefix(k, kongConfigPrefix)
			processKongAnnotation(&ing, shortKey, v)
		}
	}

	// Resolve plugins referenced via konghq.com/plugins annotation
	pluginNames := ing.Annotations[kongAnnotationPrefix+"plugins"]
	if pluginNames != "" {
		for _, name := range strings.Split(pluginNames, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			ing.Plugins = append(ing.Plugins, name)

			// Try namespaced plugin first, then cluster plugin
			key := ing.Namespace + "/" + name
			if plugin, ok := pluginMap[key]; ok && !plugin.Disabled {
				resolveKongPlugin(&ing, plugin)
			} else if plugin, ok := clusterPluginMap[name]; ok && !plugin.Disabled {
				resolveKongPlugin(&ing, plugin)
			}
		}
	}

	ing.Complexity = classifyKongComplexity(&ing)
	return ing
}

// processKongAnnotation handles direct Kong annotations on the Ingress.
func processKongAnnotation(info *IngressInfo, key, value string) {
	switch key {
	case "strip-path":
		if value == "true" {
			info.NginxAnnotations["rewrite-target"] = "/"
		}
	case "protocols":
		if strings.Contains(value, "https") {
			info.NginxAnnotations["ssl-redirect"] = "true"
		}
	case "https-redirect-status-code":
		info.NginxAnnotations["ssl-redirect"] = "true"
	case "preserve-host":
		// Kong preserves host by default; if false, similar to upstream-vhost
		if value == "false" {
			info.NginxAnnotations["upstream-vhost"] = "upstream"
		}
	case "connect-timeout", "proxy-connect-timeout":
		info.NginxAnnotations["proxy-connect-timeout"] = value
	case "read-timeout", "proxy-read-timeout":
		info.NginxAnnotations["proxy-read-timeout"] = value
	case "write-timeout", "proxy-send-timeout":
		info.NginxAnnotations["proxy-send-timeout"] = value
	case "retries":
		info.NginxAnnotations["proxy-next-upstream-tries"] = value
	case "path-handling":
		// v0 or v1 — affects path matching behavior
		if value == "v0" {
			info.NginxAnnotations["use-regex"] = "true"
		}
	case "methods":
		// Kong can restrict to specific HTTP methods
		info.NginxAnnotations["cors-allow-methods"] = value
	case "snis":
		info.NginxAnnotations["ssl-passthrough"] = "true"
	}
}

// resolveKongPlugin converts a KongPlugin spec into pseudo-annotations.
func resolveKongPlugin(info *IngressInfo, plugin kongPluginSpec) {
	cfg := plugin.Config
	if cfg == nil {
		cfg = make(map[string]interface{})
	}

	switch plugin.PluginName {
	case "rate-limiting", "rate-limiting-advanced":
		if second, ok := getConfigFloat(cfg, "second"); ok {
			info.NginxAnnotations["limit-rps"] = fmt.Sprintf("%d", int(second))
		} else if minute, ok := getConfigFloat(cfg, "minute"); ok {
			info.NginxAnnotations["limit-rpm"] = fmt.Sprintf("%d", int(minute))
		}

	case "cors":
		info.NginxAnnotations["enable-cors"] = "true"
		if origins, ok := getConfigStringSlice(cfg, "origins"); ok && len(origins) > 0 {
			info.NginxAnnotations["cors-allow-origin"] = strings.Join(origins, ",")
		}
		if methods, ok := getConfigStringSlice(cfg, "methods"); ok && len(methods) > 0 {
			info.NginxAnnotations["cors-allow-methods"] = strings.Join(methods, ",")
		}
		if headers, ok := getConfigStringSlice(cfg, "headers"); ok && len(headers) > 0 {
			info.NginxAnnotations["cors-allow-headers"] = strings.Join(headers, ",")
		}
		if creds, ok := getConfigBool(cfg, "credentials"); ok && creds {
			info.NginxAnnotations["cors-allow-credentials"] = "true"
		}
		if maxAge, ok := getConfigFloat(cfg, "max_age"); ok {
			info.NginxAnnotations["cors-max-age"] = fmt.Sprintf("%d", int(maxAge))
		}
		if exposed, ok := getConfigStringSlice(cfg, "exposed_headers"); ok && len(exposed) > 0 {
			info.NginxAnnotations["cors-expose-headers"] = strings.Join(exposed, ",")
		}

	case "ip-restriction":
		if allow, ok := getConfigStringSlice(cfg, "allow"); ok && len(allow) > 0 {
			info.NginxAnnotations["whitelist-source-range"] = strings.Join(allow, ",")
		}
		if deny, ok := getConfigStringSlice(cfg, "deny"); ok && len(deny) > 0 {
			info.NginxAnnotations["denylist-source-range"] = strings.Join(deny, ",")
		}

	case "key-auth", "jwt", "oauth2":
		info.NginxAnnotations["auth-type"] = plugin.PluginName

	case "basic-auth":
		info.NginxAnnotations["auth-type"] = "basic"

	case "request-size-limiting":
		if size, ok := getConfigFloat(cfg, "allowed_payload_size"); ok {
			info.NginxAnnotations["proxy-body-size"] = fmt.Sprintf("%dm", int(size))
		}

	case "request-transformer", "request-transformer-advanced":
		// Check for header additions/removals
		if add, ok := getConfigStringSlice(cfg, "add.headers"); ok && len(add) > 0 {
			info.NginxAnnotations["custom-headers"] = strings.Join(add, ",")
		}

	case "response-transformer", "response-transformer-advanced":
		if add, ok := getConfigStringSlice(cfg, "add.headers"); ok && len(add) > 0 {
			info.NginxAnnotations["custom-headers"] = strings.Join(add, ",")
		}

	case "proxy-cache":
		// Proxy caching — no direct NGINX equivalent but note it
		info.NginxAnnotations["proxy-buffering"] = "on"

	case "bot-detection":
		info.NginxAnnotations["enable-modsecurity"] = "true"

	case "acl":
		// ACL — closest to IP restriction / auth
		info.NginxAnnotations["auth-type"] = "acl"

	case "tcp-log", "http-log", "file-log", "syslog", "datadog", "prometheus":
		info.NginxAnnotations["enable-access-log"] = "true"

	case "opentelemetry":
		info.NginxAnnotations["enable-opentelemetry"] = "true"

	case "grpc-web", "grpc-gateway":
		info.NginxAnnotations["grpc-backend"] = "true"
		info.NginxAnnotations["backend-protocol"] = "GRPC"

	case "websocket-size-limit", "websocket-validator":
		info.NginxAnnotations["websocket-services"] = "true"

	case "redirect":
		if statusCode, ok := getConfigFloat(cfg, "status_code"); ok {
			if int(statusCode) == 301 {
				info.NginxAnnotations["permanent-redirect"] = "true"
			} else {
				info.NginxAnnotations["temporal-redirect"] = "true"
			}
		}
	}
}

// Config helper functions
func getConfigFloat(cfg map[string]interface{}, key string) (float64, bool) {
	if v, ok := cfg[key]; ok {
		switch val := v.(type) {
		case float64:
			return val, true
		case int64:
			return float64(val), true
		}
	}
	return 0, false
}

func getConfigBool(cfg map[string]interface{}, key string) (bool, bool) {
	if v, ok := cfg[key]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func getConfigStringSlice(cfg map[string]interface{}, key string) ([]string, bool) {
	// Handle nested keys like "add.headers"
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 2 {
		if nested, ok := cfg[parts[0]].(map[string]interface{}); ok {
			return getConfigStringSlice(nested, parts[1])
		}
		return nil, false
	}

	if v, ok := cfg[key]; ok {
		switch val := v.(type) {
		case []interface{}:
			var result []string
			for _, item := range val {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result, len(result) > 0
		case string:
			return []string{val}, true
		}
	}
	return nil, false
}

// classifyKongComplexity assigns complexity for Kong Ingresses.
func classifyKongComplexity(info *IngressInfo) string {
	if len(info.NginxAnnotations) == 0 && len(info.Plugins) == 0 {
		return "simple"
	}

	complexKeys := map[string]bool{
		"auth-type": true, "limit-rps": true, "limit-rpm": true,
		"whitelist-source-range": true, "denylist-source-range": true,
		"enable-cors": true, "rewrite-target": true, "ssl-passthrough": true,
		"grpc-backend": true, "enable-modsecurity": true,
	}

	for k := range info.NginxAnnotations {
		if complexKeys[k] {
			return "complex"
		}
	}

	if len(info.Plugins) > 2 {
		return "complex"
	}

	return "simple"
}
