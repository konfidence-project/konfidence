# scripts

Developer scripts for the konfidence release process.

## `release.sh`

Bumps the Helm chart versions, creates a conventional commit, tags the release,
and pushes to origin. Also supports creating pre-release tags with the same
chart update + commit + tag + push flow.

### Prerequisites

- Hermit environment active: `source ./bin/activate-hermit`
- `yq` available on PATH (provided by Hermit)
- Clean working tree with no uncommitted changes
- On `main` branch (or an agreed hotfix branch)
- Push access to `origin` (including tag push rights)

### Usage

```sh
# Semver release - bumps Chart.yaml, commits, tags, pushes
scripts/release.sh patch            # 0.1.1 -> 0.1.2 - bug fix or non-functional change
scripts/release.sh minor            # 0.1.1 -> 0.2.0 - new backwards-compatible feature
scripts/release.sh major            # 0.1.1 -> 1.0.0 - breaking change

# Pre-release tag — updates Chart.yaml, commits, tags, and pushes
scripts/release.sh tag 1.0.0-rc.1
scripts/release.sh tag 0.2.0-beta.1

# Flags
scripts/release.sh patch --dry-run          # preview without modifying anything
scripts/release.sh patch --yes              # skip confirmation prompts
scripts/release.sh tag 1.0.0-rc.1 --dry-run
```

### What it does

#### `major | minor | patch`

1. Verifies the Hermit environment is active and `yq` is available.
2. Verifies the working tree is clean (no staged or unstaged changes).
3. Reads the current `appVersion` from `charts/star/Chart.yaml` as the source of truth.
4. Computes the next version according to [Semantic Versioning](https://semver.org/).
5. Writes the new version to both `version` and `appVersion` in:
   - `charts/star/Chart.yaml`
   - `charts/galaxy/Chart.yaml`
6. Creates a commit: `chore(release): X.Y.Z`
7. Creates an annotated git tag `X.Y.Z`.
8. Runs `git push` and `git push --tags`.

> `version` and `appVersion` are always kept in sync - both fields are set to
> the same value on every release.

#### `tag <value>`

1. Verifies the working tree is clean.
2. Validates the tag matches `X.Y.Z-<suffix>` (e.g. `1.0.0-rc.1`).
   Bare `X.Y.Z` tags are reserved for `major|minor|patch` releases.
3. Writes the pre-release version to both `version` and `appVersion` in:
   - `charts/star/Chart.yaml`
   - `charts/galaxy/Chart.yaml`
4. Creates a commit: `chore(release): X.Y.Z-<suffix>`.
5. Creates an annotated git tag `X.Y.Z-<suffix>` on that commit.
6. Runs `git push` and `git push --tags`.
