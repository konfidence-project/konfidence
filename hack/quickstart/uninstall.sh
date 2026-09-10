#!/bin/sh

set -eu
set -x

cluster_name=${KIND_CLUSTER_NAME:-konfidence-quickstart}

kind delete cluster --name "$cluster_name"
