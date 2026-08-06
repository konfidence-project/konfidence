package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionConfigKind is kind of the VectorPromotionConfig resource.
	VectorPromotionConfigKind = "VectorPromotionConfig"

	// VectorPromotionConfigReadyCondition reports whether the config's references resolve.
	VectorPromotionConfigReadyCondition = "Ready"

	// VectorPromotionConfigTargetResolvedReason indicates the target Stage resolved.
	VectorPromotionConfigTargetResolvedReason = "TargetResolved"
	// VectorPromotionConfigLandscapeNotFoundReason indicates the referenced Landscape does not exist.
	VectorPromotionConfigLandscapeNotFoundReason = "LandscapeNotFound"
	// VectorPromotionConfigLandscapeNotReadyReason indicates the referenced Landscape has no managed namespace yet.
	VectorPromotionConfigLandscapeNotReadyReason = "LandscapeNotReady"
	// VectorPromotionConfigStageNotFoundReason indicates the referenced Stage does not exist in the landscape namespace.
	VectorPromotionConfigStageNotFoundReason = "StageNotFound"
	// VectorPromotionConfigSourceNotFoundReason indicates the referenced source resource does not exist.
	VectorPromotionConfigSourceNotFoundReason = "SourceNotFound"
)

// PromotionSourceReference identifies the in-cluster resource whose current
// vector is promoted from. The source is resolved by the config reconciler,
// which pins the concrete vector into `VectorPromotionSpec.Vector`; the
// execution controller never reads it.
//
// +kubebuilder:validation:XValidation:rule="(self.kind == 'Stage') == has(self.landscape)",message="landscape is required for Stage references and must be omitted for VectorTemplate references"
//
//nolint:lll // Kubebuilder annotations are intentionally long.
type PromotionSourceReference struct {
	// Kind is the kind of the source resource. A `VectorTemplate` source
	// promotes its latest assembled vector automatically; a `Stage` source
	// promotes the vector currently active on that stage and requires approval.
	// +kubebuilder:validation:Enum=VectorTemplate;Stage
	Kind string `json:"kind"`

	// Name is the name of the source resource.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Landscape is the name of the `Landscape` in the config's namespace whose
	// namespace hosts the referenced `Stage`. Required for `Stage` references;
	// must be omitted for `VectorTemplate` references, which are resolved in
	// the config's namespace.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +optional
	Landscape string `json:"landscape,omitempty"`
}

// PromotionTargetReference identifies the `Stage` whose `spec.vector` is the
// promotion target.
type PromotionTargetReference struct {
	// Kind is the kind of the target resource.
	// +kubebuilder:validation:Enum=Stage
	Kind string `json:"kind"`

	// Name is the name of the target `Stage`.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`

	// Landscape is the name of the `Landscape` in the config's namespace whose
	// namespace hosts the target `Stage`.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Landscape string `json:"landscape"`
}

// VectorPromotionConfigSpec defines the desired state of VectorPromotionConfig.
type VectorPromotionConfigSpec struct {
	// Source references the resource to promote from.
	Source PromotionSourceReference `json:"source"`

	// Target references the Stage to promote to.
	Target PromotionTargetReference `json:"target"`

	// TTLAfterFinished will be copied onto every VectorPromotion the drift
	// controller creates for this config (pending the ADR-0032 rework). See
	// `VectorPromotionSpec.TTLAfterFinished`.
	// +optional
	TTLAfterFinished *metav1.Duration `json:"ttlAfterFinished,omitempty"`

	// KeepLastPromotions bounds how many terminal VectorPromotions are
	// retained per config; the oldest beyond the bound are deleted. Retention
	// by count keeps an audit trail even when `ttlAfterFinished` is short.
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=0
	// +optional
	KeepLastPromotions *int32 `json:"keepLastPromotions,omitempty"`
}

// VectorPromotionConfigStatus defines the observed state of VectorPromotionConfig.
type VectorPromotionConfigStatus struct {
	// Conditions reports on the config itself, e.g. whether its references
	// resolve to existing resources. Promotion results are reported separately
	// in `LastPromotionConditions`.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastPromotionConditions contains the result of the most recent VectorPromotion execution
	LastPromotionConditions []metav1.Condition `json:"lastPromotionConditions,omitempty"`
	// LastSuccessfulPromotionConditions contains the result of the most recent VectorPromotion execution, that was successful
	LastSuccessfulPromotionConditions []metav1.Condition `json:"lastSuccessfulPromotionConditions,omitempty"`

	// Sequence is the monotonic counter of promotions created for this config.
	// The config reconciler increments it and stamps the value into each
	// created promotion's `spec.sequence`.
	// +optional
	Sequence int64 `json:"sequence,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="Indicates if the config's references resolve"
// +kubebuilder:printcolumn:name="Last-Succeeded",type=string,JSONPath=".status.lastPromotionConditions[?(@.type==\"Succeeded\")].status",description="Last promotion succeeded"
// +kubebuilder:printcolumn:name="Last-Condition-Reason",type=string,JSONPath=".status.lastPromotionConditions[?(@.type==\"Succeeded\")].reason",description="Last promotion condition reason"
// +kubebuilder:printcolumn:name="Last-Time",type=date,JSONPath=".status.lastPromotionConditions[?(@.type==\"Succeeded\")].lastTransitionTime",description="Time of the last promotion"

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
