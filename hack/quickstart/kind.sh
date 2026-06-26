#!/bin/sh

# kind.sh creates a disposable kind cluster and installs Konfidence into it.

set -eu
set -x

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

cluster_name=${KIND_CLUSTER_NAME:-konfidence-quickstart}

if ! kind get clusters 2>/dev/null | grep -qx "$cluster_name"; then
  kind create cluster \
    --name "$cluster_name" \
    --wait 120s
fi

kind export kubeconfig --name "$cluster_name"

if [ -x "$script_dir/install.sh" ]; then
  "$script_dir/install.sh"
else
  curl -L "${KONFIDENCE_QUICKSTART_BASE_URL:-https://raw.githubusercontent.com/konfidence-project/konfidence/main/hack/quickstart}/install.sh" | sh
fi
