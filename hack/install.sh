#!/bin/sh
# Install the kden CLI on any Unix (Linux, macOS).
#
# Usage:
#   curl -fsSL https://konfidence.cloud/install.sh | sh
#
# Environment overrides:
#   KDEN_VERSION      install a specific release tag (e.g. v0.3.0); default: latest
#   KDEN_GIT_REF      build from a git ref (branch/tag/sha) instead of a release;
#                     requires Go on PATH. Takes precedence over KDEN_VERSION.
#   KDEN_INSTALL_DIR  install location; default: $HOME/.local/bin
#   GITHUB_TOKEN      used for the GitHub API to avoid rate limits (optional)

set -eu

REPO="konfidence-project/konfidence"
API="https://api.github.com/repos/${REPO}"
INSTALL_DIR="${KDEN_INSTALL_DIR:-$HOME/.local/bin}"

# --- helpers ---------------------------------------------------------------

die() {
	echo "kden install: $*" >&2
	exit 1
}

info() {
	echo "kden install: $*" >&2
}

have() {
	command -v "$1" >/dev/null 2>&1
}

# http_get URL -> stdout. Follows redirects, fails on 4xx/5xx.
http_get() {
	if have curl; then
		curl -fsSL "$@"
	elif have wget; then
		# wget doesn't take our curl-style extra flags; callers pass only a URL.
		wget -qO- "$1"
	else
		die "need curl or wget"
	fi
}

# http_download URL OUTFILE
http_download() {
	if have curl; then
		curl -fsSL -o "$2" "$1"
	elif have wget; then
		wget -qO "$2" "$1"
	else
		die "need curl or wget"
	fi
}

# --- platform detection ----------------------------------------------------

detect_platform() {
	os=$(uname -s | tr '[:upper:]' '[:lower:]')
	case "$os" in
	linux | darwin) ;;
	*) die "unsupported OS: $os (only linux and darwin are supported)" ;;
	esac

	arch=$(uname -m)
	case "$arch" in
	x86_64 | amd64) arch=x86_64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) die "unsupported arch: $arch" ;;
	esac

	# No darwin/amd64 artifact is published (goreleaser ignore list). Intel Macs
	# run arm64 binaries under Rosetta 2, so probe for it and use the arm64 asset.
	if [ "$os" = darwin ] && [ "$arch" = x86_64 ]; then
		if [ "$(sysctl -n hw.optional.arm64 2>/dev/null || echo 0)" = 1 ]; then
			info "Intel-reported macOS on Apple Silicon (Rosetta); using arm64 build"
			arch=arm64
		else
			die "no darwin/amd64 build is published; install Go and re-run with KDEN_GIT_REF=main to build from source"
		fi
	fi

	# Mirrors .goreleaser.yaml name_template: kden-cli-<os>-<arch>.tar.gz
	asset="kden-cli-${os}-${arch}.tar.gz"
}

# --- release resolution ----------------------------------------------------

api_get() {
	# $1 = API path suffix. Adds auth header when GITHUB_TOKEN is set.
	if [ -n "${GITHUB_TOKEN:-}" ] && have curl; then
		curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" "${API}/$1"
	elif [ -n "${GITHUB_TOKEN:-}" ] && have wget; then
		wget -qO- --header="Authorization: Bearer ${GITHUB_TOKEN}" "${API}/$1"
	else
		http_get "${API}/$1"
	fi
}

# Extract the "tag_name" from a GitHub release JSON blob on stdin.
# Uses grep -o so a single-line array yields the FIRST tag, not the last
# (a greedy sed would skip to the final tag_name and pick an older release).
extract_tag() {
	grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' |
		head -n1 |
		sed 's/.*"\([^"]*\)"$/\1/'
}

latest_release() {
	# /releases/latest excludes prereleases and 404s on prerelease-only repos.
	# Fall back to the newest of all releases (which includes prereleases).
	tag=$(api_get "releases/latest" 2>/dev/null | extract_tag || true)
	if [ -z "$tag" ]; then
		tag=$(api_get "releases?per_page=1" 2>/dev/null | extract_tag || true)
	fi
	[ -n "$tag" ] || return 1
	echo "$tag"
}

# --- checksum verification -------------------------------------------------

sha256_of() {
	if have sha256sum; then
		sha256sum "$1" | awk '{print $1}'
	elif have shasum; then
		shasum -a 256 "$1" | awk '{print $1}'
	else
		die "need sha256sum or shasum to verify the download"
	fi
}

