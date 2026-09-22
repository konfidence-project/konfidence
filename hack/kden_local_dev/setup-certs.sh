#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CERT_DIR="${DEV_CERT_DIR:-${REPO_ROOT}/local/certs}"

mkdir -p "${CERT_DIR}"

mkcert -install

if [[ ! -s "${CERT_DIR}/local-dev.pem" || ! -s "${CERT_DIR}/local-dev-key.pem" ]]; then
  mkcert \
    -cert-file "${CERT_DIR}/local-dev.pem" \
    -key-file "${CERT_DIR}/local-dev-key.pem" \
    auth.localhost \
    ui.localhost \
    host.docker.internal \
    api.localhost \
    id.localhost
fi

cp "$(mkcert -CAROOT)/rootCA.pem" "${CERT_DIR}/rootCA.pem"

if [[ ! -s "${CERT_DIR}/oidc-signing-cert.pem" || ! -s "${CERT_DIR}/oidc-signing-key.pem" ]]; then
  openssl req \
    -x509 \
    -newkey rsa:3072 \
    -sha256 \
    -days 825 \
    -nodes \
    -subj "/CN=Konfidence Local OIDC Signing" \
    -keyout "${CERT_DIR}/oidc-signing-key.pem" \
    -out "${CERT_DIR}/oidc-signing-cert.pem"
fi

chmod 0600 "${CERT_DIR}/local-dev-key.pem" "${CERT_DIR}/oidc-signing-key.pem"

echo "Local development certificates are ready in ${CERT_DIR}."
