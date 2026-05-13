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
	// StageVersionKind is the kind for StageVersion resources.
	StageVersionKind = "StageVersion"

	// VectorMigrationCreatedCondition indicates that the VectorMigration resource has been created successfully.
	VectorMigrationCreatedCondition string = "VectorMigrationCreated"

	// VectorActivationCreatedCondition indicates that the VectorActivation resource has been created successfully.
	VectorActivationCreatedCondition string = "VectorActivationCreated"

	// VectorActivatedCondition indicates that the vector has been activated in the stage.
	VectorActivatedCondition = "VectorActivated"

	// StageVersionReady indicates that the stage version is ready for use.
	StageVersionReady = "Ready"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StageVersionSpec defines the desired state of StageVersion
type StageVersionSpec struct {
	// Vector points to the OCM component version that contains the deployment vector for this stage.
	// +kubebuilder:validation:MinLength=1
	Vector string `json:"vector"`

	// the object generation of the stage that created this stage version
	// +kubebuilder:validation:Minimum=1
	StageGeneration int64 `json:"stageGeneration"`

	// stageRef references the Stage this StageVersion belongs to
	// +kubebuilder:validation:required
	StageRef *StageReference `json:"stageRef"`
}

// StageVersionStatus defines the observed state of StageVersion.
type StageVersionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// StageVersion is the Schema for the stageversions API
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Stage",type=string,JSONPath=".spec.stageRef.name",description="Name of the referenced Stage"
// +kubebuilder:printcolumn:name="Stage-Generation",type=integer,JSONPath=".spec.stageGeneration",description="The object generation of the stage"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=".spec.vector",description="The deployment vector for this stage"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Time since creation of StageVersion"
type StageVersion struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="StageVersion spec is immutable after it has been set"
	// Spec defines the desired state of the StageVersion and is immutable after it has been set
	// +required
	Spec StageVersionSpec `json:"spec"`

	// status defines the observed state of StageVersion
	// +optional
	Status StageVersionStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// StageVersionList contains a list of StageVersion
type StageVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StageVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StageVersion{}, &StageVersionList{})
}
