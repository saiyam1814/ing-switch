package scanner

import (
	"context"
	"sort"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const nginxAnnotationPrefix = "nginx.ingress.kubernetes.io/"

// Scan performs a full cluster scan and returns a ScanResult.
func (s *Scanner) Scan(namespace string) (*ScanResult, error) {
	ingresses, err := s.listIngresses(namespace)
	if err != nil {
		return nil, err
	}

	controller, err := s.detectController()
	if err != nil {
		// Non-fatal: controller may not be detectable with limited permissions
		controller = ControllerInfo{Detected: false, Type: "unknown"}
	}

	namespaces := extractNamespaces(ingresses)

	return &ScanResult{
		ClusterName: s.clusterName,
		Controller:  controller,
		Ingresses:   ingresses,
		Namespaces:  namespaces,
	}, nil
}

func (s *Scanner) listIngresses(namespace string) ([]IngressInfo, error) {
	list, err := s.client.NetworkingV1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var infos []IngressInfo
	for _, ing := range list.Items {
		infos = append(infos, parseIngress(ing))
	}

	// Sort for deterministic output
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Namespace != infos[j].Namespace {
			return infos[i].Namespace < infos[j].Namespace
		}
		return infos[i].Name < infos[j].Name
	})

	return infos, nil
}

func parseIngress(ing networkingv1.Ingress) IngressInfo {
	info := IngressInfo{
		Namespace:        ing.Namespace,
		Name:             ing.Name,
		Annotations:      ing.Annotations,
		NginxAnnotations: make(map[string]string),
	}

	// IngressClass
	if ing.Spec.IngressClassName != nil {
		info.IngressClass = *ing.Spec.IngressClassName
	} else if cls, ok := ing.Annotations["kubernetes.io/ingress.class"]; ok {
		info.IngressClass = cls
	}

	// TLS
	if len(ing.Spec.TLS) > 0 {
		info.TLSEnabled = true
		for _, tls := range ing.Spec.TLS {
			if tls.SecretName != "" {
				info.TLSSecrets = append(info.TLSSecrets, tls.SecretName)
			}
		}
	}

	// Hosts and paths
	serviceSet := make(map[string]ServiceRef)
	hostSet := make(map[string]bool)

	for _, rule := range ing.Spec.Rules {
		if rule.Host != "" {
			hostSet[rule.Host] = true
		}
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			pi := PathInfo{
				Host: rule.Host,
				Path: path.Path,
			}
			if path.PathType != nil {
				pi.PathType = string(*path.PathType)
			}
			if path.Backend.Service != nil {
				pi.ServiceName = path.Backend.Service.Name
				if path.Backend.Service.Port.Number != 0 {
					pi.ServicePort = path.Backend.Service.Port.Number
				}
				key := ing.Namespace + "/" + path.Backend.Service.Name
				serviceSet[key] = ServiceRef{
					Namespace: ing.Namespace,
					Name:      path.Backend.Service.Name,
					Port:      path.Backend.Service.Port.Number,
				}
			}
			info.Paths = append(info.Paths, pi)
		}
	}

	for h := range hostSet {
		info.Hosts = append(info.Hosts, h)
	}
	sort.Strings(info.Hosts)

	for _, svc := range serviceSet {
		info.Services = append(info.Services, svc)
	}

	// Extract nginx annotations
	for k, v := range ing.Annotations {
		if strings.HasPrefix(k, nginxAnnotationPrefix) {
			shortKey := strings.TrimPrefix(k, nginxAnnotationPrefix)
			info.NginxAnnotations[shortKey] = v
		}
	}

	info.Complexity = classifyComplexity(info.NginxAnnotations)
	return info
}

// classifyComplexity assigns a complexity tier based on which annotations are used.
func classifyComplexity(nginx map[string]string) string {
	if len(nginx) == 0 {
		return "simple"
	}

	unsupportedInTraefik := map[string]bool{
		"proxy-body-size":         true,
		"client-body-buffer-size": true,
		"snippets":                true,
		"lua-resty-waf":           true,
		"modsecurity-snippet":     true,
	}

	complexAnnotations := map[string]bool{
		"auth-url":              true,
		"auth-response-headers": true,
		"canary":                true,
		"canary-weight":         true,
		"limit-rps":             true,
		"limit-connections":     true,
		"rewrite-target":        true,
		"use-regex":             true,
		"affinity":              true,
		"whitelist-source-range": true,
		"denylist-source-range":  true,
		"proxy-read-timeout":    true,
		"proxy-connect-timeout": true,
	}

	for k := range nginx {
		if unsupportedInTraefik[k] {
			return "unsupported"
		}
	}

	for k := range nginx {
		if complexAnnotations[k] {
			return "complex"
		}
	}

	return "simple"
}

func extractNamespaces(ingresses []IngressInfo) []string {
	nsSet := make(map[string]bool)
	for _, ing := range ingresses {
		nsSet[ing.Namespace] = true
	}
	var ns []string
	for n := range nsSet {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	return ns
}
