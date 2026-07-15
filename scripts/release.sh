#!/usr/bin/env bash
# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and konfidence contributors
#
# scripts/release.sh - local release helper for the konfidence repository.
#
# Usage:
#   scripts/release.sh <major|minor|patch|tag <value>> [--yes] [--dry-run]
#
# What it does (major|minor|patch):
#   1. Validates the working tree is clean.
#   2. Reads the current appVersion from charts/star/Chart.yaml (source of truth).
#   3. Computes the next semver according to the requested bump type.
#   4. Updates both `appVersion` and `version` to the new value in
#      charts/star/Chart.yaml and charts/galaxy/Chart.yaml (kept in sync).
#   5. Creates a conventional commit: chore(release): X.Y.Z
#   6. Creates an annotated git tag X.Y.Z.
#   7. Runs `git push` and `git push origin refs/tags/X.Y.Z`.
#
# What it does (tag <value>):
#   1. Validates the working tree is clean.
#   2. Validates the tag value matches X.Y.Z-<suffix> (e.g. 1.0.0-rc.1).
#   3. Updates both `appVersion` and `version` to the tag value in
#      charts/star/Chart.yaml and charts/galaxy/Chart.yaml.
#   4. Creates a conventional commit: chore(release): X.Y.Z-<suffix>.
#   5. Creates an annotated git tag X.Y.Z-<suffix>.
#   6. Runs `git push` and `git push origin refs/tags/<value>`.
#
# Prerequisites:
#   - Hermit environment active (source ./bin/activate-hermit)
#   - yq available on PATH (provided by Hermit)
#   - Push access to origin

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
STAR_CHART="${REPO_ROOT}/charts/star/Chart.yaml"
GALAXY_CHART="${REPO_ROOT}/charts/galaxy/Chart.yaml"

# ── helpers ──────────────────────────────────────────────────────────────────

usage() {
  echo "Usage: scripts/release.sh <major|minor|patch|tag <value>> [--yes] [--dry-run]"
  echo ""
  echo "  major          Bump the major version component (X.0.0)"
  echo "  minor          Bump the minor version component (x.Y.0)"
  echo "  patch          Bump the patch version component (x.y.Z)"
  echo "  tag <value>    Create a pre-release tag (e.g. 1.0.0-rc.1), updating"
  echo "                 Chart.yaml, committing, tagging, and pushing"
  echo ""
  echo "  --yes          Skip confirmation prompts (non-interactive / CI use)"
  echo "  --dry-run      Print what would happen without modifying anything"
  exit 1
}

info()  { echo "[release] $*"; }
error() { echo "[release] ERROR: $*" >&2; exit 1; }

check_hermit() {
  command -v hermit >/dev/null 2>&1 || error "Hermit is not installed. See https://cashapp.github.io/hermit/"
  hermit status >/dev/null 2>&1    || error "Hermit environment is not active. Run: source ./bin/activate-hermit"
}

check_yq() {
  command -v yq >/dev/null 2>&1 || error "yq not found on PATH. Make sure Hermit is active."
}

check_clean_tree() {
  if ! git -C "${REPO_ROOT}" diff --exit-code --quiet; then
    error "Working tree has unstaged changes. Commit or stash them before releasing."
  fi
  if ! git -C "${REPO_ROOT}" diff --cached --exit-code --quiet; then
    error "Working tree has staged changes. Commit or stash them before releasing."
  fi
}

check_branch() {
  local branch
  branch="$(git -C "${REPO_ROOT}" rev-parse --abbrev-ref HEAD)"
  if [[ "${branch}" != "main" ]]; then
    echo "[release] WARNING: you are on branch '${branch}', not 'main'."
    echo "[release]          Proceed only if this is intentional (e.g. a hotfix branch)."
  fi
}

bump_version() {
  local current="$1"
  local bump_type="$2"

  # Strip leading 'v' if present
  current="${current#v}"

  # Strip pre-release suffix (e.g. -rc.1, -ci-test-2) before splitting
  current="${current%%-*}"

  local maj min pat
  IFS='.' read -r maj min pat <<< "${current}"

  # Validate all three components are integers
  [[ "${maj}" =~ ^[0-9]+$ ]] || error "Unexpected version format: '${current}' (major component not an integer)"
  [[ "${min}" =~ ^[0-9]+$ ]] || error "Unexpected version format: '${current}' (minor component not an integer)"
  [[ "${pat}" =~ ^[0-9]+$ ]] || error "Unexpected version format: '${current}' (patch component not an integer)"

  case "${bump_type}" in
    major) echo "$((maj + 1)).0.0" ;;
    minor) echo "${maj}.$((min + 1)).0" ;;
    patch) echo "${maj}.${min}.$((pat + 1))" ;;
    *)     error "Unknown bump type: '${bump_type}'" ;;
  esac
}

