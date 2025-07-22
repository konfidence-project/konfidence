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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StageKind = "Stage"

	// TODO use condition resolvers to automatically set these conditions based on their relationships?

	// FetchFailedCondition indicates an fetch failure of another resource.
	FetchFailedCondition string = "FetchFailed"

	// VectorDeploymentCreatedCondition indicates that the VectorDeployment resource has been created successfully.
	VectorDeploymentCreatedCondition string = "VectorDeploymentCreated"

	// VectorDeployedCondition indicates that all artifacts of the vector have been successfully deployed
	// and assigned in the stage.
	VectorDeployedCondition = "VectorDeployed"

	// VectorMigratedCondition indicates that the migration tasks for the vector have been completed successfully.
	VectorMigratedCondition = "VectorMigrated"

	// VectorActivatedCondition indicates that the vector has been activated in the stage.
	VectorActivatedCondition = "VectorActivated"

	// StageReady indicates that the stage is ready for use. Same as VectorActivatedCondition.
	StageReady = "Ready"
)

// StageSpec defines the desired state of Stage.
type StageSpec struct {
	// Name is the name of the stage.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Vector points to the OCM component version that contains the deployment vector for this stage.
	Vector string `json:"vector"`
}

// StageStatus defines the observed state of Stage.
type StageStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	VectorHistory             []string                    `json:"vectorHistory,omitempty"`
	LatestVectorDeploymentRef corev1.TypedObjectReference `json:"latestVectorDeploymentRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=`.spec.vector`

// Stage is the Schema for the stages API.
type Stage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StageSpec   `json:"spec,omitempty"`
	Status StageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StageList contains a list of Stage.
type StageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Stage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Stage{}, &StageList{})
}
