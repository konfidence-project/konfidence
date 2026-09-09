#!/usr/bin/env bash
# Generate Linux screenshots into the component tree, or an optional output directory.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-$root/packages/konfidence-design-system/src/components}"
mkdir -p "$output"
output="$(cd "$output" && pwd)"
container=""
trap 'if [[ -n "$container" ]]; then docker rm -f "$container" >/dev/null; fi' EXIT

docker build --platform linux/arm64 \
	-f "$root/packages/konfidence-design-system/Dockerfile.screenshots" \
	-t konfidence-design-system-screenshots "$root"
container=$(docker create --platform linux/arm64 --ipc=host \
	konfidence-design-system-screenshots bash -e -o pipefail -c '
		pnpm ds:test
		mkdir /screenshots
		cd packages/konfidence-design-system/src/components
		find . -type f -path "*/__screenshots__/*-linux.png" -exec cp --parents -t /screenshots {} +
	')
docker start --attach "$container"
status=$(docker inspect --format '{{.State.ExitCode}}' "$container")
if [[ "$status" != 0 ]]; then
	exit "$status"
fi

# Replace only Linux PNGs after the tests and export have succeeded.
find "$output" -type f -path '*/__screenshots__/*-linux.png' -delete
docker cp "$container:/screenshots/." "$output"
