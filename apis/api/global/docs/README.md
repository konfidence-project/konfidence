# API Reference

## Packages
- [global.konfidence.cloud/v1alpha1](#globalkonfidencecloudv1alpha1)


## global.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the global v1alpha1 API group.

### Resource Types
- [StageConfiguration](#stageconfiguration)
- [StageConfigurationList](#stageconfigurationlist)
- [VectorTemplate](#vectortemplate)
- [VectorTemplateList](#vectortemplatelist)



#### Component



Component defines a component of a VectorTemplate.
A struct is used for future expansion.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |


#### StageConfiguration



StageConfiguration is the Schema for the stageConfigurations API.



_Appears in:_
- [StageConfigurationList](#stageconfigurationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageConfiguration` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageConfigurationSpec](#stageconfigurationspec)_ |  |  |  |
| `status` _[StageConfigurationStatus](#stageconfigurationstatus)_ |  |  |  |


#### StageConfigurationList



StageConfigurationList contains a list of StageConfiguration.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageConfigurationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageConfiguration](#stageconfiguration) array_ |  |  |  |


#### StageConfigurationSpec



StageConfigurationSpec defines the desired state of StageConfiguration.



_Appears in:_
- [StageConfiguration](#stageconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the stage name |  |  |
| `vector` _string_ | Vector points to the OCM component that contains the deployment vector for this stage. |  |  |


#### StageConfigurationStatus



StageConfigurationStatus defines the observed state of StageConfiguration.



_Appears in:_
- [StageConfiguration](#stageconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorTemplate



VectorTemplate is the Schema for the vectortemplates API



_Appears in:_
- [VectorTemplateList](#vectortemplatelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorTemplate` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorTemplateSpec](#vectortemplatespec)_ | spec defines the desired state of VectorTemplate |  |  |
| `status` _[VectorTemplateStatus](#vectortemplatestatus)_ | status defines the observed state of VectorTemplate |  |  |


#### VectorTemplateList



VectorTemplateList contains a list of VectorTemplate





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorTemplateList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |  |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorTemplate](#vectortemplate) array_ |  |  |  |


#### VectorTemplateSpec



VectorTemplateSpec defines the desired state of VectorTemplate.
VectorTemplateSpec defines the components of which a vector is composed.
From a VectorTemplate an OCM component is created which contains the latest version of all listed components.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `reconcileInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | ReconcileInterval defines how often the assembly controller should check for drift.<br />If not set, the controller's default reconcile interval will be used. |  | Optional: \{\} <br /> |
| `uploadTarget` _string_ | UploadTarget defines the target OCM component where the assembled vector will be uploaded. |  |  |
| `base` _string_ | Base represents an optional base component version to build upon. |  | Optional: \{\} <br /> |
| `components` _[Component](#component) array_ | Components lists the components to be included in the vector. |  | MinItems: 1 <br /> |


#### VectorTemplateStatus



VectorTemplateStatus defines the observed state of VectorTemplate.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


