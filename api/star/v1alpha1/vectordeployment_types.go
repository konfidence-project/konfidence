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

	// VectorConfigCommittedCondition indicates that the vector-scoped configuration ConfigMap has been written to the
	// landscape namespace and reflects both the optional authored configuration carried in the vector OCM
	// ComponentVersion and the aggregated DeploymentResults of the underlying ArtifactDeployments. The condition is set
	// to True even when neither authored data nor deployment results are present, so that downstream gating can be
	// expressed without special-casing the empty case.
	VectorConfigCommittedCondition = "VectorConfigCommitted"

	// VectorReadyCondition indicates that the vector deployment is ready for use.
	VectorReadyCondition = "Ready"

	// VectorDataFinalizer guards the vector-scoped configuration ConfigMap that the deployment controller writes
	// during the VectorConfigCommitted phase. The controller adds this finalizer on first reconciliation and removes
	// it only after the corresponding ConfigMap has been explicitly deleted, so that teardown happens deterministically
	// even when the controller-owner reference cascade is delayed or unobservable.
	VectorDataFinalizer = "konfidence.cloud/vector-data-cleanup"
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
//  5. Writing the vector-scoped configuration ConfigMap into the landscape namespace -> VectorConfigCommittedCondition
//  6. Marking the vector as ready for use -> VectorReadyCondition
type VectorDeploymentStatus struct {

	// Conditions represents the current set of status conditions for this vector
	// deployment. These conditions track progress through the lifecycle stages.
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResolvedVectorOcm contains the fully materialized content of the OCM ComponentVersion after it has been
	// downloaded and resolved from the OCI registry. Unlike the Spec.Vector value, which is only a reference (URL),
	// this field stores the actual resolved vector content as provided by OCM, including all artifacts and metadata.
	// It is not a reference but the inlined representation of the component version at reconciliation time.
	ResolvedVectorOcm string `json:"resolvedVectorOcm,omitempty"`

	// ResolvedVectorConfig contains the raw bytes of the optional vector-scoped configuration resource carried in the
	// vector OCM ComponentVersion (the singleton resource named "cloud-konfidence-vector-config" produced by the
	// galaxy assembly side). The field is empty when the vector does not declare such a resource. Persisted on the
	// status to avoid re-fetching the blob from OCM on every reconciliation. The value is fixed for the lifetime of
	// the VectorDeployment because the referenced vector is immutable.
	ResolvedVectorConfig string `json:"resolvedVectorConfig,omitempty"`

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

// VectorDeployment is the Schema for the vectordeployments API.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.spec) || has(self.spec)", message="Spec is required once set"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=".spec.vector",description="The deployment vector"
// +kubebuilder:printcolumn:name="Vector-Ready",type=string,JSONPath=".status.conditions[?(@.type==\"VectorReady\")].status",description="Indicates if the vector is ready"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"VectorReady\")].reason",description="The reason of the VectorReady condition"
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=".status.conditions[?(@.type==\"VectorReady\")].message",description="The message of the VectorReady condition"
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
