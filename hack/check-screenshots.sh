#!/usr/bin/env bash
# Regenerate in isolation and compare with colocated Linux baselines.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# -ne 1 || ! -f "$root/$1/package.json" ]]; then
	printf 'Usage: %s <workspace-directory>\n' "${0##*/}" >&2
	exit 2
fi
workspace="$1"
committed="$root/$workspace/src"
generated=$(mktemp -d)
trap 'rm -rf "$generated"' EXIT
bash "$root/hack/generate-screenshots.sh" "$workspace" "$generated"

artifacts=""
preserve_drift() {
	local relative="$1"
	if [[ -z "$artifacts" ]]; then
		mkdir -p "$root/.tmp/screenshot-drift/$workspace"
		artifacts=$(mktemp -d "$root/.tmp/screenshot-drift/$workspace/run.XXXXXX")
	fi
	if [[ -f "$committed/$relative" ]]; then
		mkdir -p "$artifacts/expected/$(dirname "$relative")"
		cp "$committed/$relative" "$artifacts/expected/$relative"
	fi
	if [[ -f "$generated/$relative" ]]; then
		mkdir -p "$artifacts/actual/$(dirname "$relative")"
		cp "$generated/$relative" "$artifacts/actual/$relative"
	fi
}

drift=0
while IFS= read -r -d '' file; do
	relative="${file#"$generated/"}"
	if [[ ! -f "$committed/$relative" ]]; then
		printf 'Missing baseline: %s\n' "$relative"
		preserve_drift "$relative"
		drift=1
	elif ! cmp -s "$file" "$committed/$relative"; then
		printf 'Changed baseline: %s\n' "$relative"
		preserve_drift "$relative"
		drift=1
	fi
done < <(find "$generated" -type f -path '*/__screenshots__/*-linux.png' -print0)
while IFS= read -r -d '' file; do
	relative="${file#"$committed/"}"
	if [[ ! -f "$generated/$relative" ]]; then
		printf 'Obsolete baseline: %s\n' "$relative"
		preserve_drift "$relative"
		drift=1
	fi
done < <(find "$committed" -type f -path '*/__screenshots__/*-linux.png' -print0)

if [[ "$drift" != 0 ]]; then
	printf 'Screenshot drift artifacts: %s\n' "$artifacts"
	printf "::error::Linux screenshots are stale. Run 'bash hack/generate-screenshots.sh %s' and review the PNG changes.\n" "$workspace"
	exit 1
fi
