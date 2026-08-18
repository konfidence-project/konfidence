#!/usr/bin/env bash
# Inject Helm-templated name and labels into a controller-gen ClusterRole.
#
# Reads a role YAML produced by `controller-gen rbac` and emits a copy
# whose ClusterRole name calls into the chart's fullname helper, whose
# metadata block carries the chart labels, and whose document is wrapped
# in a `controller.install` guard.
#
# Usage: patch-clusterrole.sh <chart-name> <src.yaml> <dst.yaml>
#
# Example (regenerate the ClusterRole after `make manifests`):
#   charts/patch-clusterrole.sh konfidence "config/rbac/role.yaml" \
#       "charts/konfidence/templates/clusterrole.yaml"

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $(basename "$0") <chart-name> <src.yaml> <dst.yaml>" >&2
  exit 2
fi

CHART="$1"
SRC="$2"
DST="$3"

if [[ ! -f "$SRC" ]]; then
  echo "error: source file not found: $SRC" >&2
  exit 1
fi

mkdir -p "$(dirname "$DST")"

NAME_TEMPLATE="{{ include \"${CHART}.fullname\" . }}-manager"
LABEL_INCLUDE="{{- include \"${CHART}.labels\" . | nindent 4 }}"

# controller-gen emits a single ClusterRole document with `metadata:` ->
# `name: <chart>-manager`. Replace the literal name with the fullname
# template and inject `labels:` immediately above it. The metadata block
# has no `annotations:` to worry about.
awk -v name="$NAME_TEMPLATE" -v label="$LABEL_INCLUDE" '
  BEGIN { in_meta = 0 }
  /^metadata:/                          { print; in_meta = 1; next }
  in_meta && /^  name:/ {
    print "  labels:"
    print label
    print "  name: " name
    in_meta = 0
    next
  }
  in_meta && /^[^ ]/ { in_meta = 0 }
  { print }
' "$SRC" > "$DST.tmp"

# Wrap the document in a controller.install guard. Strip a leading `---`
# so the guard sits cleanly above the document marker.
{
  echo '{{- if .Values.controller.install -}}'
  echo '---'
  if [[ "$(head -n1 "$DST.tmp")" == "---" ]]; then
    tail -n +2 "$DST.tmp"
  else
    cat "$DST.tmp"
  fi
  echo '{{- end }}'
} > "$DST"

rm -f "$DST.tmp"
