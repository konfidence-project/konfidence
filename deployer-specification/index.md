# Deployer specification

## 1. Scope and language

* Interface consists of resources in `konfidence.cloud/v1alpha1` and their status. Kubernetes is the control-plane API, even when the deployment destination is not Kubernetes.
* Go library as convenience framework, taking care of boilerplate that will be similar for most deployers. Use of Go library is not mandatory, deployers can work without it.  
* Spec assumes reader is familiar with Konfidence concepts
* Use RFC 2119 wording to ensure requirement levels are correctly understood

## 2. Shared lifecycle and ownership

* Konfidence core create vector; deployer realizes its artifacts and related data on a target runtime
* deployer installation + configuration: `DeploymentClass` -> `DeploymentTarget`
* core deployment lifecycle: `VectorDeployment` -> `ArtifactDeployment` ->  deployment results -> `VectorData` 
* core migration lifecycle: `VectorMigration` -> `TaskExecution` 
* core activation lifecycle: `ActivationTaskRegistration` -> `ActivationTaskExecution`

Resource-specific sections use this template: 
* ownership and resource selection 
* interpretation of inputs / generic fields
* processing and guarantees, failure/retry handling
* Status reporting via conditions 
* resource deletion and undeployment
* Optional behavior 


## 3. DeploymentClass: registration and dispatch

