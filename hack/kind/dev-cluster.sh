#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0
#
# Create or tear down a kind cluster with a local OCI registry.
# Usage: dev-cluster.sh up|down

set -euo pipefail

CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
KIND="${KIND:-kind}"
CLUSTER_NAME="konfidence-dev"
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5001"
KIND_CONFIG="$(dirname "$0")/kind-config.yaml"

# Cluster-level prerequisites the kubernetes-landscape-orchestrator needs to
# deploy workloads. Versions mirror hack/quickstart/install.sh. Set
# SKIP_CLUSTER_DEPS=1 to skip them when working only on the API or CLI.
SKIP_CLUSTER_DEPS="${SKIP_CLUSTER_DEPS:-}"
GATEWAY_API_VERSION="${GATEWAY_API_VERSION:-v1.4.1}"

up() {
  if [ "$("${CONTAINER_TOOL}" inspect -f '{{.State.Running}}' "${REGISTRY_NAME}" 2>/dev/null || true)" != 'true' ]; then
    echo "Starting local registry container '${REGISTRY_NAME}' on localhost:${REGISTRY_PORT}..."
    "${CONTAINER_TOOL}" run -d --restart=always \
      -p "127.0.0.1:${REGISTRY_PORT}:5000" \
      --network bridge \
      --name "${REGISTRY_NAME}" \
      registry:2
  else
    echo "Registry container '${REGISTRY_NAME}' already running."
  fi

  if ! "${KIND}" get clusters | grep -qx "${CLUSTER_NAME}"; then
    echo "Creating kind cluster '${CLUSTER_NAME}'..."
    "${KIND}" create cluster --config "${KIND_CONFIG}"
  else
    echo "kind cluster '${CLUSTER_NAME}' already exists."
  fi

  # Per-node registry mirror config, read via containerd's config_path.
  REGISTRY_DIR="/etc/containerd/certs.d/localhost:${REGISTRY_PORT}"
  for node in $("${KIND}" get nodes --name "${CLUSTER_NAME}"); do
    "${CONTAINER_TOOL}" exec "${node}" mkdir -p "${REGISTRY_DIR}"
    cat <<EOF | "${CONTAINER_TOOL}" exec -i "${node}" cp /dev/stdin "${REGISTRY_DIR}/hosts.toml"
[host."http://${REGISTRY_NAME}:5000"]
EOF
  done

  if [ "$("${CONTAINER_TOOL}" inspect -f='{{json .NetworkSettings.Networks.kind}}' "${REGISTRY_NAME}")" = 'null' ]; then
    echo "Connecting '${REGISTRY_NAME}' to the kind network..."
    "${CONTAINER_TOOL}" network connect kind "${REGISTRY_NAME}"
  fi

  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:${REGISTRY_PORT}"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
EOF

  if [ -z "${SKIP_CLUSTER_DEPS}" ]; then
    install_cluster_deps
  fi

  echo "Done. Push images to localhost:${REGISTRY_PORT}/<name>:<tag> and reference them from the cluster as such."
}

# Gateway API and Flux are prerequisites of the landscape orchestrator, not of
# the konfidence operator itself; without them nothing turns a VectorDeployment
# into running workloads.
install_cluster_deps() {
  echo "Installing Gateway API ${GATEWAY_API_VERSION}..."
  kubectl apply --server-side -f \
    "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml"

  echo "Installing Flux..."
  kubectl apply -f https://github.com/fluxcd/flux2/releases/latest/download/install.yaml
  kubectl wait deployment/source-controller \
    --namespace flux-system \
    --for=condition=Available \
    --timeout=180s
}

down() {
  "${KIND}" delete cluster --name "${CLUSTER_NAME}" || true
  "${CONTAINER_TOOL}" rm -f "${REGISTRY_NAME}" >/dev/null 2>&1 || true
  echo "Deleted kind cluster '${CLUSTER_NAME}' and registry container '${REGISTRY_NAME}'."
}

case "${1:-}" in
  up) up ;;
  down) down ;;
  *)
    echo "Usage: $0 up|down" >&2
    exit 1
    ;;
esac
