#!/usr/bin/env bash

set -e

# Script to setup the Kubernetes cluster with sample resources
# This script applies namespace, secret, and vector assembly configuration

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE_FILE="${SCRIPT_DIR}/namespace.yaml"
SECRET_FILE="${SCRIPT_DIR}/secret.yaml"

echo "=== Setting up Kubernetes cluster with sample resources ==="
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed or not in PATH"
    echo "Please install kubectl or ensure it's available in your PATH"
    exit 1
fi

# Check if kubectl can connect to a cluster
echo "Checking cluster connectivity..."
if ! kubectl cluster-info &> /dev/null; then
    echo "Error: Cannot connect to Kubernetes cluster"
    echo "Please ensure your cluster is running and kubectl is configured correctly"
    exit 1
fi

echo "✓ Connected to cluster"
echo ""

# Apply namespace
echo "→ Creating namespace..."
kubectl apply -f "${NAMESPACE_FILE}"
echo "✓ Namespace created"
echo ""

# Apply secret
echo "→ Creating registry credentials secret..."
kubectl apply -f "${SECRET_FILE}"
echo "✓ Secret created"
echo ""

echo "=== Setup completed successfully ==="
echo ""
echo "Resources created:"
echo "  - Namespace: konfidence-system"
echo "  - Secret: registry-credentials"
echo "  - ConfigMap: vector-assembly-configuration"
echo ""
echo "You can verify the resources with:"
echo "  kubectl get all -n konfidence-system"
echo "  kubectl get secret,configmap -n konfidence-system"