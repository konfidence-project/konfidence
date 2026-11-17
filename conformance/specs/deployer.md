# Deployer specification

## Preamble

Status: **Draft**.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** in this document are to be interpreted as described in BCP 14 ([RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174)) when, and only when, they appear in all capitals.

Open decisions and potential future changes are outlined where possible and are not conformance requirements.

## Introduction

An `ArtifactDeployment` is the immutable, desired-state record of a single artifact that Konfidence wants deployed. Konfidence core creates it, stamps it with an artifact type, and carries the artifact's payload inside it, but core never acts on that payload. The payload is an [Open Component Model (OCM)](https://ocm.software) component version. OCM is a technology-agnostic standard for packaging and identifying delivery artifacts. Deploying one is delegated to a *deployer*.

A deployer is a controller, or a set of controllers, that reconciles `ArtifactDeployment` objects into concrete deployments on a destination and reports progress and health back through their status.
A deployer owns an artifact type: it defines the type's packaging and payload contract (section 1). Each `ArtifactDeployment` declares its artifact type, which routes the object to the deployer that owns that type.
Konfidence core manages the delivery process. It assembles, verifies, and signs vectors, resolves each stage to a desired state, and aggregates per-artifact results at the vector level. It never deploys workloads itself, and it treats an artifact's payload as opaque.

A destination is whatever can host an artifact: a Kubernetes cluster is one case, but a deployer might equally publish static content to an object store or CDN, apply configuration to a managed cloud service, run a WebAssembly workload on a Wasm runtime, or drive any other runtime.
A deployer is a pluggable component: core and deployer communicate solely through the `konfidence.cloud/v1alpha1` control-plane API and its status.
This boundary is what lets a single Konfidence installation roll out a complex vector whose artifacts together power an entire application, spanning many domain-specific types, each handled by a purpose-built deployer, without core knowing how any individual deployment is performed.

This document specifies the behavior a deployer MUST, SHOULD, and MAY implement to be a conformant participant in that contract.

## 1. Artifact types and packaging contracts

### 1.1 Identifier

An artifact type consists of a descriptive type and a vendor-owned domain:

```bnf
<artifact-type>  ::= <type> "." <domain>
<domain>         ::= <label> "." <label> | <label> "." <domain>
<type>           ::= <label>
<label>          ::= <alphanumeric> | <alphanumeric> <label-middle> <alphanumeric>
<label-middle>   ::= "" | <label-character> <label-middle>
<label-character> ::= <alphanumeric> | "-"
<alphanumeric>   ::= <lowercase-letter> | <digit>
<lowercase-letter> ::= "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z"
<digit>          ::= "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
```

For example, `helm.konfidence.cloud` identifies a Helm artifact contract maintained by `konfidence.cloud`.

The `<domain>` MUST be a lowercase DNS name controlled by the deployer provider.
Each domain label MUST satisfy the DNS length limit of 63 octets.

The `<type>` is a separate identifier and MUST NOT exceed 63 characters under this grammar.
The `<type>` SHOULD describe the artifact's packaging format or deployment mechanism, such as `helm` or `kustomize`.

A provider MUST use a distinct artifact type whenever the payload contract changes in a way that a deployer supporting the original artifact type cannot safely interpret.
This also includes changes where the deployer can read the payload but cannot correctly interpret the described deployment, such as adding a field with semantics it does not support.

### 1.2 Artifact packaging

