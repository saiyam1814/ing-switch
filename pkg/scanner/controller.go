package scanner

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// detectController attempts to identify the ingress controller running in the cluster.
func (s *Scanner) detectController() (ControllerInfo, error) {
	// Search common namespaces
	candidateNamespaces := []string{
		"ingress-nginx", "nginx-ingress", "kube-system",
		"traefik", "kong", "default",
	}

	// Known controller label selectors
	type candidate struct {
		selector  string
		ctrlType  string
		namespace string
	}

	selectors := []candidate{
		{selector: "app.kubernetes.io/name=ingress-nginx", ctrlType: "ingress-nginx"},
		{selector: "app=ingress-nginx", ctrlType: "ingress-nginx"},
		{selector: "app.kubernetes.io/name=traefik", ctrlType: "traefik"},
		{selector: "app=traefik", ctrlType: "traefik"},
		{selector: "app=kong", ctrlType: "kong"},
		{selector: "app.kubernetes.io/name=kong", ctrlType: "kong"},
		{selector: "app.kubernetes.io/name=haproxy-ingress", ctrlType: "haproxy"},
	}

	for _, ns := range candidateNamespaces {
		for _, cand := range selectors {
			pods, err := s.client.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{
				LabelSelector: cand.selector,
				Limit:         1,
			})
			if err != nil || len(pods.Items) == 0 {
				continue
			}

			pod := pods.Items[0]
			version := extractVersion(pod.Spec.Containers)

			return ControllerInfo{
				Detected:  true,
				Type:      cand.ctrlType,
				Version:   version,
				Namespace: ns,
				PodName:   pod.Name,
			}, nil
		}
	}

	// Fallback: check all namespaces with broad search
	pods, err := s.client.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=ingress-nginx",
		Limit:         1,
	})
	if err == nil && len(pods.Items) > 0 {
		pod := pods.Items[0]
		return ControllerInfo{
			Detected:  true,
			Type:      "ingress-nginx",
			Version:   extractVersion(pod.Spec.Containers),
			Namespace: pod.Namespace,
			PodName:   pod.Name,
		}, nil
	}

	return ControllerInfo{Detected: false, Type: "unknown"}, nil
}

func extractVersion(containers interface{ }) string {
	// Use reflection-free approach — just return unknown for now
	// In a real cluster, we'd inspect the container image tag
	return "unknown"
}

// extractVersionFromImage parses the image tag to get a version string.
func extractVersionFromImage(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return "unknown"
	}
	return parts[len(parts)-1]
}
