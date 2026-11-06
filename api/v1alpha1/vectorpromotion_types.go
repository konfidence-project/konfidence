package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorPromotionKind is kind of the VectorPromotion resource.
	VectorPromotionKind = "VectorPromotion"

	// ConditionTypeGatesSatisfied reports whether every declared gate is
	// satisfied. Once True it never changes again, and the gates are frozen
	// from then on. Promotions without gates get True on their first reconcile.
	ConditionTypeGatesSatisfied = "GatesSatisfied"
	// ConditionTypeTargetResolved reports whether the target Stage and its
	// Landscape resolve. Execution retries while it is False.
	ConditionTypeTargetResolved = "TargetResolved"
	// ConditionTypeSucceeded reports the outcome of the promotion. It is
	// Unknown while the promotion is active and True or False once it
	// finished; it never changes after that.
	ConditionTypeSucceeded = "Succeeded"

	// ReasonGatesPending (GatesSatisfied=False) indicates at least one declared gate is still open.
	ReasonGatesPending = "GatesPending"
	// ReasonGateError (GatesSatisfied=False) indicates at least one declared gate reports an error.
	ReasonGateError = "GateError"
	// ReasonAllGatesSatisfied (GatesSatisfied=True) indicates every declared gate is satisfied.
	ReasonAllGatesSatisfied = "AllGatesSatisfied"
	// ReasonGatesBypassed (GatesSatisfied=False) indicates an operator bypassed the gates, so the promotion proceeds despite them.
	ReasonGatesBypassed = "GatesBypassed"
	// ReasonTargetResolved (TargetResolved=True) indicates the target Stage resolved.
	ReasonTargetResolved = "TargetResolved"
	// ReasonPromotionPending (Succeeded=Unknown) indicates the promotion has not started executing.
	ReasonPromotionPending = "PromotionPending"
	// ReasonPromotionRunning (Succeeded=Unknown) indicates the promotion is executing.
	ReasonPromotionRunning = "PromotionRunning"
	// ReasonPromotionSucceeded (Succeeded=True) indicates the promotion completed successfully.
	ReasonPromotionSucceeded = "PromotionSucceeded"
	// ReasonPromotionSuperseded (Succeeded=False) indicates a newer promotion for the same config replaced this one.
	ReasonPromotionSuperseded = "PromotionSuperseded"
	// ReasonPromotionOutstandingLimitReached (Succeeded=False) indicates a waiting promotion was
	// superseded after the limit of outstanding promotions was reached.
	ReasonPromotionOutstandingLimitReached = "OutstandingLimitReached"
	// ReasonPromotionRejected (Succeeded=False) indicates an operator rejected the promotion.
	ReasonPromotionRejected = "PromotionRejected"
)

// PromotionOverrideType is the operator decision recorded in `status.override`.
// +kubebuilder:validation:Enum=Reject;BypassGates
type PromotionOverrideType string

const (
	// PromotionOverrideTypeReject terminally rejects a waiting promotion.
	PromotionOverrideTypeReject PromotionOverrideType = "Reject"
	// PromotionOverrideTypeBypassGates lets a waiting promotion proceed to execution regardless of its gates.
	PromotionOverrideTypeBypassGates PromotionOverrideType = "BypassGates"
)

// PromotionOverride records an operator decision that overrides the normal
// gate flow, together with its provenance: who, when, and why.
type PromotionOverride struct {
	// Type is the operator decision.
	Type PromotionOverrideType `json:"type"`

	// SetBy is the authenticated subject that made the decision.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	SetBy string `json:"setBy"`

	// SetAt is the time the decision was made.
	SetAt metav1.Time `json:"setAt"`

	// Reason is the operator's justification for the decision.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Reason string `json:"reason"`
}

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

	// Gates is an immutable snapshot of the config's gates at creation time.
	// The promotion does not execute until every gate has a `Satisfied` entry
	// in `status.gates`. An empty list promotes immediately. The field is
	// required: creators set an empty list rather than omitting it.
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="gates are immutable after they have been set"
	Gates []PromotionGate `json:"gates"`

	// TTLAfterFinished defines how long the VectorPromotion should be kept after completion.
	// Once the TTL expires after the promotion finished (`status.completionTime`),
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

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.override) || (has(self.override) && self.override == oldSelf.override)",message="override is immutable once set"

// VectorPromotionStatus defines the observed state of VectorPromotion.
type VectorPromotionStatus struct {
	// Conditions report GatesSatisfied, TargetResolved and Succeeded.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// StartTime is when execution started, i.e. the promotion began writing
	// its vector to the target Stage. Set at most once.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the promotion finished (Succeeded True or False).
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Gates reports the state and provenance of each declared gate, keyed by
	// gate type. Entries are seeded `Pending` by the promotion controller and
	// set by the gate's writer (a gate controller or the konfidence API)
	// through `pkg/promotiongate`. A missing entry counts as `Pending`.
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MaxItems=32
	// +optional
	Gates []PromotionGateStatus `json:"gates,omitempty"`

	// Override records an operator decision that overrides the gate flow:
	// `Reject` terminally rejects the promotion, `BypassGates` lets it proceed
	// regardless of its gates (GatesSatisfied stays False with reason
	// GatesBypassed; the gate entries are left untouched). It is written only by
	// the konfidence API, at most once while the promotion waits on its gates,
	// and never cleared.
	// +optional
	Override *PromotionOverride `json:"override,omitempty"`

	// PromotedStageRef records the Stage this promotion actually wrote its
	// vector to, so the promotion is self-describing even after the config
	// changed or was deleted.
	// +optional
	PromotedStageRef *corev1.TypedObjectReference `json:"promotedStageRef,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:selectablefield:JSONPath=`.spec.vectorPromotionConfigName`
// +kubebuilder:printcolumn:name="Config",type=string,JSONPath=".spec.vectorPromotionConfigName",description="The referenced VectorPromotionConfig"
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=".spec.source.name",description="The promotion source"
// +kubebuilder:printcolumn:name="Target",type=string,JSONPath=".spec.target.name",description="The target Stage"
// +kubebuilder:printcolumn:name="Gates",type=string,JSONPath=".status.conditions[?(@.type==\"GatesSatisfied\")].reason",description="Why the gates are satisfied, pending, errored, or bypassed"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Succeeded\")].reason",description="Reason of the outcome"
// +kubebuilder:printcolumn:name="Started",type=date,JSONPath=".status.startTime",description="Execution start",priority=1
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
