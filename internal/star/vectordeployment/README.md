[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/konfidence/internal/star/vector-deployment)](https://api.reuse.software/info/github.com/konfidence-project/konfidence/internal/star/vector-deployment)

# star-vector-deployment-controller

## About this project

The Deployment Controller is responsible for managing the lifecycle of VectorDeployments, including downloading the vector image from the OCI repository and creating the related ArtifactDeployments and VectorAssignments.

### Lifecycle phases

1. **VectorDownloaded** — pull the vector OCM ComponentVersion from the OCI registry and persist its V2-serialized descriptor on the VectorDeployment status.
2. **ArtifactDeploymentsCreated / VectorDeployed** — create one ArtifactDeployment per artifact reference, then wait until each underlying deployer marks its ArtifactDeployment Ready.
3. **VectorAssignmentsCreated** — create the per-artifact VectorAssignment resources that the routing layer reconciles.
4. **VectorConfigCommitted** — materialize the vector-scoped configuration ConfigMap (see below). This step is a singleton action per vector and runs after all artifact deployments are Ready, so that aggregated `DeploymentResults` are observable.
5. **VectorReady** — set when all of the above have completed.

### Vector-scoped configuration ConfigMap

A vector may carry an optional, singleton OCM resource named `cloud-konfidence-vector-config` (matched on the resource Name; the OCM Type field is left to the vector author and is not used for discovery; dots are not permitted in OCM resource names per the schema, hence the dashes). The deployment controller materializes that resource (together with the aggregated deployment results from all underlying ArtifactDeployments) as a Kubernetes ConfigMap in the **landscape namespace** — that is, the same namespace as the `VectorDeployment` itself, per [ADR-0007](../../../../docs/pages/arc42/09-decisions/adrs/0007-lcp-multi-landscape-support.md). The LCP's own namespace (`konfidence-system`) is **not** used; one LCP installation can serve many landscapes, and each landscape sees only its own ConfigMaps.

Both inputs that feed the payload are immutable for the lifetime of a `VectorDeployment` (the OCM ComponentVersion is immutable and `ArtifactDeployment.spec` is immutable per the CRD's XValidation rule), so the controller writes the ConfigMap **at most once** per VectorDeployment. If a ConfigMap with the expected name already exists the controller honors it as-is; if it is missing — first reconcile after `VectorDeployed`, or removed out-of-band — it is created.

On VectorDeployment teardown the controller deletes the ConfigMap explicitly via the `konfidence.cloud/vector-data-cleanup` finalizer, and only then removes the finalizer so that the API server can finalize the VD object. The controller-owner reference is also set on the ConfigMap so that Kubernetes garbage collection cascades as a fallback.

| Property | Value |
|---|---|
| Name | `vector-data-<vectorDeploymentName>` |
| Namespace | `<vectorDeployment.Namespace>` |
| `Immutable` | `true` |
| Owner reference | controller-owner ref to the `VectorDeployment` (cascade delete on VD removal). |
| Finalizer on the VD | `konfidence.cloud/vector-data-cleanup`, drives explicit teardown. |
| Data keys | two independently-accessible JSON documents (see below). |
| Labels | `konfidence.cloud/managed-by=vector-deployment-controller`, `konfidence.cloud/vector-deployment-name=<vd.Name>`, `konfidence.cloud/vector-deployment-uid=<vd.UID>`. |

Data layout:

| Key | Content |
|---|---|
| `config.json` | The optional authored configuration blob, forwarded byte-for-byte from the OCM resource. The literal `null` when the vector did not declare such a resource. |
| `deployment-results.json` | The aggregated `DeploymentResults` of all underlying ArtifactDeployments, keyed `<componentName>/<resultName>`. Always materialized; an empty map (`{}`) when no artifact has produced any results. |

Two distinct keys (instead of a single bundled `data.json`) keep the two semantically-independent concerns separately accessible — both for `kubectl get cm -o jsonpath='{.data.config\.json}'` and for the future per-landscape vector data service which can serve them on independent endpoints.

The authored blob itself is **not** persisted on the `VectorDeployment.status` — only its SHA-256 hash (`status.resolvedVectorConfigHash`) is, as a small etcd-friendly fingerprint for traceability. On the rare path where the controller has to recreate a missing ConfigMap on a later reconcile, the blob is re-fetched lazily via the OCM adapter.

Payload shape (for reference; not persisted as a single document — see *Data layout* above):

```json
{
  "config": <opaque JSON forwarded from the cloud-konfidence-vector-config OCM blob, or null>,
  "deploymentResults": {
    "<componentName>/<resultName>": { "name": "...", "type": "...", "spec": { ... } }
  }
}
```

The two keys are **always written** (even when there is neither authored configuration nor any deployment results) so that downstream consumers can rely on their presence after `VectorReady` flips True.

The ConfigMap is consumed by a per-landscape vector data service (planned, see [ADR-0024](../../../../docs/pages/arc42/09-decisions/adrs/0024-vector-scoped-configuration-distribution.md)). Application pods are not expected to read the ConfigMap directly.

## Requirements and Setup

### OCM Authentication Setup

When accessing OCM artifacts the Deployment Controller needs (optional) authentication credentials to access one or more (remote) OCI registries.
First the controller extracts the domain name from the OCI registry url and then tries to lookup a Secret reference by domain name in a controller specific ConfigMap.
If a matching reference is found the credentials are extracted from the referenced Secret. If the ConfigMap does not exist or no matching entry has been found
the controller tries to directly get the Secret by domain name. If this Secret also not exists then no OCI credentials are configured.

So there are two options to configure the OCI registry credentials:
1. A Deployment Controller specific ConfigMap `vector-deployment-controller-configuration` in the `konfidence-system` namespace that contains one or more Secret references by domain name.
2. A Secret with the domain as name that exists in the namespace of the vector deployment resource

> **Note:**
> At the moment the secret has to be of type `kubernetes.io/dockerconfigjson`

#### Example with config map
1. Create a Secret with the OCI credentials of type `kubernetes.io/dockerconfigjson`. The Secret must be in the same namespace as the vector deployment resource.
```yaml
apiVersion: v1
data:
  .dockerconfigjson: eyJhdXRocyI6eyJyZWdpc3RyeS5leGFtcGxlLmNvbSI6eyJ1c2VybmFtZSI6InRlc3QiLCJwYXNzd29yZCI6InRlc3QiLCJlbWFpbCI6InRlc3RAc2FwLmNvbSIsImF1dGgiOiJkR1Z6ZERwMFpYTjAifX19
kind: Secret
metadata:
  creationTimestamp: "2026-01-05T11:36:58Z"
  name: my-secret-123
  namespace: default
  resourceVersion: "1766473"
  uid: c17e1c74-e10b-4d1f-99a3-0a3da7f4b5a8
type: kubernetes.io/dockerconfigjson
```

2. Create the ConfigMap referencing the Secret by domain name. The ConfigMap must be in the `konfidence-system` namespace.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vector-deployment-controller-configuration
  namespace: konfidence-system
data:
  authenticationSecretRefs: |
    registry.example.com: my-secret-123
```

### Setup Git hooks

We use git hooks to check the conventional-commit formatting at "commit-msg".

```bash
make install-git-hooks    # install all git hooks with prek
make uninstall-git-hooks  # uninstall all git hooks with prek
```

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.