verify_checksum() {
	# $1 = downloaded asset path, $2 = checksums.txt path, $3 = asset filename
	want=$(awk -v f="$3" '$2 == f {print $1}' "$2" | head -n1)
	[ -n "$want" ] || die "no checksum for $3 in checksums.txt"
	got=$(sha256_of "$1")
	# Guard the empty case explicitly: without pipefail, a failed hasher pipes
	# empty through awk with exit 0. The equality check below already fails
	# closed on empty, but say so plainly rather than print "got ".
	[ -n "$got" ] || die "failed to compute sha256 of $1"
	[ "$want" = "$got" ] || die "checksum mismatch for $3 (want $want, got $got)"
	info "checksum verified"
}

# --- install ---------------------------------------------------------------

place_binary() {
	# $1 = path to the built/extracted kden binary
	mkdir -p "$INSTALL_DIR"
	chmod 0755 "$1"
	# mv within a checkout can cross filesystems; try atomic mv, fall back to cp.
	mv "$1" "$INSTALL_DIR/kden" 2>/dev/null || {
		cp "$1" "$INSTALL_DIR/kden.tmp.$$" && mv "$INSTALL_DIR/kden.tmp.$$" "$INSTALL_DIR/kden"
	}
	info "installed kden to $INSTALL_DIR/kden"

	case ":${PATH}:" in
	*":${INSTALL_DIR}:"*) ;;
	*)
		echo "" >&2
		echo "  $INSTALL_DIR is not on your PATH. Add it with:" >&2
		echo "    export PATH=\"$INSTALL_DIR:\$PATH\"" >&2
		echo "" >&2
		;;
	esac
}

install_from_release() {
	tag=${KDEN_VERSION:-$(latest_release || true)}
	if [ -z "$tag" ]; then
		# Pre-release window: no release published yet. Fall back to building
		# from main if Go is available. (This fallback is retired once the first
		# release exists; a missing pinned KDEN_VERSION then becomes an error.)
		if have go && have git; then
			info "no release found; building from main (Go detected)"
			KDEN_GIT_REF=main install_from_source
			return
		fi
		die "no release found and Go/git not available to build from source"
	fi

	info "installing release $tag ($asset)"
	dl="https://github.com/${REPO}/releases/download/${tag}"
	http_download "${dl}/${asset}" "$tmp/$asset" ||
		die "failed to download $asset for $tag (does this platform have a build?)"
	http_download "${dl}/checksums.txt" "$tmp/checksums.txt" ||
		die "failed to download checksums.txt for $tag"

	verify_checksum "$tmp/$asset" "$tmp/checksums.txt" "$asset"
	tar -xzf "$tmp/$asset" -C "$tmp"
	[ -f "$tmp/kden" ] || die "archive did not contain a kden binary"
	place_binary "$tmp/kden"
}

install_from_source() {
	have go || die "KDEN_GIT_REF set but Go is not installed"
	have git || die "KDEN_GIT_REF set but git is not installed"
	ref=$KDEN_GIT_REF
	info "building kden from git ref: $ref"

	src="$tmp/src"
	# --branch handles branches and tags; fall back to full clone + checkout for
	# a bare commit sha.
	if ! git clone --depth 1 --branch "$ref" "https://github.com/${REPO}.git" "$src" 2>/dev/null; then
		info "ref is not a branch/tag; cloning and checking out $ref"
		git clone "https://github.com/${REPO}.git" "$src"
		(cd "$src" && git checkout "$ref")
	fi

	# go install of a monorepo subpath drops -X ldflags, so build explicitly.
	ldflags="-s -w -X github.com/konfidence-project/konfidence/pkg/build.Version=${ref}"
	(cd "$src" && go build -ldflags "$ldflags" -o "$tmp/kden" ./cmd/kden)
	place_binary "$tmp/kden"
}

# --- main ------------------------------------------------------------------

main() {
	tmp=$(mktemp -d 2>/dev/null || mktemp -d -t kden-install)
	trap 'rm -rf "$tmp"' EXIT INT TERM

	if [ -n "${KDEN_GIT_REF:-}" ]; then
		install_from_source
	else
		detect_platform
		install_from_release
	fi

	info "done. Run 'kden version' to verify."
}

main
