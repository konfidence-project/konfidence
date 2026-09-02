---
title: CRD
description: Custom Resource Definition specifications for Konfidence Kubernetes resources.
outline: [2, 3]
editLink: true
lastUpdated: true
---

# API Reference

## Packages
- [konfidence.cloud/v1alpha1](#konfidencecloudv1alpha1)


## konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the konfidence v1alpha1 API group.

### Resource Types
- [ActivationTaskExecution](#activationtaskexecution)
- [ActivationTaskRegistration](#activationtaskregistration)
- [ArtifactDeployment](#artifactdeployment)
- [DeploymentClass](#deploymentclass)
- [DeploymentTarget](#deploymenttarget)
- [Landscape](#landscape)
- [Project](#project)
- [Stage](#stage)
- [StageVersion](#stageversion)
- [StageVersionUsage](#stageversionusage)
- [TaskExecution](#taskexecution)
- [VectorActivation](#vectoractivation)
- [VectorAssignment](#vectorassignment)
- [VectorData](#vectordata)
- [VectorDeployment](#vectordeployment)
- [VectorMigration](#vectormigration)
- [VectorPromotion](#vectorpromotion)
- [VectorPromotionConfig](#vectorpromotionconfig)
- [VectorTemplate](#vectortemplate)



### ActivationTaskExecution



ActivationTaskExecution is the Schema for the ActivationTaskExecutions API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[ActivationTaskExecutionSpec](#activationtaskexecutionspec)_ | spec defines the desired state of ActivationTaskExecution |  | Required: \{\} <br /> |
| `status` _[ActivationTaskExecutionStatus](#activationtaskexecutionstatus)_ | status defines the observed state of ActivationTaskExecution |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ActivationTaskExecution
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: activationtaskexecution-sample
spec:
  type: k8s-job
  spec: {
    "template": "refine-me"
  }
  vectorActivation: "activation-1"
```

### ActivationTaskExecutionSpec



ActivationTaskExecutionSpec defines the desired state of ActivationTaskExecution



_Appears in:_
- [ActivationTaskExecution](#activationtaskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |
| `vectorActivation` _string_ | VectorActivation is a temporary field that contains the name of the associated vectorActivation |  |  |


### ActivationTaskExecutionStatus



ActivationTaskExecutionStatus defines the observed state of ActivationTaskExecution.



_Appears in:_
- [ActivationTaskExecution](#activationtaskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


### ActivationTaskRegistration



ActivationTaskRegistration is the Schema for the activationtaskregistrations API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskRegistration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[ActivationTaskRegistrationSpec](#activationtaskregistrationspec)_ | spec defines the desired state of ActivationTaskRegistration |  | Required: \{\} <br /> |
| `status` _[ActivationTaskRegistrationStatus](#activationtaskregistrationstatus)_ | status defines the observed state of ActivationTaskRegistration |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ActivationTaskRegistration
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: activationtaskregistration-sample
spec:
  type: custom-k8s-activation
  spec: {}
  succeeds:
    - activation-A
    - activation-B
  precedes:
    - activation-C
```

### ActivationTaskRegistrationSpec



ActivationTaskRegistrationSpec defines the desired state of ActivationTaskRegistration



_Appears in:_
- [ActivationTaskRegistration](#activationtaskregistration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | INSERT ADDITIONAL SPEC FIELDS - desired state of cluster<br />Important: Run "make" to regenerate code after modifying this file<br />The following markers will use OpenAPI v3 schema to validate the value<br />More info: https://book.kubebuilder.io/reference/markers/crd-validation.html |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |
| `succeeds` _string array_ |  |  |  |
| `precedes` _string array_ |  |  |  |


### ActivationTaskRegistrationStatus



ActivationTaskRegistrationStatus defines the observed state of ActivationTaskRegistration.



_Appears in:_
- [ActivationTaskRegistration](#activationtaskregistration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | conditions represent the current state of the ActivationTaskRegistration resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- "Available": the resource is fully functional<br />- "Progressing": the resource is being created or updated<br />- "Degraded": the resource failed to reach or maintain its desired state<br />The status of each condition is one of True, False, or Unknown. |  | Optional: \{\} <br /> |


### ArtifactDeployment



ArtifactDeployment is the Schema for the artifactdeployments API.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ArtifactDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactDeploymentSpec](#artifactdeploymentspec)_ | Spec defines the desired state of the ArtifactDeployment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[ArtifactDeploymentStatus](#artifactdeploymentstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: ArtifactDeployment
metadata:
  name: e9cea9b3764b9adda065f85acd63a42f04b69de2d42e9b7e4f9556913bb37abb
  namespace: default
spec:
  component:
    name: konfidence.cloud/podinfo-example
    resources:
      - content:
          helmChart: podinfo:6.9.1
          helmRepository: https://stefanprodan.github.io/podinfo
          type: helm
        name: sample-service-1-helm-chart
        type: helmChart
    version: 1.0.0
  manifest:
    allowReuse: true
    type: helm.konfidence.cloud
  taskManifests: [ ]
```

### ArtifactDeploymentSpec



ArtifactDeploymentSpec defines the desired state of an ArtifactDeployment. It describes the artifact to be deployed,
optional post-deployment tasks, and optional metadata derived from an OCM ComponentVersion. A deployer interprets
the specification according to the artifact type in Manifest.Type.



_Appears in:_
- [ArtifactDeployment](#artifactdeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `manifest` _[ArtifactManifest](#artifactmanifest)_ | Manifest contains information about the artifact itself and the deployer implementation responsible for handling it. |  |  |
| `taskManifests` _[TaskManifest](#taskmanifest) array_ | TaskManifests describes optional post-deployment tasks (commonly used for vector migrations such as database<br />schema updates). Tasks are executed after the artifact has been deployed and may form a dependency graph via<br />DependsOn. |  |  |
| `component` _[OCMComponent](#ocmcomponent)_ | Component contains OCM metadata associated with the artifact. This is a simplified mapping of the OCM ComponentVersion. |  |  |


### ArtifactDeploymentStatus



ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.



_Appears in:_
- [ArtifactDeployment](#artifactdeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration is the last observed generation. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions describes the state of the deployment lifecycle. The following conditions are expected:<br />  - ArtifactFetched: the artifact was successfully retrieved<br />  - ArtifactDeployed: the artifact was successfully deployed<br />  - AppHealthy: the deployer reports the workload as healthy<br />Conditions progress in a linear order:<br />ArtifactFetched -> ArtifactDeployed -> AppHealthy |  | Optional: \{\} <br /> |
| `deploymentResult` _[DeploymentResult](#deploymentresult) array_ | DeploymentResults captures structured outputs produced by the deployer during the deployment process—such as<br />computed DNS names, service endpoints, generated configuration, or other workload-specific details.<br />Results should be treated as immutable for a given generation and may be consumed by later stages of a vector<br />rollout (e.g., routing configuration).<br />Results are unique by (name, type). |  | Optional: \{\} <br /> |


### ArtifactManifest



ArtifactManifest describes the content of the artifact, thus it determines the deployer implementation responsible
for handling it.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)
- [VectorAssignmentSpec](#vectorassignmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the name of the DeploymentClass that should handle this artifact (e.g., "helm.konfidence.cloud").<br />This must match a DeploymentClass.metadata.name in the cluster.<br />Deployers implement their own interpretation of the artifact's contents. |  |  |
| `allowReuse` _boolean_ | AllowReuse indicates whether the deployed artifact instance may be shared across multiple VectorDeployments.<br />Reuse allows more efficient resource consumption but requires the artifact to be independent of vector-specific<br />runtime context. |  |  |


### Component



Component defines a component of a VectorTemplate.
A struct is used for future expansion.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |


### ComponentDeploymentResults

_Underlying type:_ _[DeploymentResult](#deploymentresult)_

ComponentDeploymentResults lists the deployment results emitted by a single component.

_Validation:_
- MaxItems: 16

_Appears in:_
- [VectorDataSpec](#vectordataspec)
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name identifies the result. |  | MaxLength: 253 <br /> |
| `type` _string_ | Type describes the structure contained in Spec. Each deployer may define multiple result types. |  | MaxLength: 63 <br /> |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Spec contains deployer-specific structured data. Its format is determined by the Type field. |  |  |


### ConnectionRef



ConnectionRef identifies a Secret or ConfigMap in the same namespace.



_Appears in:_
- [DeploymentTargetConnection](#deploymenttargetconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiGroup` _string_ | APIGroup is the group for the resource being referenced.<br />Defaults to the core API group ("") for Secret and ConfigMap.<br />For deployer-specific CRDs, set this to the appropriate API group. |  | Optional: \{\} <br /> |
| `kind` _string_ | Kind is the resource kind (e.g., "Secret", "ConfigMap", or a deployer-specific kind). |  | MaxLength: 64 <br />MinLength: 1 <br />Required: \{\} <br /> |
| `name` _string_ | Name is the name of the referenced resource. |  | MaxLength: 253 <br />MinLength: 1 <br />Required: \{\} <br /> |


### CredentialRef



CredentialRef references a Secret in the same namespace as the holding resource.



_Appears in:_
- [OCMCredentials](#ocmcredentials)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br /> |


### Credentials



Credentials holds credentials for various purposes — for example OCM
repository access and signing/verification key material.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ocm` _[OCMCredentials](#ocmcredentials)_ |  |  | Optional: \{\} <br /> |


### DeploymentClass



DeploymentClass declares a deployment capability provided by a deployer (controller). It is a cluster-scoped resource
installed by deployers to advertise their capabilities. Its immutable spec ensures that ownership of resources using
the class cannot change on the fly. Its metadata.name is the deployment class identifier referenced by
ArtifactDeployments and DeploymentTargets. The name must be unique across all DeploymentClasses in the cluster and
should follow the pattern `<class-name>.<vendor-domain>` (e.g., `helm.konfidence.cloud`).



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `DeploymentClass` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DeploymentClassSpec](#deploymentclassspec)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentClass
metadata:
  name: kustomize.konfidence.cloud
spec:
  controller: konfidence.cloud/kubernetes-landscape-orchestrator
```

### DeploymentClassSpec



DeploymentClassSpec defines the desired state of DeploymentClass. It is immutable because changing it requires
transfer of deployment ownership to a different controller. This process is currently not well-supported and
is therefor not recommended.



_Appears in:_
- [DeploymentClass](#deploymentclass)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `controller` _string_ | Controller is the name of the controller that implements this deployment class.<br />This identifies which operator/controller is responsible for reconciling<br />resources of this deployment class (e.g., "kubernetes-landscape-orchestrator"). |  | MaxLength: 253 <br />MinLength: 1 <br />Required: \{\} <br /> |


### DeploymentResult



DeploymentResult contains a single output produced by a deployer. These results are used to transport information
from the deployer to later phases of the vector lifecycle.



_Appears in:_
- [ArtifactDeploymentStatus](#artifactdeploymentstatus)
- [ComponentDeploymentResults](#componentdeploymentresults)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name identifies the result. |  | MaxLength: 253 <br /> |
| `type` _string_ | Type describes the structure contained in Spec. Each deployer may define multiple result types. |  | MaxLength: 63 <br /> |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Spec contains deployer-specific structured data. Its format is determined by the Type field. |  |  |


### DeploymentTarget



DeploymentTarget is the Schema for the deploymenttargets API. A DeploymentTarget
configures a concrete deployment destination within a landscape for a specific
deployment class. It is namespace-scoped and created in landscape namespaces.
Multiple DeploymentTargets can exist in the same landscape, but their deployment
class names must be unique within the namespace.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `DeploymentTarget` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[DeploymentTargetSpec](#deploymenttargetspec)_ |  |  |  |
| `status` _[DeploymentTargetStatus](#deploymenttargetstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: DeploymentTarget
metadata:
  name: production-kustomize
  namespace: kden-l-production-abc123
spec:
  deploymentClassName: kustomize.konfidence.cloud
  connection:
    type: kubeconfig
    ref:
      kind: Secret
      name: prod-cluster-kubeconfig
```

### DeploymentTargetConnection



DeploymentTargetConnection defines connection information for a deployment target.



_Appears in:_
- [DeploymentTargetSpec](#deploymenttargetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type is a hint for how to interpret the connection reference. It is advisory<br />and informational only. The deployer controller interprets and enforces its meaning. |  | MaxLength: 64 <br />MinLength: 1 <br />Required: \{\} <br /> |
| `ref` _[ConnectionRef](#connectionref)_ | Ref references a resource containing connection details. This can be a Secret, ConfigMap,<br />or a custom resource defined by the deployer. The deployer is responsible for interpreting<br />and validating the referenced resource. |  | Optional: \{\} <br /> |


### DeploymentTargetSpec



DeploymentTargetSpec defines the desired state of DeploymentTarget.



_Appears in:_
- [DeploymentTarget](#deploymenttarget)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `deploymentClassName` _string_ | DeploymentClassName references a DeploymentClass by its metadata.name. The referenced DeploymentClass determines<br />which controller is responsible for reconciling the DeploymentTarget resource. If there is no DeploymentClass<br />with that name, the DeploymentTarget will be marked as not ready. DeploymentClassName<br />is immutable because changing it would transfer ownership of the DeploymentTarget to a different controller.<br />There can only be one DeploymentTarget for a given deployment class in a landscape, so DeploymentClassName must be unique<br />across DeploymentTargets in the same landscape namespace. This restriction might be loosened in future releases of<br />Konfidence. |  | Required: \{\} <br /> |
| `connection` _[DeploymentTargetConnection](#deploymenttargetconnection)_ | Connection defines how to connect to this deployment target.<br />The structure and interpretation of connection details is specific to the<br />deployment class and its implementing controller. |  | Required: \{\} <br /> |


### DeploymentTargetStatus



DeploymentTargetStatus defines the observed state of DeploymentTarget.
The deployer controller responsible for this target's DeploymentClass is expected
to set the Ready condition once it has accepted the resource. What "accepted" means
is up to the deployer. It may include connectivity checks or simply validate the
configuration.



_Appears in:_
- [DeploymentTarget](#deploymenttarget)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  | Optional: \{\} <br /> |




### GlobMatch

_Underlying type:_ _string_

GlobMatch is a claim-value match pattern using glob semantics, where "*"
matches any run of characters (for example "repo:konfidence-project/*").

_Validation:_
- MaxLength: 512

_Appears in:_
- [JWKSSubject](#jwkssubject)



### JWKSSubject



JWKSSubject matches a workload token issued by a trusted OIDC provider,
narrowed to a required audience and at least one token claim.



_Appears in:_
- [Subject](#subject)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the OIDC discovery endpoint (the provider's<br />".well-known/openid-configuration" URL) used to resolve the signing keys<br />that the presented token is verified against. |  | MaxLength: 2048 <br />Pattern: `^https://.*$` <br /> |
| `audience` _string_ | Audience is the value the token's "aud" claim must carry. It is required<br />so that a token minted for a different service cannot be replayed against<br />Konfidence: the token is accepted only if it was issued for this audience. |  | MaxLength: 512 <br />MinLength: 1 <br /> |
| `claims` _object (keys:string, values:[GlobMatch](#globmatch))_ | Claims narrows the match to tokens whose claims match the given patterns.<br />It maps a claim name (for example "sub", "repository" or "ref") to a<br />glob pattern the claim value must match; all listed claims must match<br />(AND). At least one claim is required so a subject cannot inadvertently<br />match every token a provider issues. |  | MaxProperties: 32 <br />MinProperties: 1 <br /> |


### Landscape



Landscape is the Schema for the landscapes API. A Landscape owns a dedicated
namespace that serves as a deployment target for vectors. Landscapes must be
created in project namespaces. The landscape name is capped at 46 characters
so the derived namespace name stays within the 63-character Kubernetes limits.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `Landscape` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[LandscapeSpec](#landscapespec)_ |  |  |  |
| `status` _[LandscapeStatus](#landscapestatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: Landscape
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: sample-landscape
  namespace: kden-p-sample-project  # Must be created in a project namespace
spec:
  displayName: Sample Development Landscape
  # namespace: my-custom-landscape-namespace  # overrides the default "kden-l-sample-landscape-<hash>"
```

### LandscapeSpec



LandscapeSpec defines the desired state of Landscape.

The transition rule catches namespace being set or unset after creation;
changing a set namespace is caught by the field-level rule. The two rules
are split to stay within the CEL cost budget of the schema.



_Appears in:_
- [Landscape](#landscape)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `displayName` _string_ | DisplayName is the human-readable name of the landscape, shown in user<br />interfaces. It does not affect the namespace name or any label, and it<br />may be changed at any time. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `namespace` _string_ | Namespace overrides the name of the namespace created for this landscape.<br />When unset it defaults to `kden-l-<landscape-name>-<hash>`. It is<br />immutable once the Landscape exists, because the namespace and everything<br />it holds are bound to this name. |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |


### LandscapeStatus



LandscapeStatus defines the observed state of Landscape.



_Appears in:_
- [Landscape](#landscape)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `namespace` _string_ | Namespace is the name of the namespace managed for this landscape. |  | Optional: \{\} <br /> |
| `projectName` _string_ | ProjectName is the name of the project this landscape belongs to,<br />derived from the namespace where the Landscape CR was created. |  | Optional: \{\} <br /> |


### LocalArtifactDeploymentReference



LocalArtifactDeploymentReference holds a reference to an ArtifactDeployment in the same namespace.



_Appears in:_
- [VectorAssignmentSpec](#vectorassignmentspec)
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the ArtifactDeployment. Required. |  |  |
| `collisionCount` _integer_ | CollisionCount salts the ArtifactDeployment name hash to recover from a<br />(rare) hash collision with a different artifact. nil and 0 both mean "no<br />salt" and yield the original, unsalted name. Once bumped it is permanent<br />for this artifact slot. Mirrors Deployment.Status.CollisionCount. |  | Optional: \{\} <br /> |


### LocalObjectReference



LocalObjectReference references an object by name within the same namespace as the parent.



_Appears in:_
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the referenced object. |  |  |


### LocalVectorAssignmentReference



LocalVectorAssignmentReference holds a reference to a VectorAssignment in the same namespace.



_Appears in:_
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the VectorAssignment. Required. |  |  |


### LocalVectorDeploymentReference



LocalVectorDeploymentReference holds a reference to a VectorDeployment in the same namespace.



_Appears in:_
- [VectorAssignmentSpec](#vectorassignmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the VectorDeployment. Required. |  |  |


### OCMComponent



OCMComponent is a wrapper around the OCM ComponentVersion. It can be used to attach additional metadata to an
ArtifactDeployment. The component may include one or more OCM resources.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the OCM ComponentVersion name. |  |  |
| `version` _string_ | Version is the OCM ComponentVersion version. |  | Optional: \{\} <br /> |
| `resources` _[OCMResource](#ocmresource) array_ | Resources contains OCM resources belonging to this component. The structure is intentionally generic to support<br />the requirements of deployers targeting different runtimes. |  | Optional: \{\} <br /> |


### OCMCredentials



OCMCredentials lists Secrets holding `.ocmconfig` or `.dockerconfigjson` data.
All references are same-namespace.



_Appears in:_
- [Credentials](#credentials)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refs` _[CredentialRef](#credentialref) array_ |  |  | MinItems: 1 <br /> |


### OCMResource



OCMResource represents a single resource of an OCM ComponentVersion. The content and type are deployer-specific and
opaque to the API.



_Appears in:_
- [OCMComponent](#ocmcomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the resource name. |  |  |
| `content` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Content holds raw resource data, typically an embedded manifest, file, or<br />binary payload. |  |  |
| `type` _string_ | Type describes the resource type, following OCM conventions. |  |  |


### Project



Project is the Schema for the projects API. A Project owns a dedicated
namespace that stores the project's resources. The project name is
capped at 56 characters so the derived namespace name and label values
stay within the 63-character Kubernetes limits.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `Project` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ProjectSpec](#projectspec)_ |  |  |  |
| `status` _[ProjectStatus](#projectstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: Project
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: sample-project
spec:
  displayName: Sample Project
  # namespace: my-custom-namespace  # overrides the default "kden-p-sample-project"
  roleBindings:
    admin:
      - session:
          memberOf:
            - platform-admins
      - jwks:
          endpoint: https://token.actions.githubusercontent.com/.well-known/openid-configuration
          audience: https://konfidence.example/api
          claims:
            sub: repo:konfidence-project/konfidence:*
    pm:
      - session:
          memberOf:
            - sample-project-pms
    dev:
      - session:
          memberOf:
            - sample-project-devs
```

### ProjectSpec



ProjectSpec defines the desired state of Project.

The transition rule catches namespace being set or unset after creation;
changing a set namespace is caught by the field-level rule. The two rules
are split to stay within the CEL cost budget of the schema.



_Appears in:_
- [Project](#project)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `displayName` _string_ | DisplayName is the human-readable name of the project, shown in user<br />interfaces. It does not affect the namespace name or any label, and it<br />may be changed at any time. |  | MaxLength: 253 <br />MinLength: 1 <br />Optional: \{\} <br /> |
| `namespace` _string_ | Namespace overrides the name of the namespace created for this project.<br />When unset it defaults to `kden-p-<project-name>`. It is immutable<br />once the Project exists, because the namespace and everything it holds<br />are bound to this name. |  | MaxLength: 63 <br />Pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$` <br />Optional: \{\} <br /> |
| `roleBindings` _object (keys:string, values:[Subjects](#subjects))_ | RoleBindings grants project roles to callers. It maps a role name to the<br />list of subjects that hold that role; a caller holds the role if any<br />subject in the list matches (OR). The role names are a fixed, well-known<br />set for now (for example "admin", "pm", "dev"), but the field is a map so<br />the set can be extended without a schema change. See the Project<br />multi-tenancy ADR for the meaning of each role and the authorization flow.<br />RoleBindings is currently schema-only: no authorization is enforced yet. |  | MaxProperties: 32 <br />Optional: \{\} <br /> |


### ProjectStatus



ProjectStatus defines the observed state of Project.



_Appears in:_
- [Project](#project)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `namespace` _string_ | Namespace is the name of the namespace managed for this project. |  | Optional: \{\} <br /> |


### PromotionApproval



PromotionApproval records the granted approval.



_Appears in:_
- [VectorPromotionStatus](#vectorpromotionstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `approvedBy` _string_ | ApprovedBy is the identity that granted the approval, as reported by the<br />konfidence API. The value is an opaque, arbitrary string (username,<br />email, subject, ...); it is recorded verbatim and never interpreted. |  | MinLength: 1 <br /> |
| `approvedAt` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#time-v1-meta)_ | ApprovedAt is the time the approval was granted. |  |  |


### PromotionSourceReference



PromotionSourceReference identifies the in-cluster resource whose current
vector is promoted from. The source is resolved by the config reconciler,
which pins the concrete vector into `VectorPromotionSpec.Vector`; the
execution controller never reads it.



_Appears in:_
- [VectorPromotionConfigSpec](#vectorpromotionconfigspec)
- [VectorPromotionSpec](#vectorpromotionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is the kind of the source resource. A `VectorTemplate` source<br />promotes its latest assembled vector; a `Stage` source promotes the<br />vector currently configured on that stage (`spec.vector`). Whether the resulting promotion<br />requires approval is recorded on the promotion itself<br />(`VectorPromotionSpec.RequireApproval`); the controller defaults it<br />from the source kind. |  | Enum: [VectorTemplate Stage] <br /> |
| `name` _string_ | Name is the name of the source resource. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `landscape` _string_ | Landscape is the `metadata.name` of the `Landscape` in the config's<br />namespace (not its managed namespace) whose namespace hosts the<br />referenced `Stage`. Required for `Stage` references; must be omitted<br />for `VectorTemplate` references, which are resolved in the config's<br />namespace. |  | MaxLength: 63 <br />MinLength: 1 <br />Optional: \{\} <br /> |


### PromotionTargetReference



PromotionTargetReference identifies the `Stage` whose `spec.vector` is the
promotion target.



_Appears in:_
- [VectorPromotionConfigSpec](#vectorpromotionconfigspec)
- [VectorPromotionSpec](#vectorpromotionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is the kind of the target resource. Only `Stage` is supported. |  | Enum: [Stage] <br /> |
| `name` _string_ | Name is the name of the target `Stage`. |  | MaxLength: 253 <br />MinLength: 1 <br /> |
| `landscape` _string_ | Landscape is the `metadata.name` of the `Landscape` in the config's<br />namespace (not its managed namespace) whose namespace hosts the target<br />`Stage`. |  | MaxLength: 63 <br />MinLength: 1 <br /> |


### SessionSubject



SessionSubject matches an interactive user by group membership.



_Appears in:_
- [Subject](#subject)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `memberOf` _string array_ | MemberOf lists the groups that grant the role. Membership in any one of<br />the listed groups is sufficient to match (OR). |  | MaxItems: 64 <br />MinItems: 1 <br />items:MaxLength: 253 <br /> |


### Sign



Sign lists signatures the controller produces on every descriptor it
writes. Absence on a spec disables signing.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `signatures` _[Signature](#signature) array_ |  |  | MinItems: 1 <br /> |


### Signature



Signature pins parameters of one named signature on a component
descriptor. Used both for verification (matched against the fetched
descriptor) and for signing (overrides defaults of the emitted
signature).



_Appears in:_
- [Sign](#sign)
- [Verify](#verify)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the unique identifier for this signature. |  | MinLength: 1 <br /> |
| `algorithm` _string_ | Algorithm specifies the RSA signing algorithm.<br />When omitted, RSASSA-PSS is used.<br />Valid values: RSASSA-PSS, RSASSA-PKCS1-V1_5. |  | Optional: \{\} <br /> |
| `signatureMediaType` _string_ | SignatureMediaType specifies the encoding format for the signature bytes.<br />When omitted, application/x-pem-file (PEM) is used.<br />Valid values: application/x-pem-file, application/vnd.ocm.signature.rsa.pss,<br />application/vnd.ocm.signature.rsa. |  | Optional: \{\} <br /> |
| `hashAlgorithm` _string_ | HashAlgorithm specifies the digest algorithm used when hashing the component descriptor.<br />When omitted, SHA-256 is used.<br />Valid values: SHA-256, SHA-512. |  | Optional: \{\} <br /> |
| `normalisationAlgorithm` _string_ | NormalisationAlgorithm specifies the normalisation scheme applied to the descriptor<br />before hashing.<br />When omitted, jsonNormalisation/v4alpha1 is used.<br />Valid values: jsonNormalisation/v4alpha1. |  | Optional: \{\} <br /> |
| `issuer` _string_ | Issuer pins the expected certificate issuer DN for PEM-encoded signatures.<br />On the sign path the value is stamped into the descriptor alongside the signature,<br />so it is enforced automatically on the verify path even without an explicit pin here.<br />On the verify path, when set, this value overrides whatever the descriptor stored and<br />the handler rejects any signature whose leaf certificate issuer DN does not match.<br />When omitted on both paths the issuer field stays empty and no DN check is performed.<br />Must be non-empty when present. |  | Optional: \{\} <br /> |


### Stage



Stage is the Schema for the stages API.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `Stage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageSpec](#stagespec)_ |  |  |  |
| `status` _[StageStatus](#stagestatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: Stage
metadata:
  name: stage-dev
spec:
  vector: https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1
```

### StageReference



StageReference holds a reference to a Stage in the same namespace.



_Appears in:_
- [StageVersionSpec](#stageversionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Stage. Required. |  |  |


### StageSpec



StageSpec defines the desired state of Stage.



_Appears in:_
- [Stage](#stage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


### StageStatus



StageStatus defines the observed state of Stage.



_Appears in:_
- [Stage](#stage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `vectorHistory` _string array_ |  |  |  |
| `latestVectorDeploymentRef` _[TypedObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#typedobjectreference-v1-core)_ |  |  |  |
| `activeStageVersion` _[StageVersionReference](#stageversionreference)_ | ActiveStageVersion references the StageVersion whose vector is currently<br />active on this stage, mirrored from the stage's active StageVersionUsage. |  | Optional: \{\} <br /> |


### StageVersion



StageVersion is the Schema for the stageversions API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[StageVersionSpec](#stageversionspec)_ | Spec defines the desired state of the StageVersion and is immutable after it has been set |  | Optional: \{\} <br />Required: \{\} <br /> |
| `status` _[StageVersionStatus](#stageversionstatus)_ | status defines the observed state of StageVersion |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: StageVersion
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: stageversion-dev-475124
  ownerReferences:
    - apiVersion: konfidence.cloud/v1alpha1
      kind: Stage
      name: stage-dev
      uid: # TODO: add owner uid here after creating Stage resource
spec:
  vector: https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1
  stageGeneration: 1
  stageRef:
    name: stage-dev
```

### StageVersionReference



StageVersionReference holds a reference to a StageVersion in the same namespace.



_Appears in:_
- [StageStatus](#stagestatus)
- [StageVersionUsageSpec](#stageversionusagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the StageVersion. Required. |  |  |


### StageVersionSpec



StageVersionSpec defines the desired state of StageVersion



_Appears in:_
- [StageVersion](#stageversion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  | MinLength: 1 <br /> |
| `stageGeneration` _integer_ | the object generation of the stage that created this stage version |  | Minimum: 1 <br /> |
| `stageRef` _[StageReference](#stagereference)_ | stageRef references the Stage this StageVersion belongs to |  |  |


### StageVersionStatus



StageVersionStatus defines the observed state of StageVersion.



_Appears in:_
- [StageVersion](#stageversion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


### StageVersionUsage



StageVersionUsage is the Schema for the stageversionusages API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionUsage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[StageVersionUsageSpec](#stageversionusagespec)_ | spec defines the desired state of StageVersionUsage |  | ExactlyOneOf: [stageVersionRef stageVersionSelector] <br />Required: \{\} <br /> |
| `status` _[StageVersionUsageStatus](#stageversionusagestatus)_ | status defines the observed state of StageVersionUsage |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: StageVersionUsage
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: stageversionusage-sample
spec:
  reason: Target usage for stage 'stage-dev'
  stageVersionRef:
    name: stageversion-dev-475124
```

### StageVersionUsageSpec



StageVersionUsageSpec defines the desired state of StageVersionUsage

_Validation:_
- ExactlyOneOf: [stageVersionRef stageVersionSelector]

_Appears in:_
- [StageVersionUsage](#stageversionusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `reason` _string_ | Reason is human-readable description of why this StageVersion is in use, e.g. "executing vector migrations", "latest vector for stage xyz", |  | Optional: \{\} <br /> |
| `stageVersionRef` _[StageVersionReference](#stageversionreference)_ | StageVersionRef references a stageVersion |  | Optional: \{\} <br /> |
| `stageVersionSelector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#labelselector-v1-meta)_ | StageVersionSelector is a label selector to find a StageVersion when name is not provided. |  | Optional: \{\} <br /> |


### StageVersionUsageStatus



StageVersionUsageStatus defines the observed state of StageVersionUsage.



_Appears in:_
- [StageVersionUsage](#stageversionusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `resolvedStageVersions` _string array_ | ResolvedStageVersions contains the names of all resolved stageVersion resources specified by either stageVersionRef or StageVersionSelector |  |  |


### Subject



Subject identifies who is granted a role. Exactly one identity source
(session or jwks) must be set.



_Appears in:_
- [Subjects](#subjects)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `session` _[SessionSubject](#sessionsubject)_ | Session matches an interactively authenticated user by group membership,<br />for example a person signed in through the identity provider. |  | Optional: \{\} <br /> |
| `jwks` _[JWKSSubject](#jwkssubject)_ | JWKS matches a workload identity presenting a token signed by a trusted<br />OIDC provider, for example a CI pipeline's OIDC token. |  | Optional: \{\} <br /> |


### Subjects

_Underlying type:_ _[Subject](#subject)_

Subjects is the list of subjects that hold a role. A caller holds the role
if any subject matches (OR).

_Validation:_
- MaxItems: 32
- MinItems: 1

_Appears in:_
- [ProjectSpec](#projectspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `session` _[SessionSubject](#sessionsubject)_ | Session matches an interactively authenticated user by group membership,<br />for example a person signed in through the identity provider. |  | Optional: \{\} <br /> |
| `jwks` _[JWKSSubject](#jwkssubject)_ | JWKS matches a workload identity presenting a token signed by a trusted<br />OIDC provider, for example a CI pipeline's OIDC token. |  | Optional: \{\} <br /> |


### TaskExecution



TaskExecution is the Schema for the taskexecutions API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `TaskExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[TaskExecutionSpec](#taskexecutionspec)_ | spec defines the desired state of TaskExecution |  | Required: \{\} <br /> |
| `status` _[TaskExecutionStatus](#taskexecutionstatus)_ | status defines the observed state of TaskExecution |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: TaskExecution
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: taskexecution-sample
spec:
  name: service2-task-1
  type: k8s-job
  dependsOn: [ ]
  spec: {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "service1-task-1-container",
            "image": "registry.kdenv.lab/docker/sample-project/service1-task-1:0.0.1",
            "command": [
              "echo",
              "I am task 1 of service 1"
            ]
          }
        ]
      },
      "restartPolicy": "Never"
    },
    "backoffLimit": 4
  }
```

### TaskExecutionSpec



TaskExecutionSpec defines the desired state of TaskExecution



_Appears in:_
- [TaskExecution](#taskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `dependsOn` _string array_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |


### TaskExecutionStatus



TaskExecutionStatus defines the observed state of TaskExecution.



_Appears in:_
- [TaskExecution](#taskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


### TaskManifest



TaskManifest defines a post-deployment task that is executed after the artifact has been deployed. Tasks are
commonly used for vector migrations (such as database schema changes) but may represent any post-deployment action.

Tasks form a directed acyclic graph (DAG) at the *vector level* rather than only within a single ArtifactDeployment.
A task may depend on tasks belonging to other microservices or artifacts in the same VectorDeployment. These
cross-artifact dependencies allow defining a globally ordered migration or transformation workflow.

The controller responsible for the task type interprets the Spec field and performs the execution once all declared
dependencies have completed successfully.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name uniquely identifies this task within the entire vector. This name may be referenced by other tasks across<br />different artifacts. |  |  |
| `type` _string_ | Type specifies the task controller or execution runtime (e.g. "k8s-job", or any custom task runtime). Different<br />task types correspond to different task controllers, each interpreting the Spec field according to their own semantics. |  |  |
| `dependsOn` _string array_ | DependsOn lists names of other tasks that must complete before this task may run. Dependencies may reference<br />tasks within the same artifact or any other artifact that participates in the same VectorDeployment, allowing the<br />formation of a vector-wide DAG. |  | Optional: \{\} <br /> |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Spec contains task-specific configuration. The structure depends on the task Type and is interpreted by the<br />corresponding task controller. |  |  |


### VectorActivation



VectorActivation is the Schema for the vectoractivations API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorActivation` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorActivationSpec](#vectoractivationspec)_ | spec defines the desired state of VectorActivation |  | Required: \{\} <br /> |
| `status` _[VectorActivationStatus](#vectoractivationstatus)_ | status defines the observed state of VectorActivation |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorActivation
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: vectoractivation-sample
  ownerReferences:
    - apiVersion: konfidence.cloud/v1alpha1
      kind: StageVersion
      name: <OWNER-NAME> # TODO: add stage version name here
      uid: <OWNER-UID>‚ # TODO: add stage version uid here
spec:
  # TODO: clarify if stageVersion is required here since it is also in OwnerReferences
  stage: stage-dev
  stageVersion: <STAGE-VERSION-NAME> # TODO: add stage version name here
  vector: https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1
  vectorDeployment: common.konfidence.cloud.example.vector-0.0.1
```

### VectorActivationSpec



VectorActivationSpec defines the desired state of VectorActivation



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stage` _string_ |  |  |  |
| `stageVersion` _string_ |  |  |  |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |
| `vectorDeployment` _string_ |  |  |  |


### VectorActivationStatus



VectorActivationStatus defines the observed state of VectorActivation.



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


### VectorAssignment



VectorAssignment is the Schema for the vectorassignments API.

A VectorAssignment represents a single binding between a VectorDeployment and an ArtifactDeployment. It enables
an n:m mapping where a single artifact may be reused across multiple vectors. These objects are automatically
managed by the vector-deployment-controller and reconciled by deployers to apply vector-specific configuration.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorAssignment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorAssignmentSpec](#vectorassignmentspec)_ | Spec defines the desired state of the VectorAssignment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[VectorAssignmentStatus](#vectorassignmentstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorAssignment
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: vectorassignment-sample
spec:
  manifest:
    type: helm.konfidence.cloud
    allowReuse: true
  artifactDeploymentRef:
    name: artifactdeployment-1
  vectorDeploymentRef:
    name: vectordeployment-1
```

### VectorAssignmentSpec



VectorAssignmentSpec defines the desired state of a VectorAssignment.

A VectorAssignment represents one logical binding between a VectorDeployment and an ArtifactDeployment. Since a
single artifact may be reused across multiple vectors, an n:m relationship exists between vectors and artifacts.
VectorAssignment creates a concrete instance of that relationship.

VectorAssignment resources are created automatically during vector rollouts and are typically not authored by users.
Deployer implementations reconcile the VectorAssignment to perform vector-specific configuration based on the
artifact selected for this vector.

The VectorAssignmentSpec is immutable. If an artifact is replaced or added to a different vector, the old
VectorAssignment is deleted and a new one created.



_Appears in:_
- [VectorAssignment](#vectorassignment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `manifest` _[ArtifactManifest](#artifactmanifest)_ | Manifest contains the ArtifactManifest describing the artifact to be assigned to the vector. This duplicates the<br />manifest stored in the ArtifactDeployment for efficiency: deployers often need to filter or select assignments<br />by artifact type, and embedding the manifest avoids repeated API lookups. |  |  |
| `artifactDeploymentRef` _[LocalArtifactDeploymentReference](#localartifactdeploymentreference)_ | ArtifactDeploymentRef references the ArtifactDeployment instance that is associated with the vector. The<br />referenced artifact must exist in the same namespace as this VectorAssignment. |  |  |
| `vectorDeploymentRef` _[LocalVectorDeploymentReference](#localvectordeploymentreference)_ | VectorDeploymentRef references the VectorDeployment that this artifact is assigned to. This creates the explicit<br />mapping "artifact X belongs to vector Y". |  |  |


### VectorAssignmentStatus



VectorAssignmentStatus defines the observed state of a VectorAssignment.

A VectorAssignment progresses through a simple lifecycle driven by the deployer:

 1. VectorAssignment is created by the vector-deployment-controller.
 2. deployer reconciles it and configures vector-specific integration
 3. VectorAssignmentReadyCondition is set to True



_Appears in:_
- [VectorAssignment](#vectorassignment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions describes the latest observed state of the assignment. The primary condition is<br />VectorAssignmentReadyCondition, which becomes True once the deployer has finished processing the VectorAssignment. |  | Optional: \{\} <br /> |


### VectorConfig



VectorConfig defines feature flags and authored configuration values for a vector.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `features` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Features define the feature flags. |  |  |
| `authored` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Authored define the authored configuration values. |  |  |


### VectorData



VectorData is the schema for the vectordata API.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorData` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorDataSpec](#vectordataspec)_ |  |  | Optional: \{\} <br /> |
| `status` _[VectorDataStatus](#vectordatastatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorData
metadata:
  name: vectordata-sample
spec:
  features:
    checkout-v2:
      enabled: true
  authored:
    replicas: 3
  deploymentResults:
    https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector/service1:
      - name: candidates
        type: http-k8s-service
        spec:
          namespace: dev
          k8sName: candidates
          servicePorts:
            - name: http
              port: 80
```

### VectorDataSpec



VectorDataSpec is the LCP→landscape-orchestrator contract for vector-scoped data.
The vector deployment controller resolves the OCM envelope `{features, authored}` and aggregates per-AD
DeploymentResults; the landscape orchestrator materialises the payload on its
target runtime (ConfigMap on K8s, etc.).



_Appears in:_
- [VectorData](#vectordata)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `features` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Features carries the optional "features" subset of the OCM envelope, verbatim JSON. |  | Optional: \{\} <br /> |
| `authored` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Authored carries the optional "authored" subset of the OCM envelope, verbatim JSON. |  | Optional: \{\} <br /> |
| `deploymentResults` _object (keys:string, values:[ComponentDeploymentResults](#componentdeploymentresults))_ | DeploymentResults aggregated from underlying ArtifactDeployments, keyed by artifact<br />component name; the value lists every result emitted by that component. Within a<br />component's list, results are unique by (name, type). |  | MaxProperties: 64 <br />Optional: \{\} <br /> |


### VectorDataStatus







_Appears in:_
- [VectorData](#vectordata)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  | Optional: \{\} <br /> |


### VectorDeployment



VectorDeployment is the Schema for the vectordeployments API.

VectorDeployment represents the deployment of an immutable vector of artifacts into a specific environment or stage.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorDeploymentSpec](#vectordeploymentspec)_ | Spec defines the desired state of the VectorDeployment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[VectorDeploymentStatus](#vectordeploymentstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorDeployment
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: common.konfidence.cloud.example.vector-0.0.1
spec:
  vector: https://registry.konfidence.cloud/ocm/vector//common.konfidence.cloud/example/vector:0.0.1
```

### VectorDeploymentSpec



VectorDeploymentSpec defines the desired state of a VectorDeployment.

A VectorDeployment references a deployment vector stored as an OCM ComponentVersion in an OCI registry. The vector
describes a complete, immutable set of artifacts and versions that should be deployed as a unit.

The value must always be a fully qualified OCI URL and must resolve to a valid OCM ComponentVersion. The
VectorDeployment spec is intended to be immutable. Any substantive change should result in a new VectorDeployment
instance rather than updating an existing one.



_Appears in:_
- [VectorDeployment](#vectordeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector is a fully qualified URL pointing to an OCM ComponentVersion stored in an OCI registry. The referenced<br />component contains the deployment vector, which includes the complete list of artifacts and their versions. |  |  |


### VectorDeploymentStatus



VectorDeploymentStatus represents the observed state of a VectorDeployment as it progresses through the
deployment lifecycle.

The lifecycle consists of:
 1. Pulling the vector from the OCI registry and parsing its contents -> VectorDownloadedCondition
 2. Creating (or re-using) one ArtifactDeployment per artifact in the vector -> ArtifactDeploymentsCreatedCondition
 3. Waiting until all ArtifactDeployments have successfully deployed -> VectorDeployedCondition
 4. Creating all VectorAssignment resources associated with this vector -> VectorAssignmentsCreatedCondition
 5. Creating the VectorData CR with the resolved authored configuration + aggregated DeploymentResults; the
    runtime-specific implementor then materialises it (e.g. as a ConfigMap on Kubernetes) -> VectorDataCreatedCondition
 6. Marking the vector as ready for use once VectorData reports its own Ready=True -> VectorReadyCondition



_Appears in:_
- [VectorDeployment](#vectordeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions represents the current set of status conditions for this vector<br />deployment. These conditions track progress through the lifecycle stages. |  |  |
| `resolvedVectorOcm` _string_ | ResolvedVectorOcm contains the fully materialized content of the OCM ComponentVersion after it has been<br />downloaded and resolved from the OCI registry. Unlike the Spec.Vector value, which is only a reference (URL),<br />this field stores the actual resolved vector content as provided by OCM, including all artifacts and metadata.<br />It is not a reference but the inlined representation of the component version at reconciliation time. |  |  |
| `resultingVectorData` _[LocalObjectReference](#localobjectreference)_ | ResultingVectorData records the name of the VectorData object created for this VectorDeployment. The VectorData<br />CR is the contract between the vector deployment controller (which resolves the OCM payload) and the runtime-specific implementor<br />(which materialises it on the target runtime). The field is empty until step 5 of the lifecycle has produced the<br />CR. Names are stable across reconciliations. |  |  |
| `resultingArtifactDeployments` _object (keys:string, values:[LocalArtifactDeploymentReference](#localartifactdeploymentreference))_ | ResultingArtifactDeployments lists the ArtifactDeployment resources created (or re-used) for this vector. The<br />map key is the component name of the artifact as defined inside the vector. Keys remain stable across<br />reconciliations and re-creations. |  |  |
| `resultingVectorAssignments` _object (keys:string, values:[LocalVectorAssignmentReference](#localvectorassignmentreference))_ | ResultingVectorAssignments lists all VectorAssignment resources created for this vector. VectorAssignments are<br />not re-used like ArtifactDeployments, but instead each VectorDeployment results in a complete new set of<br />assignments.<br />The map key is the component name of the artifact. Keys are stable across reconcilations. |  |  |
| `deploymentResults` _object (keys:string, values:[ComponentDeploymentResults](#componentdeploymentresults))_ | DeploymentResults exposes an aggregated view of the deployment results produced<br />by all underlying ArtifactDeployments. The map key is the artifact component name;<br />the value lists every result emitted by that ArtifactDeployment. Within a component's<br />list, results are unique by (name, type). |  | MaxProperties: 64 <br /> |


### VectorMigration



VectorMigration is the Schema for the vectormigrations API



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorMigration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorMigrationSpec](#vectormigrationspec)_ | spec defines the desired state of VectorMigration |  | Required: \{\} <br /> |
| `status` _[VectorMigrationStatus](#vectormigrationstatus)_ | status defines the observed state of VectorMigration |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorMigration
metadata:
  labels:
    app.kubernetes.io/name: crds
  name: vectormigration-dev-985434
spec:
  stageVersion: stageversion-dev-475124
  vector: https://registry.konfidence.cloud/ocm/vector//common.konfidence.cloud/example/vector:0.0.1
```

### VectorMigrationSpec



VectorMigrationSpec defines the desired state of VectorMigration



_Appears in:_
- [VectorMigration](#vectormigration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stageVersion` _string_ |  |  |  |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


### VectorMigrationStatus



VectorMigrationStatus defines the observed state of VectorMigration.



_Appears in:_
- [VectorMigration](#vectormigration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


### VectorPromotion



VectorPromotion triggers a one-time execution of a promotion flow defined by a VectorPromotionConfig.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorPromotionSpec](#vectorpromotionspec)_ |  |  |  |
| `status` _[VectorPromotionStatus](#vectorpromotionstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorPromotion
metadata:
  namespace: default
  name: sample-vector-promotion
spec:
  vectorPromotionConfigName: sample-promotion-config
  source:
    kind: VectorTemplate
    name: sample-vector-template
  target:
    kind: Stage
    name: sample-stage
    landscape: sample-landscape
  vector: registry.kdenv.lab/sample-project//konfidence-project.com/constructed-vector:2026.8.5-090000000Z
  requireApproval: false
  sequence: 1
  ttlAfterFinished: 1h
```

### VectorPromotionConfig



VectorPromotionConfig describes a promotion flow for a vector between a source and a target.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotionConfig` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorPromotionConfigSpec](#vectorpromotionconfigspec)_ | Spec defines the desired state of the VectorPromotionConfig. |  | Optional: \{\} <br /> |
| `status` _[VectorPromotionConfigStatus](#vectorpromotionconfigstatus)_ |  |  |  |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorPromotionConfig
metadata:
  namespace: kden-p-sample
  name: sample-promotion-config
spec:
  source:
    kind: VectorTemplate
    name: sample-vector-template
  target:
    kind: Stage
    name: sample-stage
    landscape: sample-landscape
  ttlAfterFinished: 1h
```

### VectorPromotionConfigSpec



VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.



_Appears in:_
- [VectorPromotionConfig](#vectorpromotionconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _[PromotionSourceReference](#promotionsourcereference)_ | Source references the resource to promote from. |  |  |
| `target` _[PromotionTargetReference](#promotiontargetreference)_ | Target references the Stage to promote to. |  |  |
| `ttlAfterFinished` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | TTLAfterFinished will be copied onto every VectorPromotion the drift<br />controller creates for this config. See<br />`VectorPromotionSpec.TTLAfterFinished`. |  | Optional: \{\} <br /> |
| `keepLastPromotions` _integer_ | KeepLastPromotions bounds how many terminal VectorPromotions are<br />retained per config; the oldest beyond the bound are deleted. Retention<br />by count keeps an audit trail even when `ttlAfterFinished` is short.<br />Non-terminal promotions are never deleted and do not count toward the<br />bound. | 10 | Minimum: 0 <br />Optional: \{\} <br /> |


### VectorPromotionConfigStatus



VectorPromotionConfigStatus defines the observed state of VectorPromotionConfig.



_Appears in:_
- [VectorPromotionConfig](#vectorpromotionconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions reports on the config itself, e.g. whether its references<br />resolve to existing resources. Promotion results are reported separately<br />in `LastPromotionConditions`. |  |  |
| `lastPromotionConditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | LastPromotionConditions contains the result of the most recent VectorPromotion execution |  |  |
| `lastSuccessfulPromotionConditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | LastSuccessfulPromotionConditions contains the result of the most recent VectorPromotion execution, that was successful |  |  |
| `sequence` _integer_ | Sequence is the monotonic counter of promotions created for this config.<br />The config reconciler increments it and stamps the value into each<br />created promotion's `spec.sequence`. |  | Optional: \{\} <br /> |


### VectorPromotionSpec



VectorPromotionSpec defines the desired state of VectorPromotion.



_Appears in:_
- [VectorPromotion](#vectorpromotion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vectorPromotionConfigName` _string_ | VectorPromotionConfigName is the name of the VectorPromotionConfig that defines the promotion flow to execute. |  | MinLength: 1 <br /> |
| `source` _[PromotionSourceReference](#promotionsourcereference)_ | Source is a snapshot of the config's source reference at creation time,<br />recorded so a promotion is self-describing. |  |  |
| `target` _[PromotionTargetReference](#promotiontargetreference)_ | Target is a snapshot of the config's target reference at creation time.<br />Execution resolves and writes this target: approving a promotion approves<br />exactly this destination, regardless of later config edits. |  |  |
| `vector` _string_ | Vector is the concrete OCM component version reference<br />(`<registry>//<component>:<version>`) pinned when the promotion was created. |  | MinLength: 1 <br /> |
| `requireApproval` _boolean_ | RequireApproval is true when the promotion must be approved before<br />execution; false means the promotion is approved automatically. It is<br />independent of the source kind: the config controller defaults it to<br />true for `Stage` sources, but any combination is valid. | false | Optional: \{\} <br /> |
| `ttlAfterFinished` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | TTLAfterFinished defines how long the VectorPromotion should be kept after completion.<br />Once the TTL expires after the promotion reaches a terminal state (Completed or Failed),<br />the resource is eligible for automatic deletion. If no TTL is set, no deletion happens. |  | Optional: \{\} <br /> |
| `sequence` _integer_ | Sequence is a monotonic ordinal assigned by the creator (the config<br />reconciler, from the config's `status.sequence`). It is the sole<br />ordering between promotions of the same config; creation timestamps<br />only have second resolution and are never consulted. |  | Minimum: 0 <br /> |


### VectorPromotionState

_Underlying type:_ _string_

VectorPromotionState summarizes the promotion lifecycle for display.
Conditions are the source of truth; the state is derived from them.



_Appears in:_
- [VectorPromotionStatus](#vectorpromotionstatus)

| Field | Description |
| --- | --- |
| `Waiting` | PromotionStateWaiting means at least one gate is still open: the<br />promotion requires approval and has not been approved yet.<br /> |
| `Ready` | PromotionStateReady means every gate has passed and the promotion is<br />queued for execution. Promotions that require no approval are Ready<br />from their first reconcile.<br /> |
| `InProgress` | PromotionStateInProgress means the promotion is executing.<br /> |
| `Blocked` | PromotionStateBlocked means the promotion is ready but cannot execute<br />because its target does not resolve; see the config's Ready condition.<br /> |
| `Succeeded` | PromotionStateSucceeded means the promotion completed successfully.<br /> |
| `Failed` | PromotionStateFailed means the promotion reached a terminal state without success.<br /> |
| `Superseded` | PromotionStateSuperseded means a newer promotion replaced this one.<br />Superseded promotions are locked: they can never be approved or<br />executed afterwards. The newer promotion is the one to act on.<br /> |


### VectorPromotionStatus



VectorPromotionStatus defines the observed state of VectorPromotion.



_Appears in:_
- [VectorPromotion](#vectorpromotion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `state` _[VectorPromotionState](#vectorpromotionstate)_ | State summarizes Conditions for display. Conditions are the source of<br />truth; State is recomputed whenever conditions are written. `Superseded`<br />is a locked terminal state: a superseded promotion can never be<br />approved or executed afterwards, only its successor can. |  | Enum: [Waiting Ready InProgress Blocked Succeeded Failed Superseded] <br />Optional: \{\} <br /> |
| `approval` _[PromotionApproval](#promotionapproval)_ | Approval records the granted approval. A promotion is approved at most<br />once; re-approval attempts are rejected. |  | Optional: \{\} <br /> |
| `promotedStageRef` _[TypedObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#typedobjectreference-v1-core)_ | PromotedStageRef records the Stage this promotion actually wrote its<br />vector to, so the promotion is self-describing even after the config<br />changed or was deleted. |  | Optional: \{\} <br /> |


### VectorTemplate



VectorTemplate represents a template for assembling OCM components into an OCM component
that represents a vector.



_Appears in:_

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorTemplate` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorTemplateSpec](#vectortemplatespec)_ | spec defines the desired state of VectorTemplate |  | Required: \{\} <br /> |
| `status` _[VectorTemplateStatus](#vectortemplatestatus)_ | status defines the observed state of VectorTemplate |  | Optional: \{\} <br /> |


#### Example

```yaml
apiVersion: konfidence.cloud/v1alpha1
kind: VectorTemplate
metadata:
  namespace: dev
  name: shopping-app-latest
spec:
  base:
    kind: VectorTemplate
    name: shopping-app-base
  uploadTarget: https://registry.kdenv.lab/sample-project//konfidence-project/constructed-vector
  components:
    - name: https://registry.kdenv.lab/sample-project//konfidence-project/sample-vector/service1
    - name: https://registry.kdenv.lab/sample-project//konfidence-project/sample-vector/service1
    - name: https://registry.dwc.com/sample-project//dwc.tools.sap/dwc-project/dev/service2
    - name: https://registry.example.lab/sample-project//dwc.tools.sap/dwc-project/dev/service3
  config:
    - kind: Secret
      apiVersion: v1
      name: registry-credentials
```

### VectorTemplateReference



VectorTemplateReference holds a reference to a VectorTemplate in the same namespace,
used as the base of another VectorTemplate.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `kind` _string_ | Kind is the kind of the referenced object. Only VectorTemplate is supported for now. | VectorTemplate | Enum: [VectorTemplate] <br /> |
| `name` _string_ | Name is the name of the referenced VectorTemplate. Required. |  | MinLength: 1 <br /> |


### VectorTemplateSpec



VectorTemplateSpec defines the desired state of VectorTemplate.
VectorTemplateSpec defines the components of which a vector is composed.
From a VectorTemplate an OCM component is created which contains the latest version of all listed components.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `reconcileInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | ReconcileInterval defines how often the assembly controller should check for drift.<br />If not set, the controller's default reconcile interval will be used. |  | Optional: \{\} <br /> |
| `uploadTarget` _string_ | UploadTarget defines the target OCM component where the assembled vector will be uploaded. |  |  |
| `base` _[VectorTemplateReference](#vectortemplatereference)_ | Base references another VectorTemplate whose most recently assembled vector<br />(status.latestVector) is used as the base for this vector's assembly. |  | Optional: \{\} <br />Optional: \{\} <br /> |
| `components` _[Component](#component) array_ | Components lists the components to be included in the vector. |  | MinItems: 1 <br /> |
| `credentials` _[Credentials](#credentials)_ | Credentials supplies credentials for OCM repositories<br />and signing/verification key material. |  | Optional: \{\} <br /> |
| `verifyArtifacts` _[Verify](#verify)_ | VerifyArtifacts lists candidate signatures evaluated against every<br />artifact pulled into the assembly. Absence disables artifact<br />verification. |  | Optional: \{\} <br /> |
| `verifyVector` _[Verify](#verify)_ | VerifyVector lists candidate signatures evaluated against any<br />vector the assembly fetches (base or pre-existing upload target).<br />Absence disables vector verification. |  | Optional: \{\} <br /> |
| `signVector` _[Sign](#sign)_ | SignVector lists signatures the controller produces on the emitted<br />vector. Absence disables signing. |  | Optional: \{\} <br /> |
| `vectorConfig` _[VectorConfig](#vectorconfig)_ |  |  | Optional: \{\} <br /> |


### VectorTemplateStatus



VectorTemplateStatus defines the observed state of VectorTemplate.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `latestVector` _string_ | LatestVector is the concrete OCM component version of the most recently<br />assembled vector, in the form `<repository>//<component>:<version>`. It is<br />empty until the first successful assembly. |  | Optional: \{\} <br /> |


### Verify



Verify lists candidate signatures evaluated against every fetched
descriptor. Absence on a spec disables verification.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `signatures` _[Signature](#signature) array_ |  |  | MinItems: 1 <br /> |
