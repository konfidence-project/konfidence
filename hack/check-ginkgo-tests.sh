#!/usr/bin/env bash

set -euo pipefail

violations=""

while IFS= read -r file; do
	if [[ "$file" != *suite_test.go ]]; then
		violations+="$file: Test functions are only allowed in suite files"$'\n'
		continue
	fi

	test_count="$(rg -c '^func Test[[:alnum:]_]*[[:space:]]*\(' "$file")"
	if [ "$test_count" -ne 1 ] || ! rg -q 'RunSpecs\(' "$file"; then
		violations+="$file: suite files must contain exactly one RunSpecs bootstrap"$'\n'
	fi
done < <(rg -l --glob '*_test.go' '^func Test[[:alnum:]_]*[[:space:]]*\(' . || true)

violations="${violations%$'\n'}"

if [ -n "$violations" ]; then
	echo "Go behavior tests must use Ginkgo. Conventional tests found:"
	echo "$violations"
	exit 1
fi
