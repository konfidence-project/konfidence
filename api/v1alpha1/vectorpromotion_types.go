package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionKind is kind of the VectorPromotion resource.
	VectorPromotionKind = "VectorPromotion"

	// ConditionTypeSucceeded is the condition type for promotion results.
	ConditionTypeSucceeded = "Succeeded"
	// ConditionTypeApproved is the condition type for promotion approval.
	ConditionTypeApproved = "Approved"

	// ReasonPromotionSucceeded indicates the promotion completed successfully.
	ReasonPromotionSucceeded = "PromotionSucceeded"
	// ReasonInvalidPromotionConfiguration indicates the promotion configuration is invalid.
	ReasonInvalidPromotionConfiguration = "InvalidPromotionConfiguration"
	// ReasonPromotionConfigurationNotFound indicates the referenced VectorPromotionConfig was not found.
	ReasonPromotionConfigurationNotFound = "PromotionConfigurationNotFound"
	// ReasonPromotionSourceNotFound indicates the source vector was not found.
	ReasonPromotionSourceNotFound = "PromotionSourceNotFound"
	// ReasonPromotionFailed is a catch-all for other promotion errors.
	ReasonPromotionFailed = "PromotionFailed"
	// ReasonPromotionRunning indicates that the promotion is still running.
	ReasonPromotionRunning = "PromotionRunning"
	// ReasonPromotionSourceVerificationFailed indicates that the verification of the source vector failed.
	ReasonPromotionSourceVerificationFailed = "PromotionSourceVerificationFailed"
	// ReasonPromotionWaitingForApproval indicates the promotion waits for manual approval.
	ReasonPromotionWaitingForApproval = "WaitingForApproval"
	// ReasonPromotionManuallyApproved indicates the promotion was approved manually.
	ReasonPromotionManuallyApproved = "ManuallyApproved"
	// ReasonPromotionSuperseded indicates a newer promotion for the same config replaced this one.
	ReasonPromotionSuperseded = "PromotionSuperseded"
)

// VectorPromotionState summarizes the promotion lifecycle for display.
// Conditions are the source of truth; the state is derived from them.
type VectorPromotionState string

const (
	// PromotionStateWaiting means at least one gate is still open: the
	// promotion requires approval and has not been approved yet.
	PromotionStateWaiting VectorPromotionState = "Waiting"
	// PromotionStateReady means every gate has passed and the promotion is
	// queued for execution. Promotions that require no approval are Ready
	// from their first reconcile.
	PromotionStateReady VectorPromotionState = "Ready"
	// PromotionStateInProgress means the promotion is executing.
	PromotionStateInProgress VectorPromotionState = "InProgress"
	// PromotionStateSucceeded means the promotion completed successfully.
	PromotionStateSucceeded VectorPromotionState = "Succeeded"
	// PromotionStateFailed means the promotion reached a terminal state without success.
	PromotionStateFailed VectorPromotionState = "Failed"
	// PromotionStateSuperseded means a newer promotion replaced this one.
	// Superseded promotions are locked: they can never be approved or
	// executed afterwards. The newer promotion is the one to act on.
	PromotionStateSuperseded VectorPromotionState = "Superseded"
)

// VectorPromotionSpec defines the desired state of VectorPromotion.
type VectorPromotionSpec struct {
	// VectorPromotionConfigRef is the name of the VectorPromotionConfig that defines the promotion flow to execute.
	// +kubebuilder:validation:MinLength=1
	VectorPromotionConfigRef string `json:"vectorPromotionConfigRef"`

	// Vector is the concrete OCM component version reference
	// (`<registry>//<component>:<version>`) pinned when the promotion was created.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vector is immutable after it has been set"
	Vector string `json:"vector"`

	// RequireApproval is true when the promotion must be approved before
	// execution; false means the promotion is approved automatically. It is
	// independent of the source kind: the config controller defaults it to
	// true for `Stage` sources, but any combination is valid.
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="requireApproval is immutable after it has been set"
	// +optional
	RequireApproval bool `json:"requireApproval,omitempty"`

	// TTLAfterFinished defines how long the VectorPromotion should be kept after completion.
	// Once the TTL expires after the promotion reaches a terminal state (Completed or Failed),
	// the resource is eligible for automatic deletion. If no TTL is set, no deletion happens.
	// +kubebuilder:validation:Optional
	TTLAfterFinished *metav1.Duration `json:"ttlAfterFinished,omitempty"`
}

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// State summarizes Conditions for display. Conditions are the source of
	// truth; State is recomputed whenever conditions are written. `Superseded`
	// is a locked terminal state: a superseded promotion can never be
	// approved or executed afterwards, only its successor can.
	// +kubebuilder:validation:Enum=Waiting;Ready;InProgress;Succeeded;Failed;Superseded
	// +optional
	State VectorPromotionState `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Config",type=string,JSONPath=".spec.vectorPromotionConfigRef",description="The referenced VectorPromotionConfig"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=".spec.vector",description="The promoted vector version"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.state",description="Promotion state"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Age"

// VectorPromotion triggers a one-time execution of a promotion flow defined by a VectorPromotionConfig.
type VectorPromotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorPromotionSpec   `json:"spec,omitempty"`
	Status VectorPromotionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorPromotionList contains a list of VectorPromotion.
type VectorPromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorPromotion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorPromotion{}, &VectorPromotionList{})
}