validate_custom_tag() {
  local tag="$1"
  # Must match X.Y.Z-<pre-release>[+<build>] - bare X.Y.Z is reserved for major|minor|patch releases.
  # Uses the official semver.org regex (https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string)
  # adapted to mandate a pre-release suffix.
  [[ "${tag}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)-(((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*)(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$) ]] || \
    error "Invalid tag '${tag}'. Must match X.Y.Z-<pre-release> (e.g. 1.0.0-rc.1)."
}

assert_tag_does_not_exist() {
  local tag="$1"
  if git -C "${REPO_ROOT}" tag --list | grep -qx "${tag}"; then
    error "Tag '${tag}' already exists locally."
  fi
  if git -C "${REPO_ROOT}" ls-remote --tags origin | grep -q "refs/tags/${tag}$"; then
    error "Tag '${tag}' already exists on remote."
  fi
}

confirm_push() {
  local hint="$1"
  read -r -p "[release] Push to origin? [y/N] " reply
  case "${reply}" in
    y|Y|yes|YES) ;;
    *)
      info "Aborted push. To push manually when ready:"
      info "  ${hint}"
      exit 0
      ;;
  esac
}

# ── argument parsing ──────────────────────────────────────────────────────────

MODE=""        # bump | tag
BUMP_TYPE=""   # major | minor | patch
CUSTOM_TAG=""  # value for tag mode
YES=false
DRY_RUN=false

args=("$@")
i=0
while [[ ${i} -lt ${#args[@]} ]]; do
  arg="${args[${i}]}"
  case "${arg}" in
    major|minor|patch)
      MODE="bump"
      BUMP_TYPE="${arg}"
      ;;
    tag)
      MODE="tag"
      i=$(( i + 1 ))
      [[ ${i} -lt ${#args[@]} ]] || error "'tag' requires a value (e.g. scripts/release.sh tag 1.0.0-rc.1)"
      CUSTOM_TAG="${args[${i}]}"
      ;;
    --yes)      YES=true ;;
    --dry-run)  DRY_RUN=true ;;
    -h|--help)  usage ;;
    *)          error "Unknown argument: '${arg}'" ;;
  esac
  i=$(( i + 1 ))
done

[[ -n "${MODE}" ]] || usage

# ── pre-flight ────────────────────────────────────────────────────────────────

check_hermit
check_clean_tree
check_branch

# ── mode: tag ─────────────────────────────────────────────────────────────────

if [[ "${MODE}" == "tag" ]]; then
  validate_custom_tag "${CUSTOM_TAG}"
  assert_tag_does_not_exist "${CUSTOM_TAG}"
  check_yq

  info "Tag            : ${CUSTOM_TAG}"
  info "Commit         : $(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
  info "Charts affected:"
  info "  ${STAR_CHART}"
  info "  ${GALAXY_CHART}"
  echo ""

  if [[ "${DRY_RUN}" == true ]]; then
    info "DRY RUN — would execute:"
    info "  yq -i \".version = \\\"${CUSTOM_TAG}\\\"\"    ${STAR_CHART}"
    info "  yq -i \".appVersion = \\\"${CUSTOM_TAG}\\\"\" ${STAR_CHART}"
    info "  yq -i \".version = \\\"${CUSTOM_TAG}\\\"\"    ${GALAXY_CHART}"
    info "  yq -i \".appVersion = \\\"${CUSTOM_TAG}\\\"\" ${GALAXY_CHART}"
    info "  make manifests"
    info "  git add charts/star/ charts/galaxy/"
    info "  git commit -m \"chore(release): ${CUSTOM_TAG}\""
    info "  git tag -a ${CUSTOM_TAG} -m \"Release ${CUSTOM_TAG}\""
    info "  git push"
    info "  git push origin \"refs/tags/${CUSTOM_TAG}\""
    info "No changes made."
    exit 0
  fi

  if [[ "${YES}" != true ]]; then
    read -r -p "[release] Proceed? [y/N] " confirm
    case "${confirm}" in
      y|Y|yes|YES) ;;
      *) info "Aborted."; exit 0 ;;
    esac
  fi

  info "Updating chart versions..."
  yq -i ".version = \"${CUSTOM_TAG}\"" "${STAR_CHART}"
  yq -i ".appVersion = \"${CUSTOM_TAG}\"" "${STAR_CHART}"
  yq -i ".version = \"${CUSTOM_TAG}\"" "${GALAXY_CHART}"
  yq -i ".appVersion = \"${CUSTOM_TAG}\"" "${GALAXY_CHART}"

  info "Regenerating manifests..."
  make -C "${REPO_ROOT}" manifests

  info "Committing..."
  git -C "${REPO_ROOT}" add "${STAR_CHART}" "${GALAXY_CHART}" \
    charts/star/ charts/galaxy/
  git -C "${REPO_ROOT}" commit -m "chore(release): ${CUSTOM_TAG}"

  info "Creating annotated tag ${CUSTOM_TAG}..."
  git -C "${REPO_ROOT}" tag -a "${CUSTOM_TAG}" -m "Release ${CUSTOM_TAG}"

  COMMIT_SHA="$(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
  echo ""
  info "Commit ${COMMIT_SHA} and tag ${CUSTOM_TAG} created locally."

  if [[ "${YES}" != true ]]; then
    confirm_push "git push && git push origin \"refs/tags/${CUSTOM_TAG}\""
  fi

  info "Pushing commits..."
  git -C "${REPO_ROOT}" push

  info "Pushing tags..."
  git -C "${REPO_ROOT}" push origin "refs/tags/${CUSTOM_TAG}"

  info "Done. Tagged ${CUSTOM_TAG}."
  exit 0
