#!/bin/sh

# Quickstart 
#
# You can try Konfidence on your computer by creating a local Kubernetes
# cluster with kind. This script creates the cluster and installs Konfidence.
#
#
# Prerequisites
#
# - docker
# - kind
# - kubectl
# - helm
#
#
# Run the Quickstart
#
# To create the cluster and install Konfidence, run:
#
#   sh quickstart.sh install
#
# To remove the cluster:
#
#   sh quickstart.sh uninstall
#
#
# To open the Konfidence Dashboard run this command manually in your terminal:
#
#   kubectl -n konfidence-system port-forward svc/konfidence-api 8090:8090
#
# Keep the command running and open http://localhost:8090 in your browser.
# Select "Continue with SSO" to sign in as Local Admin without an external
# identity provider.

set -eu

# Set KIND_CLUSTER_NAME to use a different cluster name. Use the same value
# when removing the cluster.
cluster_name=${KIND_CLUSTER_NAME:-konfidence-quickstart}

# Override chart versions through their environment variables.
# Each chart provides the matching container image tags.
konfidence_version=${KONFIDENCE_VERSION:-0.0.0-5ec5de1c5903a95badcaf9a1a4bae275186d0893}
kubernetes_landscape_orchestrator_version=${KUBERNETES_LANDSCAPE_ORCHESTRATOR_VERSION:-0.0.0-4f194adf3c2e211514d41c59d1a446275bb093e3}
vector_data_service_version=${VECTOR_DATA_SERVICE_VERSION:-0.0.0-4f194adf3c2e211514d41c59d1a446275bb093e3}
namespace=konfidence-system

# Reuse the cluster if it already exists and select its kubeconfig context
# for the kubectl and Helm commands below.
create_cluster() {
  clusters=$(kind get clusters)
  if ! printf '%s\n' "$clusters" | grep -Fxq -- "$cluster_name"; then
    kind create cluster \
      --name "$cluster_name" \
      --wait 120s
  fi

  kind export kubeconfig --name "$cluster_name"
}

# Re-running the installation upgrades existing Helm releases.
install_components() {
  # Gateway API provides the resource types used by the Landscape Orchestrator
  # to configure HTTP routes.
  kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml

  # Flux reconciles Helm and Kustomize deployments. Wait for its controllers
  # so deployment reconciliation is available to Konfidence.
  kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml
  kubectl wait deployment --all \
    --namespace flux-system \
    --for=condition=Available \
    --timeout=180s

  # The chart includes the controller and API, which also serves the dashboard.
  # Local sign-in needs no external identity provider. In-memory sessions need
  # no database; sign in again if the API restarts. Allow cookies over HTTP and
  # disable admission webhooks to run locally without webhook certificates.
  helm upgrade --install konfidence oci://ghcr.io/konfidence-project/charts/konfidence \
    --version "$konfidence_version" \
    --namespace "$namespace" \
    --create-namespace \
    --set api.oidc.enabled=false \
    --set api.session.storageType=in-memory \
    --set api.session.cookie.secure=false \
    --set webhook.enabled=false \
    --wait

  # The Landscape Orchestrator uses Flux to deploy Helm charts and Kustomize
  # configurations for Konfidence.
  helm upgrade --install kubernetes-landscape-orchestrator oci://ghcr.io/konfidence-project/charts/kubernetes-landscape-orchestrator \
    --version "$kubernetes_landscape_orchestrator_version" \
    --namespace "$namespace" \
    --create-namespace \
    --wait

  # Applications use the Vector Data Service to read configuration and
  # deployment results for a vector at runtime.
  helm upgrade --install vector-data-service oci://ghcr.io/konfidence-project/charts/vector-data-service \
    --version "$vector_data_service_version" \
    --namespace "$namespace" \
    --create-namespace \
    --wait
}

# Delete the selected kind cluster and all workloads and data stored in it.
delete_cluster() {
  kind delete cluster --name "$cluster_name"
}

usage() {
  printf 'Usage: sh quickstart.sh [install|uninstall]\n'
  printf 'The default action is install.\n'
}

if [ "$#" -gt 1 ]; then
  usage >&2
  exit 1
fi

case "${1:-install}" in
  install)
    set -x
    create_cluster
    install_components
    ;;
  uninstall)
    set -x
    delete_cluster
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
