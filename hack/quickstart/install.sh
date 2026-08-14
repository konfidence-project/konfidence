#!/bin/sh

set -eu

version=${KONFIDENCE_VERSION:-0.0.1-alpha.1}
namespace=konfidence-system
image_pull_secret_args=


set -x

kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml

kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml
kubectl wait deployment/source-controller \
  --namespace flux-system \
  --for=condition=Available \
  --timeout=180s

helm upgrade --install konfidence oci://ghcr.io/konfidence-project/charts/konfidence \
  --version "$version" \
  --namespace "$namespace" \
  --create-namespace \
  --set image.repository=ghcr.io/konfidence-project/konfidence-operator \
  --set image.tag="$version" \
  $image_pull_secret_args \
  --wait

helm upgrade --install kubernetes-landscape-orchestrator oci://ghcr.io/konfidence-project/charts/kubernetes-landscape-orchestrator \
  --version "$version" \
  --namespace "$namespace" \
  --create-namespace \
  --set image.repository=ghcr.io/konfidence-project/kubernetes-landscape-orchestrator \
  --set image.tag="$version" \
  $image_pull_secret_args \
  --wait
