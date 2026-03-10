#!/usr/bin/env bash
set -euo pipefail

# ------------------------------------------------------------------------------
# This script can be used to set up a local PKI environment in a kind cluster using cert-manager.
# Designed for development and testing of OCM signing/verification features for the vector assembly controller.
# What this script does:
#  - kind cluster (created if missing)
#  - cert-manager installed (idempotent; uses latest stable manifest URL)
#  - ClusterIssuer: dev-selfsigned-bootstrap
#  - Root CA cert/key Secret: cert-manager/dev-root-ca-secret
#  - ClusterIssuer backed by that root: dev-root-ca-issuer
#  - Leaf cert/key Secret for signing: default/ocm-signing-tls
#  - Root CA PEM copied into: default/dev-root-ca ConfigMap (key: root.pem) - for ocm verifiers as trust anchor
#  - Prints + exports env vars for ocm signer/verifiers and enables them in this shell
#
# Requirements: kind, kubectl
# ------------------------------------------------------------------------------

CLUSTER_NAME="${CLUSTER_NAME:-gcp}"
CERT_MANAGER_NS="cert-manager"

SIGNING_NS="${SIGNING_NS:-default}"
SIGNING_CERT_NAME="${SIGNING_CERT_NAME:-ocm-signing-cert}"
SIGNING_SECRET_NAME="${SIGNING_SECRET_NAME:-ocm-signing-tls}"
ROOT_CM_NAME="${ROOT_CM_NAME:-dev-root-ca}"

ARTIFACT_SIGNING_CERT_NAME="${ARTIFACT_SIGNING_CERT_NAME:-ocm-artifact-signing-cert}"
ARTIFACT_SIGNING_SECRET_NAME="${ARTIFACT_SIGNING_SECRET_NAME:-ocm-artifact-signing-tls}"

need() { command -v "$1" >/dev/null 2>&1 || { echo "[error] Missing dependency: $1"; exit 1; }; }
need kubectl
need kind

# --- Ensure kind cluster exists ------------------------------------------------
if ! kind get clusters | grep -qx "${CLUSTER_NAME}"; then
  echo "[info] Creating kind cluster '${CLUSTER_NAME}'…"
  kind create cluster --name "${CLUSTER_NAME}"
else
  echo "[info] kind cluster '${CLUSTER_NAME}' already exists."
fi

KUBECONTEXT="kind-${CLUSTER_NAME}"
kubectl config use-context "${KUBECONTEXT}"

# --- Install cert-manager (idempotent) ----------------------------------------
echo "[info] Installing cert-manager (idempotent)…"
kubectl get namespace "${CERT_MANAGER_NS}" >/dev/null 2>&1 || kubectl create namespace "${CERT_MANAGER_NS}"

# Uses the official "latest" manifest (no version pinning).
echo "[info] Applying cert-manager manifest from official releases (latest)…"
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

echo "[info] Waiting for cert-manager deployments…"
kubectl -n "${CERT_MANAGER_NS}" rollout status deploy/cert-manager --timeout=180s
kubectl -n "${CERT_MANAGER_NS}" rollout status deploy/cert-manager-webhook --timeout=180s
kubectl -n "${CERT_MANAGER_NS}" rollout status deploy/cert-manager-cainjector --timeout=180s

# --- Apply PKI setup (root CA + issuer) ---------------------------------------
echo "[info] Applying PKI resources… 🔒"
kubectl apply -f - <<'YAML'
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: dev-selfsigned-bootstrap
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: dev-root-ca
  namespace: cert-manager
spec:
  isCA: true
  commonName: dev-root-ca
  secretName: dev-root-ca-secret
  privateKey:
    algorithm: RSA
    size: 2048
  issuerRef:
    name: dev-selfsigned-bootstrap
    kind: ClusterIssuer
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: dev-root-ca-issuer
spec:
  ca:
    secretName: dev-root-ca-secret
YAML

