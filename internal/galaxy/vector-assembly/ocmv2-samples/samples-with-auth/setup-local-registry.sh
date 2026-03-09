#!/usr/bin/env bash

set -e

# Script to setup the local OCM registry
# This script adds component versions to the local registry using the OCM CLI

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OCM_BINARY="${SCRIPT_DIR}/ocm/ocm-v2"
OCM_CONFIG="${SCRIPT_DIR}/ocm/ocm-config.yaml"
COMPONENT_CONSTRUCTOR="${SCRIPT_DIR}/ocm/component-constructor.yaml"
REGISTRY_URL="http://localhost:5100"

echo "=== Setting up local OCM registry ==="
echo "Registry URL: ${REGISTRY_URL}"
echo "OCM Config: ${OCM_CONFIG}"
echo "Component Constructor: ${COMPONENT_CONSTRUCTOR}"
echo ""

# Check if OCM binary exists
if [ ! -f "${OCM_BINARY}" ]; then
    echo "Error: OCM binary not found at ${OCM_BINARY}"
    echo "Please ensure the ocm-v2 binary is available in the ocm/ directory"
    exit 1
fi

# Check if OCM binary is executable
if [ ! -x "${OCM_BINARY}" ]; then
    echo "Making OCM binary executable..."
    chmod +x "${OCM_BINARY}"
fi

# Check if config file exists
if [ ! -f "${OCM_CONFIG}" ]; then
    echo "Error: OCM config file not found at ${OCM_CONFIG}"
    exit 1
fi

# Check if component constructor file exists
if [ ! -f "${COMPONENT_CONSTRUCTOR}" ]; then
    echo "Error: Component constructor file not found at ${COMPONENT_CONSTRUCTOR}"
    exit 1
fi

# Check if registry is reachable
echo "Checking if registry is reachable..."
if ! curl -s --fail "${REGISTRY_URL}/v2/" > /dev/null; then
    echo "Warning: Registry at ${REGISTRY_URL} is not reachable"
    echo "Please ensure the registry is running (e.g., using docker-compose in the docker/ directory)"
    echo ""
fi

# Add component versions to the registry
echo "Adding component versions to the local registry..."
"${OCM_BINARY}" add component-version \
    --config "${OCM_CONFIG}" \
    --repository "${REGISTRY_URL}" \
    --constructor "${COMPONENT_CONSTRUCTOR}"

echo ""
echo "=== Setup completed successfully ==="
echo "Component versions have been added to the registry at ${REGISTRY_URL}"