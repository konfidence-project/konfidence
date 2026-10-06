> The key words “MUST”, “MUST NOT”, “REQUIRED”, “SHALL”, “SHALL NOT”, “SHOULD”, “SHOULD NOT”, “RECOMMENDED”, “NOT RECOMMENDED”, “MAY”, and “OPTIONAL” in this document are to be interpreted as described in BCP 14 [RFC2119] [RFC8174] when, and only when, they appear in all capitals.

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

## 3. DeploymentClass


A `DeploymentClass` declares an artifact deployment capability and identifies the controller responsible for it.
It is cluster-scoped.

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentClass
metadata:
  name: myclass.example.com
spec:
  controller: example.com/my-deployer
```

### 3.1 Ownership and resource selection

A deployer MUST provide exactly one `DeploymentClass` for each artifact deployment class it implements.
A deployer SHOULD come with at least one `DeploymentClass`.

[//]: # (TODO should we include following sentence or not?)
Installation of the `DeploymentClass` SHOULD be performed by packaging rather than by a controller at runtime.
Before reconciling any resource, the deployer MUST verify that the referenced `DeploymentClass` exists and that its `spec.controller` equals the identifier of the reconciling controller.
It MUST NOT reconcile a resource whose class names another controller, even if the class name is otherwise recognized by the deployer.
Other resources referencing the `DeploymentClass` may already exist when the class is installed.

The deployer MUST use the `DeploymentClass.metadata.name` as the class identifier.
Other resources select it through a `spec.deploymentClassName` field.
These references MUST be matched exactly to the class name.

[//]: # (TODO rename ArtifactDeployment.spec.manifest.type to ArtifactDeployment.spec.deploymentClassName)



### 3.2 Interpretation of inputs / generic fields

[//]: # (TODO does it make sense to keep the spec.controller field? className is already vendor-specific, so class-to-controller mapping is useless)
The deployer MUST set `DeploymentClass.spec.controller` to the stable identifier used by its own reconcilers.
It MUST NOT use a different identifier for resources handled by the same registered controller.
It is immutable and MUST NOT change once the `DeploymentClass` was initially applied to the Konfidence control plane.

[//]: # (TODO DeploymentClass.spec should be mutable, spec.controller should be immutable)

`DeploymentClass.spec.controller` SHOULD be prefixed by a vendor-owned domain (e.g. `example.com/my-deployer`).
This is to avoid collisions of deployers tackling similar deployment artifacts.

Class providers SHOULD use deployment class names that distinguish their deployment types and vendor, for example `myclass.example.com`.
In this case `myclass` should describe the type of artifact being deployed, and `example.com` is a vendor-owned domain.

### 3.3 Processing and guarantees, failure/retry handling

When a `DeploymentClass` matching a previously created resource becomes available, a running deployer MUST reconcile those resources without requiring it to be deleted and recreated.
A newly installed deployer MUST also process existing resources of its classes after startup.

If a `DeploymentClass` is absent or belongs to another controller, the deployer MUST NOT reconcile the referenced resource or write its status.
Konfidence's core controllers contain handling of orphaned resources if a `DeploymentClass` becomes unavailable while still referenced by other resources.

### 3.4 Status reporting via conditions

`DeploymentClass` has no status subresource. Its existence and `spec.controller` express registration without the need to report controller status.

### 3.5 Resource deletion and undeployment

A deployer MUST NOT infer that existing resources have been deleted merely because their `DeploymentClass` is absent.
If the class is recreated with the same name and controller identifier, the deployer MUST reconcile matching resources again.

`DeploymentClass` deletion is NOT prohibited while references to it exist.
Deployers MAY use finalizers to clean up deployment state and update status conditions on related resources if an owned `DeploymentClass` is deleted.
They MUST NOT delete the related resources referencing the deleted `DeploymentClass`.

[//]: # (TODO: define proper cleanup behavior)

### 3.6 Optional behavior

A deployer MAY provide more than one `DeploymentClass` that share the same controller identifier in `DeploymentClass.spec.controller`.
Each class MUST independently satisfy the requirements described earlier.

### Open questions
* how to advertise capabilities (e.g. S3 deployer can handle ArtifactDeployments, but not TaskExecutions)
* consider controller health? should core check that and apply fallback behavior if unhealthy?
* how to handle edge case of ownership handover? should we give any guarantees for that process?

---

## 3A. Alternative proposal: remove DeploymentClass resource, deployers reconcile artifactTypes directly 

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentTarget
metadata:
  name: helm-production
  namespace: kden-l-prod-fz8g86s6
spec:
  artifactType: konfidence.cloud/helm:v1 # replaces for deploymentClassName
  connection:
    type: kubeconfig
    ref:
      kind: Secret
      name: production-kubeconfig
  configuration: {} # already proposed by https://github.com/konfidence-project/konfidence/issues/217
status:
  controller: konfidence.cloud/kubernetes-landscape-orchestrator:v1.2.3 # only informational
  capabilities: {} # to be defined 
  conditions: # proposed status, define which conditions we really need (Ready, Healthy, ConfigValid, Error) 
  - type: Ready   
    status: "True"
    reason: Accepted
```

