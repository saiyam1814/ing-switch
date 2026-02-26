package scanner

import (
	"os"

	"sigs.k8s.io/yaml"
)

// Config holds optional per-project settings loaded from .ing-switch.yaml
// in the current working directory.
type Config struct {
	// IgnoreAnnotationPrefixes lists annotation key prefixes that should be
	// silently skipped during scanning. Supports full keys or prefix strings.
	// Example:
	//   ignore_annotation_prefixes:
	//     - argocd.argoproj.io/
	//     - my-company.io/internal-
	IgnoreAnnotationPrefixes []string `json:"ignore_annotation_prefixes" yaml:"ignore_annotation_prefixes"`
}

// builtinIgnorePrefixes are system-generated annotation prefixes that are
// never meaningful for ingress migration and are always skipped.
var builtinIgnorePrefixes = []string{
	"kubectl.kubernetes.io/",    // kubectl last-applied-configuration
	"argocd.argoproj.io/",       // ArgoCD tracking metadata
	"deployment.kubernetes.io/", // Deployment controller bookkeeping
	"meta.helm.sh/",             // Helm release metadata
	"helm.sh/",                  // Helm chart annotations
	"kopf.dev/",                 // Kopf operator framework
}

const configFileName = ".ing-switch.yaml"

// loadConfig reads .ing-switch.yaml from the current directory.
// Missing file is not an error — returns an empty config.
func loadConfig() Config {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

// shouldIgnoreAnnotation returns true if the annotation key matches any
// built-in or user-configured ignore prefix.
func shouldIgnoreAnnotation(key string, userPrefixes []string) bool {
	for _, prefix := range builtinIgnorePrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	for _, prefix := range userPrefixes {
		if prefix == "" {
			continue
		}
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
