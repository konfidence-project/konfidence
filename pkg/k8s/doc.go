// Package k8s provides a shared Kubernetes client and scheme for all Konfidence
// binaries (cmd/api, cmd/kden) that need access to the Konfidence CRD space on
// the star cluster.
//
// # Scheme
//
// NewScheme registers the galaxy and star CRDs alongside the core Kubernetes
// types into a single runtime.Scheme. Any client.Client built from this scheme
// can read and write the full Konfidence resource space without additional
// registration steps in each binary.
//
// # Client
//
// NewClient builds a controller-runtime client.Client that supports both read
// and write operations. Resolution order:
//
//   - kubeconfigPath non-empty → load that file (local development with Hermit).
//   - kubeconfigPath empty     → in-cluster config via KUBERNETES_SERVICE_HOST
//     (production, running inside the star cluster pod).
package k8s
