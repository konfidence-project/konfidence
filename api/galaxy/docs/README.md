# API Reference

## Packages
- [galaxy.konfidence.cloud/v1alpha1](#galaxykonfidencecloudv1alpha1)


## galaxy.konfidence.cloud/v1alpha1

Package v1alpha1 contains API Schema definitions for the galaxy v1alpha1 API group.

### Resource Types
- [StageConfiguration](#stageconfiguration)
- [StageConfigurationList](#stageconfigurationlist)
- [StageSync](#stagesync)
- [StageSyncList](#stagesynclist)
- [VectorPromotion](#vectorpromotion)
- [VectorPromotionConfig](#vectorpromotionconfig)
- [VectorPromotionConfigList](#vectorpromotionconfiglist)
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


#### CredentialRef



CredentialRef references a Secret in the same namespace as the holding resource.



_Appears in:_
- [OCMCredentials](#ocmcredentials)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br /> |


#### Credentials



Credentials holds credentials for various purposes — for example OCM
repository access and signing/verification key material.



_Appears in:_
- [StageConfigurationSpec](#stageconfigurationspec)
- [VectorPromotionConfigSpec](#vectorpromotionconfigspec)
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ocm` _[OCMCredentials](#ocmcredentials)_ |  |  | Optional: \{\} <br /> |


#### OCMCredentials



OCMCredentials lists Secrets holding `.ocmconfig` or `.dockerconfigjson` data.
All references are same-namespace.



_Appears in:_
- [Credentials](#credentials)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `refs` _[CredentialRef](#credentialref) array_ |  |  | MinItems: 1 <br /> |


#### Sign



Sign lists signatures the controller produces on every descriptor it
writes. Absence on a spec disables signing.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `signatures` _[Signature](#signature) array_ |  |  | MinItems: 1 <br /> |


#### Signature



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


#### StageConfiguration



StageConfiguration is the Schema for the stageConfigurations API.



_Appears in:_
- [StageConfigurationList](#stageconfigurationlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
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
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
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
| `name` _string_ | Name is the stage name. |  |  |
| `vector` _string_ | Vector points to the OCM component that contains the deployment vector for this stage. |  |  |
| `targetNamespace` _string_ | TargetNamespace is the target namespace where the associated stage is created or updated |  |  |
| `targetWorkspace` _string_ | TargetWorkspace is the target workspace where the associated stage is created or updated |  | Optional: \{\} <br />Optional: \{\} <br /> |
| `credentials` _[Credentials](#credentials)_ | Credentials supplies credentials for OCM repository access<br />and vector verification key material. |  | Optional: \{\} <br /> |
| `verifyVector` _[Verify](#verify)_ | VerifyVector lists candidate signatures evaluated against the<br />fetched vector descriptor. Absence disables vector verification. |  | Optional: \{\} <br /> |


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
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
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
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `StageSyncList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[StageSync](#stagesync) array_ |  |  |  |


#### StageSyncSpec



StageSyncSpec defines the desired state of a StageSync.



_Appears in:_
- [StageSync](#stagesync)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `stageTemplate` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | StageTemplate contains the template of the stage to be created on the Star cluster. |  |  |
| `reconcileInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | ReconcileInterval defines how often the sync controller should reconcile the StageSync resource.<br />If not set, the controller's default reconcile interval will be used. |  | Optional: \{\} <br /> |


#### StageSyncStatus



StageSyncStatus defines the observed state of a StageSync.



_Appears in:_
- [StageSync](#stagesync)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |
| `stageStatus` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ |  |  |  |


#### VectorConfig



VectorConfig defines feature flags and authored configuration values for a vector.



_Appears in:_
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `features` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Features define the feature flags. |  |  |
| `authored` _[RawExtension](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#rawextension-runtime-pkg)_ | Authored define the authored configuration values. |  |  |


#### VectorPromotion



VectorPromotion triggers a one-time execution of a promotion flow defined by a VectorPromotionConfig.



_Appears in:_
- [VectorPromotionList](#vectorpromotionlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotion` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorPromotionSpec](#vectorpromotionspec)_ |  |  |  |
| `status` _[VectorPromotionStatus](#vectorpromotionstatus)_ |  |  |  |


#### VectorPromotionConfig



VectorPromotionConfig describes a promotion flow for a vector between a source and a target.



_Appears in:_
- [VectorPromotionConfigList](#vectorpromotionconfiglist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotionConfig` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[VectorPromotionConfigSpec](#vectorpromotionconfigspec)_ | Spec defines the desired state of the VectorPromotionConfig. |  | Optional: \{\} <br /> |
| `status` _[VectorPromotionConfigStatus](#vectorpromotionconfigstatus)_ |  |  |  |


#### VectorPromotionConfigList



VectorPromotionConfigList contains a list of VectorPromotionConfig.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotionConfigList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorPromotionConfig](#vectorpromotionconfig) array_ |  |  |  |


#### VectorPromotionConfigSpec



VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.



_Appears in:_
- [VectorPromotionConfig](#vectorpromotionconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `source` _string_ | Source is the OCM component reference to promote from.<br />This usually points to a version alias (e.g. :latest) that resolves to the component version to be promoted.<br />The format is <registry>//<component-name>:<version>. |  | MinLength: 1 <br />Pattern: `^[^/].+//.+:.+$` <br /> |
| `target` _string_ | Target is the OCM component reference to promote to.<br />This usually points to a version alias (e.g. :promoted). The actual version string is taken from the source component version.<br />The format is <registry>//<component-name>:<version>. |  | MinLength: 1 <br />Pattern: `^[^/].+//.+:.+$` <br /> |
| `credentials` _[Credentials](#credentials)_ | Credentials supplies credentials for OCM repository access and vector verification key material. |  | Optional: \{\} <br /> |
| `verifyVector` _[Verify](#verify)_ | VerifyVector lists candidate signatures evaluated against the<br />source vector before promotion proceeds. Absence disables vector<br />verification. |  | Optional: \{\} <br /> |


#### VectorPromotionConfigStatus



VectorPromotionConfigStatus defines the observed state of VectorPromotionConfig.



_Appears in:_
- [VectorPromotionConfig](#vectorpromotionconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastPromotionConditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | LastPromotionConditions contains the result of the most recent VectorPromotion execution |  |  |
| `lastSuccessfulPromotionConditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ | LastSuccessfulPromotionConditions contains the result of the most recent VectorPromotion execution, that was successful |  |  |


#### VectorPromotionList



VectorPromotionList contains a list of VectorPromotion.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
| `kind` _string_ | `VectorPromotionList` | | |
| `kind` _string_ | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  | Optional: \{\} <br /> |
| `apiVersion` _string_ | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources |  | Optional: \{\} <br /> |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[VectorPromotion](#vectorpromotion) array_ |  |  |  |


#### VectorPromotionSpec



VectorPromotionSpec defines the desired state of VectorPromotion.



_Appears in:_
- [VectorPromotion](#vectorpromotion)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `vectorPromotionConfigRef` _string_ | VectorPromotionConfigRef is the name of the VectorPromotionConfig that defines the promotion flow to execute. |  | MinLength: 1 <br /> |
| `ttlAfterFinished` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#duration-v1-meta)_ | TTLAfterFinished defines how long the VectorPromotion should be kept after completion.<br />Once the TTL expires after the promotion reaches a terminal state (Completed or Failed),<br />the resource is eligible for automatic deletion. If no TTL is set, no deletion happens. |  | Optional: \{\} <br /> |


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
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
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
| `apiVersion` _string_ | `galaxy.konfidence.cloud/v1alpha1` | | |
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
| `credentials` _[Credentials](#credentials)_ | Credentials supplies credentials for OCM repositories<br />and signing/verification key material. |  | Optional: \{\} <br /> |
| `verifyArtifacts` _[Verify](#verify)_ | VerifyArtifacts lists candidate signatures evaluated against every<br />artifact pulled into the assembly. Absence disables artifact<br />verification. |  | Optional: \{\} <br /> |
| `verifyVector` _[Verify](#verify)_ | VerifyVector lists candidate signatures evaluated against any<br />vector the assembly fetches (base or pre-existing upload target).<br />Absence disables vector verification. |  | Optional: \{\} <br /> |
| `signVector` _[Sign](#sign)_ | SignVector lists signatures the controller produces on the emitted<br />vector. Absence disables signing. |  | Optional: \{\} <br /> |
| `vectorConfig` _[VectorConfig](#vectorconfig)_ |  |  | Optional: \{\} <br /> |


#### VectorTemplateStatus



VectorTemplateStatus defines the observed state of VectorTemplate.



_Appears in:_
- [VectorTemplate](#vectortemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.32/#condition-v1-meta) array_ |  |  |  |


#### Verify



Verify lists candidate signatures evaluated against every fetched
descriptor. Absence on a spec disables verification.



_Appears in:_
- [StageConfigurationSpec](#stageconfigurationspec)
- [VectorPromotionConfigSpec](#vectorpromotionconfigspec)
- [VectorTemplateSpec](#vectortemplatespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `signatures` _[Signature](#signature) array_ |  |  | MinItems: 1 <br /> |