What is a deployment class?
- maps what controller should be responsible for reconciling which ArtifactDeployments
- ArtifactDeployment specifies deployment class in `ArtifactDeployment.spec.manifest.type` (rename to ArtifactDeployment.spec.manifest.deploymentClassName`?)
- DeploymentTarget provides landscape specific configuration for a deployment class specified in `DeploymentTarget.spec.deploymentClassName`

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentClass
metadata:
  name: <deployment-class-name>
spec:
  controller: <controller-name>
```

* **Ownership and resource selection:** Deployer installs one or more cluster-scoped `DeploymentClass` resources. `spec.controller` identifies the responsible controller; `ArtifactDeployment.spec.manifest.type` and `DeploymentTarget.spec.deploymentClassName` select the class. Deployer MUST only reconcile resources for its classes.
* **Interpretation of inputs / generic fields:** 
  - Class name identifies the deployment type (make explicit by renaming `type` to `deploymentClassName` in ArtifactDeployment?)
  - Class `spec.controller` is immutable
* **Processing and guarantees, failure/retry handling:** Define what happens when a class or controller is missing, including resources created before the deployer is installed. Installation order should not require recreating those resources.
  - ArtifactDeployments created before their referenced DeploymentClass is available will stay in state "DeploymentClass not found" until the DeploymentClass is created
  - Once an ArtifactDeployment has found its referenced DeploymentClass, the "DeploymentClass not found" error cannot occur, as deletion of the DeploymentClass is prevented (see "Resource deletion and undeployment")
  - A missing controller cannot be detected. If the referenced controller in the DeploymentClass is not installed or crashes, this will not be reflected on any status
* **Status reporting via conditions:** No `DeploymentClass` status exists today. Decide how core and users detect availability; the conditions on dependent resources need clear ownership and fallback behavior.
* **Resource deletion and undeployment:** Konfidence ensures a DeploymentClass can only be deleted if there are no ArtifactDeployments referencing it (using a finalizer). Therefore, no handling of DeploymentClass deletion is required in the Deployer.
* **Optional behavior:** Decide whether capabilities are advertised per class and how unsupported capabilities are represented.

## 4. DeploymentTarget: destination and connection

* **Ownership and resource selection:** Namespaced `DeploymentTarget` references a `DeploymentClass`; the class's deployer handles the target. Define how an artifact selects a target within its landscape. KLO currently expects exactly one ready target per class and namespace; decide whether this is a general rule.
* **Interpretation of inputs / generic fields:** `spec.connection.type` and optional `ref` are generic. Deployer defines supported connection types, validates referenced resources, and interprets their contents; the destination need not be Kubernetes.
* **Processing and guarantees, failure/retry handling:** Define when a target is usable and how changes to connection settings, credentials, URLs, or referenced objects affect existing deployments. Distinguish invalid configuration from transient loss of connectivity.
* **Status reporting via conditions:** Define what `Ready` means (accepted configuration or verified reachability), who writes it, and how loss of readiness affects dependent artifacts. Core currently reports a missing class on the target's `Ready` condition; resolve ownership of that condition.
* **Resource deletion and undeployment:** Define the effect of removing a target while artifacts still use it, including cleanup on a remote runtime.
* **Optional behavior:** Class-specific connection configuration can be supported without adding runtime-specific fields to the generic contract; specify how it is declared and validated.

## 5. ArtifactDeployment: deploying an artifact

* **Ownership and resource selection:** Core creates or reuses `ArtifactDeployment`; `spec.manifest.type` selects the class responsible for deploying it. Other deployers MUST NOT write its status or manage its runtime instance.
* **Interpretation of inputs / generic fields:** Immutable spec contains the artifact manifest and OCM component/resources. Each class documents which resource types and payloads it accepts. `taskManifests` is outside this chapter's deployment contract.
* **Processing and guarantees, failure/retry handling:** Define fetching, deployment, and health independently of Flux or Kubernetes workloads. Reconciliation MUST be idempotent; define responses to invalid artifacts, missing targets, unsupported types, and transient failures.
* **Status reporting via conditions:** Define the meanings and required transitions for `ArtifactFetched`, `ArtifactDeployed`, `AppHealthy`, `DeploymentResultCreated`, and `Ready`, including whether conditions are required when no health check or results exist. A failed prerequisite must not leave a misleading `Ready=True`.
* **Resource deletion and undeployment:** Define cleanup responsibility, ordering, and completion when runtime resources are on another cluster or platform.
* **Optional behavior:** Decide whether health checks, results, and reuse can be omitted by a class, and what core then needs to consider an artifact ready.

## 6. Deployment results and deployed-instance visibility

* **Ownership and resource selection:** Deployer publishes results in `ArtifactDeployment.status.deploymentResult`; core aggregates them by component in `VectorDeployment` and `VectorData`.
* **Interpretation of inputs / generic fields:** Results contain `name`, `type`, and opaque `spec`. `(name, type)` MUST be unique per artifact; producer documents each type's payload and intended consumers. KLO's `http-k8s-service` is an example, not a universal type.
* **Processing and guarantees, failure/retry handling:** Define when results are available, whether they may change after readiness, and how duplicate or malformed results are reported. Distinguish results consumed by applications from an inventory of deployed runtime instances.
* **Status reporting via conditions:** Define when `DeploymentResultCreated` is true, including zero-result artifacts. Decide which generic deployment identity and health information can be reported for UI/API display; there is no portable instance inventory field today.
* **Resource deletion and undeployment:** Define when results become invalid or are removed after undeployment.
* **Optional behavior:** Decide whether publishing results is optional for a class and how an artifact with no results advances to readiness.

## 7. Vector association and artifact reuse

* **Ownership and resource selection:** Core currently creates a `VectorAssignment` for each vector/artifact pair and uses `ArtifactManifest.allowReuse` when creating the `ArtifactDeployment`. One artifact may be referenced by several vectors; the class's deployer handles vector-specific work.
* **Interpretation of inputs / generic fields:** `VectorAssignment` references the vector and artifact. When reuse is enabled, a deployer MUST NOT infer a single vector from the artifact alone.
* **Processing and guarantees, failure/retry handling:** Vector-specific data MUST remain isolated across vectors sharing an artifact. Define what happens if an artifact requests reuse but the deployer cannot support it, and how missing references or failed assignment work are handled.
* **Status reporting via conditions:** Define what `VectorAssignment.Ready` proves, including whether a no-op assignment can be ready and what happens when the referenced artifact becomes unavailable.
* **Resource deletion and undeployment:** Removing one vector MUST NOT undeploy an artifact still in use by another; define cleanup of its vector-specific state.
* **Optional behavior:** Decide whether reuse is a class capability and whether the `VectorAssignment` CRD remains the representation of association (`konfidence` issue #83). Preserve the semantic contract if the CRD changes.

## 8. VectorData: vector-scoped data

* **Ownership and resource selection:** Core creates immutable `VectorData` per `VectorDeployment` with authored data, features, and aggregated results. Decide which implementor owns its materialization and status when several deployers/classes serve one landscape.
* **Interpretation of inputs / generic fields:** Define the mapping from vector identity to `VectorData` and how consumers select the right vector, including when an artifact is reused. Specify behavior for missing or unknown identities; storage and distribution are runtime-specific.
* **Processing and guarantees, failure/retry handling:** Specify when vector data must be available before core proceeds and how invalid data, collisions, or loss of access are retried or rejected. KLO's immutable ConfigMap and service are examples, not requirements.
* **Status reporting via conditions:** Define what `VectorData.Ready` proves, how freshness and identity are checked, and when readiness must be withdrawn. The presence of an unrelated object with the expected name is insufficient.
* **Resource deletion and undeployment:** Define cleanup of materialized data, including remote runtimes where Kubernetes owner references cannot cross API servers.
* **Optional behavior:** Decide whether materialization belongs to every class, one landscape-level implementor, or a separate capability; define how an unsupported capability affects vector readiness.

## 9. Cross-cutting status, reconciliation, and safety

For every condition, specify its owner, meaning of `True`/`False`/`Unknown`, reason codes, `observedGeneration`, and transitions after dependencies recover or disappear. Core and deployer MUST NOT leave a stale `Ready=True` that implies a known-unavailable deployment is healthy. Distinguish retryable failures from failures requiring user action; reconcile when mutable referenced resources change, even if the owning resource's spec is immutable. Define idempotency, event/watch versus periodic resync guarantees, status-update conflicts, permissions, credential handling, and whether sensitive data may appear in status or deployment results. These rules need a condition/ownership table before this draft becomes normative.

## 10. Future extension: tasks and activation

`TaskExecution`, `ActivationTaskExecution`, `VectorMigration`, and `VectorActivation` exist today; KLO implements Kubernetes Jobs and Gateway API HTTPRoutes for some executions. This draft does **not** require every deployer to perform migrations, activation, or routing. A later specification should define registration/selection of task and activation executors—possibly using a resource analogous to `DeploymentClass`—with their own status and capability contracts. The base deployment contract MUST NOT assume those operations are built into the same controller.

## 11. Versioning and conformance

Before declaring a specification version, define how it relates to CRD versions and any published Go bindings, how compatible changes are introduced, and how optional capabilities are declared. Each normative clause should receive an identifier for conformance checks. Validate the mandatory subset against KLO and a substantially different target platform; track KLO deviations as implementation gaps rather than weakening requirements to match existing behavior.

Initial decision list: target multiplicity and mutation; condition ownership/fallback; meaning of artifact and vector readiness; reuse capability; fate of `VectorAssignment`; VectorData implementor selection; result versus inventory schemas; cleanup guarantees; capability discovery and compatibility policy.
