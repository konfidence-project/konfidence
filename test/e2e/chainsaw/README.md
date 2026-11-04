# Example-app e2e test

Tests VectorTemplate assembly, promotion to Stage `dev`, deployment, and
in-cluster `/healthz` requests. See [#371](https://github.com/konfidence-project/konfidence/issues/371).

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

The orchestrator and vector-data-service use the same latest successful `main` build of
[kubernetes-landscape-orchestrator](https://github.com/konfidence-project/kubernetes-landscape-orchestrator),
resolved once with `gh`. Both charts come from that commit. Vector-data-service
is installed per landscape by the suite, using the deployed orchestrator's image tag.
Set `KLO_REF=<commit>` during `make e2e-up` to select another published build for both.

`COMPONENT_REGISTRY` defaults to `ghcr.io/konfidence-project/example-app`.

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

The workflow runs `make e2e-up` and `make test-e2e` on the PR merge commit and shows
the step results in the job summary. Vectors use a run-specific GHCR prefix:
`ghcr.io/konfidence-project/konfidence/e2e/run-<id>-<attempt>`.
This workflow is not part of `ci-gate` or required checks.

## Checks

- VectorTemplate, VectorPromotionConfig, Stage, and VectorDeployment report `Ready=True`.
- Candidates and interviews `/healthz` endpoints respond from inside the cluster.
- Pod logs, events, and resource descriptions are collected on failure.

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
restarts the controller after setup and after cleanup removes the credentials.

The kind cluster remains until `make e2e-down`. GHCR packages remain in the run prefix.

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
Individual collection failures are recorded; a nonzero exit can still leave a useful
partial archive. Kind logs remain available when the API is unreachable.

Built-in redaction is enabled, with credential-token masking everywhere and
key/value assignment masking in log files only (JSON resources stay valid).
Helm values are excluded. Treat snapshots as sensitive:
application logs, ConfigMaps, or custom resources can contain credentials the redactors miss.

CI preserves test fixtures until failure capture, uploads any resulting archive as
`e2e-snapshot-<attempt>` (7-day retention), and deletes kind afterward on success or failure.
Partial snapshots are uploaded even when collection returns nonzero.
