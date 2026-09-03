#!/usr/bin/env bash
# Verify a committed openapi-typescript schema matches the OpenAPI spec.
# Usage: check-openapi-types.sh <spec> <committed-schema>
# Invoked via `pnpm api:check` from a package dir, so openapi-typescript
# and oxfmt resolve from node_modules/.bin.
set -euo pipefail

spec="$1"
schema="$2"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

generated="$tmpdir/$(basename "$schema")"
openapi-typescript "$spec" -o "$generated"
oxfmt "$generated"

if ! cmp -s "$schema" "$generated"; then
	echo "::error::$schema is out of date with $spec. Run 'pnpm api:generate' and commit the result."
	diff -u "$schema" "$generated" | head -40 || true
	exit 1
fi
