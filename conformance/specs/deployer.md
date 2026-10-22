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
Its resources MUST contain exactly one resource of type `cloud.konfidence.artifact.manifest`, and that resource MUST be named `konfidence-manifest`.
This resource MUST contain a JSON object with a `type` property whose value is the complete artifact type:

[//]: # (TODO: can we use type: UTF8/v1 instead and use inline yaml)

```yaml
# component-constructor.yaml
resources:
  - name: konfidence-manifest
    relation: local
    type: cloud.konfidence.artifact.manifest
    input:
      type: file/v1
      path: ./konfidence-manifest.json
```

The referenced JSON file contains the artifact's type declaration:

```json
{
  "type": "kustomize.konfidence.cloud"
}
```

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

An `ArtifactDeployment` represent the instantiation of a given artifact in a target runtime by the deployer. 
A *runtime* is the destination environment in which an artifact is deployed to and where it operates, together with the resources managed there on behalf of that artifact.
A runtime may be a Kubernetes cluster, an object store or CDN, a managed cloud service, a bare-metal machine, or any other environment supported by a deployer.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ArtifactDeployment
metadata:
  name: e9cea9b3764b9adda065f85acd63a42f04b69de2d42e9b7e4f9556913bb37abb
  namespace: default
spec:
  # TODO investigate OCM controller to replace component with OCM CRD
  component:
    name: konfidence.cloud/podinfo-example
    resources:
      - content:
          imageReference: https://stefanprodan.github.io/podinfo:6.9.1
          type: ociArtifact
        name: sample-service-1-helm-chart
        type: helmChart
    version: 1.0.0
  manifest: # TODO drop manifest struct
    allowReuse: true  # TODO drop from ArtifactDeployment.spec
    type: helm.konfidence.cloud # TODO rename to spec.artifactType
  taskManifests: [ ] # TODO drop taskManifests

```

### 3.1 Ownership and selection

The type in `spec.artifactType` selects the responsible deployer and the corresponding target in the deployment's landscape namespace.
A deployer acts platform-wide and declares the artifact types it supports in its own configuration.
It is the cluster administrator's responsibility to ensure that at most one deployer reconciles a given artifact type in a control plane.
A vendor-owned identifier prevents accidental naming conflicts, but it cannot prevent two controllers from being configured for the same type.

The deployer MUST reconcile an `ArtifactDeployment` only when its artifact type matches a supported identifier exactly.
It MUST leave resources belonging to other providers or types to their responsible deployers.
It MUST reconcile matching `ArtifactDeployment` resources regardless of whether they were created before the deployer was installed or while it was stopped.

[//]: # ( TODO find a better section for this)
If runtime resources corresponding to an `ArtifactDeployment` already exist, the deployer MAY adopt and manage them instead of creating duplicate resources.
Before adoption, it MUST establish that the resources correspond to that `ArtifactDeployment` and are safe for it to manage.

### 3.2 Input validation

The deployer MUST validate the `ArtifactDeployment.spec.component` specification against the packaging contract for its artifact type.
This includes the required OCM resources and the payloads it consumes.
It MUST NOT rely on validation performed by an artifact authoring tool or by core.
Validation MAY fetch external resources or payloads that are referenced in the specification.
Validation is a best-effort determination that the inputs satisfy the packaging contract.
Successful validation does not guarantee that deployment will succeed.

### 3.3 Reconciliation and failure handling

The deployer MUST reconcile supported artifacts idempotently, including those created before it was installed or while it was stopped.
It MUST complete input validation before creating or changing runtime resources.
It MUST NOT deploy an artifact whose inputs it has determined are invalid.

It MUST report input validation failures through `ArtifactDeployment.status`, distinguishing them from failures during artifact retrieval, target access, deployment, and health evaluation.
For each reported failure, the condition `reason` and `message` MUST identify the failed operation and provide enough information to determine whether user action is required; 
`reason` SHOULD distinguish failures expected to clear on retry from failures requiring user action.
The deployer MAY have partially changed runtime resources if deployment fails after validation; it MUST report the failure and continue reconciling toward the desired state or, on deletion, cleanup.
If the target is absent or unready, the artifact MUST NOT be deployed through that target.
If a target becomes absent or unready after deployment, the deployer MUST withdraw any stale `Ready=True` by setting `Ready=False` when it knows a readiness prerequisite has failed, or `Ready=Unknown` when it cannot determine whether readiness prerequisites hold.
It MUST preserve existing runtime resources while the target is unavailable and MUST resume reconciliation when the target becomes usable again.

[//]: # (TODO mention artifactDeployment namespaced name in deployer logs)

### 3.4 Status

The deployer MUST report the artifact lifecycle through the `Ready` conditions in `ArtifactDeployment.status`.
`ArtifactFetched`, `ArtifactDeployed` are optional
`Stalled` condition -> check kstatus: https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md

* `ArtifactFetched=True` means that the deployer has retrieved and can read every OCM resource or payload required by its packaging contract. If no retrieval is required, it MUST set this condition to `True` with reason `NotRequired`.
* `ArtifactDeployed=True` means that all runtime resources required by the artifact type have been deployed or safely adopted, and all deployment results required by that type have been published in `status.deploymentResult`. An artifact type that requires no deployment results MUST set this condition to `True` with reason `NoResultsRequired` once its runtime resources are deployed or adopted.
* `Ready=True` means that all applicable prerequisites defined by the artifact type are satisfied, including successful input validation, required runtime deployment, required deployment results, and required health criteria.

For each condition, the deployer MUST set `status` to `False` when it knows the represented requirement has failed, and to `Unknown` when it cannot determine whether that requirement is satisfied. It MUST update conditions when its observation changes and MUST NOT leave `Ready=True` when it knows or cannot determine that a readiness prerequisite is satisfied.
When `Ready` is `False` or `Unknown`, its `reason` and `message` MUST identify the blocking prerequisite or uncertainty. Condition `reason` values SHOULD distinguish failures expected to clear on retry from failures requiring user action; `message` MUST describe the observed failure or uncertainty.
The deployer MUST set each condition's `observedGeneration` to the generation of the `ArtifactDeployment` whose specification it has observed.

### 3.5 Deletion and undeployment

The deployer MUST add a finalizer to an `ArtifactDeployment` before creating runtime resources for it.
On deletion, it MUST remove the runtime resources it created on behalf of that `ArtifactDeployment`, including resources on a remote runtime, before removing its finalizer.
It MUST NOT remove the finalizer while any such resources are known or reasonably believed to remain deployed.
This cleanup obligation applies only to resources the deployer created on behalf of the `ArtifactDeployment`, including resources created by an earlier instance or version of that deployer.
Adoption alone does not make an independently created resource subject to this cleanup obligation; the deployer MUST NOT delete such a resource unless it also created it on behalf of the `ArtifactDeployment`.
The deployer MUST retain the finalizer if cleanup fails, including when credentials are unavailable or access to the target runtime is lost, and MUST retry cleanup until it succeeds.
Retry timing and configuration are implementation-defined.
While cleanup is blocked, the `ArtifactDeployment` remains pending deletion (Kubernetes `metadata.deletionTimestamp` is set); the deployer SHOULD report the cleanup progress or blocking reason in status.
Consequently, deleting the target before deleting its artifact deployments can leave those deployments pending deletion until access to the original destination is restored.

### 3.6 Optional behavior

The packaging contract for each artifact type MUST state whether health criteria and deployment results are required.
An artifact type MAY define neither; in that case `AppHealthy` MUST be reported as not required, and `ArtifactDeployed` MUST be `True` once its required runtime resources have been deployed or adopted.

## 4. Deployment results and deployed-instance visibility

* **Ownership and resource selection:** Deployer publishes results in `ArtifactDeployment.status.deploymentResult`; core aggregates them by component in `VectorDeployment` and `VectorData`.
* **Interpretation of inputs / generic fields:** Results contain `name`, `type`, and opaque `spec`. `(name, type)` MUST be unique per artifact; producer documents each type's payload and intended consumers. KLO's `http-k8s-service` is an example, not a universal type.
* **Processing and guarantees, failure/retry handling:** Define when results are available, whether they may change after readiness, and how duplicate or malformed results are reported. Distinguish results consumed by applications from an inventory of deployed runtime instances.
* **Status reporting via conditions:** Section 3 defines `ArtifactDeployed` to include publication of required deployment results, including artifact types that require no results. Decide which generic deployment identity and health information can be reported for UI/API display; there is no portable instance inventory field today.
* **Resource deletion and undeployment:** Define when results become invalid or are removed after undeployment.
* **Optional behavior:** Section 3 requires each artifact type's packaging contract to state whether results are required and defines how an artifact type requiring no results reports deployment status.

## 5. Vector association and artifact reuse

* **Ownership and resource selection:** Core creates a `VectorAssignment` for each vector/artifact pair and uses `ArtifactManifest.allowReuse` when creating the `ArtifactDeployment`. One artifact may be referenced by several vectors; the deployer for the artifact type handles vector-specific work.
* **Interpretation of inputs / generic fields:** `VectorAssignment` references the vector and artifact. When reuse is enabled, a deployer MUST NOT infer a single vector from the artifact alone.
* **Processing and guarantees, failure/retry handling:** Vector-specific data MUST remain isolated across vectors sharing an artifact. Define what happens if an artifact requests reuse but the deployer cannot support it, and how missing references or failed assignment work are handled.
* **Status reporting via conditions:** Define what `VectorAssignment.Ready` proves, including whether a no-op assignment can be ready and what happens when the referenced artifact becomes unavailable.
* **Resource deletion and undeployment:** Removing one vector MUST NOT undeploy an artifact still in use by another; define cleanup of its vector-specific state.
* **Optional behavior:** Decide whether reuse is an artifact-type capability and whether the `VectorAssignment` CRD remains the representation of association (`konfidence` issue #83). Preserve the semantic contract if the CRD changes.

## 6. VectorData: vector-scoped data

* VectorData MUST NOT change for given vector. Deleting/Re-creating ArtifactDeployments may lead to different deploymentresults. Core must detect on that and report VectorDeployment as broken.
* Input validation for vector data: validate across multiple artifacts of a vector? e.g. colliding paths of CDN artifacts

* **Ownership and resource selection:** Core creates immutable `VectorData` per `VectorDeployment` with authored data, features, and aggregated results. Decide which implementor owns its materialization and status when several deployers/classes serve one landscape.
* **Interpretation of inputs / generic fields:** Define the mapping from vector identity to `VectorData` and how consumers select the right vector, including when an artifact is reused. Specify behavior for missing or unknown identities; storage and distribution are runtime-specific.
* **Processing and guarantees, failure/retry handling:** Specify when vector data must be available before core proceeds and how invalid data, collisions, or loss of access are retried or rejected. KLO's immutable ConfigMap and service are examples, not requirements.
* **Status reporting via conditions:** Define what `VectorData.Ready` proves, how freshness and identity are checked, and when readiness must be withdrawn. The presence of an unrelated object with the expected name is insufficient.
* **Resource deletion and undeployment:** Define cleanup of materialized data, including remote runtimes where Kubernetes owner references cannot cross API servers.
* **Optional behavior:** Decide whether materialization belongs to every class, one landscape-level implementor, or a separate capability; define how an unsupported capability affects vector readiness.

## 7. Cross-cutting status, reconciliation, and safety

For every condition, specify its owner, meaning of `True`/`False`/`Unknown`, reason codes, `observedGeneration`, and transitions after dependencies recover or disappear. Core and deployer MUST NOT leave a stale `Ready=True` that implies a known-unavailable deployment is healthy. Distinguish retryable failures from failures requiring user action; reconcile when mutable referenced resources change, even if the owning resource's spec is immutable. Define idempotency, event/watch versus periodic resync guarantees, status-update conflicts, permissions, credential handling, and whether sensitive data may appear in status or deployment results. These rules need a condition/ownership table before this draft becomes normative.

## 8. Future extension: tasks and activation

`TaskExecution`, `ActivationTaskExecution`, `VectorMigration`, and `VectorActivation` exist today; KLO implements Kubernetes Jobs and Gateway API HTTPRoutes for some executions. This draft does **not** require every deployer to perform migrations, activation, or routing. A later specification should define registration/selection of task and activation executors—possibly using a resource analogous to `DeploymentClass`—with their own status and capability contracts. The base deployment contract MUST NOT assume those operations are built into the same controller.

## 9. Versioning and conformance

Before declaring a specification version, define how it relates to CRD versions and any published Go bindings, how compatible changes are introduced, and how optional capabilities are declared. Each normative clause should receive an identifier for conformance checks. Validate the mandatory subset against KLO and a substantially different target platform; track KLO deviations as implementation gaps rather than weakening requirements to match existing behavior.

Initial decision list: target multiplicity and mutation; condition ownership/fallback; meaning of artifact and vector readiness; reuse capability; fate of `VectorAssignment`; VectorData implementor selection; result versus inventory schemas; cleanup guarantees; capability discovery and compatibility policy.
