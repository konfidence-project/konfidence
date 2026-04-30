[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/landscape-gcp-sync-controller)](https://api.reuse.software/info/github.com/konfidence-project/landscape-gcp-sync-controller)

# star-galaxy-sync-controller

## About this project

The Star Galaxy Sync Controller is a Kubernetes controller that keeps `Stage` objects on a **local (star) cluster** in sync with `StageSync` objects managed on a **remote (galaxy) cluster**. It bridges the two clusters by watching the remote cluster for `StageSync` resources and reconciling the desired state onto the local cluster.

## How it works

### Trigger

The controller watches `StageSync` objects on the **remote (galaxy) cluster**. Every create, update, or delete of a `StageSync` triggers a reconcile, during which a corresponding `Stage` object is created or updated on the **local (star) cluster** according to the template embedded in the `StageSync` spec. In addition, the controller re-reconciles each `StageSync` periodically (default: every 30 seconds) to correct any drift.

### Reconcile flow

For each `StageSync` on the remote cluster the controller:

1. Fetches the `StageSync` from the remote cluster.
2. Validates that the `Stage` API version referenced in the `StageSync` template is present and served on the local cluster — if not, reconciliation stops and the `StageSync` status is set to `Applied=False` with reason `APIVersionNotSupported`.
3. Checks that the target namespace specified in the template exists on the local cluster — if not, reconciliation stops and the status is set to `Applied=False` with reason `NamespaceNotFound`.
4. Checks for a conflicting unmanaged `Stage` — if a `Stage` with the same name and namespace already exists but is not owned by this `StageSync`, reconciliation stops with reason `ConflictWithUnmanagedStage`.
5. Creates or updates the `Stage` on the local cluster with the spec from the template, labelling it `managed-by: <stagesync-namespace>_<stagesync-name>`.
6. Reflects the `Stage`'s own status conditions back onto the `StageSync`.
7. Updates the `StageSync` status to `Applied=True` with reason `StageReconcileSuccessful`.

### Deletion

A finalizer (`konfidence.cloud/stage-sync-finalizer`) is added to each `StageSync` on first reconcile. When a `StageSync` is deleted, the controller removes the corresponding local `Stage` before releasing the finalizer, ensuring no orphaned resources are left behind.

### Guard rails

| Situation | Behaviour |
|---|---|
| `Stage` API version in template not served on local cluster | Reconciliation stops; `Applied=False / APIVersionNotSupported` |
| Target namespace does not exist on local cluster | Reconciliation stops; `Applied=False / NamespaceNotFound` |
| A `Stage` with the same name exists but is not managed by this `StageSync` | Reconciliation stops; `Applied=False / ConflictWithUnmanagedStage` |

## Requirements and Setup

### Environment Variables

#### `LANDSCAPE_NAME` *(required)*

The `LANDSCAPE_NAME` environment variable sets the name of the landscape (star cluster) the controller is running on. It is used to label `StageSync` objects on the remote cluster with `synced-by-star/<landscape-name>`, allowing tracking of which landscape cluster is managing a given sync.

#### `CONTROLLER_NAMESPACE` *(auto-injected)*

Injected automatically via the Kubernetes Downward API. Used to locate the remote kubeconfig Secret. Falls back to `default` when not set.

### Kubeconfig Secret for Remote Cluster (galaxy)

The controller reads the remote (galaxy) cluster kubeconfig from a Secret named `galaxy-sync-kubeconfig`, with the kubeconfig YAML stored under the key `kubeconfig`.

The lookup namespace for the secret is resolved from the `CONTROLLER_NAMESPACE` environment variable (injected via the Downward API) and falls back to `default` when not set.

> **Single-cluster mode:** if the Secret is not present, the controller uses the local cluster as both clusters — useful for development and testing.


### Secret Creation

Create the Secret from a kubeconfig file:

```bash
kubectl create secret generic galaxy-sync-kubeconfig \
  --from-file=kubeconfig=/path/to/your/kubeconfig.yaml \
  --namespace=<controller-namespace>
```

To verify the Secret was created correctly, decode it back to plain text:

```bash
kubectl get secret galaxy-sync-kubeconfig \
  --namespace=<controller-namespace> \
  -o jsonpath='{.data.kubeconfig}' | base64 --decode
```

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
