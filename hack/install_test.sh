#!/bin/sh
# Self-check for the pure-logic helpers in install.sh (no network, no Go).
# Sources install.sh with a guard so main() doesn't run, then asserts the
# parsing/mapping functions that would silently 404 or mis-verify if broken.
#
# Run: sh hack/install_test.sh

set -eu

here=$(dirname "$0")

# Prevent install.sh's main() from executing when we source it: stub `main`.
# We source in a subshell-safe way by trimming the trailing `main` invocation.
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
sed '$ { /^main$/d; }' "$here/install.sh" >"$tmp/lib.sh"
# shellcheck source=/dev/null
. "$tmp/lib.sh"

fail=0
check() {
	# check DESC EXPECTED ACTUAL
	if [ "$2" = "$3" ]; then
		echo "ok   - $1"
	else
		echo "FAIL - $1: expected [$2], got [$3]"
		fail=1
	fi
}

# --- extract_tag: parse tag_name out of release JSON ---
json='{"url":"x","tag_name":"v1.2.3","name":"rel"}'
check "extract_tag single" "v1.2.3" "$(printf '%s' "$json" | extract_tag)"

multi='[{"tag_name":"v0.9.0"},{"tag_name":"v0.8.0"}]'
check "extract_tag takes first" "v0.9.0" "$(printf '%s' "$multi" | extract_tag)"

# --- verify_checksum: matches goreleaser checksums.txt layout ---
asset="kden-cli-linux-x86_64.tar.gz"
echo "test-payload" >"$tmp/$asset"
sum=$(sha256_of "$tmp/$asset")
printf '%s  %s\n' "$sum" "$asset" >"$tmp/checksums.txt"
printf '%s  %s\n' "deadbeef" "other-file.tar.gz" >>"$tmp/checksums.txt"
# verify_checksum calls die()->exit on mismatch (correct in the real script),
# so run each call in a subshell to contain the exit.
echo "test-payload" >"$tmp/$asset"
if (verify_checksum "$tmp/$asset" "$tmp/checksums.txt" "$asset" >/dev/null 2>&1); then
	echo "ok   - verify_checksum accepts match"
else
	echo "FAIL - verify_checksum rejected a valid match"
	fail=1
fi
# tamper -> must fail
echo "tampered" >"$tmp/$asset"
if (verify_checksum "$tmp/$asset" "$tmp/checksums.txt" "$asset" >/dev/null 2>&1); then
	echo "FAIL - verify_checksum accepted a mismatch"
	fail=1
else
	echo "ok   - verify_checksum rejects mismatch"
fi

exit $fail
