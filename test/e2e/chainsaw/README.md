# Example-app e2e test

One Chainsaw test covers VectorTemplate assembly, automatic promotion to `dev`,
manual promotion to `prod`, artifact reuse, and a feature-only rollout through
both stages on the same landscape. See [#371](https://github.com/konfidence-project/konfidence/issues/371)
and [#375](https://github.com/konfidence-project/konfidence/issues/375).

## Run

Requires Docker, `gh`, Hermit (`source bin/activate-hermit`), and registry write access.

```sh
make e2e-up
export REGISTRY=registry.example/e2e
export REGISTRY_USERNAME=...
export REGISTRY_PASSWORD=...
make test-e2e
make e2e-down
```

Export the registry variables in each new shell; Docker login only stores credentials.

`make e2e-up` creates the kind cluster `konfidence-e2e` (`E2E_CLUSTER`), installs the
latest Flux and Gateway API releases, and deploys this checkout's operator with
`docker-build`, `webhook-certs`, and `deploy`. The API is not deployed.

Set `E2E_CORE_DIR=<checkout>` during `make e2e-up` to build and deploy the operator,
chart, and CRDs from another Konfidence checkout while retaining this checkout's suite.

The orchestrator and vector-data-service use the same latest successful `main` build of
[kubernetes-landscape-orchestrator](https://github.com/konfidence-project/kubernetes-landscape-orchestrator),
resolved once with `gh`. Both charts come from that commit. Vector-data-service
is installed per landscape by the suite. Bootstrap saves the SHA in `.tmp/klo/ref`;
`make test-e2e` passes it as `KLO_REF`, without looking up the deployed image.
Run `make e2e-up` first to populate the cached charts and revision.
Set `KLO_REF=<full-commit-sha>` during `make e2e-up` to select another published build
for both. The selected commit must have published KLO and VDS images.

`COMPONENT_REGISTRY` defaults to `ghcr.io/konfidence-project/example-app`.

### Latest released stack

Use a fresh cluster; run `make e2e-down` first if switching from the PR/main stack.

```sh
make e2e-up-release
make test-e2e
make e2e-down
```

Bootstrap resolves the latest published Konfidence and KLO releases once and prints
their versions. It installs their OCI charts and caches the matching VDS chart/version.
No operator build or fallback to `main` occurs. The API remains disabled.
The same suite and registry variables are used; Flux, Gateway API, and example-app
component selection are unchanged. This mode requires network access.

Older releases may not support the suite's current APIs. Such failures remain failures;
the fixtures and assertions are not changed to accommodate them.

JUnit report:

```sh
mkdir -p e2e-report
make test-e2e CHAINSAW_ARGS="--report-format JUNIT-STEP --report-path e2e-report"
```

Keep resources for debugging:

```sh
make test-e2e CHAINSAW_ARGS="--skip-delete"
```

## Pipeline

Add the `e2e` label to a same-repository PR, or select a branch under
Actions → E2E → Run workflow. New commits to a labelled PR cancel its previous run.
Fork PRs are skipped because their token cannot publish packages.

Manual runs offer **PR + main** (default) and **Latest releases**. Release mode uses
`make e2e-up-release`; label-triggered PR runs always use `make e2e-up`.
There is no automatic trigger on release publication.

**PR + main** accepts two optional inputs:

- `konfidence_ref`: full core commit SHA; empty uses the workflow branch. The operator,
  chart, and CRDs are built and deployed from a separate checkout of that commit.
- `klo_ref`: full KLO commit SHA with published KLO/VDS images; empty uses the
  latest successful `main` build for both.

The workflow and Chainsaw suite remain from the workflow branch. Both inputs are
ignored by **Latest releases**; incompatible selected commits still fail.

The workflow runs the selected bootstrap and `make test-e2e`, and shows
the step results in the job summary. Vectors go to the shared GHCR prefix
`ghcr.io/konfidence-project/konfidence/e2e`. After each run, the workflow deletes
vector versions older than 2 hours and keeps the 5 newest versions of each
component package.
This workflow is not part of `ci-gate` or required checks.

## Checks

- VectorTemplate, VectorPromotionConfig, StageVersion, Stage, and VectorDeployment report `Ready=True`.
- Each stage's active StageVersion references the expected vector.
- `prod` remains unassigned until the first explicit promotion.
- Exactly two ArtifactDeployments exist; dev and prod reference the same artifacts throughout the flag-only rollout.
- Candidates and interviews `/healthz` and `/debug` endpoints respond from inside the cluster.
- Pod logs, events, and resource descriptions are collected on failure.

| Phase | Dev flags | Prod flags |
|---|---|---|
| Initial automatic promotion | `false` | No vector |
| First manual promotion | `false` | `false` |
| Feature-only template change | `true` | Still `false` |
| Second manual promotion | `true` | `true` |

Both `allow-video-slots` and `enable-candidate-notes` are checked through
`/candidates/debug` and `/interviews/debug`, with `X-Konfidence-Routing-Key` set
to the active VectorDeployment name. The second assembly pins the initially
resolved component versions from prod so moving `edge` aliases cannot change the artifacts.
HTTP requests use `kubectl exec` in one test-owned curl pod; attaching to a short-lived
`kubectl run -i` pod can miss early response output.

Resource setup and checks use native Chainsaw operations and `x_k8s_get`/`x_k8s_list`.
HTTP responses use `command.check` with `json_parse`. Reuse is checked by count and
references, not by a same-name delete/recreate ledger.

Manual steps satisfy the pending `explicit-approval.konfidence.cloud` gate via
`kubectl patch --subresource=status`. API/CLI approval authorization is not tested.

## Cleanup

Chainsaw removes resources created during the current run in reverse order.
Pre-existing resources are patched and kept, including fixtures left by an
earlier `--skip-delete` run.

On a fresh cluster, cleanup removes the app and database fixtures, then the
registry Secret and Flux ConfigMap in `konfidence-system`. Project and Landscape
fixtures set `deletionPolicy: Delete` to remove their namespaces.
`--skip-delete` preserves newly created resources.

Registry credentials are read at controller startup
([#393](https://github.com/konfidence-project/konfidence/issues/393)); the test
restarts the controller after installing credentials, not during cleanup.

The kind cluster remains until `make e2e-down`. Local runs keep their registry uploads.
CI always attempts to delete its own GHCR run prefix after snapshots and cluster teardown.

## Failure snapshot

Install the [Troubleshoot support-bundle CLI](https://github.com/replicatedhq/troubleshoot/releases)
(tested with v0.134.3). For a failed run, keep fixtures until capture:

```sh
make test-e2e CHAINSAW_ARGS="--skip-delete"
make e2e-snapshot
make e2e-down
```

The snapshot is written to `.tmp/e2e-snapshot/snapshot.tar.gz`. Override the directory
with `E2E_SNAPSHOT_DIR`; use the same `E2E_CLUSTER` as bootstrap.
Krew installations can use `SUPPORT_BUNDLE="kubectl support-bundle"`.

The bundle includes kind node/container logs, events, custom resources, and Helm
release history (resource specs and status stand in for `kubectl describe`). It
does not deploy collectors or upload data.
`test/e2e/support-bundle.yaml` and `test/e2e/snapshot-redactors.yaml` are host CLI
specifications, not resources applied to Kubernetes.
Individual collection failures are recorded; a nonzero exit can still leave a useful
partial archive. Kind logs remain available when the API is unreachable.

Built-in redaction is enabled, with credential-token masking everywhere and
key/value assignment masking in log files only (JSON resources stay valid).
Helm values are excluded. Treat snapshots as sensitive:
application logs, ConfigMaps, or custom resources can contain credentials the redactors miss.

CI preserves test fixtures until failure capture, uploads any resulting archive as
`e2e-snapshot-<attempt>` (7-day retention), then deletes kind in an `always()` step.
Partial snapshots are uploaded even when collection returns nonzero.
