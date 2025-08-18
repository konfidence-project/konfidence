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
	StageVersionKind = "StageVersion"

	// FetchFailedCondition indicates an fetch failure of another resource.
	FetchFailedCondition string = "FetchFailed"

	// VectorDeploymentCreatedCondition indicates that the VectorDeployment resource has been created successfully.
	VectorDeploymentCreatedCondition string = "VectorDeploymentCreated"

	// VectorMigrationCreatedCondition indicates that the VectorMigration resource has been created successfully.
	VectorMigrationCreatedCondition string = "VectorMigrationCreated"
	
	// VectorMigratedCondition indicates that the migration tasks for the vector have been completed successfully.
	VectorMigratedCondition = "VectorMigrated"

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
	StageGeneration int64 `json:"stage_generation"`
}

// StageVersionStatus defines the observed state of StageVersion.
type StageVersionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StageVersion is the Schema for the stageversions API
type StageVersion struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of StageVersion
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
