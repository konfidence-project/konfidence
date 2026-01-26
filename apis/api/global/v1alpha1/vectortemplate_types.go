/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// VectorTemplateReadyCondition is the ready condition for the VectorTemplate resource.
	VectorTemplateReadyCondition = "Ready"

	// VectorTemplateVectorCreatedReason indicates that a new vector was created.
	VectorTemplateVectorCreatedReason = "VectorCreated"
	// VectorTemplateNoDriftDetectedReason indicates that no drift was detected.
	VectorTemplateNoDriftDetectedReason = "NoDriftDetected"
	// VectorTemplateVectorCreationFailedReason indicates that vector creation failed.
	VectorTemplateVectorCreationFailedReason = "VectorCreationFailed"
	// VectorTemplateDriftDetectionFailedReason indicates that drift detection failed.
	VectorTemplateDriftDetectionFailedReason = "DriftDetectionFailed"
)

// VectorTemplateSpec defines the desired state of VectorTemplate.
// VectorTemplateSpec defines the components of which a vector is composed.
// From a VectorTemplate an OCM component is created which contains the latest version of all listed components.
type VectorTemplateSpec struct {
	// ReconcileInterval defines how often the assembly controller should check for drift.
	// If not set, the controller's default reconcile interval will be used.
	// +kubebuilder:validation:Optional
	ReconcileInterval *metav1.Duration `json:"reconcileInterval,omitempty"`

	// UploadTarget defines the target OCM component where the assembled vector will be uploaded.
	UploadTarget string `json:"uploadTarget"`

	// Base represents an optional base component version to build upon.
	// +kubebuilder:validation:Optional
	// +optional
	Base *string `json:"base,omitempty"`

	// Components lists the components to be included in the vector.
	// +kubebuilder:validation:MinItems=1
	Components []Component `json:"components"`
}

// Component defines a component of a VectorTemplate.
// A struct is used for future expansion.
type Component struct {
	Name string `json:"name"`
}

// VectorTemplateStatus defines the observed state of VectorTemplate.
type VectorTemplateStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VectorTemplate is the Schema for the vectortemplates API
type VectorTemplate struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of VectorTemplate
	// +required
	Spec VectorTemplateSpec `json:"spec"`

	// status defines the observed state of VectorTemplate
	// +optional
	Status VectorTemplateStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// VectorTemplateList contains a list of VectorTemplate
type VectorTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []VectorTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorTemplate{}, &VectorTemplateList{})
}
