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
	// VectorMigrationKind is the kind for VectorMigration resources.
	VectorMigrationKind = "VectorMigration"

	// VectorMigrationFailed indicates that the vectorMigration failed.
	VectorMigrationFailed = "VectorMigrationFailed"
	// VectorMigrationSucceeded indicates that the vectorMigration reconciled successfully.
	VectorMigrationSucceeded = "VectorMigrationSucceeded"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VectorMigrationSpec defines the desired state of VectorMigration
type VectorMigrationSpec struct {
	StageVersion string `json:"stageVersion"`

	// Vector points to the OCM component version that contains the deployment vector for this stage.
	Vector string `json:"vector"`
}

// VectorMigrationStatus defines the observed state of VectorMigration.
type VectorMigrationStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VectorMigration is the Schema for the vectormigrations API
type VectorMigration struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of VectorMigration
	// +required
	Spec VectorMigrationSpec `json:"spec"`

	// status defines the observed state of VectorMigration
	// +optional
	Status VectorMigrationStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// VectorMigrationList contains a list of VectorMigration
type VectorMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorMigration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorMigration{}, &VectorMigrationList{})
}
