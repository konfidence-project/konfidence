#!/bin/bash
set -e

# Get the root of the git repository
REPO_ROOT=$(git rev-parse --show-toplevel)

# Pull docker images
docker pull --quiet alpine:3.22.1
docker pull --quiet stefanprodan/podinfo:6.9.1

# Add component versions
mkdir -p "$REPO_ROOT/ocm-samples/ocm-transfer"
ocm add componentversions --create --file "$REPO_ROOT/ocm-samples/ocm-transfer/artifact-1" "$REPO_ROOT/ocm-samples/sample-service-1.yaml"
ocm add componentversions --create --file "$REPO_ROOT/ocm-samples/ocm-transfer/artifact-2" "$REPO_ROOT/ocm-samples/sample-service-2.yaml"
ocm add componentversions --create --file "$REPO_ROOT/ocm-samples/ocm-transfer/vector1" "$REPO_ROOT/ocm-samples/vector.yaml"

# Transfer CTFs to OCI registry
ocm transfer ctf "$REPO_ROOT/ocm-samples/ocm-transfer/artifact-1" http://localhost:5100/sample-project --overwrite
ocm transfer ctf "$REPO_ROOT/ocm-samples/ocm-transfer/artifact-2" http://localhost:5100/sample-project --overwrite
ocm transfer ctf "$REPO_ROOT/ocm-samples/ocm-transfer/vector1" http://localhost:5100/sample-project --overwrite
