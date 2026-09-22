#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CERT_DIR="${CERT_DIR:-${REPO_ROOT}/local/tls}"
MKCERT="${MKCERT:-mkcert}"
CERT_FILE="${CERT_DIR}/local-dev.pem"
KEY_FILE="${CERT_DIR}/local-dev-key.pem"
CA_FILE="${CERT_DIR}/rootCA.pem"

if ! command -v "${MKCERT}" >/dev/null 2>&1; then
  echo "mkcert is required; activate the Hermit environment with 'source ./bin/activate-hermit'." >&2
  exit 1
fi

echo "Installing the mkcert development CA in the operating system trust store..."
"${MKCERT}" -install

mkdir -p "${CERT_DIR}"

echo "Generating the local development TLS certificate..."
"${MKCERT}" \
  -cert-file "${CERT_FILE}" \
  -key-file "${KEY_FILE}" \
  localhost \
  api.localhost \
  auth.localhost \
  id.localhost \
  ui.localhost \
  127.0.0.1 \
  ::1 \
  host.docker.internal
chmod 0600 "${KEY_FILE}"

cp "$("${MKCERT}" -CAROOT)/rootCA.pem" "${CA_FILE}"

echo "Local development TLS is ready in ${CERT_DIR}."
