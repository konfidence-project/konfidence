package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionConfigKind is kind of the VectorPromotionConfig resource.
	VectorPromotionConfigKind = "VectorPromotionConfig"
)

// PromotionSourceReference identifies the in-cluster resource whose current
// vector is promoted from.
type PromotionSourceReference struct {
	// Kind of the source resource. A `VectorTemplate` source promotes its
	// latest assembled vector automatically; a `Stage` source promotes the
	// vector currently active on that stage and requires approval.
	// +kubebuilder:validation:Enum=VectorTemplate;Stage
	Kind string `json:"kind"`

	// Name of the source resource.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// APIGroup of the source resource.
	// +kubebuilder:default="konfidence.cloud"
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Optional
	APIGroup string `json:"apiGroup,omitempty"`

	// Namespace of the source resource. Defaults to the namespace of the
	// VectorPromotionConfig. Cross-namespace references are accepted by the
	// schema but not yet acted on by controllers.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// PromotionTargetReference identifies the Stage whose `spec.vector` is the
// promotion target.
type PromotionTargetReference struct {
	// Kind of the target resource.
	// +kubebuilder:validation:Enum=Stage
	Kind string `json:"kind"`

	// Name of the target resource.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// APIGroup of the target resource.
	// +kubebuilder:default="konfidence.cloud"
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Optional
	APIGroup string `json:"apiGroup,omitempty"`

	// Namespace of the target resource. Defaults to the namespace of the
	// VectorPromotionConfig. Cross-namespace references are accepted by the
	// schema but not yet acted on by controllers.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.
type VectorPromotionConfigSpec struct {
	// Source references the resource to promote from.
	Source PromotionSourceReference `json:"source"`

	// Target references the Stage to promote to.
	Target PromotionTargetReference `json:"target"`

	// TTLAfterFinished is copied onto every VectorPromotion created for this
	// config. See `VectorPromotionSpec.TTLAfterFinished`.
	// +kubebuilder:validation:Optional
	TTLAfterFinished *metav1.Duration `json:"ttlAfterFinished,omitempty"`

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