echo "[info] Waiting for root CA certificate to be Ready…"
kubectl -n "${CERT_MANAGER_NS}" wait --for=condition=Ready certificate/dev-root-ca --timeout=180s

# --- Issue leaf cert in default namespace for signing --------------------------
kubectl apply -f - <<YAML
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ${SIGNING_CERT_NAME}
  namespace: ${SIGNING_NS}
spec:
  secretName: ${SIGNING_SECRET_NAME}
  commonName: ocm-signer
  privateKey:
    algorithm: RSA
    size: 2048
  usages:
    - code signing
  issuerRef:
    name: dev-root-ca-issuer
    kind: ClusterIssuer
YAML

echo "[info] Waiting for signing leaf certificate to be Ready…"
kubectl -n "${SIGNING_NS}" wait --for=condition=Ready "certificate/${SIGNING_CERT_NAME}" --timeout=180s

# --- Copy root CA cert into signing namespace as a ConfigMap -------------------
echo "[info] Publishing root CA PEM into ConfigMap ${SIGNING_NS}/${ROOT_CM_NAME} (key: root.pem)…"
ROOT_PEM="$(
  kubectl -n "${CERT_MANAGER_NS}" get secret dev-root-ca-secret \
    -o jsonpath='{.data.tls\.crt}' | base64 -d
)"

kubectl -n "${SIGNING_NS}" create configmap "${ROOT_CM_NAME}" \
  --from-literal=tls.crt="${ROOT_PEM}" \
  --dry-run=client -o yaml | kubectl apply -f -

# --- Issue separate leaf cert for artifact signing if needed ----------------
kubectl apply -f - <<YAML
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: ${ARTIFACT_SIGNING_CERT_NAME}
  namespace: ${SIGNING_NS}
spec:
  secretName: ${ARTIFACT_SIGNING_SECRET_NAME}
  commonName: ocm-artifact-signer
  privateKey:
    algorithm: RSA
    size: 2048
  usages:
    - code signing
  issuerRef:
    name: dev-root-ca-issuer
    kind: ClusterIssuer
YAML

echo "[info] Waiting for artifact signing leaf certificate to be Ready…"
kubectl -n "${SIGNING_NS}" wait --for=condition=Ready "certificate/${ARTIFACT_SIGNING_CERT_NAME}" --timeout=180s

# --- Final: export and print env vars ------------------------------------------
export OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME="${ROOT_CM_NAME}"
export OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE="${SIGNING_NS}"

export OCM_ARTIFACT_VERIFY="true"
export OCM_VECTOR_SIGN_AND_VERIFY="true"

export OCM_RSA_SIGNING_KEY_SECRET_NAME="${SIGNING_SECRET_NAME}"
export OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE="${SIGNING_NS}"

echo
echo "[done] cert-manager + dev PKI ready. ✨"
echo "       Signer Secret:     ${SIGNING_NS}/${SIGNING_SECRET_NAME}  (tls.key, tls.crt)"
echo "       Verifier trust CA: ${SIGNING_NS}/${ROOT_CM_NAME}         (root.pem)"
echo "       Artifact Signer Secret: ${SIGNING_NS}/${ARTIFACT_SIGNING_SECRET_NAME} (tls.key, tls.crt)"
echo
echo "[env] Exported in this shell:"
cat <<EOF
export OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME="${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME}"
export OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE="${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE}"

export OCM_RSA_SIGNING_KEY_SECRET_NAME="${OCM_RSA_SIGNING_KEY_SECRET_NAME}"
export OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE="${OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE}"

export OCM_ARTIFACT_VERIFY="${OCM_ARTIFACT_VERIFY}"
export OCM_VECTOR_SIGN_AND_VERIFY="${OCM_VECTOR_SIGN_AND_VERIFY}"
EOF

echo
echo "[hint] To persist these in your current terminal, run this script via:"
echo "       source ./setup-kind-certmanager-pki.sh"
echo "[hint] If you run Goland/Intellij, you can set these env vars in the Run Configuration."

