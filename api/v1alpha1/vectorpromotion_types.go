package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
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
	// ReasonPromotionSourceNotFound indicates the source vector was not found.
	// Not yet written: source misses surface on the config's Ready condition instead.
	ReasonPromotionSourceNotFound = "PromotionSourceNotFound"
	// ReasonPromotionFailed is a catch-all for other promotion errors.
	ReasonPromotionFailed = "PromotionFailed"
	// ReasonPromotionRunning indicates that the promotion is still running.
	ReasonPromotionRunning = "PromotionRunning"
	// ReasonPromotionSourceVerificationFailed indicates that the verification of the source vector failed.
	// Not yet written on this branch: reserved for source verification (ADR-0032 follow-up).
	ReasonPromotionSourceVerificationFailed = "PromotionSourceVerificationFailed"
	// ReasonPromotionWaitingForApproval indicates the promotion waits for manual approval.
	ReasonPromotionWaitingForApproval = "WaitingForApproval"
	// ReasonPromotionAutoApproved indicates the promotion was approved automatically
	// because its source is a VectorTemplate.
	ReasonPromotionAutoApproved = "AutoApproved"
	// ReasonPromotionManuallyApproved indicates the promotion was approved manually.
	ReasonPromotionManuallyApproved = "ManuallyApproved"
	// ReasonPromotionSuperseded indicates a newer promotion for the same config replaced this one.
	ReasonPromotionSuperseded = "PromotionSuperseded"
	// ReasonPromotionTimedOut indicates the promotion stayed in progress past the
	// execution deadline and was retired.
	ReasonPromotionTimedOut = "PromotionTimedOut"
	// ReasonPromotionTargetUnresolved indicates the target Stage or its Landscape does not
	// resolve yet. The promotion stays live and execution is retried.
	ReasonPromotionTargetUnresolved = "PromotionTargetUnresolved"
)

// VectorPromotionState summarizes the promotion lifecycle for display.
// Conditions are the source of truth; the state is derived from them.
type VectorPromotionState string

const (
	// PromotionStatePending means the promotion has not started yet.
	PromotionStatePending VectorPromotionState = "Pending"
	// PromotionStateWaitingForApproval means the promotion requires approval and has not been approved yet.
	PromotionStateWaitingForApproval VectorPromotionState = "WaitingForApproval"
	// PromotionStateApproved means the promotion is approved but execution has not started.
	PromotionStateApproved VectorPromotionState = "Approved"
	// PromotionStateInProgress means the promotion is executing.
	PromotionStateInProgress VectorPromotionState = "InProgress"
	// PromotionStateBlocked means the promotion is approved but cannot execute
	// because its target does not resolve; see the config's Ready condition.
	PromotionStateBlocked VectorPromotionState = "Blocked"
	// PromotionStateSucceeded means the promotion completed successfully.
	PromotionStateSucceeded VectorPromotionState = "Succeeded"
	// PromotionStateFailed means the promotion reached a terminal state without success.
	PromotionStateFailed VectorPromotionState = "Failed"
	// PromotionStateSuperseded means a newer promotion replaced this one.
	PromotionStateSuperseded VectorPromotionState = "Superseded"
)

// VectorPromotionSpec defines the desired state of VectorPromotion.
type VectorPromotionSpec struct {
	// VectorPromotionConfigRef is the name of the VectorPromotionConfig that defines the promotion flow to execute.
	// +kubebuilder:validation:MinLength=1
	VectorPromotionConfigRef string `json:"vectorPromotionConfigRef"`

	// Source is a snapshot of the config's source reference at creation time,
	// recorded so a promotion is self-describing.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="source is immutable after it has been set"
	Source PromotionSourceReference `json:"source"`

	// Target is a snapshot of the config's target reference at creation time.
	// Execution resolves and writes this target: approving a promotion approves
	// exactly this destination, regardless of later config edits.
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="target is immutable after it has been set"
	Target PromotionTargetReference `json:"target"`

	// Vector is the concrete OCM component version reference
	// (`<registry>//<component>:<version>`) pinned when the promotion was created.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^[^/].+//.+:.+$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="vector is immutable after it has been set"
	Vector string `json:"vector"`

	// RequireApproval is true when the promotion must be approved before
	// execution (source kind `Stage`); false means the promotion is
	// auto-approved (source kind `VectorTemplate`).
	// +kubebuilder:default=false
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="requireApproval is immutable after it has been set"
	// +optional
	RequireApproval bool `json:"requireApproval,omitempty"`

	// TTLAfterFinished defines how long the VectorPromotion should be kept after completion.
	// Once the TTL expires after the promotion reaches a terminal state (Completed or Failed),
	// the resource is eligible for automatic deletion. If no TTL is set, no deletion happens.
	// +kubebuilder:validation:Optional
	TTLAfterFinished *metav1.Duration `json:"ttlAfterFinished,omitempty"`

	// Sequence is a monotonic ordinal assigned by the creator (the config
	// reconciler, from the config's `status.sequence`). Promotions with a
	// higher sequence are newer regardless of creation timestamps, which only
	// have second resolution.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="sequence is immutable after it has been set"
	// +optional
	Sequence int64 `json:"sequence,omitempty"`
}

// PromotionApproval records a granted approval.
type PromotionApproval struct {
	// ApprovedBy is the identity that granted the approval, as reported by the
	// konfidence API.
	ApprovedBy string `json:"approvedBy"`

	// ApprovedAt is the time the approval was granted.
	ApprovedAt metav1.Time `json:"approvedAt"`
}

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// State summarizes Conditions for display. Conditions are the source of
	// truth; State is recomputed whenever conditions are written.
	// +kubebuilder:validation:Enum=Pending;WaitingForApproval;Approved;InProgress;Blocked;Succeeded;Failed;Superseded
	// +optional
	State VectorPromotionState `json:"state,omitempty"`

	// Approvals records every granted approval for auditing.
	// +optional
	Approvals []PromotionApproval `json:"approvals,omitempty"`

	// PromotedStageRef records the Stage this promotion actually wrote its
	// vector to, so the promotion is self-describing even after the config
	// changed or was deleted.
	// +optional
	PromotedStageRef *corev1.TypedObjectReference `json:"promotedStageRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:selectablefield:JSONPath=`.spec.vectorPromotionConfigRef`
// +kubebuilder:selectablefield:JSONPath=`.status.state`
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
