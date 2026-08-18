# Release Process

## Versioning

Versions follow [Semantic Versioning](https://semver.org/) and are computed automatically from [Conventional Commits](https://www.conventionalcommits.org/). The project is in its alpha phase: every release carries an `-alpha.N` suffix (starting at `0.0.1-alpha.1`) and is published as a normal "Latest" GitHub release (the suffix is a versioning scheme, not a stability marker), using release-please's `prerelease` versioning strategy:

- `fix:`, `perf:`, `chore:` → increments the pre-release counter (`0.0.1-alpha.1` → `0.0.1-alpha.2`). `chore` counts as releasable because it has a visible changelog section — dependency bumps should appear in the changelog of an operator.
- `feat:` and `BREAKING CHANGE` → minor bump of the base version (`0.0.1-alpha.2` → `0.1.0-alpha.2`). The counter carries over — it does not reset. Breaking changes stay pre-1.0 (`bump-minor-pre-major`); the jump to 1.x only happens via an explicit `Release-As`.
- `docs:`, `ci:`, `refactor:`, `style:`, `build:` with no releasable change alongside → no release

Graduating out of alpha later is a config change (`versioning`, `prerelease`, `prerelease-type` in `release-please-config.json`) plus a `Release-As` commit for the first stable version.

## Regular Release

1. Release-please automatically opens a release PR on every push to `main`
2. Review and merge the PR — release-please pushes the version tag and creates a draft GitHub release
3. The release pipeline runs as a chained job in the same workflow (not on the tag push), so the draft is guaranteed to exist before goreleaser attaches artifacts to it
4. Once binaries and Helm charts are published, the final job removes the draft flag — a failed pipeline leaves the release in draft, never half-published
5. The release changelog is sourced from `CHANGELOG.md` in the repository root; goreleaser runs with `mode: keep-existing` and does not touch the body

## Pre-release

Trigger the [Pre-release workflow](../workflows/prerelease-pipeline.yaml) manually via `workflow_dispatch` with the desired version. Only `-rc.*` suffixes build (the tag-triggered release pipeline is restricted to `X.Y.Z-rc.*` so it can never overlap with release-please's `-alpha.N` tags). This pushes the tag directly without touching the manifest, so the next regular release changelog will cover all changes since the last release.

## Overriding the Release Version

Use a [`Release-As` footer](https://github.com/googleapis/release-please?tab=readme-ov-file#release-as-an-alternative-to-a-changelog) to force a specific version regardless of commit history.

> **Important:** Close any open release-please PR before pushing a `Release-As` commit. If a PR is already open,
> release-please will update it in place rather than open a fresh one — this is known to produce inconsistent
> results where the version, changelog, and manifest may not align correctly.

Direct pushes to `main` are blocked, so the footer has to land through a normal PR: put it in the message of a regular commit on the PR branch and merge as usual. Avoid `--allow-empty` commits — the repository only allows rebase merges, which drop empty commits.

```bash
git commit -m "chore: prepare 0.1.0-alpha.1" -m "Release-As: 0.1.0-alpha.1"
```