fi

# ── mode: bump (major|minor|patch) ───────────────────────────────────────────

check_yq

CURRENT_VERSION="$(yq '.appVersion' "${STAR_CHART}" | tr -d '"')"
[[ -n "${CURRENT_VERSION}" ]] || error "Could not read 'appVersion' from ${STAR_CHART}"

NEXT_VERSION="$(bump_version "${CURRENT_VERSION}" "${BUMP_TYPE}")"
TAG="${NEXT_VERSION}"

assert_tag_does_not_exist "${TAG}"

info "Current appVersion : ${CURRENT_VERSION}"
info "Next version       : ${NEXT_VERSION}  (${BUMP_TYPE} bump)"
info "Git tag            : ${TAG}"
info "Charts affected    :"
info "  ${STAR_CHART}"
info "  ${GALAXY_CHART}"
echo ""

# ── dry-run exit ──────────────────────────────────────────────────────────────

if [[ "${DRY_RUN}" == true ]]; then
  info "DRY RUN - would execute:"
  info "  yq -i \".version = \\\"${NEXT_VERSION}\\\"\"    ${STAR_CHART}"
  info "  yq -i \".appVersion = \\\"${NEXT_VERSION}\\\"\" ${STAR_CHART}"
  info "  yq -i \".version = \\\"${NEXT_VERSION}\\\"\"    ${GALAXY_CHART}"
  info "  yq -i \".appVersion = \\\"${NEXT_VERSION}\\\"\" ${GALAXY_CHART}"
  info "  make manifests"
  info "  git add charts/star/ charts/galaxy/"
  info "  git commit -m \"chore(release): ${TAG}\""
  info "  git tag -a ${TAG} -m \"Release ${TAG}\""
  info "  git push"
  info "  git push origin \"refs/tags/${TAG}\""
  info "No changes made."
  exit 0
fi

# ── confirmation ──────────────────────────────────────────────────────────────

if [[ "${YES}" != true ]]; then
  read -r -p "[release] Proceed? [y/N] " confirm
  case "${confirm}" in
    y|Y|yes|YES) ;;
    *) info "Aborted."; exit 0 ;;
  esac
fi

# ── update Chart.yaml files ───────────────────────────────────────────────────

info "Updating chart versions..."

yq -i ".version = \"${NEXT_VERSION}\"" "${STAR_CHART}"
yq -i ".appVersion = \"${NEXT_VERSION}\"" "${STAR_CHART}"
yq -i ".version = \"${NEXT_VERSION}\"" "${GALAXY_CHART}"
yq -i ".appVersion = \"${NEXT_VERSION}\"" "${GALAXY_CHART}"

info "  ${STAR_CHART}   version + appVersion -> ${NEXT_VERSION}"
info "  ${GALAXY_CHART} version + appVersion -> ${NEXT_VERSION}"

info "Regenerating manifests..."
make -C "${REPO_ROOT}" manifests

# ── commit ────────────────────────────────────────────────────────────────────

info "Committing..."
git -C "${REPO_ROOT}" add \
  "${STAR_CHART}" \
  "${GALAXY_CHART}" \
  charts/star/ \
  charts/galaxy/

git -C "${REPO_ROOT}" commit -m "chore(release): ${TAG}"

info "Creating annotated tag ${TAG}..."
git -C "${REPO_ROOT}" tag -a "${TAG}" -m "Release ${TAG}"

# ── push ─────────────────────────────────────────────────────────────────────

COMMIT_SHA="$(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
echo ""
info "Commit ${COMMIT_SHA} and tag ${TAG} created locally."

if [[ "${YES}" != true ]]; then
  confirm_push "git push && git push origin \"refs/tags/${TAG}\""
fi

info "Pushing commits..."
git -C "${REPO_ROOT}" push

info "Pushing tags..."
git -C "${REPO_ROOT}" push origin "refs/tags/${TAG}"

info "Done. Released ${TAG}."
