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
	// TaskExecutionKind is the kind for TaskExecution resources.
	TaskExecutionKind = "TaskExecution"

	// TaskPending indicates that the task has been created and wait for Execution Controller to be executed.
	TaskPending = "TaskPending"

	// TaskInProgress indicates that the task has been started.
	TaskInProgress = "TaskInProgress"

	// TaskFailed indicates that the task execution results in errors.
	TaskFailed = "TaskFailed"

	// TaskSucceeded indicates that the task execution was successful.
	TaskSucceeded = "TaskSucceeded"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TaskExecutionSpec defines the desired state of TaskExecution
type TaskExecutionSpec struct {
	Name      string               `json:"name"`
	Type      string               `json:"type"`
	DependsOn []string             `json:"dependsOn,omitempty"`
	Spec      runtime.RawExtension `json:"spec"`
}

// TaskExecutionStatus defines the observed state of TaskExecution.
type TaskExecutionStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// TaskExecution is the Schema for the taskexecutions API
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=".spec.type",description="The type of the task execution"
// +kubebuilder:printcolumn:name="Condition",type=string,JSONPath=".status.conditions[-1:].type",description="The latest condition type"
// +kubebuilder:printcolumn:name="Condition-Status",type=string,JSONPath=".status.conditions[-1:].status",description="The latest condition status"
type TaskExecution struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of TaskExecution
	// +required
	Spec TaskExecutionSpec `json:"spec"`

	// status defines the observed state of TaskExecution
	// +optional
	Status TaskExecutionStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// TaskExecutionList contains a list of TaskExecution
type TaskExecutionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TaskExecution `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TaskExecution{}, &TaskExecutionList{})
}
