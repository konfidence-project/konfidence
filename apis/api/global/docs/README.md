# API Reference

## Packages
- [global.konfidence.cloud/v1alpha1](#globalkonfidencecloudv1alpha1)


## global.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the global v1alpha1 API group.

### Resource Types
- [StageConfiguration](#stageconfiguration)
- [StageConfigurationList](#stageconfigurationlist)
- [StageSync](#stagesync)
- [StageSyncList](#stagesynclist)
- [VectorPromotion](#vectorpromotion)
- [VectorPromotionList](#vectorpromotionlist)
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
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageConfigurationSpec](#stageconfigurationspec)_ |  |  |  |
| `status` _[StageConfigurationStatus](#stageconfigurationstatus)_ |  |  |  |


#### StageConfigurationList



StageConfigurationList contains a list of StageConfiguration.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageConfigurationList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
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
| `targetNamespace` _string_ | TargetNamespace is the target namespace where the associated stage is created or updated |  |  |
| `targetWorkspace` _string_ | TargetWorkspace is the target workspace where the associated stage is created or updated |  | Optional: \{\} <br />Optional: \{\} <br /> |


#### StageConfigurationStatus



StageConfigurationStatus defines the observed state of StageConfiguration.



_Appears in:_
- [StageConfiguration](#stageconfiguration)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### StageSync



StageSync is the Schema for the stageSyncs API.



_Appears in:_
- [StageSyncList](#stagesynclist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageSync` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[StageSyncSpec](#stagesyncspec)_ |  |  |  |
| `status` _[StageSyncStatus](#stagesyncstatus)_ |  |  |  |


#### StageSyncList



StageSyncList contains a list of StageSync.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageSyncList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageSync](#stagesync) array_ |  |  |  |


#### StageSyncSpec



StageSyncSpec defines the desired state of StageSync.



_Appears in:_
- [StageSync](#stagesync)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stageTemplate` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | StageTemplate contains the template of the stage to be created on the LCP cluster. |  |  |
| `reconcileInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | ReconcileInterval defines how often the sync controller should reconcile the StageSync resource.<br />If not set, the controller's default reconcile interval will be used. |  | Optional: \{\} <br /> |


#### StageSyncStatus



StageSyncStatus defines the observed state of StageSync.



_Appears in:_
- [StageSync](#stagesync)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorPromotion



VectorPromotion is the Schema for the vectorPromotions API.



_Appears in:_
- [VectorPromotionList](#vectorpromotionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorPromotionSpec](#vectorpromotionspec)_ |  |  |  |
| `status` _[VectorPromotionStatus](#vectorpromotionstatus)_ |  |  |  |


#### VectorPromotionList



VectorPromotionList contains a list of VectorPromotion.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorPromotion](#vectorpromotion) array_ |  |  |  |


#### VectorPromotionSpec



VectorPromotionSpec defines the desired state of VectorPromotion.



_Appears in:_
- [VectorPromotion](#vectorpromotion)



#### VectorPromotionStatus



VectorPromotionStatus defines the observed state of VectorPromotion.



_Appears in:_
- [VectorPromotion](#vectorpromotion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### VectorTemplate



VectorTemplate represents a template for assembling OCM components into an OCM component
that represents a vector.



_Appears in:_
- [VectorTemplateList](#vectortemplatelist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorTemplate` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  | Optional: \{\} <br /> |
| `spec` _[VectorTemplateSpec](#vectortemplatespec)_ | spec defines the desired state of VectorTemplate |  | Required: \{\} <br /> |
| `status` _[VectorTemplateStatus](#vectortemplatestatus)_ | status defines the observed state of VectorTemplate |  | Optional: \{\} <br /> |


#### VectorTemplateList



VectorTemplateList contains a list of VectorTemplate





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `global.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorTemplateList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
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
| `base` _string_ | Base represents an optional base component version to build upon. |  | Optional: \{\} <br />Optional: \{\} <br /> |
| `components` _[Component](#component) array_ | Components lists the components to be included in the vector. |  | MinItems: 1 <br /> |


#### VectorTemplateStatus



VectorTemplateStatus defines the observed state of VectorTemplate.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


