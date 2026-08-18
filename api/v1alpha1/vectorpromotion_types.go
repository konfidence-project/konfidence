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
	// ReasonPromotionFailed is a catch-all for other promotion errors.
	ReasonPromotionFailed = "PromotionFailed"
	// ReasonPromotionRunning indicates that the promotion is still running.
	ReasonPromotionRunning = "PromotionRunning"
	// ReasonPromotionWaitingForApproval indicates the promotion waits for manual approval.
	ReasonPromotionWaitingForApproval = "WaitingForApproval"
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
	// PromotionStateWaiting means at least one gate is still open: the
	// promotion requires approval and has not been approved yet.
	PromotionStateWaiting VectorPromotionState = "Waiting"
	// PromotionStateReady means every gate has passed and the promotion is
	// queued for execution. Promotions that require no approval are Ready
	// from their first reconcile.
	PromotionStateReady VectorPromotionState = "Ready"
	// PromotionStateInProgress means the promotion is executing.
	PromotionStateInProgress VectorPromotionState = "InProgress"
	// PromotionStateBlocked means the promotion is ready but cannot execute
	// because its target does not resolve; see the config's Ready condition.
	PromotionStateBlocked VectorPromotionState = "Blocked"
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
	// VectorPromotionConfigName is the name of the VectorPromotionConfig that defines the promotion flow to execute.
	// +kubebuilder:validation:MinLength=1
	VectorPromotionConfigName string `json:"vectorPromotionConfigName"`

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

	// Sequence is a monotonic ordinal assigned by the creator (the config
	// reconciler, from the config's `status.sequence`). It is the sole
	// ordering between promotions of the same config; creation timestamps
	// only have second resolution and are never consulted.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="sequence is immutable after it has been set"
	Sequence int64 `json:"sequence"`
}

// PromotionApproval records the granted approval.
type PromotionApproval struct {
	// ApprovedBy is the identity that granted the approval, as reported by the
	// konfidence API. The value is an opaque, arbitrary string (username,
	// email, subject, ...); it is recorded verbatim and never interpreted.
	// +kubebuilder:validation:MinLength=1
	ApprovedBy string `json:"approvedBy"`

	// ApprovedAt is the time the approval was granted.
	ApprovedAt metav1.Time `json:"approvedAt"`
}

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// State summarizes Conditions for display. Conditions are the source of
	// truth; State is recomputed whenever conditions are written. `Superseded`
	// is a locked terminal state: a superseded promotion can never be
	// approved or executed afterwards, only its successor can.
	// +kubebuilder:validation:Enum=Waiting;Ready;InProgress;Blocked;Succeeded;Failed;Superseded
	// +optional
	State VectorPromotionState `json:"state,omitempty"`

	// Approval records the granted approval. A promotion is approved at most
	// once; re-approval attempts are rejected.
	// +optional
	Approval *PromotionApproval `json:"approval,omitempty"`

	// PromotedStageRef records the Stage this promotion actually wrote its
	// vector to, so the promotion is self-describing even after the config
	// changed or was deleted.
	// +optional
	PromotedStageRef *corev1.TypedObjectReference `json:"promotedStageRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:selectablefield:JSONPath=`.spec.vectorPromotionConfigName`
// +kubebuilder:selectablefield:JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Config",type=string,JSONPath=".spec.vectorPromotionConfigName",description="The referenced VectorPromotionConfig"
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.source.name",description="The promotion source"
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.target.name",description="The target Stage"
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=".status.state",description="Promotion state"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=".spec.vector",description="The promoted vector version",priority=1
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
