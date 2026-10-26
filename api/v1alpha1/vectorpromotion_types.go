package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionKind is kind of the VectorPromotion resource.
	VectorPromotionKind = "VectorPromotion"

	// ConditionTypeSucceeded carries the promotion phase. Its status tells
	// whether the promotion finished (`Unknown` while live, `True` on success,
	// `False` on any other end); its reason is the phase. `status.state` is
	// derived from the reason and written in the same status patch.
	ConditionTypeSucceeded = "Succeeded"

	// ReasonGatesPending (Unknown) indicates at least one declared gate is still open.
	ReasonGatesPending = "GatesPending"
	// ReasonGateError (Unknown) indicates at least one declared gate reports an error.
	ReasonGateError = "GateError"
	// ReasonPromotionReady (Unknown) indicates every declared gate is satisfied
	// and the promotion is queued for execution.
	ReasonPromotionReady = "PromotionReady"
	// ReasonPromotionRunning (Unknown) indicates that the promotion is executing.
	ReasonPromotionRunning = "PromotionRunning"
	// ReasonPromotionSucceeded (True) indicates the promotion completed successfully.
	ReasonPromotionSucceeded = "PromotionSucceeded"
	// ReasonPromotionFailed (False) is a catch-all for other promotion errors.
	ReasonPromotionFailed = "PromotionFailed"
	// ReasonPromotionSuperseded (False) indicates a newer promotion for the same config replaced this one.
	ReasonPromotionSuperseded = "PromotionSuperseded"
	// ReasonPromotionOutstandingLimitReached (False) indicates an outstanding Promotion was superseded
	// after the limit of outstanding promotions was reached.
	ReasonPromotionOutstandingLimitReached = "OutstandingLimitReached"
	// ReasonPromotionTargetUnresolved (Unknown) indicates the target Stage or its Landscape does not
	// resolve yet. The promotion stays live and execution is retried.
	ReasonPromotionTargetUnresolved = "PromotionTargetUnresolved"
)

// VectorPromotionState summarizes the promotion lifecycle. It is derived from
// the reason of the Succeeded condition, which is the source of truth, and is
// written in the same status patch as that condition.
type VectorPromotionState string

const (
	// PromotionStateWaiting means at least one declared gate is not
	// satisfied yet (Pending or Error).
	PromotionStateWaiting VectorPromotionState = "Waiting"
	// PromotionStateReady means every declared gate was satisfied and the
	// promotion is queued for execution; its gates can no longer change.
	// Promotions without gates are Ready from their first reconcile.
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
	// PromotionStateSuperseded means a promotion was replaced by a newer one or
	// retired after the outstanding promotion limit was reached. Superseded
	// promotions can never be approved or executed afterward.
	PromotionStateSuperseded VectorPromotionState = "Superseded"
)

// Terminal reports whether the state is final: Succeeded, Failed or
// Superseded. A terminal promotion is never reconciled again.
func (s VectorPromotionState) Terminal() bool {
	switch s {
	case PromotionStateSucceeded, PromotionStateFailed, PromotionStateSuperseded:
		return true
	}
	return false
}

// Cleared reports whether the state belongs to a live promotion that passed
// its gates and competes for execution: Ready, InProgress or Blocked.
func (s VectorPromotionState) Cleared() bool {
	switch s {
	case PromotionStateReady, PromotionStateInProgress, PromotionStateBlocked:
		return true
	}
	return false
}

// VectorPromotionSpec defines the desired state of VectorPromotion.
//
// +kubebuilder:validation:XValidation:rule="has(self.gates) == has(oldSelf.gates) && (!has(self.gates) || self.gates == oldSelf.gates)",message="gates are immutable after they have been set"
//
//nolint:lll // Kubebuilder annotations are intentionally long.
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

	// Gates is an immutable snapshot of the config's gates at creation time.
	// The promotion stays `Waiting` until every gate has a `Satisfied` entry
	// in `status.gates`. An empty list promotes immediately.
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Gates []PromotionGate `json:"gates,omitempty"`

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

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// State summarizes the Succeeded condition. The condition's reason is the
	// source of truth; State is derived from it and written in the same status
	// patch, so the two never disagree. Gate writers may read it: gates can
	// only change while State is empty or `Waiting`. `Superseded` is a
	// terminal state: a superseded promotion can never be approved or
	// executed afterward.
	// +kubebuilder:validation:Enum=Waiting;Ready;InProgress;Blocked;Succeeded;Failed;Superseded
	// +optional
	State VectorPromotionState `json:"state,omitempty"`

	// Gates reports the state and provenance of each declared gate, keyed by
	// gate type. Entries are seeded `Pending` by the promotion controller and
	// set by the gate's writer (a gate controller or the konfidence API)
	// through `pkg/promotiongate`. A missing entry counts as `Pending`.
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=16
	// +optional
	Gates []PromotionGateStatus `json:"gates,omitempty"`

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
