#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail
umask 077

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LOCAL_DIR="${LOCAL_DIR:-${REPO_ROOT}/local}"
SECRET_DIR="${LOCAL_DIR}/secrets"
GENERATED_DIR="${LOCAL_DIR}/generated/authelia"
COMPOSE_FILE="${SCRIPT_DIR}/docker-compose.yml"
CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
YQ="${YQ:-yq}"

if [[ "${1:-}" == "reset" ]]; then
  rm -rf "${SECRET_DIR}" "${LOCAL_DIR}/generated" "${LOCAL_DIR}/dev.env"
  echo "Removed generated local development secrets."
  exit 0
fi

for tool in openssl "${CONTAINER_TOOL}" "${YQ}"; do
  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "${tool} is required to generate local development secrets." >&2
    exit 1
  fi
done

AUTHELIA_IMAGE="$("${YQ}" -r '.services.authelia.image' "${COMPOSE_FILE}")"
if [[ -z "${AUTHELIA_IMAGE}" || "${AUTHELIA_IMAGE}" == "null" ]]; then
  echo "Authelia image is not configured in ${COMPOSE_FILE}." >&2
  exit 1
fi

mkdir -p "${SECRET_DIR}" "${GENERATED_DIR}"
rm -f "${LOCAL_DIR}/dev.env"

generate_random_file() {
  local path="$1"
  local bytes="$2"
  if [[ ! -s "${path}" ]]; then
    local temporary="${path}.tmp"
    openssl rand -hex "${bytes}" >"${temporary}"
    chmod 0600 "${temporary}"
    mv "${temporary}" "${path}"
  fi
}

generate_authelia_credential() {
  local algorithm="$1"
  local secret_path="$2"
  local hash_path="$3"
  local length="$4"
  local fixed_password="${5:-}"
  if [[ -s "${secret_path}" && -s "${hash_path}" ]] && \
    { [[ -z "${fixed_password}" ]] || [[ "$(<"${secret_path}")" == "${fixed_password}" ]]; }; then
    return
  fi

  local output password digest
  if [[ -n "${fixed_password}" ]]; then
    password="${fixed_password}"
    output="$("${CONTAINER_TOOL}" run --rm "${AUTHELIA_IMAGE}" \
      authelia crypto hash generate "${algorithm}" --password "${password}" --no-confirm)"
  else
    password=""
    output="$("${CONTAINER_TOOL}" run --rm "${AUTHELIA_IMAGE}" \
      authelia crypto hash generate "${algorithm}" --random --random.charset alphabetic --random.length "${length}")"
  fi
  digest=""
  while IFS= read -r line; do
    case "${line}" in
      "Random Password: "*)
        if [[ -z "${fixed_password}" ]]; then
          password="${line#Random Password: }"
        fi
        ;;
      "Digest: "*) digest="${line#Digest: }" ;;
    esac
  done <<<"${output}"
  if [[ -z "${password}" || -z "${digest}" ]]; then
    echo "Failed to generate an Authelia ${algorithm} credential." >&2
    exit 1
  fi

  printf '%s\n' "${password}" >"${secret_path}.tmp"
  printf '%s\n' "${digest}" >"${hash_path}.tmp"
  chmod 0600 "${secret_path}.tmp" "${hash_path}.tmp"
  mv "${secret_path}.tmp" "${secret_path}"
  mv "${hash_path}.tmp" "${hash_path}"
}

generate_random_file "${SECRET_DIR}/authelia-session-secret" 32
generate_random_file "${SECRET_DIR}/authelia-storage-key" 32
generate_random_file "${SECRET_DIR}/authelia-hmac-secret" 64
generate_random_file "${SECRET_DIR}/postgres-password" 24
generate_authelia_credential pbkdf2 "${SECRET_DIR}/oidc-client-secret" "${SECRET_DIR}/oidc-client-secret-hash" 32
generate_authelia_credential argon2 "${SECRET_DIR}/user-alice-password" "${SECRET_DIR}/user-alice-password-hash" 0 password
generate_authelia_credential argon2 "${SECRET_DIR}/user-devin-password" "${SECRET_DIR}/user-devin-password-hash" 0 password
generate_authelia_credential argon2 "${SECRET_DIR}/user-primo-password" "${SECRET_DIR}/user-primo-password-hash" 0 password

if [[ ! -s "${SECRET_DIR}/oidc-signing-key.pem" ]]; then
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 \
    -out "${SECRET_DIR}/oidc-signing-key.pem.tmp" 2>/dev/null
  chmod 0600 "${SECRET_DIR}/oidc-signing-key.pem.tmp"
  mv "${SECRET_DIR}/oidc-signing-key.pem.tmp" "${SECRET_DIR}/oidc-signing-key.pem"
fi

configuration_tmp="${GENERATED_DIR}/configuration.yml.tmp"
cp "${SCRIPT_DIR}/authelia/configuration.template.yml" "${configuration_tmp}"
export AUTHELIA_SESSION_SECRET="$(<"${SECRET_DIR}/authelia-session-secret")"
export AUTHELIA_STORAGE_KEY="$(<"${SECRET_DIR}/authelia-storage-key")"
export AUTHELIA_HMAC_SECRET="$(<"${SECRET_DIR}/authelia-hmac-secret")"
export OIDC_CLIENT_SECRET_HASH="$(<"${SECRET_DIR}/oidc-client-secret-hash")"
export OIDC_SIGNING_KEY_FILE="${SECRET_DIR}/oidc-signing-key.pem"
"${YQ}" -i '
  .session.secret = strenv(AUTHELIA_SESSION_SECRET) |
  .storage.encryption_key = strenv(AUTHELIA_STORAGE_KEY) |
  .identity_providers.oidc.hmac_secret = strenv(AUTHELIA_HMAC_SECRET) |
  .identity_providers.oidc.clients[0].client_secret = strenv(OIDC_CLIENT_SECRET_HASH) |
  .identity_providers.oidc.jwks[0].key = load_str(strenv(OIDC_SIGNING_KEY_FILE))
' "${configuration_tmp}"
chmod 0600 "${configuration_tmp}"
mv "${configuration_tmp}" "${GENERATED_DIR}/configuration.yml"

users_tmp="${GENERATED_DIR}/users.yml.tmp"
cp "${SCRIPT_DIR}/authelia/users.template.yml" "${users_tmp}"
export ALICE_PASSWORD_HASH="$(<"${SECRET_DIR}/user-alice-password-hash")"
export DEVIN_PASSWORD_HASH="$(<"${SECRET_DIR}/user-devin-password-hash")"
export PRIMO_PASSWORD_HASH="$(<"${SECRET_DIR}/user-primo-password-hash")"
"${YQ}" -i '
  .users.alice.password = strenv(ALICE_PASSWORD_HASH) |
  .users.devin.password = strenv(DEVIN_PASSWORD_HASH) |
  .users.primo.password = strenv(PRIMO_PASSWORD_HASH)
' "${users_tmp}"
chmod 0600 "${users_tmp}"
mv "${users_tmp}" "${GENERATED_DIR}/users.yml"

echo "Local development secrets are ready in ${SECRET_DIR}."
