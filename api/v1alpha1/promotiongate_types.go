package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// GateTypeExplicitApproval is satisfied when a caller with approval rights
	// approves the promotion through the konfidence API.
	GateTypeExplicitApproval = "explicit-approval.konfidence.cloud"
	// GateTypeActivatedOnSourceStage is satisfied when the source stage's
	// active StageVersion resolves to the promotion's vector.
	GateTypeActivatedOnSourceStage = "activated-on-source-stage.konfidence.cloud"

	// ReasonGateApprovedByUser is the gate reason recorded when a user
	// approves a promotion through the konfidence API.
	ReasonGateApprovedByUser = "ApprovedByUser"
)

// PromotionGateState is the state of a single promotion gate.
// +kubebuilder:validation:Enum=Pending;Satisfied;Error
type PromotionGateState string

const (
	// PromotionGateStatePending means the gate is still open.
	PromotionGateStatePending PromotionGateState = "Pending"
	// PromotionGateStateSatisfied means the gate has passed.
	PromotionGateStateSatisfied PromotionGateState = "Satisfied"
	// PromotionGateStateError means the gate writer detected a problem (e.g. a
	// misconfiguration) that keeps the gate from ever passing as is.
	PromotionGateStateError PromotionGateState = "Error"
)

// PromotionGate declares a gate a promotion must pass before it executes.
type PromotionGate struct {
	// Type identifies the gate as `<gate-name>.<owner-domain>`, for example
	// `explicit-approval.konfidence.cloud` or `e2e-tests.example.com`. The
	// first label names the gate, the remaining labels name the party that
	// owns the gate type and ships its writer. The promotion controller never
	// interprets the type; it only tracks whether the gate is satisfied.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?){2,}$`
	Type string `json:"type"`
}

// PromotionGateStatus reports the state of one declared gate together with
// its provenance: who set it, when, and why.
type PromotionGateStatus struct {
	// Type is the gate type this entry reports on.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Type string `json:"type"`

	// State is the current state of the gate.
	State PromotionGateState `json:"state"`

	// Reason is a machine-readable CamelCase reason for the latest state change.
	// +kubebuilder:validation:MaxLength=1024
	// +optional
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable description of the latest state change,
	// e.g. what is misconfigured when State is `Error`.
	// +kubebuilder:validation:MaxLength=32768
	// +optional
	Message string `json:"message,omitempty"`

	// SetBy is the entity that last set the gate state: a controller name or,
	// for human actions, the authenticated subject. The value is advisory and
	// not a security boundary.
	// +optional
	SetBy string `json:"setBy,omitempty"`

	// SetAt is the time the gate state was last set.
	// +optional
	SetAt *metav1.Time `json:"setAt,omitempty"`
}
