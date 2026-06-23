package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorDataKind is the Kubernetes kind for VectorData resources.
	VectorDataKind = "VectorData"

	// VectorDataReadyCondition indicates that the runtime-specific implementor has successfully materialized the
	// vector-scoped data for this VectorDeployment in whatever shape the target runtime needs (e.g. a ConfigMap on
	// Kubernetes). VectorReady on the parent VectorDeployment is gated on this condition.
	VectorDataReadyCondition = "Ready"

	// VectorDataReasonMaterialized is the canonical Reason set on VectorDataReadyCondition once the implementor has
	// committed the data on the target runtime.
	VectorDataReasonMaterialized = "Materialized"
)

// VectorDataSpec describes the vector-scoped data that needs to be made available to applications inside a vector at
// runtime. It is populated by the Star vector-deployment-controller after all ArtifactDeployments of the vector have
// reached Ready (so DeploymentResults are observable) and the optional authored configuration blob has been resolved
// from OCM.
//
// VectorDataSpec is intentionally runtime-agnostic: the runtime-specific implementor reads the bytes out of this CR
// and writes them in whatever shape its runtime expects (a Kubernetes ConfigMap, a Cloud Foundry user-provided service
// instance, a file in a per-vector PV, etc.). Centralising OCM resolution in the Star controller keeps OCM
// credentials, schemas and crypto verifiers in one place and lets runtime adapters stay thin.
//
// Both fields are typically populated together and stay immutable for the lifetime of the VectorData object: the
// VectorDeployment that owns it has an immutable Spec, the referenced vector OCM ComponentVersion is immutable, and
// DeploymentResults are documented as immutable per ArtifactDeployment generation.
type VectorDataSpec struct {
	// Config carries the optional authored vector-scoped configuration blob (the verbatim bytes of the OCM resource
	// named "cloud-konfidence-vector-config" on the vector ComponentVersion, resolved by the Star side). The field is
	// empty when the vector did not declare such a resource.
	//
	// The Star controller never parses the bytes; the implementor is free to interpret them in whatever way is
	// appropriate for the consumer (typically JSON forwarded verbatim to the application).
	// +optional
	Config []byte `json:"config,omitempty"`

	// DeploymentResults is the aggregated set of structured outputs produced by all underlying ArtifactDeployments of
	// the owning VectorDeployment, keyed "<componentName>/<resultName>". May be empty when no artifact produced any
	// results.
	// +optional
	DeploymentResults map[string]DeploymentResult `json:"deploymentResults,omitempty"`
}

// VectorDataStatus carries the observed state of the runtime-specific materialisation. A VectorData is considered
// fulfilled once VectorDataReadyCondition flips to True; the implementor is responsible for setting that condition.
type VectorDataStatus struct {
	// Conditions reports the materialisation state. Implementors should set VectorDataReadyCondition to True after
	// successfully writing the underlying artefact on the target runtime, or to False with a descriptive Reason on
	// failure paths.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// VectorData is the Schema for the vectordata API.
//
// VectorData is created by the Star vector-deployment-controller as the last lifecycle step of a VectorDeployment.
// A runtime-specific implementor — referred to as the "landscape orchestrator" in Konfidence's architecture —
// watches VectorData resources and materialises them on the target runtime. On Kubernetes the
// `kubernetes-landscape-orchestrator` writes a ConfigMap into the landscape namespace; other runtimes (e.g. Cloud
// Foundry) ship their own orchestrator with a different materialisation. The contract between Star and the
// orchestrator is the VectorData CR itself and its Ready condition.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
//
//nolint:lll // Kubebuilder annotations are intentionally long.
type VectorData struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VectorData spec is immutable after it has been set"
	// Spec is immutable: the inputs (vector OCM ComponentVersion and per-AD DeploymentResults) are themselves
	// immutable, so the Star controller writes the Spec exactly once per VectorDeployment.
	Spec   VectorDataSpec   `json:"spec,omitempty"`
	Status VectorDataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorDataList contains a list of VectorData.
type VectorDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorData `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorData{}, &VectorDataList{})
}
