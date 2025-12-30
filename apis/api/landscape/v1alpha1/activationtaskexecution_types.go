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
	// ActivationTaskExecutionKind is the kind for ActivationTaskExecution resources.
	ActivationTaskExecutionKind = "ActivationTaskExecution"

	// ActivationTaskExecutionPending indicates that the execution has been created and wait for Execution Controller to be executed.
	ActivationTaskExecutionPending = "Pending"

	// ActivationTaskExecutionInProgress indicates that the execution has been started.
	ActivationTaskExecutionInProgress = "InProgress"

	// ActivationTaskExecutionFailed indicates that the execution results in errors.
	ActivationTaskExecutionFailed = "Failed"

	// ActivationTaskExecutionSucceeded indicates that the execution was successful.
	ActivationTaskExecutionSucceeded = "Succeeded"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ActivationTaskExecutionSpec defines the desired state of ActivationTaskExecution
type ActivationTaskExecutionSpec struct {
	Type string               `json:"type"`
	Spec runtime.RawExtension `json:"spec"`

	// VectorActivation is a temporary field that contains the name of the associated vectorActivation
	VectorActivation string `json:"vectorActivation"`
}

// ActivationTaskExecutionStatus defines the observed state of ActivationTaskExecution.
type ActivationTaskExecutionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ActivationTaskExecution is the Schema for the ActivationTaskExecutions API
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type",description="The type of the task execution"
// +kubebuilder:printcolumn:name="Vector-Activation",type=string,JSONPath=".spec.vectorActivation",description="The associated VectorActivation"
// +kubebuilder:printcolumn:name="Condition",type=string,JSONPath=".status.conditions[-1:].type",description="The latest condition type"
// +kubebuilder:printcolumn:name="Condition-Status",type=string,JSONPath=".status.conditions[-1:].status",description="The latest condition status"
type ActivationTaskExecution struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of ActivationTaskExecution
	// +required
	Spec ActivationTaskExecutionSpec `json:"spec"`

	// status defines the observed state of ActivationTaskExecution
	// +optional
	Status ActivationTaskExecutionStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ActivationTaskExecutionList contains a list of ActivationTaskExecution
type ActivationTaskExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActivationTaskExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ActivationTaskExecution{}, &ActivationTaskExecutionList{})
}
