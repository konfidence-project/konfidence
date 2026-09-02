#!/usr/bin/env bash
# Verify all committed outputs derived from api/openapi.yaml are up to date.
set -euo pipefail

assert_output() {
	local config="$1"
	local expected="$2"
	local configured
	configured="$(awk '$1 == "output:" { print $2; exit }' "$config")"

	if [ "$configured" != "$expected" ]; then
		echo "::error::$config must generate $expected, found ${configured:-no output}. Update hack/check-openapi.sh with any intentional destination change."
		exit 1
	fi
}

assert_output api/codegen-server.yaml internal/api/openapi/server.go
assert_output api/codegen-client.yaml internal/kden/apiclient/client.go

generated=(
	internal/api/openapi/server.go
	internal/kden/apiclient/client.go
	packages/konfidence-api-client/src/schema.d.ts
)

if ! git diff --quiet -- "${generated[@]}"; then
	echo "::error::OpenAPI-generated code is out of date. Run 'make generate-api' and commit the result."
	git --no-pager diff --stat -- "${generated[@]}"
	exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- "${generated[@]}")"
if [ -n "$untracked" ]; then
	echo "::error::Untracked OpenAPI-generated code found. Run 'make generate-api' and commit the result:"
	echo "$untracked"
	exit 1
fi
