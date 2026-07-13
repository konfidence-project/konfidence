package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StageKind is kind of the Stage resource.
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

	// StageReady indicates that the stage is ready for use. Same as VectorActivatedCondition.
	StageReady = "Ready"
)

// StageSpec defines the desired state of Stage.
type StageSpec struct {

	// Vector points to the OCM component version that contains the deployment vector for this stage.
	Vector string `json:"vector"`
}

// StageStatus defines the observed state of Stage.
type StageStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	VectorHistory             []string                    `json:"vectorHistory,omitempty"`
	LatestVectorDeploymentRef corev1.TypedObjectReference `json:"latestVectorDeploymentRef,omitempty"`
}

// Stage is the Schema for the stages API.
//
// +kubebuilder:object:root=true
// +kubebuilder:resource:categories=konfidence;kden
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Ready status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Age"
// +kubebuilder:printcolumn:name="Vector",type=string,JSONPath=`.spec.vector`
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
