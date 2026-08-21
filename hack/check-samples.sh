#!/usr/bin/env bash
# Verify every top-level CRD Kind has a sample in SAMPLE_DIR.
# Exits non-zero listing any Kinds with no sample file.
set -euo pipefail

SAMPLE_DIR="${SAMPLE_DIR:-test/data/samples}"
CRD_DIR="${CRD_DIR:-test/data/crds}"

missing=""
have=$(for s in "$SAMPLE_DIR"/*.yaml; do
	yq -r 'select(.kind != null) | .kind' "$s"
done | sort -u)

for crd in "$CRD_DIR"/*.yaml; do
	kind=$(yq -r '.spec.names.kind' "$crd")
	echo "$have" | grep -qxF "$kind" || missing="$missing $kind"
done

if [ -n "$missing" ]; then
	echo "::error::top-level CRD Kinds with no sample in $SAMPLE_DIR:$missing"
	echo "Add a sample per Kind so it gets a validated #### Example in the CRD reference."
	exit 1
fi

echo "All top-level CRD Kinds have a sample."
