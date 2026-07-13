package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionConfigKind is kind of the VectorPromotionConfig resource.
	VectorPromotionConfigKind = "VectorPromotionConfig"
)

// VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.
type VectorPromotionConfigSpec struct {
	// Source is the OCM component reference to promote from.
	// This usually points to a version alias (e.g. :latest) that resolves to the component version to be promoted.
	// The format is <registry>//<component-name>:<version>.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[^/].+//.+:.+$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="source is immutable after it has been set"
	Source string `json:"source"`

	// Target is the OCM component reference to promote to.
	// This usually points to a version alias (e.g. :promoted). The actual version string is taken from the source component version.
	// The format is <registry>//<component-name>:<version>.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[^/].+//.+:.+$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="target is immutable after it has been set"
	Target string `json:"target"`

	// Credentials supplies credentials for OCM repository access and vector verification key material.
	// +optional
	Credentials *Credentials `json:"credentials,omitempty"`

	// VerifyVector lists candidate signatures evaluated against the
	// source vector before promotion proceeds. Absence disables vector
	// verification.
	// +optional
	VerifyVector *Verify `json:"verifyVector,omitempty"`
}

// VectorPromotionConfigStatus defines the observed state of VectorPromotionConfig.
type VectorPromotionConfigStatus struct {
	// LastPromotionConditions contains the result of the most recent VectorPromotion execution
	LastPromotionConditions []metav1.Condition `json:"lastPromotionConditions,omitempty"`
	// LastSuccessfulPromotionConditions contains the result of the most recent VectorPromotion execution, that was successful
	LastSuccessfulPromotionConditions []metav1.Condition `json:"lastSuccessfulPromotionConditions,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Last-Succeeded",type=string,JSONPath=".status.lastPromotionConditions[0].status",description="Last promotion succeeded"
// +kubebuilder:printcolumn:name="Last-Condition-Reason",type=string,JSONPath=".status.lastPromotionConditions[0].reason",description="Last promotion condition reason"
// +kubebuilder:printcolumn:name="Last-Time",type=date,JSONPath=".status.lastPromotionConditions[0].lastTransitionTime",description="Time of the last promotion"

// VectorPromotionConfig describes a promotion flow for a vector between a source and a target.
type VectorPromotionConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the VectorPromotionConfig.
	// +kubebuilder:validation:Optional
	Spec   VectorPromotionConfigSpec   `json:"spec,omitempty"`
	Status VectorPromotionConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorPromotionConfigList contains a list of VectorPromotionConfig.
type VectorPromotionConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorPromotionConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorPromotionConfig{}, &VectorPromotionConfigList{})
}
