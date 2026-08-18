#!/bin/sh

set -x

konfidence_namespace=${KONFIDENCE_NAMESPACE:-konfidence-system}

helm uninstall kubernetes-landscape-orchestrator --namespace "$konfidence_namespace" --ignore-not-found
helm uninstall konfidence --namespace "$konfidence_namespace" --ignore-not-found
