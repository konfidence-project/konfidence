# API Reference

## Packages
- [landscape.konfidence.cloud/v1alpha1](#landscapekonfidencecloudv1alpha1)


## landscape.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the landscape v1alpha1 API group.

### Resource Types
- [ActivationTaskExecution](#activationtaskexecution)
- [ActivationTaskExecutionList](#activationtaskexecutionlist)
- [ActivationTaskRegistration](#activationtaskregistration)
- [ActivationTaskRegistrationList](#activationtaskregistrationlist)
- [ArtifactDeployment](#artifactdeployment)
- [ArtifactDeploymentList](#artifactdeploymentlist)
- [Stage](#stage)
- [StageList](#stagelist)
- [StageVersion](#stageversion)
- [StageVersionList](#stageversionlist)
- [StageVersionUsage](#stageversionusage)
- [StageVersionUsageList](#stageversionusagelist)
- [TaskExecution](#taskexecution)
- [TaskExecutionList](#taskexecutionlist)
- [VectorActivation](#vectoractivation)
- [VectorActivationList](#vectoractivationlist)
- [VectorAssignment](#vectorassignment)
- [VectorAssignmentList](#vectorassignmentlist)
- [VectorDeployment](#vectordeployment)
- [VectorDeploymentList](#vectordeploymentlist)
- [VectorMigration](#vectormigration)
- [VectorMigrationList](#vectormigrationlist)



#### ActivationTaskExecution



ActivationTaskExecution is the Schema for the ActivationTaskExecutions API



_Appears in:_
- [ActivationTaskExecutionList](#activationtaskexecutionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[ActivationTaskExecutionSpec](#activationtaskexecutionspec)_ | spec defines the desired state of ActivationTaskExecution |  | Required: \{\} <br /> |
| `status` _[ActivationTaskExecutionStatus](#activationtaskexecutionstatus)_ | status defines the observed state of ActivationTaskExecution |  | Optional: \{\} <br /> |


#### ActivationTaskExecutionList



ActivationTaskExecutionList contains a list of ActivationTaskExecution





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskExecutionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ActivationTaskExecution](#activationtaskexecution) array_ |  |  |  |


#### ActivationTaskExecutionSpec



ActivationTaskExecutionSpec defines the desired state of ActivationTaskExecution



_Appears in:_
- [ActivationTaskExecution](#activationtaskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |
| `vectorActivation` _string_ | VectorActivation is a temporary field that contains the name of the associated vectorActivation |  |  |


#### ActivationTaskExecutionStatus



ActivationTaskExecutionStatus defines the observed state of ActivationTaskExecution.



_Appears in:_
- [ActivationTaskExecution](#activationtaskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### ActivationTaskRegistration



ActivationTaskRegistration is the Schema for the activationtaskregistrations API



_Appears in:_
- [ActivationTaskRegistrationList](#activationtaskregistrationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskRegistration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[ActivationTaskRegistrationSpec](#activationtaskregistrationspec)_ | spec defines the desired state of ActivationTaskRegistration |  | Required: \{\} <br /> |
| `status` _[ActivationTaskRegistrationStatus](#activationtaskregistrationstatus)_ | status defines the observed state of ActivationTaskRegistration |  | Optional: \{\} <br /> |


#### ActivationTaskRegistrationList



ActivationTaskRegistrationList contains a list of ActivationTaskRegistration





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationTaskRegistrationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ActivationTaskRegistration](#activationtaskregistration) array_ |  |  |  |


#### ActivationTaskRegistrationSpec



ActivationTaskRegistrationSpec defines the desired state of ActivationTaskRegistration



_Appears in:_
- [ActivationTaskRegistration](#activationtaskregistration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | INSERT ADDITIONAL SPEC FIELDS - desired state of cluster<br />Important: Run "make" to regenerate code after modifying this file<br />The following markers will use OpenAPI v3 schema to validate the value<br />More info: https://book.kubebuilder.io/reference/markers/crd-validation.html |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |
| `succeeds` _string array_ |  |  |  |
| `precedes` _string array_ |  |  |  |


#### ActivationTaskRegistrationStatus



ActivationTaskRegistrationStatus defines the observed state of ActivationTaskRegistration.



_Appears in:_
- [ActivationTaskRegistration](#activationtaskregistration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | conditions represent the current state of the ActivationTaskRegistration resource.<br />Each condition has a unique type and reflects the status of a specific aspect of the resource.<br />Standard condition types include:<br />- "Available": the resource is fully functional<br />- "Progressing": the resource is being created or updated<br />- "Degraded": the resource failed to reach or maintain its desired state<br />The status of each condition is one of True, False, or Unknown. |  | Optional: \{\} <br /> |


#### ArtifactDeployment



ArtifactDeployment is the Schema for the artifactdeployments API.



_Appears in:_
- [ArtifactDeploymentList](#artifactdeploymentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ArtifactDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactDeploymentSpec](#artifactdeploymentspec)_ | Spec defines the desired state of the ArtifactDeployment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[ArtifactDeploymentStatus](#artifactdeploymentstatus)_ |  |  |  |


#### ArtifactDeploymentList



ArtifactDeploymentList contains a list of ArtifactDeployment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ArtifactDeploymentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ArtifactDeployment](#artifactdeployment) array_ |  |  |  |


#### ArtifactDeploymentSpec



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


#### ArtifactDeploymentStatus



ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.



_Appears in:_
- [ArtifactDeployment](#artifactdeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration is the last observed generation. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions describes the state of the deployment lifecycle. The following conditions are expected:<br />  - ArtifactFetched: the artifact was successfully retrieved<br />  - ArtifactDeployed: the artifact was successfully deployed<br />  - AppHealthy: the deployer reports the workload as healthy<br />Conditions progress in a linear order:<br />ArtifactFetched -> ArtifactDeployed -> AppHealthy |  | Optional: \{\} <br /> |
| `deploymentResult` _[DeploymentResult](#deploymentresult) array_ | DeploymentResults captures structured outputs produced by the deployer during the deployment process—such as<br />computed DNS names, service endpoints, generated configuration, or other workload-specific details.<br />Results should be treated as immutable for a given generation and may be consumed by later stages of a vector<br />rollout (e.g., routing configuration).<br />Each result must have a unique Name. |  | Optional: \{\} <br /> |


#### ArtifactManifest



ArtifactManifest describes the content of the artifact, thus it determines the deployer implementation responsible
for handling it.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)
- [VectorAssignmentSpec](#vectorassignmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | Type specifies the deployer that should handle this artifact (e.g., "cloud.konfidence.flux.helm",<br />"cloud.konfidence.flux.kustomize", or any custom deployer type). Deployers implement their own interpretation<br />of the artifact's contents. |  |  |
| `allowReuse` _boolean_ | AllowReuse indicates whether the deployed artifact instance may be shared across multiple VectorDeployments.<br />Reuse allows more efficient resource consumption but requires the artifact to be independent of vector-specific<br />runtime context. |  |  |


#### DeploymentResult



DeploymentResult contains a single output produced by a deployer. These results are used to transport information
from the deployer to later phases of the vector lifecycle.



_Appears in:_
- [ArtifactDeploymentStatus](#artifactdeploymentstatus)
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is a unique identifier for the result within an ArtifactDeploymentStatus. |  |  |
| `type` _string_ | Type describes the structure contained in Spec. Each deployer may define multiple result types. |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Spec contains deployer-specific structured data. Its format is determined by the Type field. |  |  |


#### LocalArtifactDeploymentReference



LocalArtifactDeploymentReference holds a reference to an ArtifactDeployment in the same namespace.



_Appears in:_
- [VectorAssignmentSpec](#vectorassignmentspec)
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the ArtifactDeployment. Required. |  |  |


#### LocalVectorAssignmentReference



LocalVectorAssignmentReference holds a reference to a VectorAssignment in the same namespace.



_Appears in:_
- [VectorDeploymentStatus](#vectordeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the VectorAssignment. Required. |  |  |


#### LocalVectorDeploymentReference



LocalVectorDeploymentReference holds a reference to a VectorDeployment in the same namespace.



_Appears in:_
- [VectorAssignmentSpec](#vectorassignmentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name	is the name of the VectorDeployment. Required. |  |  |


#### OCMComponent



OCMComponent is a wrapper around the OCM ComponentVersion. It can be used to attach additional metadata to an
ArtifactDeployment. The component may include one or more OCM resources.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the OCM ComponentVersion name. |  |  |
| `version` _string_ | Version is the OCM ComponentVersion version. |  | Optional: \{\} <br /> |
| `resources` _[OCMResource](#ocmresource) array_ | Resources contains OCM resources belonging to this component. The structure is intentionally generic to support<br />the requirements of deployers targeting different runtimes. |  | Optional: \{\} <br /> |


#### OCMResource



OCMResource represents a single resource of an OCM ComponentVersion. The content and type are deployer-specific and
opaque to the API.



_Appears in:_
- [OCMComponent](#ocmcomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the resource name. |  |  |
| `content` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Content holds raw resource data, typically an embedded manifest, file, or<br />binary payload. |  |  |
| `type` _string_ | Type describes the resource type, following OCM conventions. |  |  |


#### Stage



Stage is the Schema for the stages API.



_Appears in:_
- [StageList](#stagelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `Stage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageSpec](#stagespec)_ |  |  |  |
| `status` _[StageStatus](#stagestatus)_ |  |  |  |


#### StageList



StageList contains a list of Stage.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Stage](#stage) array_ |  |  |  |


#### StageReference



StageReference holds a reference to a Stage in the same namespace.



_Appears in:_
- [StageVersionSpec](#stageversionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the Stage. Required. |  |  |


#### StageSpec



StageSpec defines the desired state of Stage.



_Appears in:_
- [Stage](#stage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


#### StageStatus



StageStatus defines the observed state of Stage.



_Appears in:_
- [Stage](#stage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `vectorHistory` _string array_ |  |  |  |
| `latestVectorDeploymentRef` _[TypedObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#typedobjectreference-v1-core)_ |  |  |  |


#### StageVersion



StageVersion is the Schema for the stageversions API



_Appears in:_
- [StageVersionList](#stageversionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[StageVersionSpec](#stageversionspec)_ | Spec defines the desired state of the StageVersion and is immutable after it has been set |  | Optional: \{\} <br />Required: \{\} <br /> |
| `status` _[StageVersionStatus](#stageversionstatus)_ | status defines the observed state of StageVersion |  | Optional: \{\} <br /> |


#### StageVersionList



StageVersionList contains a list of StageVersion





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageVersion](#stageversion) array_ |  |  |  |


#### StageVersionReference



StageVersionReference holds a reference to a StageVersion in the same namespace.



_Appears in:_
- [StageVersionUsageSpec](#stageversionusagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the StageVersion. Required. |  |  |


#### StageVersionSpec



StageVersionSpec defines the desired state of StageVersion



_Appears in:_
- [StageVersion](#stageversion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  | MinLength: 1 <br /> |
| `stageGeneration` _integer_ | the object generation of the stage that created this stage version |  | Minimum: 1 <br /> |
| `stageRef` _[StageReference](#stagereference)_ | stageRef references the Stage this StageVersion belongs to |  |  |


#### StageVersionStatus



StageVersionStatus defines the observed state of StageVersion.



_Appears in:_
- [StageVersion](#stageversion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### StageVersionUsage



StageVersionUsage is the Schema for the stageversionusages API



_Appears in:_
- [StageVersionUsageList](#stageversionusagelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionUsage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[StageVersionUsageSpec](#stageversionusagespec)_ | spec defines the desired state of StageVersionUsage |  | ExactlyOneOf: [stageVersionRef stageVersionSelector] <br />Required: \{\} <br /> |
| `status` _[StageVersionUsageStatus](#stageversionusagestatus)_ | status defines the observed state of StageVersionUsage |  | Optional: \{\} <br /> |


#### StageVersionUsageList



StageVersionUsageList contains a list of StageVersionUsage





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionUsageList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageVersionUsage](#stageversionusage) array_ |  |  |  |


#### StageVersionUsageSpec



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


#### StageVersionUsageStatus



StageVersionUsageStatus defines the observed state of StageVersionUsage.



_Appears in:_
- [StageVersionUsage](#stageversionusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `resolvedStageVersions` _string array_ | ResolvedStageVersions contains the names of all resolved stageVersion resources specified by either stageVersionRef or StageVersionSelector |  |  |


#### TaskExecution



TaskExecution is the Schema for the taskexecutions API



_Appears in:_
- [TaskExecutionList](#taskexecutionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `TaskExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[TaskExecutionSpec](#taskexecutionspec)_ | spec defines the desired state of TaskExecution |  | Required: \{\} <br /> |
| `status` _[TaskExecutionStatus](#taskexecutionstatus)_ | status defines the observed state of TaskExecution |  | Optional: \{\} <br /> |


#### TaskExecutionList



TaskExecutionList contains a list of TaskExecution





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `TaskExecutionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[TaskExecution](#taskexecution) array_ |  |  |  |


#### TaskExecutionSpec



TaskExecutionSpec defines the desired state of TaskExecution



_Appears in:_
- [TaskExecution](#taskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `dependsOn` _string array_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |


#### TaskExecutionStatus



TaskExecutionStatus defines the observed state of TaskExecution.



_Appears in:_
- [TaskExecution](#taskexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### TaskManifest



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


#### VectorActivation



VectorActivation is the Schema for the vectoractivations API



_Appears in:_
- [VectorActivationList](#vectoractivationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorActivation` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorActivationSpec](#vectoractivationspec)_ | spec defines the desired state of VectorActivation |  | Required: \{\} <br /> |
| `status` _[VectorActivationStatus](#vectoractivationstatus)_ | status defines the observed state of VectorActivation |  | Optional: \{\} <br /> |


#### VectorActivationList



VectorActivationList contains a list of VectorActivation





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorActivationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorActivation](#vectoractivation) array_ |  |  |  |


#### VectorActivationSpec



VectorActivationSpec defines the desired state of VectorActivation



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stage` _string_ |  |  |  |
| `stageVersion` _string_ |  |  |  |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |
| `vectorDeployment` _string_ |  |  |  |


#### VectorActivationStatus



VectorActivationStatus defines the observed state of VectorActivation.



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorAssignment



VectorAssignment is the Schema for the vectorassignments API.

A VectorAssignment represents a single binding between a VectorDeployment and an ArtifactDeployment. It enables
an n:m mapping where a single artifact may be reused across multiple vectors. These objects are automatically
managed by the vector-deployment-controller and reconciled by deployers to apply vector-specific configuration.



_Appears in:_
- [VectorAssignmentList](#vectorassignmentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorAssignment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorAssignmentSpec](#vectorassignmentspec)_ | Spec defines the desired state of the VectorAssignment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[VectorAssignmentStatus](#vectorassignmentstatus)_ |  |  |  |


#### VectorAssignmentList



VectorAssignmentList contains a list of VectorAssignment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorAssignmentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorAssignment](#vectorassignment) array_ |  |  |  |


#### VectorAssignmentSpec



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


#### VectorAssignmentStatus



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


#### VectorDeployment



VectorDeployment is the Schema for the vectordeployments API.

VectorDeployment represents the deployment of an immutable vector of artifacts into a specific environment or stage.



_Appears in:_
- [VectorDeploymentList](#vectordeploymentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorDeploymentSpec](#vectordeploymentspec)_ | Spec defines the desired state of the VectorDeployment and is immutable after it has been set |  | Optional: \{\} <br /> |
| `status` _[VectorDeploymentStatus](#vectordeploymentstatus)_ |  |  |  |


#### VectorDeploymentList



VectorDeploymentList contains a list of VectorDeployment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeploymentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorDeployment](#vectordeployment) array_ |  |  |  |


#### VectorDeploymentSpec



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


#### VectorDeploymentStatus



VectorDeploymentStatus represents the observed state of a VectorDeployment as it progresses through the
deployment lifecycle.

The lifecycle consists of:
 1. Pulling the vector from the OCI registry and parsing its contents -> VectorDownloadedCondition
 2. Creating (or re-using) one ArtifactDeployment per artifact in the vector -> ArtifactDeploymentsCreatedCondition
 3. Waiting until all ArtifactDeployments have successfully deployed -> VectorDeployedCondition
 4. Creating all VectorAssignment resources associated with this vector -> VectorAssignmentsCreatedCondition
 5. Marking the vector as ready for use -> VectorReadyCondition



_Appears in:_
- [VectorDeployment](#vectordeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions represents the current set of status conditions for this vector<br />deployment. These conditions track progress through the lifecycle stages. |  |  |
| `resolvedVectorOcm` _string_ | ResolvedVectorOcm contains the fully materialized content of the OCM ComponentVersion after it has been<br />downloaded and resolved from the OCI registry. Unlike the Spec.Vector value, which is only a reference (URL),<br />this field stores the actual resolved vector content as provided by OCM, including all artifacts and metadata.<br />It is not a reference but the inlined representation of the component version at reconciliation time. |  |  |
| `resultingArtifactDeployments` _object (keys:string, values:[LocalArtifactDeploymentReference](#localartifactdeploymentreference))_ | ResultingArtifactDeployments lists the ArtifactDeployment resources created (or re-used) for this vector. The<br />map key is the component name of the artifact as defined inside the vector. Keys remain stable across<br />reconciliations and re-creations. |  |  |
| `resultingVectorAssignments` _object (keys:string, values:[LocalVectorAssignmentReference](#localvectorassignmentreference))_ | ResultingVectorAssignments lists all VectorAssignment resources created for this vector. VectorAssignments are<br />not re-used like ArtifactDeployments, but instead each VectorDeployment results in a complete new set of<br />assignments.<br />The map key is the component name of the artifact. Keys are stable across reconcilations. |  |  |
| `deploymentResults` _object (keys:string, values:[DeploymentResult](#deploymentresult))_ | DeploymentResults exposes an aggregated view of the deployment results produced<br />by all underlying ArtifactDeployments. The map key is composed of the component<br />name and the individual result name, ensuring uniqueness. |  |  |


#### VectorMigration



VectorMigration is the Schema for the vectormigrations API



_Appears in:_
- [VectorMigrationList](#vectormigrationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorMigration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorMigrationSpec](#vectormigrationspec)_ | spec defines the desired state of VectorMigration |  | Required: \{\} <br /> |
| `status` _[VectorMigrationStatus](#vectormigrationstatus)_ | status defines the observed state of VectorMigration |  | Optional: \{\} <br /> |


#### VectorMigrationList



VectorMigrationList contains a list of VectorMigration





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorMigrationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorMigration](#vectormigration) array_ |  |  |  |


#### VectorMigrationSpec



VectorMigrationSpec defines the desired state of VectorMigration



_Appears in:_
- [VectorMigration](#vectormigration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stageVersion` _string_ |  |  |  |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


#### VectorMigrationStatus



VectorMigrationStatus defines the observed state of VectorMigration.



_Appears in:_
- [VectorMigration](#vectormigration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


