# API Reference

## Packages
- [common.konfidence.cloud/v1alpha1](#commonkonfidencecloudv1alpha1)


## common.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the common v1alpha1 API group.

### Resource Types
- [Stage](#stage)
- [StageList](#stagelist)



#### Stage



Stage is the Schema for the stages API.



_Appears in:_
- [StageList](#stagelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `common.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `Stage` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageSpec](#stagespec)_ |  |  |  |
| `status` _[StageStatus](#stagestatus)_ |  |  |  |


#### StageList



StageList contains a list of Stage.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `common.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Stage](#stage) array_ |  |  |  |


#### StageSpec



StageSpec defines the desired state of Stage.



_Appears in:_
- [Stage](#stage)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the name of the stage. |  | MinLength: 1 <br /> |
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


