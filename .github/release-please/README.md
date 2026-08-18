# Release Process

## Versioning

Versions follow [Semantic Versioning](https://semver.org/) and are computed automatically from [Conventional Commits](https://www.conventionalcommits.org/). The project is in its alpha phase: every release carries an `-alpha.N` suffix (starting at `0.0.1-alpha.1`) and is marked as a GitHub pre-release, using release-please's `prerelease` versioning strategy:

- `fix:`, `perf:` (and any other releasable change) → increments the pre-release counter (`0.0.1-alpha.1` → `0.0.1-alpha.2`)
- `feat:` → minor bump of the base version, counter resets (`0.1.0-alpha.1`)
- `BREAKING CHANGE` footer → major bump of the base version, counter resets
- `chore:`, `ci:`, `docs:`, etc. with no releasable change alongside → no release

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

```bash
git commit --allow-empty -m "chore: release 0.1.0" -m "Release-As: 0.1.0"
git push origin main
```