### 3A.1 Ownership and resource selection

An artifact type MUST follow this Backus–Naur-form grammar:

```bnf
<artifact-type> ::= <domain> "/" <type> ":" <version>
<domain>        ::= <label> "." <label> | <label> "." <domain>
<type>          ::= <label>
<label>         ::= <alphanumeric> | <alphanumeric> <label-middle> <alphanumeric>
<label-middle>  ::= "" | <label-character> <label-middle>
<label-character> ::= <alphanumeric> | "-"
<alphanumeric>  ::= <lowercase-letter> | <digit>
<lowercase-letter> ::= "a" | ... | "z"
<digit>         ::= "0" | ... | "9"
<version>       ::= "v" <positive-integer>
<positive-integer> ::= <nonzero-digit> | <nonzero-digit> <digits>
<digits>        ::= <digit> | <digit> <digits>
<nonzero-digit> ::= "1" | ... | "9"
```

For example, `example.com/mytype:v1` identifies contract version `v1` of the artifact type `mytype` under `example.com`.

The `<domain>` MUST be controlled by the maintainer of the deployer and it MUST be a valid lowercase DNS name according to [RFC 1035](https://www.rfc-editor.org/info/rfc1035/).
Its purpose is to namespace artifact types so that independent deployer providers can use the same descriptive type without claiming the same identifier.

The `<type>` SHOULD describe the artifact format or deployment mechanism in one concise label, such as `helm` or `kustomize`.
It identifies what the deployer is expected to find inside the artifact or how the content in the artifact is packaged. 

The `<version>` identifies the contract for interpreting that artifact type including its expected payload.
Versions MUST start at `v1` and increase for each incompatible contract change.
Changes to the artifact's schema, that an older deployer cannot safely interpret, require incrementing the version.

A deployer MUST match the complete artifact type, including its version, when deciding whether it understands an artifact.
It MUST reconcile only resources whose `spec.artifactType` matches one of those artifact types exactly.
Deployers MUST reject resources whose `spec.artifactType` matches domain and type, but not version.
Reject means write a status condition `Ready=False` with reason `Unsupported`. (or a status condition `DeployerVersionMismatch=True`?)

One deployer MAY explicitly support several versions simultaneously so existing immutable artifacts can continue to deploy while new artifacts adopt a newer contract.

Deployers MUST ignore resources whose `spec.artifactType` contain domain or type unknown to the deployer. 

A deployer SHOULD document the exact type strings it supports, the artifact payloads it accepts, and how to configure a target for each type.

### 3A.2 Target configuration

[//]: # (TODO replace `DeploymentTarget.spec.deploymentClassName` with `spec.artifactType` )
[//]: # (TODO how to deal with versions in deploymenttargets? just ignore for now?)
`DeploymentTarget.spec.artifactType` identifies the artifact type supported by that target in its landscape.
The deployer MUST select targets by `domain/type` string, ignoring the version part of the identifier. 

Deployers can expect that there is either zero or one `DeploymentTarget` resource in a single landscape.
Support for multiple `DeploymentTargets` in a single landscape is still undecided and may come with a later version of this specification. 

The deployer MUST validate and interpret `spec.connection`. 
`spec.connection.type` contains a type hint that tells the deployer what to expect and how to interpret the contents of the resource referenced in `spec.connection.ref`.
A deployer MUST support at least one type of connection, and it SHOULD document which types it supports and how to configure them. 
If the connection type or referenced resource is not valid or not known to the deployer, it MUST write a status condition `InvalidConnection=True` and `Ready=False`. 

[//]: # (TODO: how to handle changes in connection? rotating credentials is fine, changing the target cluster is disaster. We cannot generically distinguish those cases. )

`DeploymentTarget.spec.configuration` contains schemaless configuration to be interpreted by the deployer. 
It may contain configuration values to consider for the deployment itself, such as timeouts, retries, drift detection, rollback and undeployment behavior.  
It MUST validate the configuration with its own schema and set a condition `InvalidConfiguration=True` and `Ready=False` if the validation fails.

Global defaults for configuration MAY be supplied by the deployer's installation configuration.
Those default will be applied to all landscapes in all projects of a Konfidence installation.
The deployer SHOULD document the configuration schema, defaults, and handling of mutable settings such as timeouts and credentials.

### 3A.3 Deployment and version upgrades

A newly installed or restarted deployer MUST discover and reconcile existing resources of its supported artifact types.
If a deployer's supported type set changes, it MUST reconsider affected existing resources.
One deployer MAY support several versions of the same artifact type at once.

[//]: # (TODO discuss this idea)
A deployer MUST preserve the meaning of already-created immutable `ArtifactDeployment` by recording the used deployer version into the status of the resource.
An upgrade MUST NOT silently reinterpret an existing artifact as a newer type or move it to another target.

### 3A.4 Status and deployer availability

[//]: # (TODO can we require ping / credentials check for Ready=True condition? some deployers may not be able to perform that check explicitly; maybe with dummy deployment?)
For a supported `DeploymentTarget.spec.artifactType`, the deployer MUST report whether it accepts the target through its `Ready` condition.
The `Ready=True` condition is only set when the deployer fully validated the `DeploymentTarget` resource and it expects deployments to this target to succeed. 
For the same domain and type with an unsupported version, the rejection conditions proposed in section 3A.1 apply.
For an unknown domain or type, the deployer ignores the resource and MUST NOT write its status.

`DeploymentTarget.status.controller` identifies the deployer that last wrote status and MAY be used for diagnosis.
The shape and meaning of `status.capabilities` are not yet defined.

---

## 4. DeploymentTarget: destination and connection

What is a deployment target?
- DeploymentTarget provides landscape specific configuration for a deployment class specified in `DeploymentTarget.spec.deploymentClassName`
- The exact configuration is dependent on the deployer (for example connection details to the target)

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentTarget
metadata:
  name: <deployment-target-name>
  namespace: <landscape-namespace>
spec:
  deploymentClassName: <deployment-class-name>
  connection:
    type: <connection-type> # defined by deployer
    ref:
      # deployer specifies which resource kinds must be referenced
      apiGroup:
      kind:
      name:
status:
  conditions:
  - type: Ready
    status: "True"
    reason: Accepted
```

* **Ownership and resource selection:** Namespaced `DeploymentTarget` references a `DeploymentClass`; the class's deployer handles the target. Define how an artifact selects a target within its landscape. KLO currently expects exactly one ready target per class and namespace; decide whether this is a general rule.
* **Interpretation of inputs / generic fields:** `spec.connection.type` and optional `ref` are generic. Deployer defines supported connection types, validates referenced resources, and interprets their contents; the destination need not be Kubernetes.
* **Processing and guarantees, failure/retry handling:** Define when a target is usable and how changes to connection settings, credentials, URLs, or referenced objects affect existing deployments. Distinguish invalid configuration from transient loss of connectivity.
  - The connection settings are mutable but the user must not change the actual target (e.g the target Kubernetes cluster or target S3 bucket). Therefore, the deployer does not have to deal with transferring already deployed artifacts to another target. The Deployer MUST handle changes to other chandes to the connection settings like updated credentials.
* **Status reporting via conditions:** Define what `Ready` means (accepted configuration or verified reachability), who writes it, and how loss of readiness affects dependent artifacts. Core currently reports a missing class on the target's `Ready` condition; resolve ownership of that condition.
  - The deployer that is selected by the referenced DeploymentClass MUST set the Ready condition with Reason `"Accpeted"` once it has accepted the resource. What "accepted" means is up to the deployer. It may include connectivity checks or simply validate the configuration.
  - If the deployer does not accept the DeploymentTarget due to errors in the connection configuration, the deployer MUST set the Ready condition to `"False"` with reason `ConnectionInvalid`
  - If the DeploymentTarget is invalid, the deployer stops performing any actions for ArtifactDeployments using that deployment class
* **Resource deletion and undeployment:** Define the effect of removing a target while artifacts still use it, including cleanup on a remote runtime.
* **Optional behavior:** Class-specific connection configuration can be supported without adding runtime-specific fields to the generic contract; specify how it is declared and validated.

## 5. ArtifactDeployment: deploying an artifact

[//]: # (TODO deployer spec must require deployers to specify how their artifacts look like; deployer is not only a k8s-controller but also some info/tooling to create artifacts)
idea: "plugin" architecture for our CLI, deployer must provide some info on what it expects for its artifacts; maybe just resource validation (e.g. helm artifact must include exactly one helm resource in its OCM component)

* **Ownership and resource selection:** Core creates or reuses `ArtifactDeployment`; `spec.manifest.type` selects the class responsible for deploying it. Other deployers MUST NOT write its status or manage its runtime instance.
* **Interpretation of inputs / generic fields:** Immutable spec contains the artifact manifest and OCM component/resources. Each class documents which resource types and payloads it accepts. `taskManifests` is outside this chapter's deployment contract.
* **Processing and guarantees, failure/retry handling:** Define fetching, deployment, and health independently of Flux or Kubernetes workloads. Reconciliation MUST be idempotent; define responses to invalid artifacts, missing targets, unsupported types, and transient failures.
  - ArtifactDeployments created before their referenced DeploymentClass is available will stay in state "DeploymentClass not found" until the DeploymentClass is created
  - Once an ArtifactDeployment has found its referenced DeploymentClass, the "DeploymentClass not found" error cannot occur, as deletion of the DeploymentClass is prevented (see "Resource deletion and undeployment")
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
