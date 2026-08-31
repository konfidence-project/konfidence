#!/usr/bin/env bash
# Verify all generated API references in api/docs/ are committed and up to date.
# Run after `make docs`; exits non-zero on drift or untracked files.
set -euo pipefail

if ! git diff --quiet -- api/docs; then
	echo "::error::api/docs is out of date. Run 'make docs' and commit the result."
	git --no-pager diff --stat -- api/docs
	exit 1
fi

untracked="$(git ls-files --others --exclude-standard -- api/docs)"
if [ -n "$untracked" ]; then
	echo "::error::Untracked generated docs in api/docs. Run 'make docs' and commit the result:"
	echo "$untracked"
	exit 1
fi
