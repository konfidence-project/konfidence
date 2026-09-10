#!/bin/sh

set -eu

konfidence_version=${KONFIDENCE_VERSION:-0.0.0-5ec5de1c5903a95badcaf9a1a4bae275186d0893}
kubernetes_landscape_orchestrator_version=${KUBERNETES_LANDSCAPE_ORCHESTRATOR_VERSION:-0.0.0-4f194adf3c2e211514d41c59d1a446275bb093e3}
vector_data_service_version=${VECTOR_DATA_SERVICE_VERSION:-0.0.0-4f194adf3c2e211514d41c59d1a446275bb093e3}
namespace=konfidence-system

set -x

kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml

kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml
kubectl wait deployment --all \
  --namespace flux-system \
  --for=condition=Available \
  --timeout=180s

helm upgrade --install konfidence oci://ghcr.io/konfidence-project/charts/konfidence \
  --version "$konfidence_version" \
  --namespace "$namespace" \
  --create-namespace \
  --set api.oidc.enabled=false \
  --set api.session.storageType=in-memory \
  --set api.session.cookie.secure=false \
  --set webhook.enabled=false \
  --wait

helm upgrade --install kubernetes-landscape-orchestrator oci://ghcr.io/konfidence-project/charts/kubernetes-landscape-orchestrator \
  --version "$kubernetes_landscape_orchestrator_version" \
  --namespace "$namespace" \
  --create-namespace \
  --wait

helm upgrade --install vector-data-service oci://ghcr.io/konfidence-project/charts/vector-data-service \
  --version "$vector_data_service_version" \
  --namespace "$namespace" \
  --create-namespace \
  --wait
