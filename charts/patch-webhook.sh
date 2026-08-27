#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and Konfidence contributors
# SPDX-License-Identifier: Apache-2.0
#
# Inject Helm-templated values into a controller-gen ValidatingWebhookConfiguration.
#
# Reads webhook manifests produced by `controller-gen webhook` and emits a Helm
# template with conditional rendering, dynamic service reference, and chart labels.
#
# Usage: patch-webhook.sh <chart-name> <src-dir> <dst.yaml>
#
# Example:
#   charts/patch-webhook.sh konfidence config/webhook \
#       charts/konfidence/templates/validatingwebhookconfiguration.yaml

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $(basename "$0") <chart-name> <src-dir> <dst.yaml>" >&2
  exit 2
fi

CHART="$1"
SRC_DIR="$2"
DST="$3"

# Find the ValidatingWebhookConfiguration manifest
SRC=$(find "$SRC_DIR" -name "manifests.yaml" -o -name "*.yaml" | head -n1)

if [[ ! -f "$SRC" ]]; then
  echo "error: no webhook manifest found in: $SRC_DIR" >&2
  exit 1
fi

mkdir -p "$(dirname "$DST")"

# Add Helm conditional header
echo '{{- if and .Values.controller.install .Values.webhook.enabled -}}' > "$DST"

# Process the source file line by line
in_client_config=false
in_service=false

while IFS= read -r line; do
  # Skip the document separator
  if [[ "$line" == "---" ]]; then
    continue
  fi
  
  # Replace metadata name
  if [[ "$line" =~ ^[[:space:]]*name:[[:space:]]*validating-webhook-configuration ]]; then
    echo "  name: {{ include \"$CHART.fullname\" . }}-validating-webhook-configuration" >> "$DST"
    # Add labels after name
    echo "  labels:" >> "$DST"
    echo "    {{- include \"$CHART.operatorLabels\" . | nindent 4 }}" >> "$DST"
    echo "    {{- with .Values.webhook.labels }}" >> "$DST"
    echo "    {{- toYaml . | nindent 4 }}" >> "$DST"
    echo "    {{- end }}" >> "$DST"
    # Add annotations for cert-manager CA injection support
    echo "  {{- with .Values.webhook.annotations }}" >> "$DST"
    echo "  annotations:" >> "$DST"
    echo "    {{- toYaml . | nindent 4 }}" >> "$DST"
    echo "  {{- end }}" >> "$DST"
    continue
  fi
  
  # Detect clientConfig block
  if [[ "$line" =~ ^[[:space:]]+clientConfig: ]]; then
    in_client_config=true
    echo "$line" >> "$DST"
    continue
  fi
  
  # Detect service block inside clientConfig
  if $in_client_config && [[ "$line" =~ ^[[:space:]]+service: ]]; then
    in_service=true
    echo "$line" >> "$DST"
    continue
  fi
  
  # Replace service name
  if $in_service && [[ "$line" =~ ^[[:space:]]+name:[[:space:]]+ ]]; then
    echo "      name: {{ include \"$CHART.fullname\" . }}-webhook-service" >> "$DST"
    continue
  fi
  
  # Replace service namespace and add caBundle
  if $in_service && [[ "$line" =~ ^[[:space:]]+namespace:[[:space:]]+ ]]; then
    echo "      namespace: {{ .Release.Namespace }}" >> "$DST"
    # Peek at next line to see if we need to add path before caBundle
    next_line=""
    in_service=false
    # We'll add caBundle after the next non-service line (path or other field)
    # Set a flag to add caBundle after path
    add_ca_bundle_after_path=true
    continue
  fi
  
  # Add caBundle after the path field (which comes after service block)
  if [[ -n "${add_ca_bundle_after_path:-}" ]] && [[ "$line" =~ ^[[:space:]]+path: ]]; then
    echo "$line" >> "$DST"
    echo "    {{- if .Values.webhook.caBundle }}" >> "$DST"
    echo "    caBundle: {{ .Values.webhook.caBundle | quote }}" >> "$DST"
    echo "    {{- end }}" >> "$DST"
    add_ca_bundle_after_path=""
    continue
  fi
  
  # Exit clientConfig when we see another top-level field
  if $in_client_config && [[ "$line" =~ ^[[:space:]]+[a-zA-Z]+: ]] && ! [[ "$line" =~ ^[[:space:]]+service: ]]; then
    in_client_config=false
  fi
  
  # Replace failurePolicy
  if [[ "$line" =~ ^[[:space:]]+failurePolicy: ]]; then
    echo "  failurePolicy: {{ .Values.webhook.failurePolicy }}" >> "$DST"
    continue
  fi
  
  # Output all other lines as-is
  echo "$line" >> "$DST"
  
done < "$SRC"

# Add Helm conditional footer
echo "{{- end }}" >> "$DST"

echo "Patched webhook configuration written to $DST"
