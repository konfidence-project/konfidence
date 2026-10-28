#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
export PATH="$root/bin:$PATH"
source "$root/hack/e2e/versions.env"

cluster=${E2E_CLUSTER_NAME:-konfidence-e2e}
state="$root/.tmp/e2e"
export KUBECONFIG="$state/kubeconfig"

if [[ ! "$cluster" =~ ^konfidence-e2e(-[a-z0-9]+(-[a-z0-9]+)*)?$ ]]; then
  echo "E2E_CLUSTER_NAME must be konfidence-e2e or start with konfidence-e2e-." >&2
  exit 1
fi

up() {
  local ref="$KUBERNETES_LANDSCAPE_ORCHESTRATOR_REF"
  if [[ ! "$ref" =~ ^[0-9a-f]{40}$ ]]; then
    echo "KUBERNETES_LANDSCAPE_ORCHESTRATOR_REF must be a full commit SHA." >&2
    exit 1
  fi

  local orchestrator="$state/orchestrator/$ref"
  mkdir -p "$orchestrator" "$root/bin/linux-build/e2e"
  if [[ ! -f "$orchestrator/go.mod" ]]; then
    curl --fail --location --retry 3 \
      "https://github.com/konfidence-project/kubernetes-landscape-orchestrator/archive/$ref.tar.gz" \
      -o "$state/orchestrator.tar.gz"
    tar -xzf "$state/orchestrator.tar.gz" --strip-components=1 -C "$orchestrator"
  fi

  local commit tag arch built ldflags
  commit=$(git -C "$root" rev-parse HEAD)
  tag="e2e-${commit:0:12}-$(date -u +%Y%m%d%H%M%S)"
  built=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  arch=$(GOWORK=off go env GOARCH)
  ldflags="-s -w -X github.com/konfidence-project/konfidence/pkg/build.Version=$tag -X github.com/konfidence-project/konfidence/pkg/build.GitCommit=$commit -X github.com/konfidence-project/konfidence/pkg/build.Date=$built"
  export CGO_ENABLED=0 GOOS=linux GOARCH="$arch"

  cd "$root"
  pnpm install --frozen-lockfile
  pnpm ui:build
  GOWORK=off go build -trimpath -ldflags "$ldflags" \
    -o bin/linux-build/e2e/konfidence ./cmd/konfidence
  GOWORK=off go build -trimpath -ldflags "$ldflags" \
    -o bin/linux-build/e2e/api ./cmd/api

  # Build the pinned deployer against this checkout's API and library.
  printf 'go %s\n' "$(awk '/^go / { print $2; exit }' "$orchestrator/go.mod")" > "$state/go.work"
  (
    cd "$orchestrator"
    GOWORK="$state/go.work" bin/go work use "$root" "$orchestrator"
    GOWORK="$state/go.work" bin/go build -trimpath \
      -ldflags "-s -w -X github.com/konfidence-project/kubernetes-landscape-orchestrator/pkg/build.Version=e2e-${ref:0:12} -X github.com/konfidence-project/kubernetes-landscape-orchestrator/pkg/build.GitCommit=$ref" \
      -o "$root/bin/linux-build/e2e/kubernetes-landscape-orchestrator" .
  )

  local operator_image="konfidence-e2e/operator:$tag"
  local api_image="konfidence-e2e/api:$tag"
  local orchestrator_image="konfidence-e2e/orchestrator:$tag"
  docker build -f Dockerfile --build-arg TARGETPLATFORM=bin/linux-build/e2e \
    --build-arg OPERATOR_NAME=konfidence -t "$operator_image" .
  docker build -f Dockerfile.api --build-arg TARGETPLATFORM=bin/linux-build/e2e \
    -t "$api_image" .
  docker build -f Dockerfile --build-arg TARGETPLATFORM=bin/linux-build/e2e \
    --build-arg OPERATOR_NAME=kubernetes-landscape-orchestrator -t "$orchestrator_image" .

  if ! kind get clusters | grep -Fxq "$cluster"; then
    kind create cluster --name "$cluster" --image "$KIND_NODE_IMAGE" --wait 180s
  fi
  kind export kubeconfig --name "$cluster"
  kubectl config use-context "kind-$cluster"
  kind load docker-image "$operator_image" "$api_image" "$orchestrator_image" --name "$cluster"

  kubectl apply --server-side -f \
    "https://github.com/kubernetes-sigs/gateway-api/releases/download/$GATEWAY_API_VERSION/standard-install.yaml"
  flux install --export --version "$FLUX_VERSION" \
    --components=source-controller,kustomize-controller,helm-controller > "$state/flux.yaml"
  kubectl apply --server-side -f "$state/flux.yaml"
  if ! kubectl wait deployment/source-controller deployment/kustomize-controller deployment/helm-controller \
    -n flux-system --for=condition=Available --timeout=300s; then
    kubectl get pods -n flux-system
    exit 1
  fi

  helm upgrade --install konfidence "$root/charts/konfidence" \
    --namespace konfidence-system --create-namespace \
    --set image.repository=konfidence-e2e/operator --set-string "image.tag=$tag" \
    --set image.pullPolicy=Never \
    --set api.image.repository=konfidence-e2e/api --set-string "api.image.tag=$tag" \
    --set api.image.pullPolicy=Never \
    --set api.oidc.enabled=false --set api.session.cookie.secure=false \
    --set webhook.enabled=false --wait --timeout 300s
  helm upgrade --install kubernetes-landscape-orchestrator \
    "$orchestrator/charts/kubernetes-landscape-orchestrator" \
    --namespace konfidence-system \
    --set image.repository=konfidence-e2e/orchestrator --set-string "image.tag=$tag" \
    --set image.pullPolicy=Never --wait --timeout 300s

  printf 'Built commit: %s\nKubeconfig: %s\n' "$commit" "$KUBECONFIG"
}

case "${1:-}" in
  up) up ;;
  down) kind delete cluster --name "$cluster" ;;
  *) echo "Usage: bash hack/e2e/cluster.sh up|down" >&2; exit 1 ;;
esac
