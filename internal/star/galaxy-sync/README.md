[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/landscape-gcp-sync-controller)](https://api.reuse.software/info/github.com/konfidence-project/landscape-gcp-sync-controller)

# landscape-gcp-sync-controller

## About this project

The Landscape GCP Sync Controller is a Kubernetes controller that keeps `Stage` objects on a **local (LCP) cluster** in sync with `StageSync` objects managed on a **remote (GCP) cluster**. It bridges the two clusters by watching the remote cluster for `StageSync` resources and reconciling the desired state onto the local cluster.

## How it works

### Trigger

The controller watches `StageSync` objects on the **remote (GCP) cluster**. Every create, update, or delete of a `StageSync` triggers a reconcile, during which a corresponding `Stage` object is created or updated on the **local (LCP) cluster** according to the template embedded in the `StageSync` spec. In addition, the controller re-reconciles each `StageSync` periodically (default: every 30 seconds) to correct any drift.

### Reconcile flow

For each `StageSync` on the remote cluster the controller:

1. Fetches the `StageSync` from the remote cluster.
2. Validates that the `Stage` API version referenced in the `StageSync` template is present and served on the local cluster — if not, reconciliation stops and the `StageSync` status is set to `Applied=False` with reason `APIVersionNotSupported`.
3. Checks that the target namespace specified in the template exists on the local cluster — if not, reconciliation stops and the status is set to `Applied=False` with reason `NamespaceNotFound`.
4. Checks for a conflicting unmanaged `Stage` — if a `Stage` with the same name and namespace already exists but is not owned by this `StageSync`, reconciliation stops with reason `ConflictWithUnmanagedStage`.
5. Creates or updates the `Stage` on the local cluster with the spec from the template, labelling it `managed-by: <stagesync-namespace>_<stagesync-name>`.
6. Reflects the `Stage`'s own status conditions back onto the `StageSync`.
7. Updates the `StageSync` status to `Applied=True` with reason `StageCreationSuccessful`.

### Deletion

A finalizer (`konfidence.cloud/stage-sync-finalizer`) is added to each `StageSync` on first reconcile. When a `StageSync` is deleted, the controller removes the corresponding local `Stage` before releasing the finalizer, ensuring no orphaned resources are left behind.

### Guard rails

| Situation | Behaviour |
|---|---|
| `Stage` API version in template not served on local cluster | Reconciliation stops; `Applied=False / APIVersionNotSupported` |
| Target namespace does not exist on local cluster | Reconciliation stops; `Applied=False / NamespaceNotFound` |
| A `Stage` with the same name exists but is not managed by this `StageSync` | Reconciliation stops; `Applied=False / ConflictWithUnmanagedStage` |

## Requirements and Setup

### Kubeconfig Secret for Remote Cluster (GCP)

The controller reads the kubeconfig for the remote (GCP) cluster from a Kubernetes Secret named `gcp-sync-kubeconfig` in the local cluster.
The Secret must contain a key named `kubeconfig` with the kubeconfig YAML as its value.

Create the Secret from a kubeconfig file:

```bash
kubectl create secret generic gcp-sync-kubeconfig \
  --from-file=kubeconfig=/path/to/your/kubeconfig.yaml \
  --namespace=<controller-namespace>
```

The controller resolves the lookup namespace from the `CONTROLLER_NAMESPACE` environment variable, which is automatically injected via the Downward API when deployed with the provided manifests. It falls back to `default` when the variable is not set (e.g. during local development).

> **Single-cluster mode:** if the Secret is not present, the controller uses the local cluster as both the local and the remote cluster. This is useful for development and testing.


### Setup Git hooks

We use git hooks to check the conventional-commit formatting at "commit-msg".

```bash
make install-git-hooks    # install all git hooks with prek
make uninstall-git-hooks  # uninstall all git hooks with prek
```

## Support, Feedback, Contributing

This project is open to feature requests, suggestions, and bug reports via [GitHub issues](https://github.com/konfidence-project/landscape-gcp-sync-controller/issues). Contributions are encouraged and always welcome. For more information on how to contribute, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure

If you find a bug that may be a security problem, please follow the instructions in our [security policy](https://github.com/konfidence-project/landscape-gcp-sync-controller/security/policy) on how to report it. Please do not create GitHub issues for security-related concerns.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2026 SAP SE or an SAP affiliate company and landscape-gcp-sync-controller contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available via the [REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/landscape-gcp-sync-controller).
