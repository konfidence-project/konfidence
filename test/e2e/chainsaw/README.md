# Chainsaw e2e test

Tests VectorTemplate assembly, promotion to Stage `dev`, and deployment of the
example app. See [#371](https://github.com/konfidence-project/konfidence/issues/371).

## Run

Requires Chainsaw, kubectl, Helm, and registry write access. Use a disposable
cluster with konfidence, Kubernetes Landscape Orchestrator, and Flux installed.

```sh
sh hack/quickstart/quickstart.sh install
export REGISTRY=registry.example/e2e
export REGISTRY_USERNAME=...
export REGISTRY_PASSWORD=...
make test-e2e
```

`COMPONENT_REGISTRY` defaults to `ghcr.io/konfidence-project/example-app`.
`VECTOR_DATA_SERVICE_VERSION` overrides the pinned chart version.

JUnit report:

```sh
mkdir -p e2e-report
make test-e2e CHAINSAW_ARGS="--report-format JUNIT-TEST --report-path e2e-report"
```

Keep resources for debugging:

```sh
make test-e2e CHAINSAW_ARGS="--skip-delete"
```

## Checks

- VectorTemplate, VectorPromotionConfig, Stage, and VectorDeployment report `Ready=True`.
- Candidates and interviews `/healthz` endpoints respond from inside the cluster.
- Pod logs, events, and resource descriptions are collected on failure.

## Cleanup

Chainsaw removes applied resources in reverse order. Project and Landscape
controllers remove their namespaces. This also removes the database and app.
The registry Secret and Flux ConfigMap in `konfidence-system` are applied first
and removed last. `--skip-delete` preserves the resources.

Registry credentials are read at controller startup
([#393](https://github.com/konfidence-project/konfidence/issues/393)); the test
restarts the controller after setup and cleanup.

The kind cluster remains. Remove it with:

```sh
sh hack/quickstart/quickstart.sh uninstall
```
