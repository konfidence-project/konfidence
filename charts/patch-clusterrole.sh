#!/usr/bin/env bash
# Inject Helm-templated name and labels into a controller-gen ClusterRole.
#
# Reads a role YAML produced by `controller-gen rbac` and emits a copy
# whose ClusterRole name calls into the chart's fullname helper, whose
# metadata block carries the chart labels, and whose document is wrapped
# in a `controller.install` guard.
#
# Usage: patch-clusterrole.sh <chart-name> <src.yaml> <dst.yaml> [extra-src.yaml [guard]]
#
# When extra-src.yaml is given, the rules it contains beyond those in
# src.yaml (set difference, computed with yq) are appended wrapped in a
# Helm guard (default `.Values.galaxy.enabled`). This keeps the
# galaxy-only RBAC rules out of star-only installs while the base rules
# stay unconditional.
#
# Example (regenerate the ClusterRole after `make manifests`):
#   charts/patch-clusterrole.sh konfidence "config/rbac/star/role.yaml" \
#       "charts/konfidence/templates/clusterrole.yaml" \
#       "config/rbac/galaxy/role.yaml" ".Values.galaxy.enabled"

set -euo pipefail

if [[ $# -lt 3 || $# -gt 5 ]]; then
  echo "usage: $(basename "$0") <chart-name> <src.yaml> <dst.yaml> [extra-src.yaml [guard]]" >&2
  exit 2
fi

CHART="$1"
SRC="$2"
DST="$3"
EXTRA_SRC="${4:-}"
GUARD="${5:-.Values.galaxy.enabled}"

if [[ ! -f "$SRC" ]]; then
  echo "error: source file not found: $SRC" >&2
  exit 1
fi

if [[ -n "$EXTRA_SRC" ]]; then
  if [[ ! -f "$EXTRA_SRC" ]]; then
    echo "error: extra source file not found: $EXTRA_SRC" >&2
    exit 1
  fi
  command -v yq >/dev/null 2>&1 || {
    echo "error: yq is required to merge the guarded rules" >&2
    exit 1
  }
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

# Rules present in the extra role but not in the base role, wrapped in
# the guard so they only render when the corresponding controller set is
# enabled. controller-gen emits the rules at top-level list indentation,
# and so does yq, so the guarded block aligns with the base rules.
EXTRA_RULES=""
if [[ -n "$EXTRA_SRC" ]]; then
  # The sed dedents yq's sequence style ("    - x") to controller-gen's
  # ("  - x") so the guarded block matches the base rules' formatting.
  EXTRA_RULES="$(yq eval-all '[select(fileIndex == 0).rules[]] - [select(fileIndex == 1).rules[]]' "$EXTRA_SRC" "$SRC" | sed 's/^    - /  - /')"
  if [[ "$EXTRA_RULES" == "[]" ]]; then
    EXTRA_RULES=""
  fi
fi

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
  if [[ -n "$EXTRA_RULES" ]]; then
    echo "{{- if ${GUARD} }}"
    echo "$EXTRA_RULES"
    echo '{{- end }}'
  fi
  echo '{{- end }}'
} > "$DST"

rm -f "$DST.tmp"
