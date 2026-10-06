#!/usr/bin/env bash
# Verify the Vite-prerendered CLI pages match the committed Go embed files.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"
generated=(internal/kden/auth/pages/generated/success.html internal/kden/auth/pages/generated/failure.html)

if ! git diff --quiet -- "${generated[@]}"; then
	echo "::error::CLI pages are out of date. Run 'pnpm cli-pages:build' and commit the generated HTML."
	git --no-pager diff --stat -- "${generated[@]}"
	exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- "${generated[@]}")"
if [[ -n "$untracked" ]]; then
	echo "::error::Untracked CLI pages found. Run 'pnpm cli-pages:build' and commit the generated HTML:"
	echo "$untracked"
	exit 1
fi
