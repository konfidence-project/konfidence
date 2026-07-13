package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StageConfigurationKind is kind of the StageConfiguration resource.
	StageConfigurationKind = "StageConfiguration"

	// StageConfigurationReadyCondition is the ready condition for the StageConfiguration resource.
	StageConfigurationReadyCondition = "Ready"
)

// StageConfigurationSpec defines the desired state of StageConfiguration.
type StageConfigurationSpec struct {
	// Name is the stage name.
	Name string `json:"name"`

	// Vector points to the OCM component that contains the deployment vector for this stage.
	Vector string `json:"vector"`

	// TargetNamespace is the target namespace where the associated stage is created or updated
	TargetNamespace string `json:"targetNamespace"`

	// Credentials supplies credentials for OCM repository access
	// and vector verification key material.
	// +optional
	Credentials *Credentials `json:"credentials,omitempty"`

	// VerifyVector lists candidate signatures evaluated against the
	// fetched vector descriptor. Absence disables vector verification.
	// +optional
	VerifyVector *Verify `json:"verifyVector,omitempty"`
}

// StageConfigurationStatus defines the observed state of StageConfiguration.
type StageConfigurationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=`.spec.vector`

// StageConfiguration is the Schema for the stageConfigurations API.
type StageConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StageConfigurationSpec   `json:"spec,omitempty"`
	Status StageConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StageConfigurationList contains a list of StageConfiguration.
type StageConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StageConfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StageConfiguration{}, &StageConfigurationList{})
}
