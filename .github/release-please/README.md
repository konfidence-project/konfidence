# Release Process

## Versioning

Versions follow [Semantic Versioning](https://semver.org/) and are computed automatically from [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` → minor bump
- `fix:`, `perf:` → patch bump
- `BREAKING CHANGE` footer → major bump
- All other types (`chore:`, `ci:`, etc.) → no release

## Regular Release

1. Release-please automatically opens a release PR on every push to `main`
2. Review and merge the PR — release-please pushes the version tag, triggering the release pipeline
3. The release changelog is sourced from `CHANGELOG.md` in this directory

## Pre-release

Trigger the [Pre-release workflow](../workflows/prerelease-pipeline.yaml) manually via `workflow_dispatch` with the desired version (e.g. `0.1.0-rc.1`). This pushes the tag directly without touching the manifest, so the next regular release changelog will cover all changes since the last stable release.

## Overriding the Release Version

Push an empty commit to `main` with a `Release-As` footer before the release PR is merged:

```bash
git commit --allow-empty -m "chore: release 1.0.0" -m "Release-As: 1.0.0"
```
