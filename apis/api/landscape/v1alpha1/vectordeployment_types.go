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
	// VectorDeploymentKind is the kind of the VectorDeployment resource.
	VectorDeploymentKind = "VectorDeployment"

	// VectorDownloadedCondition indicates that the vector has been successfully downloaded from the OCI repository.
	VectorDownloadedCondition = "VectorDownloaded"

	// ArtifactDeploymentsCreatedCondition indicates that all ArtifactDeployment resources have been created successfully.
	ArtifactDeploymentsCreatedCondition = "ArtifactDeploymentsCreated"

	// VectorDeployedCondition indicates that all artifacts of the vector have been successfully deployed.
	VectorDeployedCondition = "VectorDeployed"

	// VectorAssignmentsCreatedCondition indicates that all VectorAssignment resources have been created successfully.
	VectorAssignmentsCreatedCondition = "VectorAssignmentsCreated"

	// VectorReadyCondition indicates that the vector deployment is ready for use.
	VectorReadyCondition = "VectorReady"
)

// VectorDeploymentSpec defines the desired state of VectorDeployment.
type VectorDeploymentSpec struct {
	// Vector points to the OCM component version that contains the deployment vector for this stage.
	Vector string `json:"vector"`
}

// VectorDeploymentStatus defines the observed state of VectorDeployment.
type VectorDeploymentStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VectorDeployment is the Schema for the vectordeployments API.
type VectorDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorDeploymentSpec   `json:"spec,omitempty"`
	Status VectorDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorDeploymentList contains a list of VectorDeployment.
type VectorDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorDeployment{}, &VectorDeploymentList{})
}
