#!/usr/bin/env bash
# Verify the generated Helm chart manifests are in sync with the Go types.
# Run after `make manifests`; exits non-zero on drift or untracked files.
set -euo pipefail

PATHS=(
	charts/konfidence/templates/crds
	charts/konfidence/templates/clusterrole.yaml
	charts/konfidence/templates/validatingwebhookconfiguration.yaml
	charts/konfidence/README.md
)

if ! git diff --quiet -- "${PATHS[@]}"; then
	echo "::error::Generated manifests are out of date. Run 'make manifests' and commit the result."
	git --no-pager diff --stat -- "${PATHS[@]}"
	exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- "${PATHS[@]}")"
if [ -n "$untracked" ]; then
	echo "::error::Untracked generated manifests. Run 'make manifests' and commit the result:"
	echo "$untracked"
	exit 1
fi
