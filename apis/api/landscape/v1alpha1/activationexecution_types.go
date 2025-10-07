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
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ActivationExecutionKind is the kind for ActivationExecution resources.
	ActivationExecutionKind = "ActivationExecution"

	// ActivationExecutionPending indicates that the execution has been created and wait for Execution Controller to be executed.
	ActivationExecutionPending = "ActivationExecutionPending"

	// ActivationExecutionInProgress indicates that the execution has been started.
	ActivationExecutionInProgress = "ActivationExecutionInProgress"

	// ActivationExecutionFailed indicates that the execution results in errors.
	ActivationExecutionFailed = "ActivationExecutionFailed"

	// ActivationExecutionSucceeded indicates that the execution was successful.
	ActivationExecutionSucceeded = "ActivationExecutionSucceeded"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActivationExecutionSpec defines the desired state of ActivationExecution
type ActivationExecutionSpec struct {
	Name string               `json:"name"`
	Type string               `json:"type"`
	Spec runtime.RawExtension `json:"spec"`
}

// ActivationExecutionStatus defines the observed state of ActivationExecution.
type ActivationExecutionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ActivationExecution is the Schema for the activationexecutions API
type ActivationExecution struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ActivationExecution
	// +required
	Spec ActivationExecutionSpec `json:"spec"`

	// status defines the observed state of ActivationExecution
	// +optional
	Status ActivationExecutionStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ActivationExecutionList contains a list of ActivationExecution
type ActivationExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActivationExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActivationExecution{}, &ActivationExecutionList{})
}
