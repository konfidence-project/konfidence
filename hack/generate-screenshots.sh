#!/usr/bin/env bash
# Generate a workspace's Linux screenshots into its source tree, or an optional output directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ $# -lt 1 || $# -gt 2 || ! -f "$root/$1/package.json" ]]; then
	printf 'Usage: %s <workspace-directory> [output-directory]\n' "${0##*/}" >&2
	exit 2
fi
workspace="$1"
output="${2:-$root/$workspace/src}"
mkdir -p "$output"
output="$(cd "$output" && pwd)"
generated="$(mktemp -d)"
container=""
cleanup() {
	rm -rf "$generated"
	if [[ -n "$container" ]]; then docker rm -f "$container" >/dev/null; fi
}
trap cleanup EXIT

docker build --platform linux/arm64 \
	-f "$root/hack/Dockerfile.screenshots" \
	-t konfidence-screenshots "$root"
container=$(docker create --platform linux/arm64 --ipc=host \
	konfidence-screenshots bash -e -o pipefail -c '
		pnpm --dir "$1" test:screenshots
		mkdir /screenshots
		cd "$1/src"
		find . -type f -path "*/__screenshots__/*-linux.png" -exec cp --parents -t /screenshots {} +
	' _ "$workspace")
docker start --attach "$container"
status=$(docker inspect --format '{{.State.ExitCode}}' "$container")
if [[ "$status" != 0 ]]; then
	exit "$status"
fi

docker cp "$container:/screenshots/." "$generated"
if [[ -z "$(find "$generated" -type f -path '*/__screenshots__/*-linux.png' -print -quit)" ]]; then
	printf '%s\n' 'No Linux screenshots were generated.' >&2
	exit 1
fi

# Replace only Linux PNGs after the tests and export have succeeded.
find "$output" -type f -path '*/__screenshots__/*-linux.png' -delete
cp -R "$generated/." "$output"
