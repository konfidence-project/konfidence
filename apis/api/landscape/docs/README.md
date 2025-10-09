# API Reference

## Packages
- [landscape.konfidence.cloud/v1alpha1](#landscapekonfidencecloudv1alpha1)


## landscape.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the landscape v1alpha1 API group.

### Resource Types
- [ActivationExecution](#activationexecution)
- [ActivationExecutionList](#activationexecutionlist)
- [ArtifactDeployment](#artifactdeployment)
- [ArtifactDeploymentList](#artifactdeploymentlist)
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
- [VectorDeploymentUsage](#vectordeploymentusage)
- [VectorDeploymentUsageList](#vectordeploymentusagelist)
- [VectorMigration](#vectormigration)
- [VectorMigrationList](#vectormigrationlist)



#### ActivationExecution



ActivationExecution is the Schema for the activationexecutions API



_Appears in:_
- [ActivationExecutionList](#activationexecutionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ActivationExecutionSpec](#activationexecutionspec)_ | spec defines the desired state of ActivationExecution |  |  |
| `status` _[ActivationExecutionStatus](#activationexecutionstatus)_ | status defines the observed state of ActivationExecution |  |  |


#### ActivationExecutionList



ActivationExecutionList contains a list of ActivationExecution





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ActivationExecutionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ActivationExecution](#activationexecution) array_ |  |  |  |


#### ActivationExecutionSpec



ActivationExecutionSpec defines the desired state of ActivationExecution



_Appears in:_
- [ActivationExecution](#activationexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |


#### ActivationExecutionStatus



ActivationExecutionStatus defines the observed state of ActivationExecution.



_Appears in:_
- [ActivationExecution](#activationexecution)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### ArtifactDeployment



ArtifactDeployment is the Schema for the artifactdeployments API.



_Appears in:_
- [ArtifactDeploymentList](#artifactdeploymentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ArtifactDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[ArtifactDeploymentSpec](#artifactdeploymentspec)_ |  |  |  |
| `status` _[ArtifactDeploymentStatus](#artifactdeploymentstatus)_ |  |  |  |


#### ArtifactDeploymentList



ArtifactDeploymentList contains a list of ArtifactDeployment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `ArtifactDeploymentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[ArtifactDeployment](#artifactdeployment) array_ |  |  |  |


#### ArtifactDeploymentSpec



ArtifactDeploymentSpec defines the desired state of ArtifactDeployment.



_Appears in:_
- [ArtifactDeployment](#artifactdeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `manifest` _[ArtifactManifest](#artifactmanifest)_ |  |  |  |
| `taskManifests` _[TaskManifest](#taskmanifest) array_ |  |  |  |
| `component` _[OCMComponent](#ocmcomponent)_ |  |  |  |


#### ArtifactDeploymentStatus



ArtifactDeploymentStatus defines the observed state of ArtifactDeployment.



_Appears in:_
- [ArtifactDeployment](#artifactdeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `observedGeneration` _integer_ | ObservedGeneration is the last observed generation. |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | Conditions holds the conditions for the ArtifactDeployment. |  |  |
| `deploymentResult` _[DeploymentResult](#deploymentresult)_ |  |  |  |


#### ArtifactManifest



ArtifactManifest defines the manifest for an artifact.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  |  |
| `allowReuse` _boolean_ |  |  |  |


#### DeploymentResult



DeploymentResult contains the result of an artifact deployment.



_Appears in:_
- [ArtifactDeploymentStatus](#artifactdeploymentstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `url` _string_ |  |  |  |


#### OCMComponent



OCMComponent is a wrapper around the OCM ComponentVersion.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `version` _string_ |  |  |  |
| `resources` _[OCMResource](#ocmresource) array_ |  |  |  |


#### OCMResource



OCMResource represent a single resource in an OCM ComponentVersion.



_Appears in:_
- [OCMComponent](#ocmcomponent)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `image` _string_ |  |  |  |
| `version` _string_ |  |  |  |
| `type` _string_ |  |  |  |


#### StageVersion



StageVersion is the Schema for the stageversions API



_Appears in:_
- [StageVersionList](#stageversionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageVersionSpec](#stageversionspec)_ | spec defines the desired state of StageVersion |  |  |
| `status` _[StageVersionStatus](#stageversionstatus)_ | status defines the observed state of StageVersion |  |  |


#### StageVersionList



StageVersionList contains a list of StageVersion





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageVersion](#stageversion) array_ |  |  |  |


#### StageVersionRef



StageVersionRef references a stageVersion



_Appears in:_
- [StageVersionUsageSpec](#stageversionusagespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the stageVersion |  |  |


#### StageVersionSpec



StageVersionSpec defines the desired state of StageVersion



_Appears in:_
- [StageVersion](#stageversion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  | MinLength: 1 <br /> |
| `stage_generation` _integer_ | the object generation of the stage that created this stage version |  | Minimum: 1 <br /> |


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
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageVersionUsageSpec](#stageversionusagespec)_ | spec defines the desired state of StageVersionUsage |  |  |
| `status` _[StageVersionUsageStatus](#stageversionusagestatus)_ | status defines the observed state of StageVersionUsage |  |  |


#### StageVersionUsageList



StageVersionUsageList contains a list of StageVersionUsage





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageVersionUsageList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageVersionUsage](#stageversionusage) array_ |  |  |  |


#### StageVersionUsageSpec



StageVersionUsageSpec defines the desired state of StageVersionUsage



_Appears in:_
- [StageVersionUsage](#stageversionusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `reason` _string_ | Reason is human-readable description of why this StageVersion is in use, e.g. "executing vector migrations", "latest vector for stage xyz", |  |  |
| `stageVersionRef` _[StageVersionRef](#stageversionref)_ | StageVersionRef references a stageVersion |  |  |


#### StageVersionUsageStatus



StageVersionUsageStatus defines the observed state of StageVersionUsage.



_Appears in:_
- [StageVersionUsage](#stageversionusage)



#### TaskExecution



TaskExecution is the Schema for the taskexecutions API



_Appears in:_
- [TaskExecutionList](#taskexecutionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `TaskExecution` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[TaskExecutionSpec](#taskexecutionspec)_ | spec defines the desired state of TaskExecution |  |  |
| `status` _[TaskExecutionStatus](#taskexecutionstatus)_ | status defines the observed state of TaskExecution |  |  |


#### TaskExecutionList



TaskExecutionList contains a list of TaskExecution





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `TaskExecutionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
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



TaskManifest defines the manifest for a task.



_Appears in:_
- [ArtifactDeploymentSpec](#artifactdeploymentspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `type` _string_ |  |  |  |
| `dependsOn` _string array_ |  |  |  |
| `spec` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |


#### VectorActivation



VectorActivation is the Schema for the vectoractivations API



_Appears in:_
- [VectorActivationList](#vectoractivationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorActivation` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorActivationSpec](#vectoractivationspec)_ | spec defines the desired state of VectorActivation |  |  |
| `status` _[VectorActivationStatus](#vectoractivationstatus)_ | status defines the observed state of VectorActivation |  |  |


#### VectorActivationList



VectorActivationList contains a list of VectorActivation





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorActivationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorActivation](#vectoractivation) array_ |  |  |  |


#### VectorActivationSpec



VectorActivationSpec defines the desired state of VectorActivation



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stageVersion` _string_ |  |  |  |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


#### VectorActivationStatus



VectorActivationStatus defines the observed state of VectorActivation.



_Appears in:_
- [VectorActivation](#vectoractivation)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorAssignment



VectorAssignment is the Schema for the vectorassignments API.



_Appears in:_
- [VectorAssignmentList](#vectorassignmentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorAssignment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorAssignmentSpec](#vectorassignmentspec)_ |  |  |  |
| `status` _[VectorAssignmentStatus](#vectorassignmentstatus)_ |  |  |  |


#### VectorAssignmentList



VectorAssignmentList contains a list of VectorAssignment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorAssignmentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorAssignment](#vectorassignment) array_ |  |  |  |


#### VectorAssignmentSpec



VectorAssignmentSpec defines the desired state of VectorAssignment.



_Appears in:_
- [VectorAssignment](#vectorassignment)



#### VectorAssignmentStatus



VectorAssignmentStatus defines the observed state of VectorAssignment.



_Appears in:_
- [VectorAssignment](#vectorassignment)



#### VectorDeployment



VectorDeployment is the Schema for the vectordeployments API.



_Appears in:_
- [VectorDeploymentList](#vectordeploymentlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeployment` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorDeploymentSpec](#vectordeploymentspec)_ |  |  |  |
| `status` _[VectorDeploymentStatus](#vectordeploymentstatus)_ |  |  |  |


#### VectorDeploymentList



VectorDeploymentList contains a list of VectorDeployment.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeploymentList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorDeployment](#vectordeployment) array_ |  |  |  |


#### VectorDeploymentSpec



VectorDeploymentSpec defines the desired state of VectorDeployment.



_Appears in:_
- [VectorDeployment](#vectordeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


#### VectorDeploymentStatus



VectorDeploymentStatus defines the observed state of VectorDeployment.



_Appears in:_
- [VectorDeployment](#vectordeployment)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `resolvedVectorOcm` _string_ |  |  |  |
| `resultingArtifactDeployments` _object (keys:string, values:[TypedObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#typedobjectreference-v1-core))_ |  |  |  |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorDeploymentUsage



VectorDeploymentUsage is the Schema for the vectordeploymentusages API.



_Appears in:_
- [VectorDeploymentUsageList](#vectordeploymentusagelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeploymentUsage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorDeploymentUsageSpec](#vectordeploymentusagespec)_ |  |  |  |
| `status` _[VectorDeploymentUsageStatus](#vectordeploymentusagestatus)_ |  |  |  |


#### VectorDeploymentUsageList



VectorDeploymentUsageList contains a list of VectorDeploymentUsage.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorDeploymentUsageList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorDeploymentUsage](#vectordeploymentusage) array_ |  |  |  |


#### VectorDeploymentUsageSpec



VectorDeploymentUsageSpec defines the desired state of VectorDeploymentUsage.



_Appears in:_
- [VectorDeploymentUsage](#vectordeploymentusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vector` _string_ | Vector points to the OCM component version that contains the deployment vector for this stage. |  |  |


#### VectorDeploymentUsageStatus



VectorDeploymentUsageStatus defines the observed state of VectorDeploymentUsage.



_Appears in:_
- [VectorDeploymentUsage](#vectordeploymentusage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `vectorDeployment` _[TypedLocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#typedlocalobjectreference-v1-core)_ |  |  |  |


#### VectorMigration



VectorMigration is the Schema for the vectormigrations API



_Appears in:_
- [VectorMigrationList](#vectormigrationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorMigration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorMigrationSpec](#vectormigrationspec)_ | spec defines the desired state of VectorMigration |  |  |
| `status` _[VectorMigrationStatus](#vectormigrationstatus)_ | status defines the observed state of VectorMigration |  |  |


#### VectorMigrationList



VectorMigrationList contains a list of VectorMigration





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `landscape.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorMigrationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
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


