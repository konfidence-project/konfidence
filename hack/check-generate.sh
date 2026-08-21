#!/usr/bin/env bash
# Verify the CRD reference doc is committed and up to date.
# Run after `make docs-reference`; exits non-zero on drift or untracked docs.
set -euo pipefail

if ! git diff --quiet -- api/docs; then
	echo "::error::api/docs/crd.md is out of date. Run 'make docs-reference' and commit the result."
	git --no-pager diff --stat -- api/docs
	exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- api/docs)"
if [ -n "$untracked" ]; then
	echo "::error::Untracked generated docs. Run 'make docs-reference' and commit the result:"
	echo "$untracked"
	exit 1
fi
