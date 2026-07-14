#!/usr/bin/env bash
# Inject Helm-templated annotations and labels into a controller-gen CRD.
#
# Reads a CRD YAML produced by controller-gen and emits a copy whose
# metadata block calls into the chart's crdAnnotations / crdLabels helpers
# and whose document is wrapped in a `crd.install` guard.
#
# Usage: patch-crd.sh <chart-name> <src.yaml> <dst.yaml> [guard]
#
# The optional guard is the Helm condition wrapping the document; it
# defaults to `.Values.crd.install`. Galaxy CRDs pass
# `and .Values.crd.install .Values.galaxy.enabled` so they are skipped
# on star-only installs.
#
# Example (patch generated CRDs in place):
#   for f in charts/konfidence/templates/crds/*.yaml; do
#     charts/patch-crd.sh konfidence "$f" "$f"
#   done

set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo "usage: $(basename "$0") <chart-name> <src.yaml> <dst.yaml> [guard]" >&2
  exit 2
fi

CHART="$1"
SRC="$2"
DST="$3"
GUARD="${4:-.Values.crd.install}"

if [[ ! -f "$SRC" ]]; then
  echo "error: source file not found: $SRC" >&2
  exit 1
fi

mkdir -p "$(dirname "$DST")"

ANNOT_INCLUDE="{{- include \"${CHART}.crdAnnotations\" . | nindent 4 }}"
LABEL_INCLUDE="{{- include \"${CHART}.crdLabels\" . | nindent 4 }}"

# The metadata block has two shapes:
#   1. controller-gen emits `annotations:` (with its version stamp) — we
#      insert the templated annotation include as the first entry.
#   2. some CRDs only have `name:` — we still inject `labels:` before it.
# In both cases we add the labels block immediately above `name:`.
awk -v annot="$ANNOT_INCLUDE" -v label="$LABEL_INCLUDE" '
  BEGIN { in_meta = 0; in_annot = 0 }
  /^metadata:/                          { print; in_meta = 1; next }
  in_meta && /^  annotations:/          { print; in_annot = 1; print annot; next }
  in_meta && in_annot && /^    [a-z]/   { print; next }
  in_meta && in_annot && /^  [a-z]/     { in_annot = 0 }
  in_meta && /^  name:/ {
    print "  labels:"
    print label
    print
    in_meta = 0
    next
  }
  in_meta && /^[^ ]/ { in_meta = 0 }
  { print }
' "$SRC" > "$DST.tmp"

# Wrap the document in its install guard. Strip a leading `---` so the
# guard sits cleanly above the document marker.
{
  echo "{{- if ${GUARD} -}}"
  echo '---'
  if [[ "$(head -n1 "$DST.tmp")" == "---" ]]; then
    tail -n +2 "$DST.tmp"
  else
    cat "$DST.tmp"
  fi
  echo '{{- end }}'
} > "$DST"

rm -f "$DST.tmp"
