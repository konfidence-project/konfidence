package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LandscapeKind is kind of the Landscape resource.
	LandscapeKind = "Landscape"

	// LandscapeReadyCondition is the ready condition for the Landscape resource.
	LandscapeReadyCondition = "Ready"
	// LandscapeNamespaceReadyCondition reports on the managed landscape namespace.
	LandscapeNamespaceReadyCondition = "NamespaceReady"

	// LandscapeReconciledReason indicates that the landscape was fully reconciled.
	LandscapeReconciledReason = "LandscapeReconciled"
	// LandscapeTerminatingReason indicates that the landscape is being deleted and is
	// waiting for its namespace to terminate.
	LandscapeTerminatingReason = "Terminating"
	// LandscapeNamespaceReconciledReason indicates that the landscape namespace is in the desired state.
	LandscapeNamespaceReconciledReason = "NamespaceReconciled"
	// LandscapeNamespaceConflictReason indicates that the landscape namespace exists but is not managed by the landscape.
	LandscapeNamespaceConflictReason = "NamespaceConflict"
	// LandscapeNamespaceTerminatingReason indicates that the landscape namespace is currently terminating.
	LandscapeNamespaceTerminatingReason = "NamespaceTerminating"
	// LandscapeNamespaceCreateFailedReason indicates that creating or updating the landscape namespace failed.
	LandscapeNamespaceCreateFailedReason = "NamespaceCreateFailed"
	// LandscapeInvalidNamespaceReason indicates that the landscape was created in a namespace that is not a project namespace.
	LandscapeInvalidNamespaceReason = "InvalidNamespace"

	// LandscapeNamespacePrefix is the prefix of the default landscape namespace name.
	LandscapeNamespacePrefix = "kden-l-"
)

// LandscapeSpec defines the desired state of Landscape.
//
// The transition rule catches namespace being set or unset after creation;
// changing a set namespace is caught by the field-level rule. The two rules
// are split to stay within the CEL cost budget of the schema.
// +kubebuilder:validation:XValidation:rule="has(self.namespace) == has(oldSelf.namespace)",message="namespace is immutable"
type LandscapeSpec struct {
	// DisplayName is the human-readable name of the landscape, shown in user
	// interfaces. It does not affect the namespace name or any label, and it
	// may be changed at any time.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// Namespace overrides the name of the namespace created for this landscape.
	// When unset it defaults to `kden-l-<landscape-name>-<hash>`. It is
	// immutable once the Landscape exists, because the namespace and everything
	// it holds are bound to this name.
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="namespace is immutable"
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// LandscapeStatus defines the observed state of Landscape.
type LandscapeStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Namespace is the name of the namespace managed for this landscape.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// ProjectName is the name of the project this landscape belongs to,
	// derived from the namespace where the Landscape CR was created.
	// +optional
	ProjectName string `json:"projectName,omitempty"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 46",message="landscape name must be at most 46 characters"
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.status.projectName`,description="The project this landscape belongs to"
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.status.namespace`,description="The landscape namespace"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Landscape is the Schema for the landscapes API. A Landscape owns a dedicated
// namespace that serves as a deployment target for vectors. Landscapes must be
// created in project namespaces. The landscape name is capped at 46 characters
// so the derived namespace name stays within the 63-character Kubernetes limits.
type Landscape struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LandscapeSpec   `json:"spec,omitempty"`
	Status LandscapeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LandscapeList contains a list of Landscape.
type LandscapeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Landscape `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Landscape{}, &LandscapeList{})
}