# Optionally copy env vars to clipboard (if pbcopy/xclip available)
read -p "Do you want to copy the env vars to clipboard? (y/n) " -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
  ENV_VARS="OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME=${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME};OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE=${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE};OCM_RSA_SIGNING_KEY_SECRET_NAME=${OCM_RSA_SIGNING_KEY_SECRET_NAME};OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE=${OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE};OCM_ARTIFACT_VERIFY=${OCM_ARTIFACT_VERIFY};OCM_VECTOR_SIGN_AND_VERIFY=${OCM_VECTOR_SIGN_AND_VERIFY}"
  if command -v pbcopy >/dev/null 2>&1; then
    printf "%s" "${ENV_VARS}" | pbcopy
    printf "\n[info] Env vars copied to clipboard."
  elif command -v xclip >/dev/null 2>&1; then
  ENV_VARS="OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME=${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAME};OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE=${OCM_VERIFIER_TRUST_ANCHOR_CONFIGMAP_NAMESPACE};OCM_RSA_SIGNING_KEY_SECRET_NAME=${OCM_RSA_SIGNING_KEY_SECRET_NAME};OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE=${OCM_RSA_SIGNING_KEY_SECRET_NAMESPACE};OCM_ARTIFACT_VERIFY=${OCM_ARTIFACT_VERIFY};OCM_VECTOR_SIGN_AND_VERIFY=${OCM_VECTOR_SIGN_AND_VERIFY}"
    xclip -selection clipboard <<< "${ENV_VARS}"
    printf "\n[info] Env vars copied to clipboard."
  else
    echo "[warning] No clipboard utility found (pbcopy/xclip). Please copy the env vars from above manually."
  fi
fi

# Optionally copy the artifact signing cert/key info to a tmp directly for easy access by ocm cli
echo
read -p "Do you want to boostrap the ocm cli signer config for artifacts and print them to stdout? (y/n) " -n 1 -r
if [[ $REPLY =~ ^[Yy]$ ]]; then
  # fetch cert and key from the artifact signing secret
  ARTIFACT_SIGNING_KEY=$(kubectl -n "${SIGNING_NS}" get secret "${ARTIFACT_SIGNING_SECRET_NAME}" -o jsonpath='{.data.tls\.key}' | base64 -d | awk '{print "                " $0}')
  ARTIFACT_SIGNING_CERT=$(kubectl -n "${SIGNING_NS}" get secret "${ARTIFACT_SIGNING_SECRET_NAME}" -o jsonpath='{.data.tls\.crt}' | base64 -d | awk '{print "                " $0}')
  echo
  echo "ocm config:"
  echo "-------------------"
  cat <<EOF
type: generic.config.ocm.software/v1
configurations:
  - type: credentials.config.ocm.software
    consumers:
      - identity:
          type: OCIRepository
          hostname: localhost
          port: "5100"
          scheme: http
        credentials:
          - type: Credentials/v1
            properties:
              username: ""
              password: ""
      - identity:
          type: RSA/v1alpha1
          algorithm: RSASSA-PSS
          signature: konfidence.cloud.signature.artifact.upload
        credentials:
          - type: Credentials/v1
            properties:
              private_key_pem: |
                ${ARTIFACT_SIGNING_KEY}
              public_key_pem: |
                ${ARTIFACT_SIGNING_CERT}
EOF
  echo
  echo
  echo "ocm signer spec:"
  echo "-------------------"
cat <<EOF
type: RSASigningConfiguration/v1alpha1
signatureEncodingPolicy: PEM
signatureAlgorithm: RSASSA-PSS
EOF
echo
echo "example usage with ocm cli: ocm sign component-version ghcr.io/myregistry/mycomponent:v1.0.0 --signature my-signature --config .ocmconfig --signer-spec rsassa-pss.yaml"
fi
