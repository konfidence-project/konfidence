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
	metav1 "k8s.io/apimachinery/pkg/api/meta/v1"
)

const (
	// VectorActivationKind is the kind for VectorActivation resources.
	VectorActivationKind = "VectorActivation"

	// ActivationFailed indicates that the vectorActivation failed.
	ActivationFailed = "Failed"

	// ActivationSucceeded indicates that the vectorActivation reconciled successfully.
	ActivationSucceeded = "Succeeded"

	// ActivationSkipped indicates that the vectorActivation was skipped.
	ActivationSkipped = "Skipped"

	// ActivationInProgress indicates that the vectorActivation is in progress.
	ActivationInProgress = "InProgress"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VectorActivationSpec defines the desired state of VectorActivation
type VectorActivationSpec struct {
	Stage string `json:"stage"`

	StageVersion string `json:"stageVersion"`

	// Vector points to the OCM component version that contains the deployment vector for this stage.
	Vector string `json:"vector"`

	VectorDeployment string `json:"vectorDeployment"`
}

// VectorActivationStatus defines the observed state of VectorActivation.
type VectorActivationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// VectorActivation is the Schema for the vectoractivations API
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:printcolumn:name="Stage",type=string,JSONPath=".spec.stage",description="The stage of the activation"
// +kubebuilder:printcolumn:name="Stage-Version",type=string,JSONPath=".spec.stageVersion",description="The version of the stage"
// +kubebuilder:printcolumn:name="Condition",type=string,JSONPath=".status.conditions[-1:].type",description="The latest condition type"
// +kubebuilder:printcolumn:name="Condition-Status",type=string,JSONPath=".status.conditions[-1:].status",description="The latest condition status"
type VectorActivation struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of VectorActivation
	// +required
	Spec VectorActivationSpec `json:"spec"`

	// status defines the observed state of VectorActivation
	// +optional
	Status VectorActivationStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// VectorActivationList contains a list of VectorActivation
type VectorActivationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorActivation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorActivation{}, &VectorActivationList{})
}
