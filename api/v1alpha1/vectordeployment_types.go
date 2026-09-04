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

	// VectorAssignmentsCreatedCondition indicates that all VectorAssignment resources have been created successfully.
	VectorAssignmentsCreatedCondition = "VectorAssignmentsCreated"

	// VectorDataCreatedCondition indicates that the vector deployment controller has created the VectorData CR. Materialisation of the data
	// (e.g. as a ConfigMap on Kubernetes) is reported separately on the VectorData object and reflected on the
	// parent VectorDeployment through VectorReadyCondition.
	VectorDataCreatedCondition = "VectorDataCreated"

	// VectorReadyCondition indicates that the vector deployment is ready for use.
	VectorReadyCondition = "Ready"
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
//  5. Creating the VectorData CR with the resolved authored configuration + aggregated DeploymentResults; the
//     runtime-specific implementor then materialises it (e.g. as a ConfigMap on Kubernetes) -> VectorDataCreatedCondition
//  6. Marking the vector as ready for use once VectorData reports its own Ready=True -> VectorReadyCondition
type VectorDeploymentStatus struct {

	// Conditions represents the current set of status conditions for this vector
	// deployment. These conditions track progress through the lifecycle stages.
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResolvedVectorOcm contains the fully materialized content of the OCM ComponentVersion after it has been
	// downloaded and resolved from the OCI registry. Unlike the Spec.Vector value, which is only a reference (URL),
	// this field stores the actual resolved vector content as provided by OCM, including all artifacts and metadata.
	// It is not a reference but the inlined representation of the component version at reconciliation time.
	ResolvedVectorOcm string `json:"resolvedVectorOcm,omitempty"`

	// ResultingVectorData records the name of the VectorData object created for this VectorDeployment. The VectorData
	// CR is the contract between the vector deployment controller (which resolves the OCM payload) and the runtime-specific implementor
	// (which materialises it on the target runtime). The field is empty until step 5 of the lifecycle has produced the
	// CR. Names are stable across reconciliations.
	ResultingVectorData *LocalObjectReference `json:"resultingVectorData,omitempty"`

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
	// by all underlying ArtifactDeployments. The map key is the artifact component name;
	// the value lists every result emitted by that ArtifactDeployment. Within a component's
	// list, results are unique by (name, type).
	// +kubebuilder:validation:MaxProperties=64
	// +kubebuilder:validation:XValidation:rule="self.all(k,self[k].all(a,self[k].exists_one(b,b.name==a.name&&b.type==a.type)))"
	DeploymentResults map[string]ComponentDeploymentResults `json:"deploymentResults,omitempty"`
}

// LocalObjectReference references an object by name within the same namespace as the parent.
type LocalObjectReference struct {
	// Name of the referenced object.
	Name string `json:"name"`
}

// VectorDeployment is the Schema for the vectordeployments API.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=".spec.vector",description="The deployment vector"
// +kubebuilder:printcolumn:name="Vector-Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status",description="Indicates if the vector is ready"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason",description="The reason of the Ready condition"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].message",description="The message of the Ready condition"
//
// VectorDeployment represents the deployment of an immutable vector of artifacts into a specific environment or stage.
//
//nolint:lll // Kubebuilder annotations are intentionally long.
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
