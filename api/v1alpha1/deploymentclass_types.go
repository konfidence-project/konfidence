package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DeploymentClassKind is kind of the DeploymentClass resource.
	DeploymentClassKind = "DeploymentClass"
)

// DeploymentClassSpec defines the desired state of DeploymentClass. It is immutable because changing it requires
// transfer of deployment ownership to a different controller. This process is currently not well-supported and
// is therefor not recommended.
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="DeploymentClass spec is immutable"
type DeploymentClassSpec struct {
	// Controller is the name of the controller that implements this deployment class.
	// This identifies which operator/controller is responsible for reconciling
	// resources of this deployment class (e.g., "kubernetes-landscape-orchestrator").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Controller string `json:"controller"`
}

//nolint:lll // Kubebuilder annotations are intentionally long.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories=konfidence;kden
// +kubebuilder:printcolumn:name="Controller",type=string,JSONPath=`.spec.controller`,description="Controller name"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DeploymentClass declares a deployment capability provided by a deployer (controller). It is a cluster-scoped resource
// installed by deployers to advertise their capabilities. Its immutable spec ensures that ownership of resources using
// the class cannot change on the fly. Its metadata.name is the deployment class identifier referenced by
// ArtifactDeployments and DeploymentTargets. The name must be unique across all DeploymentClasses in the cluster and
// should follow the pattern `<class-name>.<vendor-domain>` (e.g., `helm.konfidence.cloud`).
type DeploymentClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DeploymentClassSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// DeploymentClassList contains a list of DeploymentClass.
type DeploymentClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeploymentClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DeploymentClass{}, &DeploymentClassList{})
}
