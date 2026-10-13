# Deployer specification

## Preamble

Status: **Draft**.

The key words **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** in this document are to be interpreted as described in BCP 14 ([RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and [RFC 8174](https://www.rfc-editor.org/rfc/rfc8174)) when, and only when, they appear in all capitals.

Open decisions and potential future changes are outlined where possible and are not conformance requirements.
This specification defines the behavior expected of a deployer for the resources created by Konfidence core.
The communication between Konfidence core and a deployer happens solely via the resources in the `konfidence.cloud/v1alpha1` control-plane API and their status.

## Introduction

An `ArtifactDeployment` is the immutable, desired-state record of a single artifact that Konfidence wants deployed. Konfidence core creates it, stamps it with an artifact type, and carries the artifact's payload — an [Open Component Model (OCM)](https://ocm.software) component version — inside it, but core never acts on that payload. OCM gives Konfidence a technology-agnostic, uniquely identified package for delivery artifacts while leaving how they are deployed to tools built on top of it. Reconciling an `ArtifactDeployment` into a running deployment is delegated to a *deployer*.

A deployer is a controller, or a set of controllers, that reconciles `ArtifactDeployment` objects into concrete deployments on a destination and reports progress and health back through their status.
A deployer owns an artifact type: it defines the type's packaging and payload contract, including its supported versions (section 1). Each `ArtifactDeployment` declares its artifact type, which routes the object to the deployer that owns that type.
Konfidence core manages the delivery process — it assembles and validates vectors, resolves each stage to a desired state, and aggregates per-artifact results back into vector-level resources — but it never deploys workloads itself, and it treats an artifact's payload as opaque.

A destination can be a Kubernetes cluster, but it need not be: a deployer might publish static content to an object store or CDN, apply configuration to a managed cloud service, or drive any other destination that can host an artifact.
A deployer is a pluggable component: core and deployer communicate solely through the `konfidence.cloud/v1alpha1` control-plane API and its status.
This boundary is what lets a single Konfidence installation roll out complex vectors whose artifacts span many domain-specific types, each handled by a purpose-built deployer, without core knowing how any individual deployment is performed.

This document specifies the behavior a deployer MUST, SHOULD, and MAY implement to be a conformant participant in that contract.

## 1. Artifact types and packaging contracts

### 1.1 Identifier

An artifact type consists of a vendor-owned domain, a descriptive type, and a contract version:

```bnf
<artifact-type>  ::= <type-name> "/" <version>
<type-name>      ::= <type> "." <domain>
<domain>         ::= <label> "." <label> | <label> "." <domain>
<type>           ::= <label>
<label>          ::= <alphanumeric> | <alphanumeric> <label-middle> <alphanumeric>
<label-middle>   ::= "" | <label-character> <label-middle>
<label-character> ::= <alphanumeric> | "-"
<alphanumeric>   ::= <lowercase-letter> | <digit>
<lowercase-letter> ::= "a" | "b" | "c" | "d" | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n" | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x" | "y" | "z"
<digit>          ::= "0" | <nonzero-digit>
<version>        ::= "v" <positive-integer> | "v" <positive-integer> "alpha" <positive-integer> | "v" <positive-integer> "beta" <positive-integer>
<positive-integer> ::= <nonzero-digit> | <nonzero-digit> <digits>
<digits>         ::= <digit> | <digit> <digits>
<nonzero-digit>  ::= "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
```

For example, `helm.konfidence.cloud/v1alpha1` names the first alpha version of the Helm artifact contract maintained by `konfidence.cloud`.

The `<domain>` MUST be a lowercase DNS name controlled by the deployer provider.
Each domain label MUST satisfy the DNS length limit of 63 octets.

The `<type>` is a separate identifier and MUST NOT exceed 63 characters under this grammar.
The `<type>` SHOULD describe the artifact's packaging format or deployment mechanism, such as `helm` or `kustomize`.

A provider MUST use a distinct version identifier whenever the payload contract changes in a way that a deployer supporting the previous identifier cannot safely interpret.
This also includes changes where the deployer is generally able to read the payload, but may not be able to correctly interpret the described deployment (e.g. when adding a new field).

### 1.2 Artifact packaging

An artifact is an OCM component version following the [OCM spec](https://github.com/open-component-model/ocm-spec).
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
  "type": "kustomize.konfidence.cloud/v1alpha1"
}
```

The component version MAY contain additional resources that a deployer uses to understand what needs to be deployed.

The provider of an artifact type SHOULD document the packaging and validation requirements for each supported contract version.
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
Its `spec.artifactType` contains the unversioned `<type-name>`, so one target can serve multiple versions of the same artifact contract.

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
  supportedArtifactTypeVersions:
    - v1
  conditions:
    - type: Ready
      status: "True"
      reason: Accepted
```

### 2.1 Ownership and selection

A deployer MUST reconcile only targets whose `spec.artifactType` matches a `<type-name>` it supports.
It MUST NOT update the status of targets belonging to other providers or types.
Within a landscape namespace, at most one target for a given `<type-name>` is allowed.
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

### 2.3 Status and supported versions

The deployer MUST set the target's `Ready` condition after validating the parameters it needs to accept the target.
`Ready=True` means the deployer accepts the target for deployment.
This does not guarantee that a future artifact deployment will succeed.

The deployer MAY perform additional connectivity or credential checks and report their results in separate deployer-owned conditions.
It MUST set `Ready=False` with a descriptive reason and message when required parameters are invalid.

The deployer MUST report the versions it can handle for the `DeploymentTarget.spec.artifactType` in `status.supportedArtifactTypeVersions`.
Every reported version MUST correspond to a supported handler for that type and the target's configuration.
It MUST remove a version from that list if it ceases to support it.

Core MUST check that the target is ready and that the artifact's version appears in this list before treating the target as suitable for deployment.
The deployer SHOULD publish its name and release version in `status.controller`.

The exact shape of `status.capabilities` remains open for now.

### 2.4 Deletion

Deleting a `DeploymentTarget` MUST NOT by itself undeploy artifacts or remove resources at its destination.
Runtime resources remain deployed, but the deployer may lose access to them and can no longer guarantee their continued reconciliation.
If another target for the same artifact type is created, the deployer MUST NOT assume that it refers to the same destination or silently adopt the existing runtime resources.
How core reports the state of affected `ArtifactDeployment` resources after target deletion remains to be specified; their previous readiness MUST NOT be taken as evidence that they are still being reconciled.

## 3. ArtifactDeployment

Core creates an immutable `ArtifactDeployment` specification containing an artifact manifest and OCM component resources.

### 3.1 Ownership and selection

The type in `spec.manifest.type` selects the responsible deployer and the corresponding target in the deployment's landscape namespace.
A deployer acts platform-wide and declares the artifact types it supports in its own configuration.
It is the cluster administrator's responsibility to ensure that at most one deployer reconciles a given artifact type in a control plane.
A vendor-owned identifier prevents accidental naming conflicts, but it cannot prevent two controllers from being configured for the same type.

The deployer MUST reconcile an `ArtifactDeployment` only when its artifact type matches a supported identifier exactly.
It MUST leave resources belonging to other providers or types to their responsible deployers.
An unsupported version MUST NOT be interpreted as another version of the same type.

### 3.2 Input validation

The deployer MUST validate the `ArtifactDeployment` specification against the packaging contract for its exact artifact type and version.
This includes the required OCM resources and the payloads it consumes.
It MUST NOT rely on validation performed by an artifact authoring tool or by core.
Task manifests attached to an artifact are outside the base artifact-deployment contract.

### 3.3 Reconciliation and failure handling

The deployer MUST reconcile supported artifacts idempotently, including those created before it was installed or while it was stopped.
It MUST complete input validation before creating runtime resources.
An invalid payload MUST be reported through `ArtifactDeployment.status` rather than deployed.
If the target is absent, unready, or does not report support for the artifact's version, the artifact MUST NOT be deployed through that target.

### 3.4 Status

The deployer reports artifact fetching, deployment, application health, deployment results, and overall readiness through `ArtifactDeployment.status`.
The meanings and transition rules for `ArtifactFetched`, `ArtifactDeployed`, `AppHealthy`, `DeploymentResultCreated`, and `Ready` require further definition.
A deployer MUST NOT leave `Ready=True` when it knows that a prerequisite for readiness has failed.

### 3.5 Deletion and undeployment

The deployer MUST add a finalizer to an `ArtifactDeployment` before creating runtime resources for it.
On deletion, it MUST remove the runtime resources it manages, including resources on a remote runtime, before removing its finalizer.
It MUST NOT remove the finalizer while any of those resources are known to remain deployed.
If cleanup cannot complete, including because the target connection is unavailable, the deployer MUST retain the finalizer and retry cleanup when access is restored.
Consequently, deleting the target before deleting its artifact deployments can leave those deployments pending deletion until access to the original destination is restored.

### 3.6 Optional behavior

Whether a health check or deployment result is required for every artifact type, and how an artifact without either advances to readiness, remain open decisions.

## 4. Deployment results and deployed-instance visibility

* **Ownership and resource selection:** Deployer publishes results in `ArtifactDeployment.status.deploymentResult`; core aggregates them by component in `VectorDeployment` and `VectorData`.
* **Interpretation of inputs / generic fields:** Results contain `name`, `type`, and opaque `spec`. `(name, type)` MUST be unique per artifact; producer documents each type's payload and intended consumers. KLO's `http-k8s-service` is an example, not a universal type.
* **Processing and guarantees, failure/retry handling:** Define when results are available, whether they may change after readiness, and how duplicate or malformed results are reported. Distinguish results consumed by applications from an inventory of deployed runtime instances.
* **Status reporting via conditions:** Define when `DeploymentResultCreated` is true, including zero-result artifacts. Decide which generic deployment identity and health information can be reported for UI/API display; there is no portable instance inventory field today.
* **Resource deletion and undeployment:** Define when results become invalid or are removed after undeployment.
* **Optional behavior:** Decide whether publishing results is optional for an artifact type and how an artifact with no results advances to readiness.

## 5. Vector association and artifact reuse

* **Ownership and resource selection:** Core creates a `VectorAssignment` for each vector/artifact pair and uses `ArtifactManifest.allowReuse` when creating the `ArtifactDeployment`. One artifact may be referenced by several vectors; the deployer for the artifact type handles vector-specific work.
* **Interpretation of inputs / generic fields:** `VectorAssignment` references the vector and artifact. When reuse is enabled, a deployer MUST NOT infer a single vector from the artifact alone.
* **Processing and guarantees, failure/retry handling:** Vector-specific data MUST remain isolated across vectors sharing an artifact. Define what happens if an artifact requests reuse but the deployer cannot support it, and how missing references or failed assignment work are handled.
* **Status reporting via conditions:** Define what `VectorAssignment.Ready` proves, including whether a no-op assignment can be ready and what happens when the referenced artifact becomes unavailable.
* **Resource deletion and undeployment:** Removing one vector MUST NOT undeploy an artifact still in use by another; define cleanup of its vector-specific state.
* **Optional behavior:** Decide whether reuse is an artifact-type capability and whether the `VectorAssignment` CRD remains the representation of association (`konfidence` issue #83). Preserve the semantic contract if the CRD changes.

## 6. VectorData: vector-scoped data

* VectorData MUST NOT change for given vector. Deleting/Re-creating ArtifactDeployments may lead to different deploymentresults. Core must detect on that and report VectorDeployment as broken. 

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
