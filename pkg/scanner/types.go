package scanner

// ScanResult is the complete output of a cluster scan.
type ScanResult struct {
	ClusterName string          `json:"clusterName"`
	Controller  ControllerInfo  `json:"controller"`
	Ingresses   []IngressInfo   `json:"ingresses"`
	Namespaces  []string        `json:"namespaces"`
}

// ControllerInfo describes the detected ingress controller.
type ControllerInfo struct {
	Detected  bool   `json:"detected"`
	Type      string `json:"type"`      // "ingress-nginx" | "traefik" | "kong" | "unknown"
	Version   string `json:"version"`
	Namespace string `json:"namespace"`
	PodName   string `json:"podName"`
}

// SourceType identifies what kind of resource was scanned.
type SourceType string

const (
	SourceNginxIngress      SourceType = "nginx-ingress"
	SourceTraefikIngressRoute SourceType = "traefik-ingressroute"
)

// IngressInfo holds parsed data for a single Ingress or IngressRoute resource.
type IngressInfo struct {
	Namespace        string            `json:"namespace"`
	Name             string            `json:"name"`
	SourceType       SourceType        `json:"sourceType,omitempty"` // "nginx-ingress" | "traefik-ingressroute"
	IngressClass     string            `json:"ingressClass"`
	Hosts            []string          `json:"hosts"`
	Paths            []PathInfo        `json:"paths"`
	TLSEnabled       bool              `json:"tlsEnabled"`
	TLSSecrets       []string          `json:"tlsSecrets"`
	Annotations      map[string]string `json:"annotations"`       // All annotations
	NginxAnnotations map[string]string `json:"nginxAnnotations"`  // Extracted feature annotations (nginx or traefik pseudo-annotations)
	Middlewares      []string          `json:"middlewares,omitempty"` // Traefik middleware names referenced by this route
	Services         []ServiceRef      `json:"services"`
	Complexity       string            `json:"complexity"` // "simple" | "complex" | "unsupported"
}

// PathInfo describes a single path rule in an Ingress.
type PathInfo struct {
	Host        string `json:"host"`
	Path        string `json:"path"`
	PathType    string `json:"pathType"`
	ServiceName string `json:"serviceName"`
	ServicePort int32  `json:"servicePort"`
}

// ServiceRef is a reference to a backend service.
type ServiceRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Port      int32  `json:"port"`
}
