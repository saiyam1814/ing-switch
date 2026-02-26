package scanner

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Scanner holds the Kubernetes client and configuration.
type Scanner struct {
	client      kubernetes.Interface
	clusterName string
}

// NewScanner creates a Scanner connected to the Kubernetes cluster.
func NewScanner(kubeconfigPath, context string) (*Scanner, error) {
	var loadingRules *clientcmd.ClientConfigLoadingRules
	if kubeconfigPath != "" {
		// Explicit --kubeconfig flag: treat as a single file path.
		loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	} else {
		// No explicit path: use the default rules which correctly split
		// KUBECONFIG on ":" (or ";" on Windows) and merge all files,
		// falling back to ~/.kube/config when KUBECONFIG is unset.
		loadingRules = clientcmd.NewDefaultClientConfigLoadingRules()
	}

	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, err
	}

	// If current-context is empty and no --context flag was given, pick
	// the first available context rather than letting ClientConfig() fail.
	if rawConfig.CurrentContext == "" && context == "" {
		for name := range rawConfig.Contexts {
			configOverrides.CurrentContext = name
			break
		}
		if configOverrides.CurrentContext == "" {
			return nil, fmt.Errorf("kubeconfig has no contexts defined")
		}
		clientConfig = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	clusterName := rawConfig.CurrentContext
	if configOverrides.CurrentContext != "" {
		clusterName = configOverrides.CurrentContext
	}

	return &Scanner{client: client, clusterName: clusterName}, nil
}
