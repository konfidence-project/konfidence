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

// VectorDeploymentSpec defines the desired state of a VectorDeployment.
//
// A VectorDeployment references a deployment vector stored as an OCM ComponentVersion in an OCI registry. The vector
// describes a complete, immutable set of artifacts and versions that should be deployed as a unit.
//
// The value must always be a fully qualified OCI URL and must resolve to a valid OCM ComponentVersion. The
// VectorDeployment spec is intended to be immutable. Any substantive change should result in a new VectorDeployment
// instance rather than updating an existing one.
type VectorDeploymentSpec struct {
	// Vector is a fully qualified URL pointing to an OCM ComponentVersion stored in an OCI registry. The referenced
	// component contains the deployment vector, which includes the complete list of artifacts and their versions.
	Vector string `json:"vector"`
}

// VectorDeploymentStatus represents the observed state of a VectorDeployment as it progresses through the
// deployment lifecycle.
//
// The lifecycle consists of:
//  1. Pulling the vector from the OCI registry and parsing its contents -> VectorDownloadedCondition
//  2. Creating (or re-using) one ArtifactDeployment per artifact in the vector -> ArtifactDeploymentsCreatedCondition
//  3. Waiting until all ArtifactDeployments have successfully deployed -> VectorDeployedCondition
//  4. Creating all VectorAssignment resources associated with this vector -> VectorAssignmentsCreatedCondition
//  5. Marking the vector as ready for use -> VectorReadyCondition
type VectorDeploymentStatus struct {

	// Conditions represents the current set of status conditions for this vector
	// deployment. These conditions track progress through the lifecycle stages.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResolvedVectorOcm contains the fully materialized content of the OCM ComponentVersion after it has been
	// downloaded and resolved from the OCI registry. Unlike the Spec.Vector value, which is only a reference (URL),
	// this field stores the actual resolved vector content as provided by OCM, including all artifacts and metadata.
	// It is not a reference but the inlined representation of the component version at reconciliation time.
	ResolvedVectorOcm string `json:"resolvedVectorOcm,omitempty"`

	// ResultingArtifactDeployments lists the ArtifactDeployment resources created (or re-used) for this vector. The
	// map key is the component name of the artifact as defined inside the vector. Keys remain stable across
	// reconciliations and re-creations.
	ResultingArtifactDeployments map[string]LocalArtifactDeploymentReference `json:"resultingArtifactDeployments,omitempty"`

	// ResultingVectorAssignments lists all VectorAssignment resources created for this vector. VectorAssignments are
	// not re-used like ArtifactDeployments, but instead each VectorDeployment results in a complete new set of
	// assignments.
	//
	// The map key is the component name of the artifact. Keys are stable across reconcilations.
	ResultingVectorAssignments map[string]LocalVectorAssignmentReference `json:"resultingVectorAssignments,omitempty"`

	// DeploymentResults exposes an aggregated view of the deployment results produced
	// by all underlying ArtifactDeployments. The map key is composed of the component
	// name and the individual result name, ensuring uniqueness.
	DeploymentResults map[string]DeploymentResult `json:"deploymentResults,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"

// VectorDeployment is the Schema for the vectordeployments API.
//
// VectorDeployment represents the deployment of an immutable vector of artifacts into a specific environment or stage.
type VectorDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="VectorDeployment spec is immutable after it has been set"
	// Spec defines the desired state of the VectorDeployment and is immutable after it has been set
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
