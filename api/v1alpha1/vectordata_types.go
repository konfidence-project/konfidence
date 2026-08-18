package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	VectorDataKind = "VectorData"

	// VectorDataReadyCondition is set to True by the landscape orchestrator once the
	// runtime-specific materialisation (a ConfigMap on Kubernetes) is in place.
	VectorDataReadyCondition = "Ready"

	// VectorDataReasonMaterialized is the canonical Reason on VectorDataReadyCondition=True.
	VectorDataReasonMaterialized = "Materialized"
)

// VectorDataSpec is the LCP→landscape-orchestrator contract for vector-scoped data.
// The vector deployment controller resolves the OCM envelope `{features, authored}` and aggregates per-AD
// DeploymentResults; the landscape orchestrator materialises the payload on its
// target runtime (ConfigMap on K8s, etc.).
type VectorDataSpec struct {
	// Features carries the optional "features" subset of the OCM envelope, verbatim JSON.
	// +optional
	Features *runtime.RawExtension `json:"features,omitempty"`

	// Authored carries the optional "authored" subset of the OCM envelope, verbatim JSON.
	// +optional
	Authored *runtime.RawExtension `json:"authored,omitempty"`

	// DeploymentResults aggregated from underlying ArtifactDeployments, keyed by artifact
	// component name; the value lists every result emitted by that component.
	// +optional
	DeploymentResults map[string][]DeploymentResult `json:"deploymentResults,omitempty"`
}

type VectorDataStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// VectorData is the schema for the vectordata API.
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
	Spec   VectorDataSpec   `json:"spec,omitempty"`
	Status VectorDataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type VectorDataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorData `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorData{}, &VectorDataList{})
}
