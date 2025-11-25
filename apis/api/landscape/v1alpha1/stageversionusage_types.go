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
	// StageVersionUsageKind is the kind for StageVersionUsage resources.
	StageVersionUsageKind = "StageVersionUsage"

	// StageVersionNotFound indicates that the referenced stage version does not exist.
	StageVersionNotFound = "StageVersionNotFound"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:validation:ExactlyOneOf=stageVersionRef;stageVersionSelector

// StageVersionUsageSpec defines the desired state of StageVersionUsage
type StageVersionUsageSpec struct {
	// Reason is human-readable description of why this StageVersion is in use, e.g. "executing vector migrations", "latest vector for stage xyz",
	// +optional
	Reason string `json:"reason,omitempty"`

	// StageVersionRef references a stageVersion
	// +optional
	StageVersionRef *StageVersionReference `json:"stageVersionRef,omitempty"`

	// StageVersionSelector is a label selector to find a StageVersion when name is not provided.
	// +optional
	StageVersionSelector *metav1.LabelSelector `json:"stageVersionSelector,omitempty"`
}

// StageVersionUsageStatus defines the observed state of StageVersionUsage.
type StageVersionUsageStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResolvedStageVersions contains the names of all resolved stageVersion resources specified by either stageVersionRef or StageVersionSelector
	ResolvedStageVersions []string `json:"resolvedStageVersions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StageVersionUsage is the Schema for the stageversionusages API
type StageVersionUsage struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of StageVersionUsage
	// +required
	Spec StageVersionUsageSpec `json:"spec"`

	// status defines the observed state of StageVersionUsage
	// +optional
	Status StageVersionUsageStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// StageVersionUsageList contains a list of StageVersionUsage
type StageVersionUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StageVersionUsage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StageVersionUsage{}, &StageVersionUsageList{})
}
