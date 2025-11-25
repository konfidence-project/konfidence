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
	// VectorAssignedCondition indicates that the deployer has successfully processed the VectorAssignment. This usually
	// means that any assignment-specific configuration such as routing configuration has been created or updated.
	//
	// A VectorAssignment is considered complete once this condition is True.
	VectorAssignedCondition = "VectorAssigned"
)

// VectorAssignmentSpec defines the desired state of a VectorAssignment.
//
// A VectorAssignment represents one logical binding between a VectorDeployment and an ArtifactDeployment. Since a
// single artifact may be reused across multiple vectors, an n:m relationship exists between vectors and artifacts.
// VectorAssignment creates a concrete instance of that relationship.
//
// VectorAssignment resources are created automatically during vector rollouts and are typically not authored by users.
// Deployer implementations reconcile the VectorAssignment to perform vector-specific configuration based on the
// artifact selected for this vector.
//
// The VectorAssignmentSpec is immutable. If an artifact is replaced or added to a different vector, the old
// VectorAssignment is deleted and a new one created.
type VectorAssignmentSpec struct {
	// Manifest contains the ArtifactManifest describing the artifact to be assigned to the vector. This duplicates the
	// manifest stored in the ArtifactDeployment for efficiency: deployers often need to filter or select assignments
	// by artifact type, and embedding the manifest avoids repeated API lookups.
	Manifest ArtifactManifest `json:"manifest"`

	// ArtifactDeploymentRef references the ArtifactDeployment instance that is associated with the vector. The
	// referenced artifact must exist in the same namespace as this VectorAssignment.
	ArtifactDeploymentRef LocalArtifactDeploymentReference `json:"artifactDeploymentRef"`

	// VectorDeploymentRef references the VectorDeployment that this artifact is assigned to. This creates the explicit
	// mapping "artifact X belongs to vector Y".
	VectorDeploymentRef LocalVectorDeploymentReference `json:"vectorDeploymentRef"`
}

// VectorAssignmentStatus defines the observed state of a VectorAssignment.
//
// A VectorAssignment progresses through a simple lifecycle driven by the deployer:
//
//  1. VectorAssignment is created by the vector-deployment-controller.
//  2. deployer reconciles it and configures vector-specific integration
//  3. VectorAssignedCondition is set to True
type VectorAssignmentStatus struct {
	// Conditions describes the latest observed state of the assignment. The primary condition is
	// VectorAssignedCondition, which becomes True once the deployer has finished processing the VectorAssignment.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VectorAssignment is the Schema for the vectorassignments API.
//
// A VectorAssignment represents a single binding between a VectorDeployment and an ArtifactDeployment. It enables
// an n:m mapping where a single artifact may be reused across multiple vectors. These objects are automatically
// managed by the vector-deployment-controller and reconciled by deployers to apply vector-specific configuration.
type VectorAssignment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorAssignmentSpec   `json:"spec,omitempty"`
	Status VectorAssignmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VectorAssignmentList contains a list of VectorAssignment.
type VectorAssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorAssignment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorAssignment{}, &VectorAssignmentList{})
}
