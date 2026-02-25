package scanner

import (
	"os"
	"path/filepath"

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
	if kubeconfigPath == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfigPath = env
		} else {
			home, _ := os.UserHomeDir()
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	configOverrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		configOverrides.CurrentContext = context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, err
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
	if context != "" {
		clusterName = context
	}

	return &Scanner{client: client, clusterName: clusterName}, nil
}
