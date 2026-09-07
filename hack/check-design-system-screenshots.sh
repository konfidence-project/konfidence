#!/usr/bin/env bash
# Regenerate in isolation and compare with colocated Linux baselines.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
committed="$root/packages/konfidence-design-system/src/components"
generated=$(mktemp -d)
trap 'rm -rf "$generated"' EXIT
bash "$root/hack/generate-design-system-screenshots.sh" "$generated"

drift=0
while IFS= read -r -d '' file; do
	relative="${file#"$generated/"}"
	if [[ ! -f "$committed/$relative" ]]; then
		printf 'Missing baseline: %s\n' "$relative"
		drift=1
	elif ! cmp -s "$file" "$committed/$relative"; then
		printf 'Changed baseline: %s\n' "$relative"
		drift=1
	fi
done < <(find "$generated" -type f -path '*/__screenshots__/*-linux.png' -print0)
while IFS= read -r -d '' file; do
	relative="${file#"$committed/"}"
	if [[ ! -f "$generated/$relative" ]]; then
		printf 'Obsolete baseline: %s\n' "$relative"
		drift=1
	fi
done < <(find "$committed" -type f -path '*/__screenshots__/*-linux.png' -print0)

if [[ "$drift" != 0 ]]; then
	echo "::error::Linux screenshots are stale. Run 'pnpm ds:screenshots:generate', review the PNG changes, and commit them."
	exit 1
fi
