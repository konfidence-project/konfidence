package k8s

import (
	"fmt"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient builds a controller-runtime client.Client backed by the shared
// Konfidence scheme (galaxy CRDs + star CRDs + core Kubernetes types).
//
// Resolution order:
//  1. kubeconfigPath non-empty → load that file (local development).
//  2. kubeconfigPath empty     → in-cluster config via KUBERNETES_SERVICE_HOST
//     (production, running inside the star cluster pod)
//
// The returned client supports both read (List, Get) and write (Create, Update,
// Patch, Delete) operations against all registered Konfidence resources
func NewClient(kubeconfigPath string) (client.Client, error) {
	cfg, err := restConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("build REST config: %w", err)
	}

	c, err := client.New(cfg, client.Options{Scheme: NewScheme()})
	if err != nil {
		return nil, fmt.Errorf("build k8s client: %w", err)
	}

	return c, nil
}

func restConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig %q: %w", kubeconfigPath, err)
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config unavailable (set --kubeconfig for local dev): %w", err)
	}
	return cfg, nil
}