An artifact is an OCM component version following the [OCM spec](https://github.com/open-component-model/ocm-spec).
The OCM component version identifies the packaged artifact; it is separate from the artifact type.
Its resources MUST contain exactly one resource with the OCM resource type `cloud.konfidence.artifact.manifest`, and that resource MUST be named `konfidence-manifest`.
The content of this resource is the *artifact manifest*.
The artifact manifest MUST be a JSON or YAML object with a `type` property whose value is the complete artifact type as defined in section 1.1.

In an OCM component constructor, the artifact manifest can be written inline with the [`UTF8/v1`](https://ocm.software/docs/reference/input-and-access-types/#utf8v1) input:

```yaml
# component-constructor.yaml
resources:
  - name: konfidence-manifest
    relation: local
    type: cloud.konfidence.artifact.manifest # OCM resource type
    input:
      type: UTF8/v1                          # OCM input type
      yaml:
        type: kustomize.konfidence.cloud     # artifact type
```

The OCM input type only tells the OCM tooling how to read the content when the component version is built. It is not part of this contract.
A `file/v1` input that reads the artifact manifest from a JSON file, such as `./manifest.json` containing `{"type": "kustomize.konfidence.cloud"}`, is equally valid.

The component version MAY contain additional resources that a deployer uses to understand what needs to be deployed.

The provider of an artifact type SHOULD document its packaging and validation requirements.
These requirements MUST identify required and optional OCM resources, their names or types, cardinality, accepted access methods, and expected payload formats.
The provider MUST also document semantic constraints that cannot be checked from the component descriptor alone.

A deployer MUST validate the resources and payloads it relies on before deploying an artifact.
It MUST report invalid artifacts through `ArtifactDeployment.status`, as specified in section 3.

A provider MAY publish a machine-readable schema for structural validation on the resolved OCM component descriptor.
The schema format, how schemas are published and selected by artifact type, and which checks can be performed before payloads are fetched remain to be defined.
Future versions of this specification may introduce requirements about how deployers provide that information to enable seamless integration into Konfidence tooling. 

A provider MAY offer custom tools to create or validate artifacts for their supported types.
Conformance MUST NOT depend on artifacts having been created with provider-specific tooling.

## 2. DeploymentTarget

A `DeploymentTarget` configures a destination for an artifact type in a landscape namespace.
Its `spec.artifactType` contains the artifact type defined in section 1.1.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentTarget
metadata:
  name: helm-production
  namespace: kden-l-prod-fz8g86s6
spec:
  artifactType: helm.konfidence.cloud
  parameters:
    apiGroup: ""
    kind: Secret
    name: production-kubeconfig
status:
  controller: konfidence.cloud/kubernetes-landscape-orchestrator:v1.2.3
  conditions:
    - type: Ready
      status: "True"
      reason: Accepted
```

### 2.1 Ownership and selection

A deployer MUST reconcile only targets whose `spec.artifactType` matches an `<artifact-type>` it supports.
It MUST NOT update the status of targets belonging to other providers or types.
Within a landscape namespace, at most one target for a given `<artifact-type>` is allowed.
If no target exists, an artifact of that type cannot be deployed into the landscape.

### 2.2 Parameters and validation

The deployer defines the interpretation and validation of `spec.parameters`.
Parameters MAY point to any resource a deployer needs to establish a connection to the target runtime for a deployment, including Secrets, ConfigMaps, or deployer-owned custom resources.
It MAY use installation-wide defaults, which apply to every landscape served by that deployer.
It MUST validate parameters before use, and it MUST NOT try to deploy to targets that have validation errors.

It SHOULD document the parameters it accepts and their defaults, and which changes can be applied to an existing target.
Since the exact configuration for establishing a connection is left to the deployer, Konfidence cannot distinguish safe operations (like credential rotations) from harmful operations (like moving to a different cluster).
The deployer SHOULD implement best-effort validation to prevent the user from changing parameters that may cause a service downtime for the deployed applications. 

Some changes require reconciliation even when the `DeploymentTarget.spec` does not change.
The deployer MUST re-evaluate a target when a referenced resource that it relies on changes.

### 2.3 Status

The deployer MUST set the target's `Ready` condition after validating the parameters it needs to accept the target.
`Ready=True` means the deployer accepts the target for deployment.
This does not guarantee that a future artifact deployment will succeed.

The deployer MAY perform additional connectivity or credential checks and report their results in separate deployer-owned conditions.
It MUST set `Ready=False` with a descriptive reason and message when required parameters are invalid.

Core MUST check that the target is ready and its artifact type matches the artifact before treating the target as suitable for deployment.
The deployer SHOULD publish its name and release version in `status.controller`.

The exact shape of `status.capabilities` remains open for now.

### 2.4 Deletion

Deleting a `DeploymentTarget` MUST NOT by itself undeploy artifacts or remove resources at its destination.
Runtime resources remain deployed, but the deployer may lose access to them and can no longer guarantee their continued reconciliation.
If another target for the same artifact type is created, the deployer MUST NOT assume that it refers to the same destination or silently adopt the existing runtime resources.
How core reports the state of affected `ArtifactDeployment` resources after target deletion remains to be specified; their previous readiness MUST NOT be taken as evidence that they are still being reconciled.

## 3. ArtifactDeployment

An `ArtifactDeployment` represents the instantiation of a given artifact in a target runtime by the deployer. 
A *runtime* is the destination environment in which an artifact is deployed to and where it operates, together with the resources managed there on behalf of that artifact.
A runtime may be a Kubernetes cluster, an object store or CDN, a managed cloud service, a bare-metal machine, or any other environment supported by a deployer.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ArtifactDeployment
metadata:
  name: e9cea9b3764b9adda065f85acd63a42f04b69de2d42e9b7e4f9556913bb37abb
  namespace: default
spec:
  artifactType: helm.konfidence.cloud
  component: # TODO: will be refined with https://github.com/konfidence-project/konfidence/issues/397
    name: konfidence.cloud/podinfo-example
    resources:
      - content:
          imageReference: https://stefanprodan.github.io/podinfo:6.9.1
          type: ociArtifact
        name: sample-service-1-helm-chart
        type: helmChart
    version: 1.0.0
```

### 3.1 Ownership and selection

The type in `spec.artifactType` selects the responsible deployer and the corresponding target in the deployment's landscape namespace.
A deployer acts platform-wide and declares the artifact types it supports in its own configuration.
It is the cluster administrator's responsibility to ensure that at most one deployer reconciles a given artifact type in a control plane.
A vendor-owned identifier prevents accidental naming conflicts, but it cannot prevent two controllers from being configured for the same type.

The deployer MUST reconcile an `ArtifactDeployment` only when its artifact type matches a supported identifier exactly.
It MUST do so regardless of whether they were created before the deployer was installed or while it was stopped.
It MUST NOT reconcile resources whose `spec.artifactType` is not known to the deployer.

### 3.2 Input validation

The deployer MUST validate the `ArtifactDeployment.spec.component` specification against the packaging contract for its artifact type.
This includes the required OCM resources and the payloads it consumes.
It MUST NOT solely rely on validation performed by an artifact authoring tool or by core.

Validation MAY fetch external resources or payloads that are referenced in the specification.
Validation is a best-effort determination to make sure that the inputs satisfy the packaging contract.
Successful validation does not guarantee that deployment of the artifact will succeed.

### 3.3 Reconciliation

The deployer MUST reconcile supported artifacts idempotently, including those created before it was installed or while it was stopped.
For each `ArtifactDeployment`, the deployer MUST:

1. Validate its component resources and payloads as described in section 3.2.
2. Find the `DeploymentTarget` for the artifact type in the deployment's landscape namespace as described in section 2.
   The target MUST have a `Ready=True` condition and its `spec.artifactType` MUST match exactly.
   If no such target exists, the deployer MUST write a `Stalled=True` condition with reason `NoReadyDeploymentTarget`.
3. Use the target's parameters to connect to the target runtime.
   The deployer MAY cache the credentials or reuse connections across different reconciliations, including across targets that refer to the same runtime infrastructure.
4. Apply the resources described by `ArtifactDeployment.spec.component` to the target runtime.
   The deployer MAY create intermediate resources (including custom resources on the control plane) to track or identify the runtime resources managed by the `ArtifactDeployment`.
   If runtime resources corresponding to an `ArtifactDeployment` already exist, the deployer MAY adopt and manage them instead of creating duplicate resources.
   Before adoption, it MUST establish that the resources correspond to that exact `ArtifactDeployment` and are safe for it to manage.
5. Publish deployment results in `ArtifactDeployment.status.deploymentResult` as specified in section 4.

The deployer MAY record events on an `ArtifactDeployment` to help users understand the deployment process. 
Events are informational and do not replace the status conditions used to report deployment state.

### 3.4 Status

The deployer MUST report reconciliation progress using the `Reconciling`, `Stalled`, and `Ready` conditions. 
These conditions are mutually exclusive.

The deployer MUST maintain the following conditions:

* It MUST ensure that no more than one of `Reconciling`, `Stalled`, and `Ready` is `True` simultaneously.
* `Ready` MUST be present after the initial status update and MUST have a status of `True` or `False`.
* `Reconciling` MUST be present with `status=True` while the deployment has not reached the desired state and the deployer expects automatic reconciliation to make further progress. 
It SHOULD be removed once that is no longer the case (abnormal-true condition).
* `Stalled` MUST be present with `status=True` when reconciliation is stalled and SHOULD be removed when reconciliation is no longer stalled (abnormal-true condition).

The deployer MAY report additional deployment-related information in separate deployer-owned conditions.
These conditions MUST NOT change the meaning of `Reconciling`, `Stalled`, or `Ready`.

The deployer SHOULD NOT report `Reconciling=False` or `Stalled=False`. 
Their absence MUST be interpreted as `False`.

#### Condition semantics

**`Reconciling=True`**

Deployment of the artifact is in progress.
It MUST be set while the deployer is actively applying changes or waiting for the target runtime to converge to the desired state.
This condition is independent of whether a controller is currently executing its `Reconcile` method.
It MAY remain `True` across multiple reconcile calls, including while the deployer is waiting for a runtime resource to become ready.
It MUST NOT be set when further automatic reconciliation cannot make progress.
Its `reason` and `message` MAY be used to describe the current activity of the deployer.  

**`Stalled=True`** 

Reconciliation cannot make further progress. 
The deployer has exhausted its attempts, and reaching the desired state requires an external change or intervention.
It MUST be set when the deployer determines that it cannot reach the desired state through further automatic reconciliation.
It MUST NOT be used for transient failures that the deployer continues to retry automatically.
Its `reason` SHOULD be used to identify the cause of the failure and provide an actionable `message` explaining the problems.

**`Ready=True`** 

The desired state has been reached in the target runtime.
It MUST be set when the deployer has successfully reconciled the resource and verified that the target runtime satisfies the desired state.
It MUST NOT be set merely because deployment operations have been submitted successfully.
Before setting `Ready=True` for the first time, the deployer MUST finish determining the complete set of deployment results required by the artifact's packaging contract, as specified in section 4.

#### Examples

| Condition | Reason | Message |
|---|---|---|
| `Reconciling=True` | `ApplyingResources` | Applying deployment resources to the target cluster. |
| `Reconciling=True` | `WaitingForReadiness` | Waiting for Deployment `backend` to become available. |
| `Reconciling=True` | `RetryingOperation` | Failed to connect to the target cluster. Retrying automatically. |
| `Stalled=True` | `InvalidConfiguration` | Deployment configuration is invalid: required field `image` is missing. |
| `Stalled=True` | `AuthenticationFailed` | Authentication to the target cluster failed. |
| `Stalled=True` | `RetryLimitExceeded` | Deployment failed after 5 attempts. |
| `Ready=True` | `ReconciliationSucceeded` | Artifact successfully deployed in target runtime. |

### 3.5 Deletion and undeployment

When an `ArtifactDeployment` is deleted, the deployer MUST remove the runtime resources it manages on its behalf, including resources on a remote runtime.
The deployer MUST NOT consider deletion complete while any such resources are known or reasonably believed to remain deployed.
The deployer SHOULD use a Kubernetes finalizer or an equivalent mechanism to keep the `ArtifactDeployment` available until cleanup succeeds.

If cleanup fails, including because credentials are unavailable or access to the target runtime is lost, the deployer MUST continue with its cleanup procedure until it succeeds.
While cleanup is blocked, the deployer SHOULD report cleanup progress with the `Reconciling=True` condition and an appropriate `reason` and `message`.

Deleting the `DeploymentTarget` before deleting associated `ArtifactDeployments` can leave those deployments pending deletion until access to the original destination is restored.

## 4. Deployment results

A deployment result carries information that are discovered after or during the deployment of an artifact.
It is intended to be used by a later deployment step or during service runtime.
Deployment results are published in `ArtifactDeployment.status.deploymentResult`.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ArtifactDeployment
# ...
status: 
  deploymentResults:
    - name: http-api
      type: http-k8s-service
      spec:
        K8sName: candidates-0-1-0-f486ecb-hd4d84jb4k
        Namespace: kden-l-dev
        ServicePorts:
         - name: http
           port: 80
           protocol: TCP
           targetPort: http
```

### 4.1 Format

A deployment result consists of a `name`, a `type`, and an opaque `spec`:

- `name` identifies the result within an artifact.
- `type` identifies what the result contains and how `spec` is to be interpreted
- `spec` holds the result’s structured data, its format is defined by the result's `type`.

The pair `(name, type)` MUST be unique within an `ArtifactDeployment`.

The deployer MUST document the payload and intended consumers of each result type it publishes.
This includes whether its results are meaningful only within a particular runtime or if they can be used across runtimes.

### 4.2 Publishing

The deployer MUST specify which deployment result types it supports and under which conditions they are reported.
For each deployment result type, a deployer MUST document how results are selected for publication: 

- it MAY publish a result automatically for every artifact it deploys
- or publish a result only when the artifact’s payload requests or supplies it.
 
The deployer MUST validate and publish the deployment result according to its documented contract of the result type.
If its contract requires a result, the deployer MUST NOT report `ArtifactDeployment` as `Ready=True` until that result has been discovered and published.
The deployer MUST determine the complete result set and report it to `ArtifactDeployment.status.deploymentResults` before setting the `Ready=True` status condition.
Before reporting `Ready=True` for the first time, the deployer MAY publish and revise provisional results.

An artifact MAY produce no deployment results at all.

### 4.3 Immutability

The complete deployment result set MUST NOT change after the first time the `ArtifactDeployment` reports `Ready=True`.
Results are compared as a set identified by `(name, type)`.
Changing a result's `spec`, adding or removing a result, or replacing an empty set with a nonempty one changes that set.
Changing only the order of results does not change the set.

If the deployer encounters a situation where it reported `Ready=True` and detects that a deployment result is no longer intact, it MUST try to re-establish the exact runtime resources that caused that deployment result. 
If that is not possible, the deployer must report `Stalled=True` with reason `DeploymentResultChanged`.
This causes the entire vector to be unhealthy, because Konfidence can no longer ensure consistent behavior within that vector. 

## 5. VectorData

`VectorData` is an immutable, vector-scoped snapshot of authored configuration, feature settings, and deployment results from the vector’s artifacts. 
Deployers make it available in their runtimes so artifacts can use the data for that specific vector.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorData
metadata:
  name: dev-stable-features-44d8v8p5s8ls7
  namespace: kden-l-dev
spec:
  deploymentResults:
    github.com/konfidence-project/example-app/candidates: [ ... ]
  features:
    allow-video-slots: false
    enable-candidate-notes: false
```

### 5.1 Ownership and selection

A deployer MUST reconcile a `VectorData` when `spec.artifactTypes` contains at least one artifact type it supports.
For each such type, it MUST use the corresponding `DeploymentTarget` in the landscape namespace to determine the destination.
Each deployer defines the runtime boundary for its destinations and determines whether its supported types use vector-scoped data there.

A single `VectorData` resource can be reconciled by multiple deployers concurrently.
Deployer MUST make sure to gracefully handle conflicting status updates. 

### 5.2 Input validation

The deployer MUST validate `VectorData` before provisioning.
Validation includes general checks if the deployer is able to provision the `VectorData` into the desired destination, as well as validating the set of `spec.deploymentResults` across the vector. 
It is the deployer's responsibility to find conflicts in deployment results, that a single artifact cannot detect (e.g. if two different artifacts claim the same route in their deployment results). 
For that, the deployer MUST only interpret deployment results it understands itself. 

If validation fails, the deployer MUST NOT begin provisioning and MUST NOT report readiness for the affected artifact type.

### 5.3 Reconciliation 

Provisioning of `VectorData` is OPTIONAL, depending on the needs and capabilities of the runtime. 
Where the runtime can use vector-scoped data, the deployer MUST provision the complete `VectorData` for that runtime so artifacts can access it by the means defined for that runtime.
Where the runtime cannot use vector-scoped data, the deployer MAY omit runtime provisioning, but MUST still perform validation and report readiness for each supported artifact type in the vector.

The deployer SHOULD make the complete snapshot available in one operation where the runtime permits it.
It MAY use multiple operations if the runtime does not provide a mechanisms to do it in a single operation. 
It MUST NOT report readiness until the entire snapshot is available and verified by the means defined for that runtime.
Artifacts may cache the contents of `VectorData` indefinitely, so the deployer MUST make sure that the complete snapshot is available before reporting readiness. 

If the deployer discovers that provisioned data is unavailable, does not match the immutable `VectorData`, or cannot be verified, it MUST withdraw its readiness contribution and reconcile toward the immutable snapshot.

### 5.4 Status

[//]: # (TODO: https://github.com/konfidence-project/konfidence/issues/395)

The deployer MUST report a separate readiness contribution for each artifact type it supports in `VectorData.spec.artifactTypes`.
Each contribution MUST identify both the artifact type and the deployer.

The exact representation of status reporting remain open in [konfidence#395](https://github.com/konfidence-project/konfidence/issues/395).

### 5.5 Deletion

When `VectorData` is deleted, the deployer MUST remove the data it manages on its behalf.
It MUST NOT consider deletion complete while any such data is known or reasonably believed to remain.
It SHOULD use a finalizer or equivalent mechanism to retain `VectorData` until cleanup succeeds.
If target connection information is lost, deletion MUST remain pending until cleanup can succeed.

## 6. Vector association and artifact reuse

* **Ownership and resource selection:** Core records each vector/artifact association in `VectorDeployment.status.resultingArtifactDeployments` and uses `ArtifactManifest.allowReuse` when creating the `ArtifactDeployment`. One artifact may be referenced by several vectors; the deployer for the artifact type handles vector-specific work.
* **Interpretation of inputs / generic fields:** A reusable `ArtifactDeployment` MUST NOT depend on vector-specific runtime context. Vector-scoped authored data and aggregated deployment results are transported through `VectorData`. When reuse is enabled, a deployer MUST NOT infer a single vector from the artifact alone.
* **Processing and guarantees, failure/retry handling:** Vector-specific data MUST remain isolated across vectors sharing an artifact. Define what happens if an artifact requests reuse but the deployer cannot support it, and how missing references or failed assignment work are handled.
* **Status reporting via conditions:** `ArtifactDeployment.status` is the source of artifact readiness and deployment results. `VectorDeployment` reflects the aggregate readiness of its referenced artifact deployments.
* **Resource deletion and undeployment:** Removing one vector MUST NOT undeploy an artifact still in use by another; define cleanup of its vector-specific state.
* **Optional behavior:** Decide whether reuse is an artifact-type capability.

## 7. Cross-cutting status, reconciliation, and safety

[//]: # (TODO mention namespaced name of reconciled resource in deployer logs?)

For every condition, specify its owner, meaning of `True`/`False`/`Unknown`, reason codes, `observedGeneration`, and transitions after dependencies recover or disappear. Core and deployer MUST NOT leave a stale `Ready=True` that implies a known-unavailable deployment is healthy. Distinguish retryable failures from failures requiring user action; reconcile when mutable referenced resources change, even if the owning resource's spec is immutable. Define idempotency, event/watch versus periodic resync guarantees, status-update conflicts, permissions, credential handling, and whether sensitive data may appear in status or deployment results. These rules need a condition/ownership table before this draft becomes normative.

## 8. Future extension: tasks and activation

`TaskExecution`, `ActivationTaskExecution`, `VectorMigration`, and `VectorActivation` exist today; KLO implements Kubernetes Jobs and Gateway API HTTPRoutes for some executions. This draft does **not** require every deployer to perform migrations, activation, or routing. A later specification should define registration/selection of task and activation executors—possibly using a resource analogous to `DeploymentClass`—with their own status and capability contracts. The base deployment contract MUST NOT assume those operations are built into the same controller.

## 9. Versioning and conformance

Before declaring a specification version, define how it relates to CRD versions and any published Go bindings, how compatible changes are introduced, and how optional capabilities are declared. Each normative clause should receive an identifier for conformance checks. Validate the mandatory subset against KLO and a substantially different target platform; track KLO deviations as implementation gaps rather than weakening requirements to match existing behavior.

Initial decision list: target multiplicity and mutation; condition ownership/fallback; meaning of artifact and vector readiness; reuse capability; VectorData implementor selection; result versus inventory schemas; cleanup guarantees; capability discovery and compatibility policy.
