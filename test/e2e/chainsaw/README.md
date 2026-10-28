# Example-app e2e test

Tests VectorTemplate assembly, promotion to Stage `dev`, deployment, and
in-cluster `/healthz` requests. See [#371](https://github.com/konfidence-project/konfidence/issues/371).

## Run

Requires Docker, Python 3, and registry write access. Activate Hermit for the
pinned build and cluster tools. The bootstrap builds the operator, API, and UI
from the working tree and installs the local chart.

```sh
source bin/activate-hermit
export KUBECONFIG="$PWD/.tmp/e2e/kubeconfig"
export REGISTRY=registry.example/e2e
export REGISTRY_USERNAME=...
export REGISTRY_PASSWORD=...
make e2e-up
make test-e2e
```

Export the registry variables in each new shell; Docker login only stores credentials.

`COMPONENT_REGISTRY` defaults to `ghcr.io/konfidence-project/example-app`.
`VECTOR_DATA_SERVICE_VERSION` overrides the pinned chart version.

Flux, Gateway API, the kind node image, and the orchestrator commit are pinned in
`hack/e2e/versions.env`. The orchestrator is built against this checkout's deployer
library. It currently uses the compatible commit from orchestrator PR #86.

JUnit report:

```sh
mkdir -p e2e-report
make test-e2e CHAINSAW_ARGS="--report-format JUNIT-TEST --report-path e2e-report"
```

Keep resources for debugging:

```sh
make test-e2e CHAINSAW_ARGS="--skip-delete"
```

## Pipeline

Add the `e2e` label to a same-repository PR, or select a branch under
Actions → E2E → Run workflow. New commits to a labelled PR cancel its previous run.
Fork PRs are skipped because their token cannot publish packages.

The workflow builds the checked-out PR merge commit, runs the suite in fresh kind,
and publishes JUnit results and logs. Vectors use a run-specific GHCR prefix:
`ghcr.io/konfidence-project/konfidence/e2e/run-<id>-<attempt>`.
PR images stay local to kind. This workflow is not part of `ci-gate` or required
checks. Whole-cluster snapshots are not included.

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

The kind cluster remains. Remove it with:

```sh
make e2e-down
```

`E2E_CLUSTER_NAME` defaults to `konfidence-e2e`. Use the same name for setup and teardown.
The workflow always tears down its cluster; GHCR packages remain in the run prefix.
