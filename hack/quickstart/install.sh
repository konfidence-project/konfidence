#!/bin/sh

set -eu

version=${KONFIDENCE_VERSION:-0.0.1-alpha.1}
namespace=konfidence-system
image_pull_secret_args=

# TODO(blocker): remove this private GHCR auth block once quickstart artifacts are public.
# BEGIN private GHCR auth block
if [ -n "${GHCR_TOKEN:-}" ]; then
  if [ -z "${GHCR_USERNAME:-}" ]; then
    echo "GHCR_USERNAME must be set when GHCR_TOKEN is set" >&2
    exit 1
  fi
  printf '%s' "$GHCR_TOKEN" | helm registry login ghcr.io \
    --username "$GHCR_USERNAME" \
    --password-stdin
  kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
  kubectl create secret docker-registry ghcr-auth \
    --namespace "$namespace" \
    --docker-server=ghcr.io \
    --docker-username="$GHCR_USERNAME" \
    --docker-password="$GHCR_TOKEN" \
    --dry-run=client \
    -o yaml | kubectl apply -f -
  image_pull_secret_args="--set imagePullSecrets[0].name=ghcr-auth"
fi
# END private GHCR auth block

set -x

kubectl apply --server-side -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.1/standard-install.yaml

kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml
kubectl wait deployment/source-controller \
  --namespace flux-system \
  --for=condition=Available \
  --timeout=180s

helm upgrade --install galaxy oci://ghcr.io/konfidence-project/charts/galaxy \
  --version "$version" \
  --namespace "$namespace" \
  --create-namespace \
  --set image.repository=ghcr.io/konfidence-project/galaxy-operator \
  --set image.tag="$version" \
  $image_pull_secret_args \
  --wait

helm upgrade --install star oci://ghcr.io/konfidence-project/charts/star \
  --version "$version" \
  --namespace "$namespace" \
  --create-namespace \
  --set image.repository=ghcr.io/konfidence-project/star-operator \
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
