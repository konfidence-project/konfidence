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
	// VectorDeploymentUsageKind is the kind of the VectorDeploymentUsage resource.
	VectorDeploymentUsageKind = "VectorDeploymentUsage"

	// VectorDeploymentAssignedCondition indicates that a VectorDeployment has been assigned to the usage.
	VectorDeploymentAssignedCondition = "VectorDeploymentAssigned"
)

// VectorDeploymentUsageSpec defines the desired state of VectorDeploymentUsage.
type VectorDeploymentUsageSpec struct {
	// VectorRef points to the OCM component version that contains the deployment vector for this stage.
	VectorRef corev1.TypedLocalObjectReference `json:"vectorRef"`
}

// VectorDeploymentUsageStatus defines the observed state of VectorDeploymentUsage.
type VectorDeploymentUsageStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	VectorDeploymentRef corev1.TypedLocalObjectReference `json:"vectorDeployment,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=`.spec.vector`

// VectorDeploymentUsage is the Schema for the vectordeploymentusages API.
type VectorDeploymentUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorDeploymentUsageSpec   `json:"spec,omitempty"`
	Status VectorDeploymentUsageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorDeploymentUsageList contains a list of VectorDeploymentUsage.
type VectorDeploymentUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorDeploymentUsage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorDeploymentUsage{}, &VectorDeploymentUsageList{})
}
