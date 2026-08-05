package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorTemplateKind is kind of the VectorTemplate resource.
	VectorTemplateKind = "VectorTemplate"

	// VectorTemplateReadyCondition is the ready condition for the VectorTemplate resource.
	VectorTemplateReadyCondition = "Ready"

	// VectorTemplateVectorCreatedReason indicates that a new vector was created.
	VectorTemplateVectorCreatedReason = "VectorCreated"
	// VectorTemplateNoDriftDetectedReason indicates that no drift was detected.
	VectorTemplateNoDriftDetectedReason = "NoDriftDetected"
	// VectorTemplateVectorCreationFailedReason indicates that vector creation failed.
	VectorTemplateVectorCreationFailedReason = "VectorCreationFailed"
	// VectorTemplateDriftDetectionFailedReason indicates that drift detection failed.
	VectorTemplateDriftDetectionFailedReason = "DriftDetectionFailed"
	// VectorTemplateWaitingForBaseReason indicates assembly is waiting for the base
	// VectorTemplate to assemble a vector (its status.latestVector is not set yet).
	VectorTemplateWaitingForBaseReason = "WaitingForBase"
)

// VectorTemplateSpec defines the desired state of VectorTemplate.
// VectorTemplateSpec defines the components of which a vector is composed.
// From a VectorTemplate an OCM component is created which contains the latest version of all listed components.
type VectorTemplateSpec struct {
	// ReconcileInterval defines how often the assembly controller should check for drift.
	// If not set, the controller's default reconcile interval will be used.
	// +kubebuilder:validation:Optional
	ReconcileInterval *metav1.Duration `json:"reconcileInterval,omitempty"`

	// UploadTarget defines the target OCM component where the assembled vector will be uploaded.
	UploadTarget string `json:"uploadTarget"`

	// Base references another VectorTemplate whose most recently assembled vector
	// (status.latestVector) is used as the base for this vector's assembly.
	// +kubebuilder:validation:Optional
	// +optional
	Base *VectorTemplateReference `json:"base,omitempty"`

	// Components lists the components to be included in the vector.
	// +kubebuilder:validation:MinItems=1
	Components []Component `json:"components"`

	// Credentials supplies credentials for OCM repositories
	// and signing/verification key material.
	// +optional
	Credentials *Credentials `json:"credentials,omitempty"`

	// VerifyArtifacts lists candidate signatures evaluated against every
	// artifact pulled into the assembly. Absence disables artifact
	// verification.
	// +optional
	VerifyArtifacts *Verify `json:"verifyArtifacts,omitempty"`

	// VerifyVector lists candidate signatures evaluated against any
	// vector the assembly fetches (base or pre-existing upload target).
	// Absence disables vector verification.
	// +optional
	VerifyVector *Verify `json:"verifyVector,omitempty"`

	// SignVector lists signatures the controller produces on the emitted
	// vector. Absence disables signing.
	// +optional
	SignVector *Sign `json:"signVector,omitempty"`

	// +kubebuilder:validation:Optional
	VectorConfig *VectorConfig `json:"vectorConfig,omitempty"`
}

// Component defines a component of a VectorTemplate.
// A struct is used for future expansion.
type Component struct {
	Name string `json:"name"`
}

// VectorTemplateStatus defines the observed state of VectorTemplate.
type VectorTemplateStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LatestVector is the concrete OCM component version of the most recently
	// assembled vector, in the form <repository>//<component>:<version>. It is
	// empty until the first successful assembly. 
	// +optional
	LatestVector string `json:"latestVector,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="Indicates if the vector template is ready"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason",description="The reason of the Ready condition"
// +kubebuilder:printcolumn:name="Upload-Target",type=string,JSONPath=".spec.uploadTarget",description="The upload target of the vector template"
// +kubebuilder:printcolumn:name="Latest-Vector",type=string,JSONPath=".status.latestVector",description="The most recently assembled concrete vector version"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Time since creation of the vector template"

// VectorTemplate represents a template for assembling OCM components into an OCM component
// that represents a vector.
type VectorTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of VectorTemplate
	// +required
	Spec VectorTemplateSpec `json:"spec"`

	// status defines the observed state of VectorTemplate
	// +optional
	Status VectorTemplateStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// VectorTemplateList contains a list of VectorTemplate
type VectorTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []VectorTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorTemplate{}, &VectorTemplateList{})
}
