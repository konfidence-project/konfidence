#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

CERT_DIR="${CERT_DIR:-/tmp/k8s-webhook-server/serving-certs}"
SERVICE_NAME="${SERVICE_NAME:-konfidence-webhook-service}"
NAMESPACE="${NAMESPACE:-konfidence-system}"

echo "Generating webhook server certificates using mkcert..."
echo "  Certificate directory: ${CERT_DIR}"
echo "  Service DNS names:"
echo "    - ${SERVICE_NAME}.${NAMESPACE}.svc"
echo "    - ${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"
echo ""

# Create cert directory if it doesn't exist
mkdir -p "${CERT_DIR}"

# Generate webhook server certificate
echo "Generating webhook server certificate..."
mkcert \
  -cert-file "${CERT_DIR}/tls.crt" \
  -key-file "${CERT_DIR}/tls.key" \
  "${SERVICE_NAME}.${NAMESPACE}.svc" \
  "${SERVICE_NAME}.${NAMESPACE}.svc.cluster.local"

# Copy CA certificate for Kubernetes caBundle injection
echo "Copying CA certificate..."
CA_ROOT="$(mkcert -CAROOT)"
cp "${CA_ROOT}/rootCA.pem" "${CERT_DIR}/ca.crt"

echo ""
echo "✓ Certificates generated successfully"
echo "  CA Certificate: ${CERT_DIR}/ca.crt"
echo "  Server Certificate: ${CERT_DIR}/tls.crt"
echo "  Server Private Key: ${CERT_DIR}/tls.key"
echo ""
